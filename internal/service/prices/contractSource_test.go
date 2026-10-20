package prices

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type pricePoolFixture struct {
	coins    []common.Address
	balances []*big.Int
	supply   *big.Int
	meta     bool
}

type contractPriceFixture struct {
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
