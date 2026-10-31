package stakedao

import (
	"ABIevm/internal/service/prices"
	dto "ABIevm/internal/service/stakedao/dto"
	"context"
	"encoding/json"
	"errors"
	"github.com/shopspring/decimal"
	"math"
	"math/big"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type fixtureRPC struct {
	mu                   sync.Mutex
	controller           common.Address
	latest, deployment   uint64
	logs                 []types.Log
	queries              []ethereum.FilterQuery
	callBlocks           []uint64
	multicallSizes       []int
	directCalls          int
	vaults, assets       map[common.Address]common.Address
	callFailure          string
	filterFailure        error
	headerReads          int
	codeReads            int
	reorg                bool
	closed               bool
	accountant           common.Address
	controllerAccountant common.Address
	rewardToken          common.Address
	fullPrimary          bool
	sidecar              bool
	sidecarIdentity      map[string]common.Address
	policy               uint8
	extraTokens          []common.Address
	extraRate            *big.Int
	extraFinish          uint32
	killed               bool
	physical             *big.Int
	totalAssets          *big.Int
	shutdown             map[common.Address]bool
}

func (f *fixtureRPC) ChainID(context.Context) (*big.Int, error)   { return big.NewInt(1), nil }
func (f *fixtureRPC) BlockNumber(context.Context) (uint64, error) { return f.latest, nil }
func (f *fixtureRPC) HeaderByNumber(_ context.Context, number *big.Int) (*types.Header, error) {
	f.headerReads++
	h := &types.Header{Number: new(big.Int).Set(number)}
	if f.reorg && f.headerReads > 1 {
		h.Extra = []byte("reorg")
	}
	return h, nil
}
func (f *fixtureRPC) CodeAt(_ context.Context, _ common.Address, number *big.Int) ([]byte, error) {
	f.codeReads++
	if number.Uint64() >= f.deployment {
		return []byte{1}, nil
	}
	return nil, nil
}
func (f *fixtureRPC) FilterLogs(_ context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries = append(f.queries, query)
	if f.filterFailure != nil {
		return nil, f.filterFailure
	}
	out := make([]types.Log, 0)
	for _, entry := range f.logs {
		if entry.BlockNumber >= query.FromBlock.Uint64() && entry.BlockNumber <= query.ToBlock.Uint64() {
			out = append(out, entry)
		}
	}
	return out, nil
}
func (f *fixtureRPC) Close() { f.closed = true }
func (f *fixtureRPC) CallContract(_ context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	f.callBlocks = append(f.callBlocks, block.Uint64())
	if call.To != nil && *call.To == stakeDAOMulticallAddress {
		parsed, _ := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
		args, err := parsed.Methods["aggregate3"].Inputs.Unpack(call.Data[4:])
		if err != nil {
			return nil, err
		}
		calls := *abi.ConvertType(args[0], new([]multicallInput)).(*[]multicallInput)
		f.multicallSizes = append(f.multicallSizes, len(calls))
		results := make([]multicallOutput, len(calls))
		for i, subcall := range calls {
			data, err := f.contractResult(ethereum.CallMsg{To: &subcall.Target, Data: subcall.CallData})
			results[i] = multicallOutput{Success: err == nil, ReturnData: data}
		}
		return parsed.Methods["aggregate3"].Outputs.Pack(results)
	}
	f.directCalls++
	return f.contractResult(call)
}

func (f *fixtureRPC) contractResult(call ethereum.CallMsg) ([]byte, error) {
	for _, abiJSON := range []string{protocolControllerABI, erc20NameABI, erc20SymbolABI, erc20DecimalsABI, rewardVaultERC4626ABI, rewardVaultAccountantABI, accountantRewardTokenABI, vaultV2ABI, accountantV2ABI, strategyV2ABI, sidecarV2ABI, allocatorV2ABI, curveGaugeABI, curveControllerABI, convexPoolABI, convexBoosterABI} {
		parsed, _ := abi.JSON(strings.NewReader(abiJSON))
		method, err := parsed.MethodById(call.Data[:4])
		if err != nil {
			continue
		}
		if value, ok := f.sidecarIdentity[method.Name]; ok && *call.To == common.HexToAddress("0x1236") {
			return method.Outputs.Pack(value)
		}
		if method.Name == f.callFailure {
			return nil, errors.New("fixture call failed")
		}
		switch method.Name {
		case "strategy":
			address := common.Address{}
			if f.fullPrimary {
				address = common.HexToAddress("0x1234")
			}
			return method.Outputs.Pack(address)
		case "PROTOCOL_CONTROLLER":
			return method.Outputs.Pack(f.controller)
		case "BOOSTER":
			return method.Outputs.Pack(common.HexToAddress(convexBoosterEthereum))
		case "pid":
			return method.Outputs.Pack(big.NewInt(1))
		case "rewardReceiver":
			return method.Outputs.Pack(common.HexToAddress("0x44"))
		case "poolInfo":
			values := []common.Address{common.HexToAddress("0xcc"), common.HexToAddress("0xaa"), common.HexToAddress("0x1237")}
			for i, key := range []string{"poolAsset", "poolGauge", "poolRewards"} {
				if value, ok := f.sidecarIdentity[key]; ok {
					values[i] = value
				}
			}
			return method.Outputs.Pack(values[0], common.HexToAddress("0x1238"), values[1], values[2], common.Address{}, false)
		case "LOCKER":
			return method.Outputs.Pack(common.HexToAddress("0x99"))
		case "allocator":
			return method.Outputs.Pack(common.HexToAddress("0x1235"))
		case "getAllocationTargets":
			targets := []common.Address{common.HexToAddress("0x99")}
			if f.sidecar {
				targets = append(targets, common.HexToAddress("0x1236"))
			}
			return method.Outputs.Pack(targets)
		case "POLICY":
			return method.Outputs.Pack(f.policy)
		case "getProtocolFeePercent":
			return method.Outputs.Pack(big.NewInt(150000000000000000))
		case "getHarvestFeePercent":
			return method.Outputs.Pack(big.NewInt(1000000000000000))
		case "is_killed":
			return method.Outputs.Pack(f.killed)
		case "inflation_rate":
			return method.Outputs.Pack(big.NewInt(4000000000000000000))
		case "gauge_relative_weight":
			return method.Outputs.Pack(big.NewInt(500000000000000000))
		case "working_balances":
			return method.Outputs.Pack(big.NewInt(400000))
		case "working_supply":
			return method.Outputs.Pack(big.NewInt(4000000))
		case "gauge":
			if *call.To == f.controller {
				args, err := method.Inputs.Unpack(call.Data[4:])
				if err != nil {
					return nil, err
				}
				gauge := args[0].(common.Address)
				return method.Outputs.Pack(f.vaults[gauge], f.assets[gauge], common.HexToAddress("0x44"), [4]byte{0xc7, 0x15, 0xe3, 0x73}, f.shutdown[gauge])
			}
			return method.Outputs.Pack(common.BytesToAddress(f.logs[0].Topics[1].Bytes()))
		case "baseRewardPool":
			return method.Outputs.Pack(common.HexToAddress("0x1237"))
		case "rewardToken":
			return method.Outputs.Pack(f.rewardToken)
		case "balanceOf":
			if *call.To == common.HexToAddress("0x1234") {
				if f.physical != nil {
					return method.Outputs.Pack(f.physical)
				}
				return method.Outputs.Pack(big.NewInt(1234567))
			}
			return method.Outputs.Pack(big.NewInt(500000))
		case "totalSupply":
			return method.Outputs.Pack(big.NewInt(1000000))
		case "rewardRate":
			return method.Outputs.Pack(big.NewInt(2000000000000000000))
		case "periodFinish":
			return method.Outputs.Pack(big.NewInt(10000))
		case "getRewardTokens":
			return method.Outputs.Pack(f.extraTokens)
		case "rewardData":
			rate := f.extraRate
			if rate == nil {
				rate = new(big.Int)
			}
			return method.Outputs.Pack(common.HexToAddress("0x44"), uint32(0), f.extraFinish, rate, new(big.Int))
		case "ACCOUNTANT":
			return method.Outputs.Pack(f.accountant)
		case "accountant":
			return method.Outputs.Pack(f.controllerAccountant)
		case "REWARD_TOKEN":
			return method.Outputs.Pack(f.rewardToken)
		case "vault", "asset":
			if len(method.Inputs) == 0 {
				return method.Outputs.Pack(common.HexToAddress("0xcc"))
			}
			args, err := method.Inputs.Unpack(call.Data[4:])
			if err != nil {
				return nil, err
			}
			gauge := args[0].(common.Address)
			address := f.vaults[gauge]
			if method.Name == "asset" {
				address = f.assets[gauge]
			}
			return method.Outputs.Pack(address)
		case "name":
			return method.Outputs.Pack("Stake DAO fixture Vault")
		case "symbol":
			return method.Outputs.Pack("FIX")
		case "decimals":
			if f.fullPrimary && *call.To == f.rewardToken {
				return method.Outputs.Pack(uint8(18))
			}
			return method.Outputs.Pack(uint8(6))
		case "totalAssets":
			if f.totalAssets != nil {
				return method.Outputs.Pack(f.totalAssets)
			}
			return method.Outputs.Pack(big.NewInt(1234567))
		}
	}
	return nil, errors.New("unexpected call")
}

func registration(t *testing.T, controller, gauge, vault, asset common.Address, block uint64, index uint) types.Log {
	t.Helper()
	parsed, err := abi.JSON(strings.NewReader(vaultRegisteredABI))
	if err != nil {
		t.Fatal(err)
	}
	event := parsed.Events["VaultRegistered"]
	data, err := event.Inputs.NonIndexed().Pack(common.HexToAddress("0x44"), [4]byte{0xc7, 0x15, 0xe3, 0x73})
	if err != nil {
		t.Fatal(err)
	}
	return types.Log{
		Address: controller, BlockNumber: block, Index: index, Data: data,
		Topics: []common.Hash{event.ID, common.BytesToHash(gauge.Bytes()), common.BytesToHash(vault.Bytes()), common.BytesToHash(asset.Bytes())},
	}
}

func TestVaultRegisteredMatchesRepositoryABI(t *testing.T) {
	data, err := os.ReadFile("../../../contracts/stakedao/CreateVault.json")
	if err != nil {
		t.Fatal(err)
	}
	repositoryABI, err := abi.JSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	minimalABI, _ := abi.JSON(strings.NewReader(vaultRegisteredABI))
	if !reflect.DeepEqual(repositoryABI.Events["VaultRegistered"].Inputs, minimalABI.Events["VaultRegistered"].Inputs) || repositoryABI.Events["VaultRegistered"].ID != minimalABI.Events["VaultRegistered"].ID {
		t.Fatal("discovery ABI differs from checked-in Controller ABI")
	}
}

func TestDiscoveryChunksHistoryAndKeepsLatestRegistration(t *testing.T) {
	controller := common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)
	gauge, other := common.HexToAddress("0x11"), common.HexToAddress("0x22")
	vault, oldVault, asset := common.HexToAddress("0x33"), common.HexToAddress("0x55"), common.HexToAddress("0x66")
	removed := registration(t, controller, gauge, oldVault, asset, 4009, 1)
	removed.Removed = true
	f := &fixtureRPC{logs: []types.Log{
		registration(t, controller, gauge, vault, asset, 2010, 3),
		registration(t, controller, gauge, oldVault, asset, 2010, 1),
		registration(t, controller, gauge, oldVault, asset, 10, 0),
		registration(t, controller, other, vault, asset, 4008, 0), removed,
	}}
	result, err := DiscoverStakeDAOVaults(context.Background(), f, controller, 10, 4010)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 || result[0].Gauge != gauge || result[0].Vault != vault || result[0].RewardReceiver != common.HexToAddress("0x44") || result[0].ProtocolID != [4]byte{0xc7, 0x15, 0xe3, 0x73} {
		t.Fatalf("wrong registrations: %+v", result)
	}
	wantRanges := [][2]uint64{{10, 1509}, {1510, 3009}, {3010, 4010}}
	sort.Slice(f.queries, func(i, j int) bool { return f.queries[i].FromBlock.Cmp(f.queries[j].FromBlock) < 0 })
	if len(f.queries) != len(wantRanges) {
		t.Fatalf("queries: %d", len(f.queries))
	}
	for i, query := range f.queries {
		if query.FromBlock.Uint64() != wantRanges[i][0] || query.ToBlock.Uint64() != wantRanges[i][1] || query.Addresses[0] != controller || len(query.Topics[0]) != 1 {
			t.Fatalf("incorrect query: %+v", query)
		}
	}
}

func TestDiscoveryRejectsInvalidLogsAndPropagatesErrors(t *testing.T) {
	controller := common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)
	valid := registration(t, controller, common.HexToAddress("0x11"), common.HexToAddress("0x22"), common.HexToAddress("0x33"), 10, 0)
	for _, field := range []string{"topics", "signature", "data", "address", "padding", "zero"} {
		t.Run(field, func(t *testing.T) {
			entry := valid
			entry.Topics = append([]common.Hash(nil), valid.Topics...)
			switch field {
			case "topics":
				entry.Topics = entry.Topics[:1]
			case "signature":
				entry.Topics[0] = common.Hash{}
			case "data":
				entry.Data = []byte{1}
			case "address":
				entry.Address = common.Address{}
			case "padding":
				entry.Topics[1][0] = 1
			case "zero":
				entry.Topics[1] = common.Hash{}
			}
			_, err := DiscoverStakeDAOVaults(context.Background(), &fixtureRPC{logs: []types.Log{entry}}, controller, 10, 10)
			if err == nil {
				t.Fatal("invalid log was accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &fixtureRPC{}
	_, err := DiscoverStakeDAOVaults(ctx, f, controller, 0, 10)
	if !errors.Is(err, context.Canceled) || len(f.queries) != 0 {
		t.Fatalf("cancellation: %v", err)
	}
	_, err = DiscoverStakeDAOVaults(context.Background(), &fixtureRPC{filterFailure: context.DeadlineExceeded}, controller, 0, 10)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("filter error lost: %v", err)
	}
}

func newPoolFixture(t *testing.T) (*StakeDAOService, *fixtureRPC) {
	t.Helper()
	controller := common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)
	gauge, vault, asset := common.HexToAddress("0xaa"), common.HexToAddress("0xbb"), common.HexToAddress("0xcc")
	inactive := common.HexToAddress("0xdd")
	f := &fixtureRPC{
		controller: controller, latest: 4500, deployment: 123,
		vaults: map[common.Address]common.Address{gauge: vault}, assets: map[common.Address]common.Address{gauge: asset},
		logs: []types.Log{registration(t, controller, gauge, vault, asset, 124, 0), registration(t, controller, inactive, vault, asset, 125, 0)},
	}
	f.accountant = common.HexToAddress("0x77")
	f.controllerAccountant = f.accountant
	f.rewardToken = common.HexToAddress("0x88")
	parsed, _ := abi.JSON(strings.NewReader(vaultRegisteredABI))
	for i := range f.logs {
		data, err := parsed.Events["VaultRegistered"].Inputs.NonIndexed().Pack(common.HexToAddress("0x44"), [4]byte{0xc7, 0x15, 0xe3, 0x73})
		if err != nil {
			t.Fatal(err)
		}
		f.logs[i].Data = data
	}
	store := prices.NewMemoryStore()
	if err := store.Set(context.Background(), prices.TokenKey{ChainID: 1, Address: asset}, prices.Quote{USD: decimal.NewFromInt(2), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(context.Background(), prices.TokenKey{ChainID: 1, Address: f.rewardToken}, prices.Quote{USD: decimal.NewFromInt(3), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	s := NewStakeDAOService(prices.NewTokenPriceService(store, nil))
	config := s.networks["ethereum"]
	config.deploymentBlock = f.deployment
	s.networks["ethereum"] = config
	s.dial = func(context.Context, string) (stakeDAORPC, error) { return f, nil }
	return s, f
}

func TestGetPoolsUsesDiscoveredVaultsAndOneSnapshot(t *testing.T) {
	s, f := newPoolFixture(t)
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(f.queries, func(i, j int) bool { return f.queries[i].FromBlock.Cmp(f.queries[j].FromBlock) < 0 })
	if len(result.Pools) != 1 || len(result.Errors) != 1 || result.Errors[0].Field != "apr" || result.BlockNumber != f.latest || f.queries[0].FromBlock.Uint64() != f.deployment || !f.closed {
		t.Fatalf("unexpected result: %+v, queries: %+v, closed: %v", result, f.queries, f.closed)
	}
	pool := result.Pools[0]
	if pool.Name != "Stake DAO fixture Vault" || pool.AssetAmount == nil || *pool.AssetAmount != "1.234567" || pool.TVL == nil || *pool.TVL != 2.469134 || pool.APY != nil || pool.APR != nil || pool.APRStatus != "unavailable" {
		t.Fatalf("wrong metadata: %+v", pool)
	}
	if pool.AccountantAddress == nil || *pool.AccountantAddress != f.accountant || pool.RewardToken == nil || *pool.RewardToken != f.rewardToken || pool.RewardMechanism != "accountant" {
		t.Fatalf("wrong reward mechanism: %+v", pool)
	}
	for _, block := range f.callBlocks {
		if block != f.latest {
			t.Fatalf("call used block %d", block)
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"apr":null`) || !strings.Contains(string(encoded), `"apy":null`) {
		t.Fatalf("unknown values were presented as zero: %s", encoded)
	}
}

func TestGetPoolsReportsMissingMetadataAndDecimals(t *testing.T) {
	for _, method := range []string{"name", "decimals", "totalAssets"} {
		t.Run(method, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.callFailure = method
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Pools) != 1 || len(result.Errors) < 2 {
				t.Fatalf("missing failure diagnostics: %+v", result)
			}
			if method == "name" && result.Pools[0].Name != "FIX" {
				t.Fatal("symbol fallback missing")
			}
			if method != "name" && result.Pools[0].AssetAmount != nil {
				t.Fatal("amount computed from missing inputs")
			}
			if method == "decimals" && result.Pools[0].AssetDecimals != nil {
				t.Fatal("decimals guessed")
			}
		})
	}
}

func TestGetPoolsRejectsStaleRegistrationAndReorganizedSnapshot(t *testing.T) {
	s, f := newPoolFixture(t)
	f.vaults[common.HexToAddress("0xaa")] = common.HexToAddress("0xff")
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil || len(result.Pools) != 0 || len(result.Errors) != 1 || result.Errors[0].Field != "registration" {
		t.Fatalf("stale registration: %+v, %v", result, err)
	}
	s, f = newPoolFixture(t)
	f.reorg = true
	if _, err := s.GetPools(context.Background(), "ethereum"); err == nil {
		t.Fatal("reorganized snapshot accepted")
	}
	s, f = newPoolFixture(t)
	f.callFailure = "gauge"
	if _, err := s.GetPools(context.Background(), "ethereum"); err == nil || !f.closed {
		t.Fatal("failed Controller read was silently skipped")
	}
}

func TestUnsupportedNetworkDoesNotDial(t *testing.T) {
	s, f := newPoolFixture(t)
	_, err := s.GetPools(context.Background(), "base")
	if !errors.Is(err, ErrUnsupportedNetwork) || f.closed {
		t.Fatalf("unsupported network: %v", err)
	}
}

func TestTVLRemainsUnavailableWhenLPPriceIsMissing(t *testing.T) {
	s, _ := newPoolFixture(t)
	s.prices = prices.NewTokenPriceService(prices.NewMemoryStore(), nil)
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Pools) != 1 || result.Pools[0].TVL != nil || result.Pools[0].AssetAmount == nil {
		t.Fatalf("unknown TVL misrepresented: %+v", result)
	}
	found := false
	for _, issue := range result.Errors {
		if issue.Field == "tvl" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing price was silently ignored")
	}
}

func TestAccountantFailuresDoNotInventRewardsOrAPR(t *testing.T) {
	for _, method := range []string{"ACCOUNTANT", "accountant", "REWARD_TOKEN"} {
		t.Run(method, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.callFailure = method
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			pool := result.Pools[0]
			if pool.RewardToken != nil || pool.APR != nil || pool.APY != nil || pool.APRStatus != "unavailable" || pool.TVL == nil {
				t.Fatalf("reward failure produced invented values: %+v", pool)
			}
		})
	}
	s, f := newPoolFixture(t)
	f.controllerAccountant = common.HexToAddress("0x99")
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil || result.Pools[0].RewardToken != nil {
		t.Fatalf("mismatched Accountant accepted: %+v %v", result, err)
	}
}

func TestTVLZeroAssetsAndFractionalDecimals(t *testing.T) {
	store := prices.NewMemoryStore()
	asset := common.HexToAddress("0x11")
	if err := store.Set(context.Background(), prices.TokenKey{ChainID: 1, Address: asset}, prices.Quote{USD: decimal.RequireFromString("1.23"), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	s := &StakeDAOService{chainID: 1, prices: prices.NewTokenPriceService(store, nil)}
	for amount, want := range map[string]float64{"0": 0, "123.456789": 151.85185047, "0.000000000000000001": 1.23e-18} {
		pool := dto.StakeDAOPool{GaugeAddress: asset, AssetAddress: asset, AssetAmount: &amount}
		if issues := s.valuePool(context.Background(), &pool); len(issues) != 0 || pool.TVL == nil || *pool.TVL != want {
			t.Fatalf("TVL for %s: %+v %+v", amount, pool, issues)
		}
	}
}

type recordingPriceProvider struct {
	tokens []common.Address
	reads  []common.Address
}

func (p *recordingPriceProvider) EnsurePrices(_ context.Context, _ int64, tokens []common.Address) error {
	p.tokens = append(p.tokens, tokens...)
	return nil
}

func (p *recordingPriceProvider) GetTokenPrice(_ context.Context, token common.Address, _ int64) (decimal.Decimal, error) {
	p.reads = append(p.reads, token)
	return decimal.NewFromInt(2), nil
}

func TestEmptyVaultHasZeroTVLWithoutFetchingOrReadingPrices(t *testing.T) {
	s, fixture := newPoolFixture(t)
	fixture.totalAssets = new(big.Int)
	provider := &recordingPriceProvider{}
	s.prices = provider
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil || result == nil || len(result.Pools) != 1 {
		t.Fatalf("empty vault load failed: result=%+v err=%v", result, err)
	}
	pool := result.Pools[0]
	if pool.TVL == nil || *pool.TVL != 0 || pool.APR != nil || pool.APRStatus != "unavailable" || pool.Source != "onchain" {
		t.Fatalf("empty vault value misrepresented: %+v", pool)
	}
	if len(provider.tokens) != 0 || len(provider.reads) != 0 {
		t.Fatalf("empty vault requested USD quotes: fetch=%v reads=%v", provider.tokens, provider.reads)
	}
	for _, issue := range result.Errors {
		if issue.Field == "tvl" {
			t.Fatalf("known zero TVL reported as unknown: %+v", issue)
		}
	}
}

func TestPriceRequestsIncludeOnlyKnownPositiveRewardRates(t *testing.T) {
	for _, kind := range []string{"expired", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			s, fixture := newPoolFixture(t)
			fixture.fullPrimary = true
			fixture.extraTokens = []common.Address{common.HexToAddress("0xeeee")}
			fixture.extraFinish, fixture.extraRate = 0, big.NewInt(1000000000000000000)
			if kind == "unreadable" {
				fixture.callFailure = "rewardData"
			}
			provider := &recordingPriceProvider{}
			s.prices = provider
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil || result == nil || len(result.Pools) != 1 {
				t.Fatalf("reward test load failed: %+v %v", result, err)
			}
			pool := result.Pools[0]
			if !reflect.DeepEqual(provider.tokens, []common.Address{pool.AssetAddress, fixture.rewardToken}) {
				t.Fatalf("unused reward priced or active primary omitted: %v", provider.tokens)
			}
			if pool.TVL == nil || pool.APR == nil {
				t.Fatalf("active primary lost its valuation: %+v", pool)
			}
			if kind == "unreadable" && pool.APRStatus != "partial" {
				t.Fatal("unknown extra reward rate became complete/zero APR")
			}
		})
	}
}

func TestCurvePrimaryProjectionAppliesFeesToCorrectTargets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		sidecar bool
		policy  uint8
		want    string
	}{
		{"locker-checkpoint", false, 0, "0.1698"},
		{"locker-and-convex-checkpoint", true, 0, "1.1688"},
		{"locker-and-convex-harvest", true, 1, "1.17"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.fullPrimary = true
			f.sidecar = tc.sidecar
			f.policy = tc.policy
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			pool := result.Pools[0]
			if len(result.Errors) != 0 || len(pool.Rewards) != 1 || pool.Rewards[0].RatePerSecond == nil || *pool.Rewards[0].RatePerSecond != tc.want || pool.APRStatus != "available" || pool.APR == nil || pool.APY != nil {
				t.Fatalf("incorrect projection: %+v, issues %+v", pool, result.Errors)
			}
			eps, _ := decimal.RequireFromString(tc.want).Float64()
			want := eps * 3 * 31536000 / 2.469134 * 100
			if math.Abs(*pool.APR-want)/want > 1e-12 {
				t.Fatalf("APR %g want %g", *pool.APR, want)
			}
		})
	}
}

func TestVaultExtraRateScalingAndExpiredRewards(t *testing.T) {
	for _, expired := range []bool{false, true} {
		s, f := newPoolFixture(t)
		token := common.HexToAddress("0x55")
		f.extraTokens = []common.Address{token}
		f.extraFinish = 1
		// 0.01 reward token/sec with 6 token decimals and an additional 1e18 vault scale.
		f.extraRate = new(big.Int).Mul(big.NewInt(10000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
		if expired {
			f.extraFinish = 0
		}
		store := prices.NewMemoryStore()
		for address, value := range map[common.Address]int64{common.HexToAddress("0xcc"): 2, token: 7} {
			if err := store.Set(context.Background(), prices.TokenKey{ChainID: 1, Address: address}, prices.Quote{USD: decimal.NewFromInt(value), UpdatedAt: time.Now()}); err != nil {
				t.Fatal(err)
			}
		}
		s.prices = prices.NewTokenPriceService(store, nil)
		result, err := s.GetPools(context.Background(), "ethereum")
		if err != nil {
			t.Fatal(err)
		}
		pool := result.Pools[0]
		wantRate := "0.01"
		if expired {
			wantRate = "0"
		}
		if len(pool.Rewards) != 1 || pool.Rewards[0].RatePerSecond == nil || *pool.Rewards[0].RatePerSecond != wantRate || pool.APR == nil || pool.APRStatus != "partial" || pool.APY != nil {
			t.Fatalf("incorrect extra stream: %+v", pool)
		}
		wantAPR := 0.01 * 7 * 31536000 / 2.469134 * 100
		if expired {
			wantAPR = 0
		}
		if math.Abs(*pool.APR-wantAPR) > 1e-6 {
			t.Fatalf("APR %g want %g", *pool.APR, wantAPR)
		}
	}
}

func TestMissingExtraListDoesNotMarkPrimaryAPRComplete(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary = true
	f.callFailure = "getRewardTokens"
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	if result.Pools[0].APR == nil || result.Pools[0].APRStatus != "partial" || len(result.Errors) == 0 {
		t.Fatalf("missing reward list treated as complete: %+v", result)
	}
}

func TestKilledGaugeHasKnownZeroPrimaryRate(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary = true
	f.killed = true
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	pool := result.Pools[0]
	if pool.APR == nil || *pool.APR != 0 || pool.APRStatus != "available" {
		t.Fatalf("killed gauge: %+v %+v", pool, result.Errors)
	}
}

func TestSidecarIdentityDoesNotRequireGaugeAndRejectsUnrelatedPools(t *testing.T) {
	for _, tc := range []struct {
		name, override, stage string
		value                 common.Address
	}{
		{name: "common-getters-without-gauge"},
		{name: "wrong-asset", override: "asset", stage: "sidecar_asset", value: common.HexToAddress("0xdead")},
		{name: "wrong-receiver", override: "rewardReceiver", stage: "sidecar_reward_receiver", value: common.HexToAddress("0xdead")},
		{name: "wrong-controller", override: "PROTOCOL_CONTROLLER", stage: "sidecar_controller", value: common.HexToAddress("0xdead")},
		{name: "wrong-booster", override: "BOOSTER", stage: "sidecar_booster", value: common.HexToAddress("0xdead")},
		{name: "zero-reward-pool", override: "baseRewardPool", stage: "sidecar_reward_pool"},
		{name: "wrong-registered-asset", override: "poolAsset", stage: "sidecar_pool_identity", value: common.HexToAddress("0xdead")},
		{name: "wrong-registered-gauge", override: "poolGauge", stage: "sidecar_pool_identity", value: common.HexToAddress("0xdead")},
		{name: "wrong-registered-rewards", override: "poolRewards", stage: "sidecar_pool_identity", value: common.HexToAddress("0xdead")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.fullPrimary, f.sidecar = true, true
			f.sidecarIdentity = map[string]common.Address{}
			if tc.override != "" {
				f.sidecarIdentity[tc.override] = tc.value
			}
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			pool := result.Pools[0]
			if tc.override == "" {
				if len(result.Errors) != 0 || pool.APRStatus != "available" || len(pool.Rewards) != 1 || pool.Rewards[0].RatePerSecond == nil || *pool.Rewards[0].RatePerSecond != "1.1688" {
					t.Fatalf("Sidecar without gauge failed: %+v", result)
				}
			} else if len(result.Errors) != 1 || result.Errors[0].Stage != tc.stage || pool.APR != nil || pool.APRStatus != "unavailable" {
				t.Fatalf("incorrect Sidecar identity accepted: %+v", result)
			}
		})
	}
}

func TestPrimaryRewardDiagnosticsIdentifyTheFailedStep(t *testing.T) {
	for _, tc := range []struct{ method, stage, contract string }{
		{"inflation_rate", "gauge_inflation", "0xaa"},
		{"working_supply", "gauge_working_supply", "0xaa"},
		{"getAllocationTargets", "allocation_targets", "0x1235"},
		{"getProtocolFeePercent", "protocol_fee", "0x77"},
		{"getHarvestFeePercent", "harvest_fee", "0x77"},
		{"POLICY", "harvest_policy", "0xbb"},
		{"baseRewardPool", "sidecar_reward_pool", "0x1236"},
		{"rewardRate", "convex_reward_rate", "0x1237"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.fullPrimary, f.sidecar, f.callFailure = true, true, tc.method
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil || len(result.Errors) != 1 {
				t.Fatalf("failed diagnostic load: result=%+v err=%v", result, err)
			}
			issue := result.Errors[0]
			if issue.Code != "primary_reward_"+tc.stage || issue.Stage != tc.stage || issue.Method != tc.method || issue.Contract == nil || *issue.Contract != common.HexToAddress(tc.contract) || issue.GaugeAddress != common.HexToAddress("0xaa") || issue.Field != "apr" {
				t.Fatalf("wrong failure location: %+v", issue)
			}
			if result.Pools[0].APR != nil || result.Pools[0].APRStatus != "unavailable" || result.Pools[0].TVL == nil {
				t.Fatal("diagnostics changed the reward calculation")
			}
		})
	}
}

func TestPhysicalAllocationMismatchDoesNotOverstateAPR(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary = true
	f.physical = big.NewInt(9999999)
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	if result.Pools[0].APR != nil || result.Pools[0].APRStatus != "unavailable" || result.Pools[0].TVL == nil {
		t.Fatalf("mismatched allocation accepted: %+v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Stage != "allocation_match" || result.Errors[0].Method != "balanceOf" {
		t.Fatalf("allocation mismatch reason was lost: %+v", result.Errors)
	}
}

func TestMissingActiveExtraPriceMakesAPRPartial(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary = true
	f.extraTokens = []common.Address{common.HexToAddress("0x55")}
	f.extraFinish = 1
	f.extraRate = new(big.Int).Mul(big.NewInt(10000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	pool := result.Pools[0]
	if pool.APR == nil || pool.APRStatus != "partial" || len(pool.Rewards) != 2 || pool.Rewards[1].APR != nil {
		t.Fatalf("missing active price treated as zero/complete: %+v", pool)
	}
}
