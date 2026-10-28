package prices

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
)

type pricePoolFixture struct {
	coins    []common.Address
	balances []*big.Int
	supply   *big.Int
	meta     bool
}

type contractPriceFixture struct {
	mu          sync.Mutex
	t           *testing.T
	methods     abi.ABI
	timestamp   uint64
	feeds       map[common.Address]common.Address
	answers     map[common.Address]*big.Int
	updated     map[common.Address]uint64
	precision   map[common.Address]uint8
	assets      map[common.Address]common.Address
	conversions map[common.Address]*big.Int
	pools       map[common.Address]pricePoolFixture
	calls       map[string]int
	badRound    bool
	fail        string
}

func priceAddress(n int64) common.Address { return common.BigToAddress(big.NewInt(n)) }

func newContractPriceFixture(t *testing.T) (*TokenPriceService, *contractPriceFixture, *MemoryStore, Snapshot) {
	t.Helper()
	methods, err := abi.JSON(strings.NewReader(contractPriceABI))
	if err != nil {
		t.Fatal(err)
	}
	f := &contractPriceFixture{t: t, methods: methods, timestamp: uint64(time.Now().Unix()), feeds: make(map[common.Address]common.Address), answers: make(map[common.Address]*big.Int), updated: make(map[common.Address]uint64), precision: make(map[common.Address]uint8), assets: make(map[common.Address]common.Address), conversions: make(map[common.Address]*big.Int), pools: make(map[common.Address]pricePoolFixture), calls: make(map[string]int)}
	store := NewMemoryStore()
	return NewTokenPriceService(store, NewContractSource(store)), f, store, Snapshot{Caller: f, Block: 12345, Timestamp: f.timestamp}
}

func (f *contractPriceFixture) addFeed(token common.Address, raw int64, precision uint8) common.Address {
	feed := priceAddress(int64(len(f.feeds) + 10000))
	f.feeds[token], f.answers[feed], f.precision[feed], f.updated[feed] = feed, big.NewInt(raw), precision, f.timestamp-60
	return feed
}

func (f *contractPriceFixture) CallContract(_ context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if block == nil || block.Uint64() != 12345 {
		f.t.Fatal("price query did not use the selected block")
	}
	m, err := f.methods.MethodById(call.Data[:4])
	if err != nil {
		return nil, err
	}
	args, err := m.Inputs.Unpack(call.Data[4:])
	if err != nil {
		return nil, err
	}
	target := *call.To
	f.calls[target.Hex()+":"+m.Name]++
	if f.fail == m.Name {
		return nil, errors.New("RPC transport failure")
	}
	switch m.Name {
	case "getFeed":
		if target != feedRegistry || args[1].(common.Address) != usdDenomination {
			f.t.Fatal("wrong USD registry lookup")
		}
		feed := f.feeds[args[0].(common.Address)]
		if feed == (common.Address{}) {
			return nil, errors.New("execution reverted: Feed not found")
		}
		return m.Outputs.Pack(feed)
	case "latestRoundData":
		answered := int64(5)
		if f.badRound {
			answered = 4
		}
		return m.Outputs.Pack(big.NewInt(5), f.answers[target], new(big.Int).SetUint64(f.updated[target]), new(big.Int).SetUint64(f.updated[target]), big.NewInt(answered))
	case "decimals":
		d, ok := f.precision[target]
		if !ok {
			return nil, errors.New("decimals unavailable")
		}
		return m.Outputs.Pack(d)
	case "get_implementation_address":
		if target != curveNGFactory {
			f.t.Fatal("unverified Curve factory")
		}
		impl := common.Address{}
		if _, ok := f.pools[args[0].(common.Address)]; ok {
			impl = priceAddress(9999)
		}
		return m.Outputs.Pack(impl)
	case "is_meta":
		return m.Outputs.Pack(f.pools[args[0].(common.Address)].meta)
	case "asset":
		asset, ok := f.assets[target]
		if !ok {
			return nil, errors.New("execution reverted")
		}
		return m.Outputs.Pack(asset)
	case "convertToAssets":
		unit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(f.precision[target])), nil)
		if args[0].(*big.Int).Cmp(unit) != 0 {
			f.t.Fatal("ERC4626 conversion used the wrong share unit")
		}
		return m.Outputs.Pack(f.conversions[target])
	case "N_COINS":
		return m.Outputs.Pack(big.NewInt(int64(len(f.pools[target].coins))))
	case "coins":
		return m.Outputs.Pack(f.pools[target].coins[args[0].(*big.Int).Int64()])
	case "balances":
		return m.Outputs.Pack(f.pools[target].balances[args[0].(*big.Int).Int64()])
	case "totalSupply":
		return m.Outputs.Pack(f.pools[target].supply)
	}
	return nil, errors.New("unexpected contract price method")
}

type contractCallerFunc func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)

func (f contractCallerFunc) CallContract(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	return f(ctx, call, block)
}

func repeatedLPFixture(t *testing.T, count int, missing bool) (*TokenPriceService, *contractPriceFixture, *MemoryStore, Snapshot, []common.Address, common.Address) {
	t.Helper()
	s, f, store, snapshot := newContractPriceFixture(t)
	known, component := priceAddress(1), priceAddress(2)
	f.addFeed(known, 102000000, 8)
	f.precision[known], f.precision[component] = 6, 6
	if !missing {
		f.addFeed(component, 98000000, 8)
	}
	var tokens []common.Address
	for i := 0; i < count; i++ {
		token := priceAddress(int64(100 + i))
		tokens = append(tokens, token)
		f.precision[token] = 18
		f.pools[token] = pricePoolFixture{coins: []common.Address{known, component}, balances: []*big.Int{big.NewInt(50000000), big.NewInt(50000000)}, supply: decimal.RequireFromString("10000000000000000000").BigInt()}
	}
	return s, f, store, snapshot, tokens, component
}

func TestParallelPricesPreserveLPValuesAndDeduplicateMissingComponents(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "priced", true: "missing-anchor"}[missing], func(t *testing.T) {
			old, oldRPC, _, snapshot, tokens, _ := repeatedLPFixture(t, 20, missing)
			// Reproduce the previous sequential resolver without the RPC cache.
			r := &contractResolver{source: old.source.(*ContractSource), ctx: context.Background(), snapshot: snapshot, memo: make(map[common.Address]Quote), visiting: make(map[common.Address]bool), decimals: make(map[common.Address]uint8)}
			expected := make(map[common.Address]Quote)
			for _, token := range tokens {
				q, err := r.price(token, 0)
				if missing && err == nil || !missing && err != nil {
					t.Fatalf("unexpected baseline price: %s %v", token, err)
				}
				if err == nil {
					expected[token] = q
				}
			}
			s, f, _, parallel, tokens, component := repeatedLPFixture(t, 20, missing)
			parallel.Workers = 8
			quotes, err := s.source.(*ContractSource).FetchAt(context.Background(), 1, tokens, parallel)
			if missing && err == nil || !missing && err != nil || len(quotes) != len(expected) {
				t.Fatalf("parallel result differs: quotes=%d expected=%d err=%v", len(quotes), len(expected), err)
			}
			for _, token := range tokens {
				got, found := quotes[token]
				want, priced := expected[token]
				if found != priced || found && (!got.USD.Equal(want.USD) || got.ValidUntil.Sub(got.UpdatedAt) != want.ValidUntil.Sub(want.UpdatedAt)) {
					t.Fatalf("LP price/deadline changed: token=%s got=%+v want=%+v", token, got, want)
				}
			}
			if missing {
				if f.calls[component.Hex()+":asset"] != 1 || f.calls[feedRegistry.Hex()+":getFeed"] != len(tokens)+2 {
					t.Fatalf("missing component probed repeatedly: %+v", f.calls)
				}
				if totalPriceCalls(f) >= totalPriceCalls(oldRPC) {
					t.Fatal("negative probes were not deduplicated")
				}
			}
			t.Logf("20 LP tokens (missing anchor=%t): RPC before=%d after=%d", missing, totalPriceCalls(oldRPC), totalPriceCalls(f))
		})
	}
}

func TestCold359TokenPricesUseAtMostEightConcurrentDirectCalls(t *testing.T) {
	s, f, _, snapshot := newContractPriceFixture(t)
	var tokens []common.Address
	for i := int64(1); i <= 359; i++ {
		token := priceAddress(i)
		tokens = append(tokens, token)
		f.addFeed(token, 100000000+i, 8)
	}
	started, release := make(chan struct{}, 8), make(chan struct{})
	var active, peak atomic.Int32
	snapshot.Workers = 8
	snapshot.Caller = contractCallerFunc(func(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		if n > 8 {
			t.Error("more than eight price RPC calls active")
		}
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if call.From != (common.Address{}) || call.To == nil || *call.To == common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11") {
			t.Error("price reads changed oracle caller semantics")
		}
		return f.CallContract(ctx, call, block)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.EnsurePricesAt(ctx, 1, tokens, snapshot) }()
	for range 8 {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("price calls remained sequential")
		}
	}
	close(release)
	if err := <-done; err != nil || active.Load() != 0 || peak.Load() != 8 {
		t.Fatalf("parallel prices failed or left active calls: err=%v active=%d peak=%d", err, active.Load(), peak.Load())
	}
	for i, token := range tokens {
		q, err := s.GetTokenPrice(context.Background(), token, 1)
		if err != nil || !q.Equal(decimal.NewFromInt(100000001+int64(i)).Shift(-8)) {
			t.Fatalf("cold token price changed: %s %s %v", token, q, err)
		}
	}
	before := totalPriceCalls(f)
	if err := s.EnsurePricesAt(context.Background(), 1, tokens, Snapshot{}); err != nil || totalPriceCalls(f) != before {
		t.Fatal("fresh prices were fetched again")
	}
	t.Logf("359 cold USD anchors: RPC=%d peak concurrency=%d", before, peak.Load())
}

func TestPriceWorkersCancelAndJoinOnProviderFailureOrRequestCancellation(t *testing.T) {
	for _, status := range []int{0, http.StatusPaymentRequired, http.StatusTooManyRequests} {
		t.Run(map[int]string{0: "canceled", 402: "quota", 429: "rate-limit"}[status], func(t *testing.T) {
			s, _, _, snapshot := newContractPriceFixture(t)
			snapshot.Workers = 8
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			started := make(chan struct{}, 8)
			var active, attempts atomic.Int32
			snapshot.Caller = contractCallerFunc(func(ctx context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
				active.Add(1)
				defer active.Add(-1)
				attempts.Add(1)
				select {
				case started <- struct{}{}:
				default:
				}
				if status != 0 {
					return nil, rpc.HTTPError{StatusCode: status, Status: "quota exhausted"}
				}
				<-ctx.Done()
				return nil, ctx.Err()
			})
			var tokens []common.Address
			for i := int64(1); i <= 30; i++ {
				tokens = append(tokens, priceAddress(i))
			}
			done := make(chan error, 1)
			go func() { done <- s.EnsurePricesAt(ctx, 1, tokens, snapshot) }()
			if status == 0 {
				for range 8 {
					select {
					case <-started:
					case <-ctx.Done():
						t.Fatal("price workers did not start")
					}
				}
				cancel()
			}
			select {
			case err := <-done:
				if active.Load() != 0 || attempts.Load() > 8 {
					t.Fatalf("failed calls retried/workers left active: active=%d attempts=%d", active.Load(), attempts.Load())
				}
				if status == 0 && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation lost: %v", err)
				}
				var httpErr rpc.HTTPError
				if status != 0 && (!errors.Is(err, ErrPriceRPCUnavailable) || !errors.As(err, &httpErr) || httpErr.StatusCode != status) {
					t.Fatalf("provider failure became missing token price: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("price workers did not stop")
			}
		})
	}
}

func TestFailedContractProbeIsRetriedInTheNextFetch(t *testing.T) {
	s, f, _, snapshot := newContractPriceFixture(t)
	token := priceAddress(1)
	if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{token}, snapshot); err == nil {
		t.Fatal("unknown token unexpectedly priced")
	}
	f.addFeed(token, 123000000, 8)
	if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{token}, snapshot); err != nil {
		t.Fatal("failed probe cache survived its request", err)
	}
	q, err := s.GetTokenPrice(context.Background(), token, 1)
	if err != nil || q.String() != "1.23" {
		t.Fatalf("updated USD anchor ignored: %s %v", q, err)
	}
}

func TestParallelCyclicRootsDoNotWaitOnEachOther(t *testing.T) {
	s, f, _, snapshot := newContractPriceFixture(t)
	a, b := priceAddress(1), priceAddress(2)
	f.assets[a], f.assets[b] = b, a
	f.precision[a], f.precision[b] = 6, 6
	f.conversions[a], f.conversions[b] = big.NewInt(1000000), big.NewInt(1000000)
	snapshot.Workers = 8
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := s.EnsurePricesAt(ctx, 1, []common.Address{a, b}, snapshot)
	if err == nil || ctx.Err() != nil {
		t.Fatalf("cyclic roots were priced or deadlocked: %v", err)
	}
	for _, token := range []common.Address{a, b} {
		if _, err := s.GetTokenPrice(context.Background(), token, 1); !errors.Is(err, ErrPriceUnavailable) {
			t.Fatal("cyclic token received a USD quote")
		}
	}
}

func totalPriceCalls(f *contractPriceFixture) int {
	n := 0
	for _, v := range f.calls {
		n += v
	}
	return n
}

func TestContractUSDCacheAndExpiry(t *testing.T) {
	s, f, store, snapshot := newContractPriceFixture(t)
	token := priceAddress(1)
	feed := f.addFeed(token, 123456789, 8)
	if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{token, token}, snapshot); err != nil {
		t.Fatal(err)
	}
	p, err := s.GetTokenPrice(context.Background(), token, 1)
	if err != nil || p.String() != "1.23456789" {
		t.Fatalf("USD normalization %s %v", p, err)
	}
	calls := totalPriceCalls(f)
	if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{token}, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	if totalPriceCalls(f) != calls {
		t.Fatal("fresh cache caused RPC calls")
	}
	q, _, _ := store.Get(context.Background(), TokenKey{1, token})
	q.UpdatedAt = time.Now().Add(-16 * time.Minute)
	if err := store.Set(context.Background(), TokenKey{1, token}, q); err != nil {
		t.Fatal(err)
	}
	f.updated[feed] = snapshot.Timestamp - 30
	if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{token}, snapshot); err != nil {
		t.Fatal(err)
	}
	if totalPriceCalls(f) <= calls {
		t.Fatal("stale cache was not refreshed")
	}
	q, _, _ = store.Get(context.Background(), TokenKey{1, token})
	q.ValidUntil = time.Now().Add(-time.Second)
	if err := store.Set(context.Background(), TokenKey{1, token}, q); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTokenPrice(context.Background(), token, 1); !errors.Is(err, ErrPriceUnavailable) {
		t.Fatal("expired oracle quote was served")
	}
}

func TestContractLPPriceUsesAllBalancesAndSupply(t *testing.T) {
	for _, lpDecimals := range []uint8{6, 18} {
		s, f, store, snapshot := newContractPriceFixture(t)
		usdc, usdt, vault, pool := priceAddress(1), priceAddress(2), priceAddress(3), priceAddress(4)
		f.addFeed(usdc, 102000000, 8)
		f.addFeed(usdt, 98000000, 8)
		f.precision[usdc], f.precision[usdt], f.precision[vault], f.precision[pool] = 6, 6, 18, lpDecimals
		f.assets[vault], f.conversions[vault] = usdt, big.NewInt(1100000)
		supply := new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(lpDecimals)), nil))
		f.pools[pool] = pricePoolFixture{coins: []common.Address{usdc, vault}, balances: []*big.Int{big.NewInt(50000000), decimal.RequireFromString("20000000000000000000").BigInt()}, supply: supply}
		if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{pool}, snapshot); err != nil {
			t.Fatal(err)
		}
		p, err := s.GetTokenPrice(context.Background(), pool, 1)
		if err != nil || !p.Equal(decimal.RequireFromString("7.256")) {
			t.Fatalf("LP decimals %d: got %s %v", lpDecimals, p, err)
		}
		component, found, _ := store.Get(context.Background(), TokenKey{1, vault})
		if !found || component.USD.String() != "1.078" {
			t.Fatal("component price not cached")
		}
		q, _, _ := store.Get(context.Background(), TokenKey{1, pool})
		if q.ValidUntil.After(component.UpdatedAt.Add(15 * time.Minute)) {
			t.Fatal("LP quote extended component freshness")
		}
		calls := totalPriceCalls(f)
		if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{pool, vault, usdc, usdt}, snapshot); err != nil {
			t.Fatal(err)
		}
		if totalPriceCalls(f) != calls {
			t.Fatal("cached components re-read contracts")
		}
	}
}

func TestContractInvalidRoundsDoNotProducePrices(t *testing.T) {
	for _, kind := range []string{"stale", "future", "zero", "negative", "incomplete", "missingDecimals"} {
		t.Run(kind, func(t *testing.T) {
			s, f, _, snapshot := newContractPriceFixture(t)
			token := priceAddress(1)
			feed := f.addFeed(token, 100000000, 8)
			switch kind {
			case "stale":
				f.updated[feed] = snapshot.Timestamp - 25*60*60
			case "future":
				f.updated[feed] = snapshot.Timestamp + 1
			case "zero":
				f.answers[feed] = big.NewInt(0)
			case "negative":
				f.answers[feed] = big.NewInt(-1)
			case "incomplete":
				f.badRound = true
			case "missingDecimals":
				delete(f.precision, feed)
			}
			if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{token}, snapshot); err == nil {
				t.Fatal("invalid oracle accepted")
			}
			if _, err := s.GetTokenPrice(context.Background(), token, 1); !errors.Is(err, ErrPriceUnavailable) {
				t.Fatal("invalid price cached")
			}
		})
	}
}

func TestContractMissingUSDAnchorKeepsOtherQuotes(t *testing.T) {
	s, f, _, snapshot := newContractPriceFixture(t)
	known, unknown := priceAddress(1), priceAddress(2)
	f.addFeed(known, 100000000, 8)
	if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{unknown, known}, snapshot); err == nil {
		t.Fatal("missing anchor not reported")
	}
	if _, err := s.GetTokenPrice(context.Background(), known, 1); err != nil {
		t.Fatal("valid quote lost", err)
	}
	if _, err := s.GetTokenPrice(context.Background(), unknown, 1); !errors.Is(err, ErrPriceUnavailable) {
		t.Fatal("unknown USD price invented")
	}
}

func TestContractUnsupportedLPAndVaultCases(t *testing.T) {
	for _, kind := range []string{"metapool", "zeroSupply", "missingComponent", "nestedLP", "cycle", "transport"} {
		t.Run(kind, func(t *testing.T) {
			s, f, _, snapshot := newContractPriceFixture(t)
			root, coin, other := priceAddress(1), priceAddress(2), priceAddress(3)
			f.precision[root], f.precision[coin], f.precision[other] = 18, 6, 6
			f.addFeed(coin, 100000000, 8)
			f.addFeed(other, 100000000, 8)
			f.pools[root] = pricePoolFixture{coins: []common.Address{coin, other}, balances: []*big.Int{big.NewInt(1000000), big.NewInt(1000000)}, supply: big.NewInt(1000000000000000000)}
			switch kind {
			case "metapool":
				p := f.pools[root]
				p.meta = true
				f.pools[root] = p
			case "zeroSupply":
				p := f.pools[root]
				p.supply = new(big.Int)
				f.pools[root] = p
			case "missingComponent":
				delete(f.feeds, other)
			case "nestedLP":
				delete(f.feeds, other)
				f.pools[other] = f.pools[root]
			case "cycle":
				delete(f.pools, root)
				f.assets[root] = root
			case "transport":
				f.fail = "getFeed"
			}
			if err := s.EnsurePricesAt(context.Background(), 1, []common.Address{root}, snapshot); err == nil {
				t.Fatal("unsupported price accepted")
			}
			if _, err := s.GetTokenPrice(context.Background(), root, 1); !errors.Is(err, ErrPriceUnavailable) {
				t.Fatal("unsupported root price cached")
			}
		})
	}
}

func TestContractSourceRequiresSnapshotAndHonorsCancellation(t *testing.T) {
	s, f, _, snapshot := newContractPriceFixture(t)
	if err := s.EnsurePrices(context.Background(), 1, []common.Address{priceAddress(1)}); err == nil {
		t.Fatal("contract source read latest implicitly")
	}
	if err := s.EnsurePricesAt(context.Background(), 8453, []common.Address{priceAddress(1)}, snapshot); err == nil {
		t.Fatal("unsupported network accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.EnsurePricesAt(ctx, 1, []common.Address{priceAddress(1)}, snapshot); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored")
	}
	if totalPriceCalls(f) != 0 {
		t.Fatal("invalid request made RPC calls")
	}
}
