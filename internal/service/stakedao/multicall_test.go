package stakedao

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ABIevm/internal/service/prices"
	"ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type multicallProbeRPC struct {
	stakeDAORPC
	calls int
	call  func(ethereum.CallMsg, *big.Int) ([]byte, error)
}

func (p *multicallProbeRPC) CallContract(_ context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	p.calls++
	return p.call(msg, block)
}

func repositoryMulticallABI(t *testing.T) abi.ABI {
	t.Helper()
	data, err := os.ReadFile("../../../contracts/curve/Multicall3Curve.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := abi.JSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func decodeMulticall(t *testing.T, parsed abi.ABI, msg ethereum.CallMsg, block *big.Int) []multicallInput {
	t.Helper()
	if msg.To == nil || *msg.To != stakeDAOMulticallAddress || block == nil || block.Cmp(big.NewInt(4500)) != 0 || msg.From != (common.Address{}) {
		t.Fatalf("wrong Multicall target/snapshot: %+v block %v", msg, block)
	}
	method, err := parsed.MethodById(msg.Data[:4])
	if err != nil || method.Name != "aggregate3" {
		t.Fatalf("wrong Multicall selector: %v", err)
	}
	args, err := method.Inputs.Unpack(msg.Data[4:])
	if err != nil {
		t.Fatal(err)
	}
	calls := *abi.ConvertType(args[0], new([]multicallInput)).(*[]multicallInput)
	for _, call := range calls {
		if !call.AllowFailure {
			t.Fatal("optional subcall would revert the entire batch")
		}
	}
	return calls
}

func TestMulticallChunksAndPreservesOrderAndSnapshot(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	getter, _ := abi.JSON(strings.NewReader(rewardVaultERC4626ABI))
	var sizes []int
	client := &multicallProbeRPC{call: func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		calls := decodeMulticall(t, parsed, msg, block)
		sizes = append(sizes, len(calls))
		results := make([]multicallOutput, len(calls))
		for i, call := range calls {
			method, err := getter.MethodById(call.CallData[:4])
			if err != nil || method.Name != "totalAssets" {
				t.Fatal("wrong subcall selector")
			}
			data, err := method.Outputs.Pack(call.Target.Big())
			if err != nil {
				t.Fatal(err)
			}
			results[i] = multicallOutput{Success: true, ReturnData: data}
		}
		return parsed.Methods["aggregate3"].Outputs.Pack(results)
	}}
	service := &StakeDAOService{client: client}
	var reads []snapshotRead
	for i := 1; i <= 49; i++ {
		reads = append(reads, readContract(common.BigToAddress(big.NewInt(int64(i))), rewardVaultERC4626ABI, "totalAssets"))
	}
	results := service.snapshotCalls(context.Background(), big.NewInt(4500), reads...)
	if !reflect.DeepEqual(sizes, []int{24, 24, 1}) {
		t.Fatalf("unbounded batches: %v", sizes)
	}
	for i, result := range results {
		value, err := result.unsigned()
		if err != nil || value.Cmp(big.NewInt(int64(i+1))) != 0 {
			t.Fatalf("result %d reordered/lost: %v %v", i, value, err)
		}
	}
}

func TestMulticallIsolatesRevertsAndMalformedGetterData(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	getter, _ := abi.JSON(strings.NewReader(erc20DecimalsABI))
	valid, _ := getter.Methods["decimals"].Outputs.Pack(uint8(6))
	client := &multicallProbeRPC{call: func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		decodeMulticall(t, parsed, msg, block)
		return parsed.Methods["aggregate3"].Outputs.Pack([]multicallOutput{
			{Success: false, ReturnData: valid},
			{Success: true, ReturnData: nil},
			{Success: true, ReturnData: valid},
		})
	}}
	read := readContract(common.HexToAddress("0xaa"), erc20DecimalsABI, "decimals")
	results := (&StakeDAOService{client: client}).snapshotCalls(context.Background(), big.NewInt(4500), read, read, read)
	for _, result := range results[:2] {
		if value, err := result.unsigned(); err == nil || value != nil {
			t.Fatalf("failed getter turned into a number: %v %v", value, err)
		}
	}
	if value, err := results[2].unsigned(); err != nil || value.Uint64() != 6 || client.calls != 1 {
		t.Fatalf("healthy getter lost or retried: %v %v calls=%d", value, err, client.calls)
	}
}

func TestMulticallRejectsInvalidBatchesWithoutFallback(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	read := readContract(common.HexToAddress("0xaa"), erc20DecimalsABI, "decimals")
	for _, mode := range []string{"transport", "malformed", "too-few", "too-many"} {
		t.Run(mode, func(t *testing.T) {
			client := &multicallProbeRPC{call: func(ethereum.CallMsg, *big.Int) ([]byte, error) {
				switch mode {
				case "transport":
					return nil, errors.New("provider unavailable")
				case "malformed":
					return []byte{1}, nil
				case "too-few":
					return parsed.Methods["aggregate3"].Outputs.Pack([]multicallOutput{})
				default:
					return parsed.Methods["aggregate3"].Outputs.Pack(make([]multicallOutput, 3))
				}
			}}
			results := (&StakeDAOService{client: client}).snapshotCalls(context.Background(), big.NewInt(4500), read, read)
			if client.calls != 1 {
				t.Fatalf("unexpected retry/fallback: %d calls", client.calls)
			}
			for _, result := range results {
				if !errors.Is(result.err, ErrRPCUnavailable) || len(result.values) != 0 {
					t.Fatalf("invalid batch accepted: %+v", result)
				}
			}
		})
	}
	client := &multicallProbeRPC{}
	service := &StakeDAOService{client: client}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := service.snapshotCalls(ctx, big.NewInt(4500), read); !errors.Is(result[0].err, context.Canceled) || client.calls != 0 {
		t.Fatalf("cancelled batch was sent: %+v", result)
	}
	if result := service.snapshotCalls(context.Background(), nil, read); result[0].err == nil || client.calls != 0 {
		t.Fatal("unfixed block accepted")
	}
}

func TestMulticallConsumesOneSampleAttemptForMultipleGetters(t *testing.T) {
	_, f := newPoolFixture(t)
	client := newSampleRPC(f)
	client.remaining = 1
	read := readContract(common.HexToAddress("0xbb"), erc20DecimalsABI, "decimals")
	results := (&StakeDAOService{client: client}).snapshotCalls(context.Background(), big.NewInt(4500), read, read, read)
	for _, result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
	}
	if client.remaining != 0 || client.exhausted || len(f.callBlocks) != 1 || !reflect.DeepEqual(f.multicallSizes, []int{3}) {
		t.Fatal("sample budget counted subcalls as separate RPC attempts")
	}
}

func TestMulticallReducesFullPoolReadsWithoutChangingRates(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.sidecar = true, true
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil || len(result.Errors) != 0 || len(result.Pools) != 1 {
		t.Fatalf("pool loading failed: %+v %v", result, err)
	}
	pool := result.Pools[0]
	if pool.AssetAmount == nil || *pool.AssetAmount != "1.234567" || pool.TVL == nil || *pool.TVL != 2.469134 || pool.Rewards[0].RatePerSecond == nil || *pool.Rewards[0].RatePerSecond != "1.1688" || pool.APRStatus != "available" {
		t.Fatalf("pool values changed: %+v", pool)
	}
	getters := f.directCalls
	for _, size := range f.multicallSizes {
		getters += size
	}
	requests := len(f.multicallSizes) + f.directCalls
	if getters < 30 || requests*2 >= getters {
		t.Fatalf("expected >2x fewer eth_call requests: %d requests for %d getters", requests, getters)
	}
	t.Logf("%d contract getters in %d eth_call requests (%d Multicall batches)", getters, requests, len(f.multicallSizes))
}

func TestSharedSnapshotReadsAreDeduplicatedWithoutSharingMutableValues(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	getter, _ := abi.JSON(strings.NewReader(rewardVaultERC4626ABI))
	started, release := make(chan struct{}), make(chan struct{})
	client := &multicallProbeRPC{call: func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		calls := decodeMulticall(t, parsed, msg, block)
		close(started)
		<-release
		if len(calls) != 1 {
			t.Errorf("shared getter was sent more than once: %d", len(calls))
		}
		data, _ := getter.Methods["totalAssets"].Outputs.Pack(big.NewInt(123))
		return parsed.Methods["aggregate3"].Outputs.Pack([]multicallOutput{{Success: true, ReturnData: data}})
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	batches := newSnapshotBatcher(ctx, client, big.NewInt(4500))
	defer batches.close()
	service := &StakeDAOService{batches: batches}
	read := readContract(common.HexToAddress("0xbb"), rewardVaultERC4626ABI, "totalAssets")
	values := make([]*big.Int, 8)
	var wg sync.WaitGroup
	for i := range values {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			values[i], err = service.snapshotCalls(ctx, big.NewInt(4500), read)[0].unsigned()
			if err != nil {
				t.Error(err)
			}
		}()
	}
	<-started
	close(release)
	wg.Wait()
	if client.calls != 1 {
		t.Fatalf("duplicate in-flight/cached reads: %d RPC attempts", client.calls)
	}
	values[0].SetInt64(999)
	for _, value := range values[1:] {
		if value == nil || value.Int64() != 123 {
			t.Fatalf("ABI consumers share mutable values: %v", values)
		}
	}
	if result := service.snapshotCalls(ctx, big.NewInt(4501), read)[0]; result.err == nil || client.calls != 1 {
		t.Fatal("scheduler accepted a different snapshot")
	}
}

func TestSnapshotCacheDoesNotSurviveTheRequestOrHideGetterFailures(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	getter, _ := abi.JSON(strings.NewReader(rewardVaultERC4626ABI))
	seen := make(map[string]int)
	client := &multicallProbeRPC{call: func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		args, err := parsed.Methods["aggregate3"].Inputs.Unpack(msg.Data[4:])
		if err != nil {
			t.Fatal(err)
		}
		calls := *abi.ConvertType(args[0], new([]multicallInput)).(*[]multicallInput)
		out := make([]multicallOutput, len(calls))
		for i, call := range calls {
			seen[block.String()+call.Target.Hex()]++
			data, _ := getter.Methods["totalAssets"].Outputs.Pack(new(big.Int).Set(block))
			out[i] = multicallOutput{Success: true, ReturnData: data}
			switch call.Target {
			case common.HexToAddress("0x1"):
				out[i].Success = false
			case common.HexToAddress("0x2"):
				out[i].ReturnData = []byte{1}
			}
		}
		return parsed.Methods["aggregate3"].Outputs.Pack(out)
	}}
	ctx := context.Background()
	for _, block := range []int64{4500, 4501} {
		batches := newSnapshotBatcher(ctx, client, big.NewInt(block))
		service := &StakeDAOService{batches: batches}
		reads := []snapshotRead{
			readContract(common.HexToAddress("0x1"), rewardVaultERC4626ABI, "totalAssets"),
			readContract(common.HexToAddress("0x2"), rewardVaultERC4626ABI, "totalAssets"),
			readContract(common.HexToAddress("0x3"), rewardVaultERC4626ABI, "totalAssets"),
		}
		for range 2 {
			result := service.snapshotCalls(ctx, big.NewInt(block), reads...)
			value, err := result[2].unsigned()
			if result[0].err == nil || result[1].err == nil || err != nil || value.Int64() != block {
				t.Fatalf("getter failures leaked or stale snapshot reused: block=%d results=%+v", block, result)
			}
		}
		batches.close()
	}
	if len(seen) != 6 {
		t.Fatalf("requests shared a cache: %+v", seen)
	}
	for key, count := range seen {
		if count != 1 {
			t.Fatalf("getter retried/fallback used: %s count=%d", key, count)
		}
	}
}

func multiPoolFixture(t *testing.T, count int) (*StakeDAOService, *fixtureRPC, []StakeDAOVaultInfo) {
	t.Helper()
	s, f := newPoolFixture(t)
	f.fullPrimary, f.logs = true, nil
	var vaults []StakeDAOVaultInfo
	for i := 0; i < count; i++ {
		gauge := common.BigToAddress(big.NewInt(int64(1000 + i)))
		vault := common.BigToAddress(big.NewInt(int64(2000 + i)))
		asset := common.HexToAddress("0xcc")
		f.vaults[gauge], f.assets[gauge] = vault, asset
		f.logs = append(f.logs, registration(t, f.controller, gauge, vault, asset, 124, uint(i)))
		vaults = append(vaults, StakeDAOVaultInfo{Gauge: gauge, Vault: vault, Asset: asset, RewardReceiver: common.HexToAddress("0x44"), ProtocolID: [4]byte{0xc7, 0x15, 0xe3, 0x73}})
	}
	sort.Slice(vaults, func(i, j int) bool { return vaults[i].Gauge.Hex() < vaults[j].Gauge.Hex() })
	return s, f, vaults
}

func TestCrossPoolMulticallPreservesValuesAndReducesRPCRequests(t *testing.T) {
	baseline, oldRPC, vaults := multiPoolFixture(t, 5)
	baseline.client, baseline.controller, baseline.chainID = oldRPC, oldRPC.controller, 1
	var expected []dto.StakeDAOPool
	for _, info := range vaults {
		if _, err := baseline.currentGaugeRegistrations(context.Background(), big.NewInt(4500), []common.Address{info.Gauge}); err != nil {
			t.Fatal(err)
		}
		pool, issues := baseline.loadPool(context.Background(), info, big.NewInt(4500))
		issues = append(issues, baseline.loadPoolRewardMechanism(context.Background(), info, big.NewInt(4500), &pool)...)
		issues = append(issues, baseline.loadRewardStreams(context.Background(), info, big.NewInt(4500), &pool)...)
		issues = append(issues, baseline.valuePool(context.Background(), &pool)...)
		issues = append(issues, baseline.valueRewards(context.Background(), &pool)...)
		if len(issues) != 0 {
			t.Fatalf("baseline is incomplete: %+v", issues)
		}
		pool.UpdatedAt = time.Time{}
		expected = append(expected, pool)
	}
	optimized, newRPC, _ := multiPoolFixture(t, 5)
	result, err := optimized.GetPools(context.Background(), "ethereum")
	if err != nil || result == nil || len(result.Errors) != 0 || len(result.Pools) != len(expected) {
		t.Fatalf("optimized load failed: %+v %v", result, err)
	}
	for i := range result.Pools {
		result.Pools[i].UpdatedAt = time.Time{}
		if !reflect.DeepEqual(result.Pools[i], expected[i]) {
			t.Fatalf("pool %d changed: got %+v want %+v", i, result.Pools[i], expected[i])
		}
	}
	oldCalls := len(oldRPC.callBlocks)
	newCalls := len(newRPC.callBlocks)
	if newCalls*2 >= oldCalls || newRPC.directCalls != 0 {
		t.Fatalf("cross-pool batching did not halve eth_call requests: before=%d after=%d direct=%d", oldCalls, newCalls, newRPC.directCalls)
	}
	for _, size := range newRPC.multicallSizes {
		if size > stakeDAOMulticallBatchSize {
			t.Fatal("oversized cross-pool batch")
		}
	}
	t.Logf("5 fixture pools, identical values/order: eth_call before=%d after=%d", oldCalls, newCalls)
}

func TestFull359PoolLoadUsesBoundedCrossPoolBatches(t *testing.T) {
	s, f, vaults := multiPoolFixture(t, 359)
	// Unlike the original warmed-price fixture, every asset now has a distinct
	// cold quote. Exercise the real price source as well as pool loading.
	for i := range f.logs {
		gauge := common.BytesToAddress(f.logs[i].Topics[1].Bytes())
		asset := common.BigToAddress(big.NewInt(int64(4000 + i)))
		f.assets[gauge] = asset
		f.logs[i].Topics[3] = common.BytesToHash(asset.Bytes())
	}
	oracle := attachColdPriceSource(t, s, f)
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil || result == nil || len(result.Pools) != len(vaults) || len(result.Errors) != 0 {
		t.Fatalf("359-pool load failed: pools=%v err=%v", result, err)
	}
	for i, pool := range result.Pools {
		if pool.GaugeAddress != vaults[i].Gauge || pool.VaultAddress != vaults[i].Vault || pool.TVL == nil || pool.APR == nil || pool.APRStatus != "available" {
			t.Fatalf("reordered/incomplete pool %d: %+v", i, pool)
		}
	}
	if len(f.callBlocks) > 5*len(vaults) {
		t.Fatalf("too many eth_call requests: %d", len(f.callBlocks))
	}
	if oracle.calls.Load() != 3*(359+1) {
		t.Fatalf("cold quotes were not fetched exactly once: %d", oracle.calls.Load())
	}
	t.Logf("359 pools with cold contract prices: pool eth_call=%d price eth_call=%d", len(f.callBlocks), oracle.calls.Load())
}

const coldOracleABI = `[
{"type":"function","name":"getFeed","inputs":[{"type":"address"},{"type":"address"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"decimals","inputs":[],"outputs":[{"type":"uint8"}]},
{"type":"function","name":"latestRoundData","inputs":[],"outputs":[{"type":"uint80"},{"type":"int256"},{"type":"uint256"},{"type":"uint256"},{"type":"uint80"}]}
]`

type coldPriceRPC struct {
	*fixtureRPC
	methods   abi.ABI
	timestamp uint64
	calls     atomic.Int32
	failure   error
	missing   common.Address
}

func attachColdPriceSource(t *testing.T, service *StakeDAOService, fixture *fixtureRPC) *coldPriceRPC {
	t.Helper()
	methods, err := abi.JSON(strings.NewReader(coldOracleABI))
	if err != nil {
		t.Fatal(err)
	}
	caller := &coldPriceRPC{fixtureRPC: fixture, methods: methods, timestamp: uint64(time.Now().Unix())}
	store := prices.NewMemoryStore()
	service.prices = prices.NewTokenPriceService(store, prices.NewContractSource(store))
	service.dial = func(context.Context, string) (stakeDAORPC, error) { return caller, nil }
	return caller
}

func (c *coldPriceRPC) HeaderByNumber(ctx context.Context, block *big.Int) (*types.Header, error) {
	header, err := c.fixtureRPC.HeaderByNumber(ctx, block)
	if header != nil {
		header.Time = c.timestamp
	}
	return header, err
}

func (c *coldPriceRPC) CallContract(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	if call.To != nil && *call.To == stakeDAOMulticallAddress {
		return c.fixtureRPC.CallContract(ctx, call, block)
	}
	c.calls.Add(1)
	if c.failure != nil {
		return nil, c.failure
	}
	if block == nil || block.Uint64() != c.latest || call.From != (common.Address{}) || call.To == nil {
		return nil, errors.New("oracle caller/snapshot changed")
	}
	method, err := c.methods.MethodById(call.Data[:4])
	if err != nil {
		return nil, err
	}
	switch method.Name {
	case "getFeed":
		args, err := method.Inputs.Unpack(call.Data[4:])
		if err != nil {
			return nil, err
		}
		token := args[0].(common.Address)
		if token == c.missing {
			return nil, priceRevertError{}
		}
		return method.Outputs.Pack(common.BigToAddress(new(big.Int).Add(token.Big(), big.NewInt(100000))))
	case "decimals":
		return method.Outputs.Pack(uint8(8))
	case "latestRoundData":
		answer := big.NewInt(200000000)
		if *call.To == common.BigToAddress(new(big.Int).Add(c.rewardToken.Big(), big.NewInt(100000))) {
			answer = big.NewInt(300000000)
		}
		updated := new(big.Int).SetUint64(c.timestamp - 60)
		return method.Outputs.Pack(big.NewInt(5), answer, updated, updated, big.NewInt(5))
	}
	return nil, errors.New("unexpected oracle method")
}

type priceRevertError struct{}

func (priceRevertError) Error() string  { return "execution reverted: Feed not found" }
func (priceRevertError) ErrorCode() int { return 3 }

func TestColdPriceFailureIsUnavailableDataButHTTP402AbortsTheLoad(t *testing.T) {
	for _, quota := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-feed", true: "quota"}[quota], func(t *testing.T) {
			logs := captureLoadLogs(t)
			s, fixture, vaults := multiPoolFixture(t, 2)
			missing := common.HexToAddress("0xeeee")
			gauge := common.BytesToAddress(fixture.logs[0].Topics[1].Bytes())
			fixture.assets[gauge], fixture.logs[0].Topics[3] = missing, common.BytesToHash(missing.Bytes())
			oracle := attachColdPriceSource(t, s, fixture)
			oracle.missing = missing
			if quota {
				oracle.failure = rpc.HTTPError{StatusCode: 402, Status: "402 Payment Required"}
			}
			result, err := s.GetPools(context.Background(), "ethereum")
			if quota {
				var httpErr rpc.HTTPError
				if result != nil || !errors.Is(err, ErrRPCUnavailable) || !errors.As(err, &httpErr) || httpErr.StatusCode != 402 || !fixture.closed {
					t.Fatalf("price quota failure hidden: result=%+v err=%v", result, err)
				}
			} else {
				if err != nil || result == nil || len(result.Pools) != len(vaults) {
					t.Fatalf("missing feed discarded otherwise valid pools: result=%+v err=%v", result, err)
				}
				for _, pool := range result.Pools {
					if pool.AssetAddress == missing {
						if pool.TVL != nil || pool.APR != nil {
							t.Fatal("missing USD price was replaced with a number")
						}
					} else if pool.TVL == nil || *pool.TVL != 2.469134 || pool.APR == nil {
						t.Fatalf("valid cold USD quote lost: %+v", pool)
					}
				}
			}
			records := loadLogRecords(t, logs)
			last := records[len(records)-1]
			if last["price_workers"] != float64(8) || last["rpc_active"] != float64(0) {
				t.Fatalf("missing price concurrency diagnostics/unfinished calls: %+v", last)
			}
			if !quota && (last["rpc_reverts"].(float64) < 1 || last["first_rpc_failure"] != "contract_revert" || last["event"] != "completed") {
				t.Fatalf("expected contract probe treated as a failed load: %+v", last)
			}
		})
	}
}

type contextPoolRPC struct {
	stakeDAORPC
	call func(context.Context) ([]byte, error)
}

func (p *contextPoolRPC) CallContract(ctx context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	return p.call(ctx)
}

func TestCrossPoolFailureStopsWorkersAndDoesNotRetryQuotaErrors(t *testing.T) {
	s, fixture, _ := multiPoolFixture(t, 20)
	var calls atomic.Int32
	s.dial = func(context.Context, string) (stakeDAORPC, error) {
		return &contextPoolRPC{stakeDAORPC: fixture, call: func(context.Context) ([]byte, error) {
			calls.Add(1)
			return nil, rpc.HTTPError{StatusCode: http.StatusPaymentRequired, Status: "402 Payment Required"}
		}}, nil
	}
	result, err := s.GetPools(context.Background(), "ethereum")
	var httpErr rpc.HTTPError
	if result != nil || !errors.As(err, &httpErr) || httpErr.StatusCode != 402 || calls.Load() != 1 || !fixture.closed {
		t.Fatalf("quota error retried/hidden or RPC left open: result=%+v err=%v calls=%d", result, err, calls.Load())
	}
}

func TestCrossPoolCancellationJoinsDispatcherAndPoolWorkers(t *testing.T) {
	s, fixture, _ := multiPoolFixture(t, 20)
	started := make(chan struct{})
	var active atomic.Int32
	s.dial = func(context.Context, string) (stakeDAORPC, error) {
		return &contextPoolRPC{stakeDAORPC: fixture, call: func(ctx context.Context) ([]byte, error) {
			active.Add(1)
			defer active.Add(-1)
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		}}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := s.GetPools(ctx, "ethereum")
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("pool dispatcher did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || active.Load() != 0 || !fixture.closed {
			t.Fatalf("cancellation left work running: err=%v active=%d closed=%v", err, active.Load(), fixture.closed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("pool workers/dispatcher did not exit")
	}
}
