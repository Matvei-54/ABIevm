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
	accountantSupply     *big.Int
	accountantSupplyFail bool
	accountantSupplyBad  bool
	assetDecimals        *uint8
	tokenDecimals        map[common.Address]uint8
	strategyAddress      common.Address
	totalAssets          *big.Int
	shutdown             map[common.Address]bool
	utilityRate          *big.Int
	utilityCalls         int
	convexSupply         *big.Int
	headerTime           uint64
	gaugeTokens          []common.Address
	gaugeRate            *big.Int
	gaugeFinish          uint64
	gaugeUpdated         uint64
	gaugeSupply          *big.Int
	gaugeCount           *big.Int
	gaugeLP              *common.Address
}

func (f *fixtureRPC) ChainID(context.Context) (*big.Int, error)   { return big.NewInt(1), nil }
func (f *fixtureRPC) BlockNumber(context.Context) (uint64, error) { return f.latest, nil }
func (f *fixtureRPC) HeaderByNumber(_ context.Context, number *big.Int) (*types.Header, error) {
	f.headerReads++
	h := &types.Header{Number: new(big.Int).Set(number), Time: f.headerTime}
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
	for _, abiJSON := range []string{protocolControllerABI, erc20NameABI, erc20SymbolABI, erc20DecimalsABI, rewardVaultERC4626ABI, rewardVaultAccountantABI, accountantRewardTokenABI, vaultV2ABI, accountantV2ABI, strategyV2ABI, sidecarV2ABI, allocatorV2ABI, curveGaugeABI, curveGaugeExtrasABI, curveControllerABI, convexPoolABI, convexBoosterABI, convexPoolUtilitiesABI} {
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
		case "lp_token":
			if f.gaugeLP != nil {
				return method.Outputs.Pack(*f.gaugeLP)
			}
			return method.Outputs.Pack(f.assets[*call.To])
		case "reward_count":
			if f.gaugeCount != nil {
				return method.Outputs.Pack(f.gaugeCount)
			}
			return method.Outputs.Pack(big.NewInt(int64(len(f.gaugeTokens))))
		case "reward_tokens":
			args, err := method.Inputs.Unpack(call.Data[4:])
			if err != nil {
				return nil, err
			}
			return method.Outputs.Pack(f.gaugeTokens[args[0].(*big.Int).Int64()])
		case "reward_data":
			rate := f.gaugeRate
			if rate == nil {
				rate = new(big.Int)
			}
			return method.Outputs.Pack(common.Address{}, common.HexToAddress("0x44"), new(big.Int).SetUint64(f.gaugeFinish), rate, new(big.Int).SetUint64(f.gaugeUpdated), new(big.Int))
		case "rewardRates":
			f.utilityCalls++
			rate := f.utilityRate
			if rate == nil {
				rate = helperCRVRate(big.NewInt(2000000000000000000), big.NewInt(1000000), 10000, 0)
			}
			cvx := common.HexToAddress("0x4e3FBD56CD56c3e72c1403e103b45Db9da5B9D2B")
			return method.Outputs.Pack([]common.Address{f.rewardToken, cvx, cvx}, []*big.Int{rate, new(big.Int), big.NewInt(7)})
		case "strategy":
			address := common.Address{}
			if f.fullPrimary {
				address = common.HexToAddress("0x1234")
				if f.strategyAddress != (common.Address{}) {
					address = f.strategyAddress
				}
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
			strategy := common.HexToAddress("0x1234")
			if f.strategyAddress != (common.Address{}) {
				strategy = f.strategyAddress
			}
			if *call.To == strategy {
				if f.physical != nil {
					return method.Outputs.Pack(f.physical)
				}
				return method.Outputs.Pack(big.NewInt(1234567))
			}
			return method.Outputs.Pack(big.NewInt(500000))
		case "totalSupply":
			if len(method.Inputs) == 0 && f.vaults[*call.To] != (common.Address{}) && f.gaugeSupply != nil {
				return method.Outputs.Pack(f.gaugeSupply)
			}
			if *call.To == common.HexToAddress("0x1237") && f.convexSupply != nil {
				return method.Outputs.Pack(f.convexSupply)
			}
			if len(method.Inputs) == 1 {
				if f.accountantSupplyFail {
					return nil, errors.New("accounting RPC failed: https://rpc.invalid/private-key")
				}
				if f.accountantSupplyBad {
					return []byte{1}, nil
				}
				if f.accountantSupply != nil {
					return method.Outputs.Pack(f.accountantSupply)
				}
				if f.totalAssets != nil {
					return method.Outputs.Pack(f.totalAssets)
				}
				return method.Outputs.Pack(big.NewInt(1234567))
			}
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
			if value, ok := f.tokenDecimals[*call.To]; ok {
				return method.Outputs.Pack(value)
			}
			if f.fullPrimary && *call.To == f.rewardToken {
				return method.Outputs.Pack(uint8(18))
			}
			if f.assetDecimals != nil {
				return method.Outputs.Pack(*f.assetDecimals)
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

func TestVaultExtraDistributionStatusDescribesOnlyTheScheduledStream(t *testing.T) {
	for _, tc := range []struct {
		name, status, rate string
		finish             uint32
		raw                int64
		failure            bool
	}{
		{"active", "active", "0.01", 1, 10000, false},
		{"not-funded", "not_started", "0", 0, 0, false},
		{"expired", "expired", "0", 0, 10000, false},
		{"zero-rate-period", "zero_rate", "0", 1, 0, false},
		{"unreadable", "unavailable", "", 1, 10000, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.extraTokens = []common.Address{common.HexToAddress("0x55")}
			f.extraFinish = tc.finish
			f.extraRate = new(big.Int).Mul(big.NewInt(tc.raw), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
			if tc.failure {
				f.callFailure = "rewardData"
			}
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Pools[0].Rewards) != 1 {
				t.Fatal("extra reward disappeared")
			}
			reward := result.Pools[0].Rewards[0]
			if reward.DistributionStatus != tc.status {
				t.Fatalf("wrong distribution state: %+v", reward)
			}
			if tc.failure {
				if reward.RatePerSecond != nil || reward.APR != nil {
					t.Fatal("unreadable stream became a confirmed zero")
				}
			} else if reward.RatePerSecond == nil || *reward.RatePerSecond != tc.rate {
				t.Fatalf("scheduled rate changed: %+v", reward)
			}
		})
	}
}

func TestConvexCVXIsCountedOnceAtTheVaultRateWithoutRepeatedFees(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.sidecar = true, true
	cvx := common.HexToAddress("0x4e3FBD56CD56c3e72c1403e103b45Db9da5B9D2B")
	f.extraTokens, f.extraFinish = []common.Address{cvx}, 1
	f.tokenDecimals = map[common.Address]uint8{cvx: 18}
	f.extraRate = decimal.RequireFromString("0.01").Shift(36).BigInt()
	store := prices.NewMemoryStore()
	for token, price := range map[common.Address]int64{common.HexToAddress("0xcc"): 2, f.rewardToken: 3, cvx: 7} {
		if err := store.Set(context.Background(), prices.TokenKey{ChainID: 1, Address: token}, prices.Quote{USD: decimal.NewFromInt(price), UpdatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	s.prices = prices.NewTokenPriceService(store, nil)
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	pool := result.Pools[0]
	if len(result.Errors) != 0 || len(pool.Rewards) != 2 || pool.APRStatus != "available" {
		t.Fatalf("incomplete or duplicated rewards: %+v", result)
	}
	var extra *dto.StakeDAOReward
	for i := range pool.Rewards {
		if pool.Rewards[i].Token == cvx {
			if extra != nil {
				t.Fatal("CVX counted more than once")
			}
			extra = &pool.Rewards[i]
		}
	}
	if extra == nil || extra.Source != "vault-extra" || extra.DistributionStatus != "active" || extra.RatePerSecond == nil || *extra.RatePerSecond != "0.01" || extra.APR == nil {
		t.Fatalf("wrong CVX stream: %+v", extra)
	}
	wantCVX := 0.01 * 7 * 31536000 / 2.469134 * 100
	wantCRV := 1.1688 * 3 * 31536000 / 2.469134 * 100
	if math.Abs(*extra.APR-wantCVX)/wantCVX > 1e-12 || pool.APR == nil || math.Abs(*pool.APR-(wantCVX+wantCRV))/(wantCVX+wantCRV) > 1e-12 {
		t.Fatalf("vault extra had repeated fees or an additional projected CVX component: %+v", pool)
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

func TestAllocationCoverageKeepsAccountedTVLAndActualNetRewardRate(t *testing.T) {
	for _, tc := range []struct {
		name, booked, physical, surplus, status string
		decimals                                uint8
	}{
		{"equal", "1.234567", "1.234567", "0.000000", "covered", 6},
		{"six-decimal-surplus", "1.234567", "1.237567", "0.003000", "surplus", 6},
		{"fxs-eth-surplus", "0.147548075075193763", "0.150548075075193763", "0.003000000000000000", "surplus", 18},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.fullPrimary = true
			f.strategyAddress = common.HexToAddress(curveSurplusStrategyEthereum)
			f.accountant = common.HexToAddress(curveSurplusAccountantEthereum)
			f.controllerAccountant = f.accountant
			f.assetDecimals = &tc.decimals
			f.totalAssets = decimal.RequireFromString(tc.booked).Shift(int32(tc.decimals)).BigInt()
			f.physical = decimal.RequireFromString(tc.physical).Shift(int32(tc.decimals)).BigInt()
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			pool := result.Pools[0]
			if len(result.Errors) != 0 || pool.Allocation == nil || pool.Allocation.Status != tc.status || pool.Allocation.AccountedAssets != tc.booked || pool.Allocation.PhysicalAssets != tc.physical || pool.Allocation.SurplusAssets != tc.surplus || pool.APRStatus != "available" || pool.APR == nil {
				t.Fatalf("wrong coverage or APR: pool=%+v errors=%+v", pool, result.Errors)
			}
			if pool.Rewards[0].RatePerSecond == nil || *pool.Rewards[0].RatePerSecond != "0.1698" {
				t.Fatalf("actual emissions were reduced by the accounted/physical ratio: %+v", pool.Rewards)
			}
			wantTVL := decimal.RequireFromString(tc.booked).Mul(decimal.NewFromInt(2))
			if pool.TVL == nil || !decimal.NewFromFloat(*pool.TVL).Sub(wantTVL).Abs().LessThan(decimal.RequireFromString("0.000000000000001")) {
				t.Fatal("physical surplus was added to vault TVL", pool.TVL)
			}
			wantAPR, _ := decimal.RequireFromString("0.1698").Mul(decimal.NewFromInt(3*31536000*100)).DivRound(wantTVL, 36).Float64()
			if math.Abs(*pool.APR-wantAPR)/wantAPR > 1e-12 {
				t.Fatalf("APR used a physical denominator: got %g want %g", *pool.APR, wantAPR)
			}
			for _, block := range f.callBlocks {
				if block != f.latest {
					t.Fatal("accounting and emissions used different snapshots")
				}
			}
		})
	}
}

func TestCoverageRejectsDeficitAndUnverifiedAccounting(t *testing.T) {
	for _, tc := range []struct{ name, stage string }{
		{"deficit", "allocation_coverage"},
		{"accounting-mismatch", "accounting_match"},
		{"supply-unavailable", "accountant_supply"},
		{"supply-malformed", "accountant_supply"},
		{"unknown-strategy", "allocation_model"},
		{"unknown-accountant", "allocation_model"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.fullPrimary = true
			f.strategyAddress = common.HexToAddress(curveSurplusStrategyEthereum)
			f.accountant = common.HexToAddress(curveSurplusAccountantEthereum)
			f.controllerAccountant = f.accountant
			f.physical = big.NewInt(1237567)
			switch tc.name {
			case "deficit":
				f.physical = big.NewInt(1234566)
			case "accounting-mismatch":
				f.accountantSupply = big.NewInt(1234568)
			case "supply-unavailable":
				f.accountantSupplyFail = true
			case "supply-malformed":
				f.accountantSupplyBad = true
			case "unknown-strategy":
				f.strategyAddress = common.HexToAddress("0x1234")
			case "unknown-accountant":
				f.accountant, f.controllerAccountant = common.HexToAddress("0x77"), common.HexToAddress("0x77")
			}
			result, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			pool := result.Pools[0]
			if len(result.Errors) != 1 || result.Errors[0].Stage != tc.stage || pool.TVL == nil || pool.APR != nil || pool.APRStatus != "unavailable" || len(pool.Rewards) != 0 {
				t.Fatalf("unverified/deficient assets got primary APR: pool=%+v errors=%+v", pool, result.Errors)
			}
			if tc.name == "deficit" && (pool.Allocation == nil || pool.Allocation.Status != "deficit" || pool.Allocation.SurplusAssets != "0.000000") {
				t.Fatal("deficit was misrepresented as surplus")
			}
			if strings.HasPrefix(tc.name, "unknown-") && (pool.Allocation == nil || pool.Allocation.Status != "unverified") {
				t.Fatal("unknown reward model was called verified")
			}
			raw, _ := json.Marshal(result)
			if strings.Contains(string(raw), "private-key") {
				t.Fatal("upstream error exposed in diagnostics")
			}
		})
	}
}

func TestCoveredEmptyVaultKeepsZeroTVLAndUnavailableAPR(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.killed = true, true
	f.totalAssets, f.physical = new(big.Int), new(big.Int)
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	pool := result.Pools[0]
	if pool.Allocation == nil || pool.Allocation.Status != "covered" || pool.TVL == nil || *pool.TVL != 0 || pool.APR != nil || pool.APRStatus != "unavailable" {
		t.Fatalf("empty vault became a positive APR: %+v", pool)
	}
}

func TestUnknownSurplusModelDoesNotOverstateAPR(t *testing.T) {
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
	if len(result.Errors) != 1 || result.Errors[0].Stage != "allocation_model" || result.Errors[0].Contract == nil {
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

func TestConvexDiagnosticsRetainSourcesAndNeverChangeRewardAmounts(t *testing.T) {
	for _, tc := range []struct{ name, status string }{
		{"matched", "matched"}, {"different-rate", "mismatch"}, {"unreadable", "unavailable"}, {"zero-supply", "zero_supply"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			s.client, s.chainID, s.snapshotTime = f, 1, 1
			f.rewardToken = common.HexToAddress("0xD533a949740bb3306d119CC777fa900bA034cd52")
			switch tc.name {
			case "different-rate":
				f.utilityRate = big.NewInt(1)
			case "unreadable":
				f.callFailure = "rewardRates"
			case "zero-supply":
				f.convexSupply = new(big.Int)
			}
			pool := dto.StakeDAOPool{GaugeAddress: common.HexToAddress("0xaa"), AssetAddress: common.HexToAddress("0xcc"), Diagnostics: &dto.StakeDAOPoolDiagnostics{
				Primary: &dto.PrimaryRateDiagnostic{ConvexGrossRate: "1", NetRate: "1.1688", Sidecars: []dto.ConvexPlacement{{Address: common.HexToAddress("0x1236"), PID: "1", RewardPool: common.HexToAddress("0x1237")}}},
			}}
			s.addPoolDiagnostics(context.Background(), big.NewInt(4500), &pool)
			d := pool.Diagnostics.Convex
			if d == nil || d.Status != tc.status || pool.Diagnostics.Primary.NetRate != "1.1688" || len(pool.Rewards) != 0 {
				t.Fatalf("control changed accounting or status: %+v", d)
			}
			if tc.name == "matched" {
				if len(d.Slots) != 3 || d.Slots[1].Token != d.Slots[2].Token || d.Slots[1].Source == d.Slots[2].Source || d.RoundingDifference == nil || *d.RoundingDifference != "0" || *d.OwnSidecarRate != "1" {
					t.Fatalf("units, duplicate source or placement lost: %+v", d)
				}
			}
			for _, number := range f.callBlocks {
				if number != 4500 {
					t.Fatal("control used another block")
				}
			}
		})
	}
}

func TestPrimaryDiagnosticBreakdownIsOptionalAndReconcilesFees(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s, f := newPoolFixture(t)
		f.fullPrimary, f.sidecar = true, true
		s.client, s.controller, s.chainID, s.diagnostics = f, f.controller, 1, enabled
		result, err := s.getPoolsSelected(context.Background(), "ethereum", &poolSample{limit: 1, gauges: []common.Address{common.HexToAddress("0xaa")}})
		if err != nil {
			t.Fatal(err)
		}
		p := result.Pools[0]
		if !enabled {
			if p.Diagnostics != nil || f.utilityCalls != 0 {
				t.Fatal("normal sample added control requests")
			}
			continue
		}
		d := p.Diagnostics.Primary
		if d == nil || d.LockerGrossRate != "0.2" || d.ConvexGrossRate != "1" || d.ProtocolFeeRate != "0.03" || d.HarvestFeeRate != "0.0012" || d.NetRate != "1.1688" {
			t.Fatalf("fee breakdown incorrect: %+v", d)
		}
		if len(p.Rewards) != 1 || p.Rewards[0].Source != "curve-primary-net" || *p.Rewards[0].RatePerSecond != d.NetRate {
			t.Fatal("diagnostic net differs from actual reward")
		}
	}
}

func TestHelperRateRespectsExpiryAndIntegerRounding(t *testing.T) {
	raw, supply := big.NewInt(7), big.NewInt(3)
	want := "2333333333333333333"
	if helperCRVRate(raw, supply, 10, 10).String() != want || helperCRVRate(raw, supply, 10, 11).Sign() != 0 || helperCRVRate(raw, new(big.Int), 10, 1).Sign() != 0 {
		t.Fatal("expiry/units/zero-supply comparison incorrect")
	}
	if raw.Int64() != 7 || supply.Int64() != 3 {
		t.Fatal("comparison mutated original inputs")
	}
}

func TestDiagnosticQuoteRetainsThePriceActuallyUsed(t *testing.T) {
	ctx := context.Background()
	store := prices.NewMemoryStore()
	token := common.HexToAddress("0xaa")
	q := prices.Quote{USD: decimal.NewFromInt(2), UpdatedAt: time.Now(), Source: "original", BlockNumber: 100}
	if err := store.Set(ctx, prices.TokenKey{ChainID: 1, Address: token}, q); err != nil {
		t.Fatal(err)
	}
	s := NewStakeDAOService(prices.NewTokenPriceService(store, nil))
	s.chainID, s.diagnostics = 1, true
	pool := dto.StakeDAOPool{AssetAddress: token}
	used, err := s.valuationPrice(ctx, &pool, token)
	if err != nil || used.String() != "2" {
		t.Fatal("initial valuation failed")
	}
	q.USD, q.Source, q.BlockNumber = decimal.NewFromInt(3), "refreshed", 101
	if err := store.Set(ctx, prices.TokenKey{ChainID: 1, Address: token}, q); err != nil {
		t.Fatal(err)
	}
	s.addPoolDiagnostics(ctx, big.NewInt(100), &pool)
	if len(pool.Diagnostics.Prices) != 1 || pool.Diagnostics.Prices[0].USD != "2" || pool.Diagnostics.Prices[0].BlockNumber != 100 {
		t.Fatal("report substituted a later quote")
	}
}

func TestReaderReferenceSeparatesGlobalRateCapAndFeePolicies(t *testing.T) {
	for _, globalRate := range []string{"1000000000000000000", "2000000000000000000"} {
		crv := common.HexToAddress("0xd533a949740bb3306d119cc777fa900ba034cd52")
		asset := common.HexToAddress("0xcc")
		amount, balance, own, net := "100", "10000000000000000000", "0.1", "1.769475"
		d18 := uint8(18)
		p := dto.StakeDAOPool{AssetAddress: asset, AssetAmount: &amount, AssetDecimals: &d18, RewardToken: &crv,
			Allocation: &dto.StakeDAOAllocation{Status: "covered"},
			Diagnostics: &dto.StakeDAOPoolDiagnostics{
				Primary: &dto.PrimaryRateDiagnostic{LockerGrossRate: "2", ConvexGrossRate: own, ProtocolFeeFraction: "0.165", HarvestFeeFraction: "0.00025", NetRate: net, Sidecars: []dto.ConvexPlacement{{PID: "1"}},
					LockerInputs: &dto.LockerRateDiagnostic{InflationRaw: "10000000000000000000", RelativeWeightRaw: "1000000000000000000", WorkingSupplyRaw: "100000000000000000000"}},
				Convex: &dto.ConvexRateDiagnostic{Status: "matched", RewardRateRaw: &globalRate, SidecarBalanceRaw: &balance},
				Prices: []dto.PriceDiagnostic{{Token: asset, USD: "1", BlockNumber: 123}, {Token: crv, USD: "1", BlockNumber: 123}},
			}}
		addReaderModel(123, &p)
		r := p.Diagnostics.ReaderModel
		if r.Status != "available" || r.ReferenceAPR == nil {
			t.Fatalf("reference missing: %+v", r)
		}
		factor := 31536000.0
		if math.Abs(*r.ActualCRVAPR-1.769475*factor) > 1e-7 || math.Abs(*r.WithoutHarvestFeeAPR-1.77*factor) > 1e-7 || math.Abs(*r.GlobalProtocolFeeAPR-1.7535*factor) > 1e-7 || math.Abs(*r.ReferenceAPR-2.505*factor) > 1e-7 {
			t.Fatalf("rate allocation / cap / fees mixed: %+v", r)
		}
		if *r.CappedReferenceSidecarCRVRate != "1" || p.Diagnostics.Primary.NetRate != net || len(p.Rewards) != 0 {
			t.Fatal("reference changed actual rewards or did not cap the whole pool rate")
		}
		p.Diagnostics.Prices[0].BlockNumber = 122
		addReaderModel(123, &p)
		if p.Diagnostics.ReaderModel.Status != "unavailable" {
			t.Fatal("reference accepted another block's price")
		}
	}
}

func TestAPRReasonsDistinguishEmptyMissingPricesAndUnsupportedModel(t *testing.T) {
	one, zero := "1", "0"
	tvl := 2.0
	gauge := common.HexToAddress("0x11")
	for _, tc := range []struct {
		name   string
		pool   dto.StakeDAOPool
		issues []dto.StakeDAOPoolError
		want   []string
	}{
		{"empty", dto.StakeDAOPool{AssetAmount: &zero}, nil, []string{"empty_vault"}},
		{"asset-price", dto.StakeDAOPool{AssetAmount: &one, RewardsComplete: true}, nil, []string{"asset_price_missing"}},
		{"reward-data", dto.StakeDAOPool{AssetAmount: &one, TVL: &tvl}, nil, []string{"reward_data_missing"}},
		{"unsupported", dto.StakeDAOPool{AssetAmount: &one, TVL: &tvl, RewardMechanism: "unsupported"}, nil, []string{"unsupported_model"}},
		{"reward-price", dto.StakeDAOPool{AssetAmount: &one, TVL: &tvl, RewardsComplete: true}, []dto.StakeDAOPoolError{{GaugeAddress: gauge, Code: "reward_price_missing"}}, []string{"reward_price_missing"}},
		{"other-gauge", dto.StakeDAOPool{AssetAmount: &one, TVL: &tvl}, []dto.StakeDAOPoolError{{GaugeAddress: common.HexToAddress("0x22"), Code: "unsupported_model"}}, []string{"reward_data_missing"}},
		{"available-zero", dto.StakeDAOPool{AssetAmount: &one, TVL: &tvl, APRStatus: "available"}, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.pool.GaugeAddress = gauge
			setAPRReasons(&tc.pool, tc.issues)
			if !reflect.DeepEqual(tc.pool.APRReasons, tc.want) {
				t.Fatalf("got %v want %v", tc.pool.APRReasons, tc.want)
			}
		})
	}
}

func TestYieldBreakdownSeparatesGaugeProjectionFromVaultPayout(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.headerTime = true, 1000
	token := common.HexToAddress("0x66")
	f.gaugeTokens, f.extraTokens = []common.Address{token}, []common.Address{token}
	f.gaugeRate, f.gaugeFinish = big.NewInt(1), 2000 // one raw 6-decimal token/sec
	f.gaugeSupply = big.NewInt(1000000000)           // 1000 six-decimal LP
	f.extraRate, f.extraFinish = big.NewInt(1000000000000000000), 2000
	f.tokenDecimals = map[common.Address]uint8{token: 6}
	store := prices.NewMemoryStore()
	for address, price := range map[common.Address]int64{common.HexToAddress("0xcc"): 2, f.rewardToken: 3, token: 3} {
		if err := store.Set(context.Background(), prices.TokenKey{ChainID: 1, Address: address}, prices.Quote{USD: decimal.NewFromInt(price), UpdatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	s.prices = prices.NewTokenPriceService(store, nil)
	r, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Pools[0]
	y := p.Yield
	if y == nil || y.GaugeExtra.APR == nil || math.Abs(*y.GaugeExtra.APR-4.7304) > 1e-10 || *y.GaugeExtra.Rewards[0].RatePerSecond != "0.000001" {
		t.Fatalf("incorrect gauge units: %+v", y)
	}
	if p.APR == nil || y.CRV.APR == nil || y.VaultExtra.APR == nil || math.Abs(*p.APR-*y.CRV.APR-*y.VaultExtra.APR) > 1e-6 {
		t.Fatalf("gauge projection was double counted: %+v", p)
	}
	if p.APRScope != "vault_rewards" || y.Status != "partial" || y.Merkl.APR != nil || y.Trading.APY != nil || y.Trading.WindowDays != nil || y.Merkl.Status != "unavailable" {
		t.Fatalf("unknown components fabricated: %+v", y)
	}
	if !reflect.DeepEqual(y.IncludedInAPR, []string{"crv", "vaultExtra"}) {
		t.Fatalf("ambiguous aggregation: %+v", y)
	}
}

func TestGaugeYieldMissingDataNeverTurnsIntoZero(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*fixtureRPC)
		status string
		zero   bool
	}{
		{"no-rewards", func(f *fixtureRPC) { f.gaugeTokens = nil }, "available", true},
		{"expired", func(f *fixtureRPC) { f.gaugeFinish = 900 }, "available", true},
		{"not-started", func(f *fixtureRPC) { f.gaugeRate = new(big.Int); f.gaugeFinish = 0 }, "available", true},
		{"future-update", func(f *fixtureRPC) { f.gaugeUpdated = 1001 }, "unavailable", false},
		{"missing-price", func(f *fixtureRPC) {}, "unavailable", false},
		{"missing-count", func(f *fixtureRPC) { f.callFailure = "reward_count" }, "unavailable", false},
		{"missing-data", func(f *fixtureRPC) { f.callFailure = "reward_data" }, "unavailable", false},
		{"duplicate-token", func(f *fixtureRPC) { f.gaugeTokens = append(f.gaugeTokens, f.gaugeTokens[0]) }, "unavailable", false},
		{"too-many-tokens", func(f *fixtureRPC) { f.gaugeCount = big.NewInt(9) }, "unavailable", false},
		{"wrong-lp", func(f *fixtureRPC) { wrong := common.HexToAddress("0xff"); f.gaugeLP = &wrong }, "unavailable", false},
		{"zero-supply", func(f *fixtureRPC) { f.gaugeSupply = new(big.Int) }, "unavailable", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.fullPrimary, f.headerTime = true, 1000
			f.gaugeTokens = []common.Address{common.HexToAddress("0x66")}
			f.gaugeRate, f.gaugeFinish = big.NewInt(1), 2000
			tc.change(f)
			r, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			p := r.Pools[0]
			g := p.Yield.GaugeExtra
			if g.Status != tc.status || tc.zero && (g.APR == nil || *g.APR != 0) || !tc.zero && g.APR != nil {
				t.Fatalf("unknown became zero or status is wrong: %+v", g)
			}
			if p.APR == nil || p.APRStatus != "available" || p.Yield.CRV.Status != "available" {
				t.Fatalf("optional gauge data corrupted vault income: %+v", p)
			}
		})
	}
}

func TestYieldComponentsAreIndependentOfPrimaryAndExtraFailures(t *testing.T) {
	zero, one := float64(0), float64(1)
	p := dto.StakeDAOPool{TVL: &one, VaultExtrasComplete: true}
	if c := streamComponent(&p, "vault-extra", "active_vault_distribution", false); c.APR == nil || *c.APR != 0 || c.Status != "available" {
		t.Fatalf("verified empty extra list depends on CRV: %+v", c)
	}
	p.VaultExtrasComplete = false
	p.Rewards = []dto.StakeDAOReward{{Source: "curve-primary-net", APR: &zero}, {Source: "vault-extra"}}
	if c := streamComponent(&p, "curve-primary-net", "net_vault_stream", true); c.APR == nil || *c.APR != 0 || c.Status != "available" {
		t.Fatalf("known zero CRV depends on missing extra price: %+v", c)
	}
	if c := streamComponent(&p, "vault-extra", "active_vault_distribution", false); c.APR != nil || c.Status != "unavailable" {
		t.Fatalf("unknown extras became zero: %+v", c)
	}
}

func TestGaugeYieldPartialPriceCoverage(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.headerTime = true, 1000
	f.gaugeTokens = []common.Address{common.HexToAddress("0x66"), f.rewardToken}
	f.gaugeRate, f.gaugeFinish = big.NewInt(1), 2000
	r, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Pools[0]
	g := p.Yield.GaugeExtra
	if g.Status != "partial" || g.APR == nil || *g.APR <= 0 || len(g.Rewards) != 2 || g.Rewards[0].APR != nil || g.Rewards[0].Reason != "price_missing" || g.Rewards[1].APR == nil {
		t.Fatalf("missing price discarded known rewards or became zero: %+v", g)
	}
	if p.APRStatus != "available" || p.APR == nil || p.Yield.CRV.APR == nil || *p.APR != *p.Yield.CRV.APR {
		t.Fatalf("gauge projection changed vault APR: %+v", p)
	}
}
