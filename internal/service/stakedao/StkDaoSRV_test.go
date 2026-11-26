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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
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
	paused               bool
	weight               *big.Int
	malformedPause       bool
	feeHistory           bool
	feeBlock             uint64
	feeScenario          string
	feeCalls             int
	feeRampTime          uint64
	feeHeaderReads       map[uint64]int
}

func (f *fixtureRPC) ChainID(context.Context) (*big.Int, error)   { return big.NewInt(1), nil }
func (f *fixtureRPC) BlockNumber(context.Context) (uint64, error) { return f.latest, nil }
func (f *fixtureRPC) HeaderByNumber(_ context.Context, number *big.Int) (*types.Header, error) {
	f.headerReads++
	h := &types.Header{Number: new(big.Int).Set(number), Time: f.headerTime}
	if f.feeHistory {
		h.Time = 1700000000 + number.Uint64()*12
		if f.feeScenario == "historical-reorg" && number.Uint64() != f.latest {
			if f.feeHeaderReads == nil {
				f.feeHeaderReads = map[uint64]int{}
			}
			f.feeHeaderReads[number.Uint64()]++
			if f.feeHeaderReads[number.Uint64()] > 1 {
				h.Extra = []byte("changed")
			}
		}
	}
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
	if f.feeHistory {
		f.feeBlock = block.Uint64()
		if f.feeScenario == "archive" && f.feeBlock < f.latest {
			return nil, errors.New("Archive requests require a personal token: https://rpc.invalid/private-key")
		}
	}
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
	for _, abiJSON := range []string{protocolControllerABI, controllerStateABI, erc20NameABI, erc20SymbolABI, erc20DecimalsABI, rewardVaultERC4626ABI, rewardVaultAccountantABI, accountantRewardTokenABI, vaultV2ABI, accountantV2ABI, strategyV2ABI, sidecarV2ABI, allocatorV2ABI, curveGaugeABI, curveGaugeExtrasABI, curveControllerABI, convexPoolABI, convexBoosterABI, convexPoolUtilitiesABI, feePoolABI} {
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
		case "get_virtual_price", "initial_A", "future_A", "initial_A_time", "future_A_time":
			f.feeCalls++
			value := big.NewInt(10000)
			if method.Name == "get_virtual_price" {
				value = new(big.Int).Add(big.NewInt(1000000000000000000), new(big.Int).Mul(new(big.Int).SetUint64(f.feeBlock), big.NewInt(1000000000)))
				if f.feeScenario == "decrease" {
					value = new(big.Int).Sub(big.NewInt(2000000000000000000), value)
				}
			}
			if method.Name == "initial_A_time" || method.Name == "future_A_time" {
				value = new(big.Int)
			}
			if f.feeRampTime > 0 && 1700000000+f.feeBlock*12 >= f.feeRampTime {
				if method.Name == "future_A" {
					value = big.NewInt(20000)
				}
				if method.Name == "initial_A_time" {
					value = new(big.Int).SetUint64(f.feeRampTime)
				}
				if method.Name == "future_A_time" {
					value = new(big.Int).SetUint64(f.feeRampTime + 86400)
				}
			}
			return method.Outputs.Pack(value)
		case "coins":
			args, _ := method.Inputs.Unpack(call.Data[4:])
			coin := common.HexToAddress("0x11")
			if args[0].(*big.Int).Int64() == 1 {
				coin = common.HexToAddress("0x22")
			}
			if f.feeScenario == "wrong-coins" {
				coin = common.HexToAddress("0xff")
			}
			return method.Outputs.Pack(coin)
		case "get_pool_asset_types":
			kinds := []uint8{0, 0}
			if f.feeScenario == "dynamic-rates" {
				kinds[0] = 3
			}
			return method.Outputs.Pack(kinds)
		case "stored_rates":
			return method.Outputs.Pack([]*big.Int{big.NewInt(1000000000000000000), new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)})
		case "isPaused":
			if f.malformedPause {
				return []byte{1}, nil
			}
			return method.Outputs.Pack(f.paused)
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
			if f.weight != nil {
				return method.Outputs.Pack(f.weight)
			}
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
	if p.APRScope != "vault_rewards" || y.Status != "partial" || y.Merkl.APR != nil || y.Trading.APY != nil || y.Trading.WindowDays == nil || *y.Trading.WindowDays != 30 || y.Merkl.Status != "unavailable" {
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

func TestPoolStateDoesNotDependOnUSDPriceOrAPR(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.weight = true, new(big.Int)
	s.prices = prices.NewTokenPriceService(prices.NewMemoryStore(), nil)
	r, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Pools[0]
	state := p.State
	if p.APR != nil || p.APRStatus != "unavailable" || state == nil || state.Registered == nil || !*state.Registered || state.Shutdown == nil || *state.Shutdown || state.HasAssets == nil || !*state.HasAssets || state.GaugeKilled == nil || *state.GaugeKilled || state.ProtocolPaused == nil || *state.ProtocolPaused || state.HasCRVWeight == nil || *state.HasCRVWeight || state.CRVRelativeWeight == nil || *state.CRVRelativeWeight != "0" || len(state.Issues) != 0 {
		t.Fatalf("price or APR availability leaked into state: %+v", p)
	}
}

func TestPoolStateUnknownGettersAreNotFalse(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		s, f := newPoolFixture(t)
		f.fullPrimary = true
		f.malformedPause = malformed
		if !malformed {
			f.callFailure = "isPaused"
		}
		r, err := s.GetPools(context.Background(), "ethereum")
		if err != nil {
			t.Fatal(err)
		}
		p := r.Pools[0]
		if p.State.ProtocolPaused != nil || len(p.State.Issues) != 1 || p.State.Issues[0].Field != "protocolPaused" || p.State.GaugeKilled == nil || *p.State.GaugeKilled || p.APRStatus != "available" {
			t.Fatalf("unknown pause became false or changed APR: %+v", p)
		}
	}
}

func TestHasAssetsDoesNotRequireTokenDecimals(t *testing.T) {
	s, f := newPoolFixture(t)
	f.callFailure = "decimals"
	r, err := s.GetGaugeStatesAt(context.Background(), "ethereum", []common.Address{common.HexToAddress("0xaa")}, 0)
	if err != nil {
		t.Fatal(err)
	}
	row := r.Gauges[0]
	if row.AssetAmount != nil || row.State.HasAssets == nil || !*row.State.HasAssets {
		t.Fatalf("raw asset presence depends on decimals: %+v", row)
	}
}

func TestPauseAndKilledFlagsDoNotMeanEmptyOrUnavailable(t *testing.T) {
	for _, killed := range []bool{false, true} {
		s, f := newPoolFixture(t)
		f.fullPrimary, f.paused, f.killed = true, true, killed
		r, err := s.GetPools(context.Background(), "ethereum")
		if err != nil {
			t.Fatal(err)
		}
		p := r.Pools[0]
		if p.State.ProtocolPaused == nil || !*p.State.ProtocolPaused || p.State.GaugeKilled == nil || *p.State.GaugeKilled != killed || p.State.HasCRVWeight == nil || !*p.State.HasCRVWeight || p.State.HasAssets == nil || !*p.State.HasAssets || p.APRStatus != "available" || p.APR == nil {
			t.Fatalf("independent state flags conflated: %+v", p)
		}
		if killed && *p.APR != 0 || !killed && *p.APR <= 0 {
			t.Fatalf("existing killed handling changed: %+v", p)
		}
	}
}

func TestStateOnlyIncludesShutdownAndUnregisteredWithoutPrices(t *testing.T) {
	s, f := newPoolFixture(t)
	f.paused, f.killed = true, true
	f.shutdown = map[common.Address]bool{common.HexToAddress("0xaa"): true}
	f.totalAssets, f.weight = new(big.Int), new(big.Int)
	provider := &recordingPriceProvider{}
	s.prices = provider
	gauges := []common.Address{common.HexToAddress("0xaa"), common.HexToAddress("0xff")}
	r, err := s.GetGaugeStatesAt(context.Background(), "ethereum", gauges, 4500)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Gauges) != 2 || r.BlockNumber != 4500 || r.BlockHash == nil || r.RPCAttempts == 0 || len(f.queries) != 0 || f.codeReads != 0 || len(provider.tokens) != 0 || len(provider.reads) != 0 || f.headerReads != 2 {
		t.Fatalf("state-only used history/prices or failed snapshot checks: %+v", r)
	}
	registered, absent := r.Gauges[0], r.Gauges[1]
	if registered.State.Registered == nil || !*registered.State.Registered || registered.State.Shutdown == nil || !*registered.State.Shutdown || registered.State.ProtocolPaused == nil || !*registered.State.ProtocolPaused || registered.State.GaugeKilled == nil || !*registered.State.GaugeKilled || registered.State.HasAssets == nil || *registered.State.HasAssets || registered.State.HasCRVWeight == nil || *registered.State.HasCRVWeight || registered.AssetAmount == nil || *registered.AssetAmount != "0" {
		t.Fatalf("shutdown/capped/killed/empty conflated: %+v", registered)
	}
	if absent.State.Registered == nil || *absent.State.Registered || absent.VaultAddress != nil || absent.AssetAmount != nil || absent.State.Shutdown != nil || absent.State.ProtocolPaused != nil || absent.State.HasAssets != nil || absent.State.GaugeKilled != nil || absent.State.CRVRelativeWeight != nil {
		t.Fatalf("absent registration became a closed/empty pool: %+v", absent)
	}
	for _, b := range f.callBlocks {
		if b != 4500 {
			t.Fatalf("different snapshot used: %d", b)
		}
	}
}

func TestStateOnlyFailureValidationAndIndependentFlags(t *testing.T) {
	for _, tc := range []struct {
		name          string
		fail          string
		invalidWeight bool
	}{
		{"registration", "gauge", false}, {"killed", "is_killed", false},
		{"weight", "gauge_relative_weight", false}, {"assets", "totalAssets", false},
		{"weight-bounds", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, f := newPoolFixture(t)
			f.callFailure = tc.fail
			if tc.invalidWeight {
				f.weight = new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil)
			}
			r, err := s.GetGaugeStatesAt(context.Background(), "ethereum", []common.Address{common.HexToAddress("0xaa")}, 0)
			if err != nil {
				t.Fatal(err)
			}
			state := r.Gauges[0].State
			if len(state.Issues) == 0 {
				t.Fatalf("missing failure: %+v", state)
			}
			switch tc.name {
			case "registration":
				if state.Registered != nil || state.Shutdown != nil {
					t.Fatalf("failed registration became false: %+v", state)
				}
			case "killed":
				if state.GaugeKilled != nil || state.HasCRVWeight == nil || state.HasAssets == nil {
					t.Fatalf("flags not independent: %+v", state)
				}
			case "weight", "weight-bounds":
				if state.HasCRVWeight != nil || state.CRVRelativeWeight != nil || state.GaugeKilled == nil {
					t.Fatalf("invalid weight became zero: %+v", state)
				}
			case "assets":
				if state.HasAssets != nil || state.Registered == nil || state.ProtocolPaused == nil {
					t.Fatalf("failed assets became empty: %+v", state)
				}
			}
		})
	}
	s, _ := newPoolFixture(t)
	for _, gauges := range [][]common.Address{nil, {{}}, {common.HexToAddress("0xaa"), common.HexToAddress("0xaa")}, {common.HexToAddress("0x1"), common.HexToAddress("0x2"), common.HexToAddress("0x3"), common.HexToAddress("0x4"), common.HexToAddress("0x5"), common.HexToAddress("0x6")}} {
		if _, err := s.GetGaugeStatesAt(context.Background(), "ethereum", gauges, 0); err == nil {
			t.Fatal("invalid gauge selection accepted")
		}
	}
	s, f := newPoolFixture(t)
	f.reorg = true
	if _, err := s.GetGaugeStatesAt(context.Background(), "ethereum", []common.Address{common.HexToAddress("0xaa")}, 0); err == nil || !strings.Contains(err.Error(), "reorganized") {
		t.Fatalf("state reorg accepted: %v", err)
	}
}

func newFeeFixture(t *testing.T, ng bool) (*StakeDAOService, *fixtureRPC) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.feeHistory, f.latest = true, true, 300000
	pool := common.HexToAddress("0xcc")
	s.trading.routes = map[common.Address]feePoolRoute{pool: {pool: pool, coins: [2]common.Address{common.HexToAddress("0x11"), common.HexToAddress("0x22")}, ng: ng}}
	return s, f
}

func TestTradingHistoryComputesSevenAndThirtyDaysWithoutMixingAPR(t *testing.T) {
	s, f := newFeeFixture(t, true)
	r, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Pools[0]
	tr := p.Yield.Trading
	if tr.Status != "available" || p.APY == nil || tr.APY == nil || p.APYScope != "trading_fees" || len(tr.Windows) != 2 || *tr.WindowDays != 30 || p.APRStatus != "available" || *p.APR != *p.Yield.CRV.APR {
		t.Fatalf("wrong yield scopes or windows: %+v", p)
	}
	for i, days := range []int{7, 30} {
		w := tr.Windows[i]
		if w.Status != "available" || w.APY == nil || w.CoveredSeconds == nil || *w.CoveredSeconds != uint64(days)*86400 || w.StartBlockHash == nil || w.EndBlock != f.latest {
			t.Fatalf("incomplete window: %+v", w)
		}
		start := f.latest - uint64(days)*7200
		ratio := float64(1000000000000000000+f.latest*1000000000) / float64(1000000000000000000+start*1000000000)
		want := math.Expm1(math.Log(ratio)*365/float64(days)) * 100
		if math.Abs(*w.APY-want) > 1e-10 {
			t.Fatalf("bad annualization: got %v want %v", *w.APY, want)
		}
		if days == 30 && *p.APY != *w.APY {
			t.Fatal("main APY is not the monthly value")
		}
	}
	before := f.feeCalls
	s.client, s.chainID, s.snapshotTime = f, 1, r.BlockTimestamp
	head, err := f.HeaderByNumber(context.Background(), new(big.Int).SetUint64(f.latest))
	if err != nil {
		t.Fatal(err)
	}
	s.addTradingHistory(context.Background(), head, r.Pools)
	if f.feeCalls != before {
		t.Fatal("hash-keyed fee snapshots were fetched twice")
	}
}

func TestTradingHistoryDoesNotSubstituteWeekForMonth(t *testing.T) {
	s, f := newFeeFixture(t, false)
	f.feeRampTime = 1700000000 + f.latest*12 - 14*86400
	r, err := s.GetPools(context.Background(), "ethereum")
	if err != nil {
		t.Fatal(err)
	}
	p := r.Pools[0]
	tr := p.Yield.Trading
	if tr.Status != "partial" || tr.Windows[0].APY == nil || tr.Windows[1].APY != nil || tr.APY != nil || p.APY != nil || tr.Windows[1].Reasons[0] != "amplification_changed_in_window" {
		t.Fatalf("week substituted for month or ramp ignored: %+v", tr)
	}
}

func TestTradingHistoryRejectsUnverifiedGrowthAndPreservesCRV(t *testing.T) {
	for _, scenario := range []string{"archive", "wrong-coins", "dynamic-rates", "decrease", "historical-reorg"} {
		t.Run(scenario, func(t *testing.T) {
			s, f := newFeeFixture(t, true)
			f.feeScenario = scenario
			r, err := s.GetPools(context.Background(), "ethereum")
			if err != nil {
				t.Fatal(err)
			}
			p := r.Pools[0]
			if p.APY != nil || p.Yield.Trading.Status != "unavailable" || len(p.Yield.Trading.Reasons) == 0 || p.APR == nil || p.APRStatus != "available" {
				t.Fatalf("invalid history became APY or invalidated CRV: %+v", p)
			}
			if scenario == "archive" && p.Yield.Trading.Reasons[0] != "archive_rpc_required" {
				t.Fatalf("archive access requirement lost: %+v", p.Yield.Trading)
			}
			raw, _ := json.Marshal(r)
			if strings.Contains(string(raw), "private-key") {
				t.Fatal("history leaked RPC credentials")
			}
		})
	}
}

func TestFeeGrowthZeroAndWindowGuards(t *testing.T) {
	point := feeSnapshot{virtualPrice: big.NewInt(1000000000000000000), initialA: big.NewInt(10000), futureA: big.NewInt(10000)}
	apy, observed, err := feeGrowth(point, point, 1000000, 1000000+7*86400)
	if err != nil || apy == nil || *apy != 0 || observed == nil || *observed != 0 {
		t.Fatalf("confirmed zero growth unavailable: %v", err)
	}
	_, _, err = feeGrowth(point, point, 1000000, 1000000)
	if err == nil {
		t.Fatal("zero duration accepted")
	}
	s, f := newFeeFixture(t, false)
	head, _ := f.HeaderByNumber(context.Background(), new(big.Int).SetUint64(f.latest))
	cache := map[uint64]*types.Header{f.latest: head}
	start, err := findFeeBoundary(context.Background(), f, head, 30, cache)
	if err != nil || start.Time > head.Time-30*86400 || head.Time-30*86400-start.Time > 12 {
		t.Fatalf("wrong timestamp boundary: %+v %v", start, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := findFeeBoundary(ctx, s.client, head, 30, map[uint64]*types.Header{}); err == nil {
		t.Fatal("canceled lookup accepted")
	}
}

// Tests live with the existing fixtures to avoid scattering new test files.
type feeModelRPC struct {
	*fixtureRPC
	scanFailure     bool
	scanError       error
	identityFailure bool
	scans           []ethereum.FilterQuery
}

func (f *feeModelRPC) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scans = append(f.scans, q)
	if f.scanError != nil {
		return nil, f.scanError
	}
	if f.scanFailure {
		return nil, errors.New("https://rpc.invalid/private-key archive unavailable")
	}
	return []types.Log{}, nil
}
func (f *feeModelRPC) CallContract(ctx context.Context, c ethereum.CallMsg, b *big.Int) ([]byte, error) {
	aggregate, _ := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
	v, err := aggregate.Methods["aggregate3"].Inputs.Unpack(c.Data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(v[0], new([]multicallInput)).(*[]multicallInput)
	out := []multicallOutput{}
	for _, c := range calls {
		var result []byte
		var last error
		for _, raw := range []string{weETHFeeABI, rewardRouteABI, feePoolABI, erc20DecimalsABI} {
			a, _ := abi.JSON(strings.NewReader(raw))
			m, e := a.MethodById(c.CallData[:4])
			if e != nil {
				continue
			}
			args, _ := m.Inputs.Unpack(c.CallData[4:])
			var values []interface{}
			switch m.Name {
			case "coins":
				i := int(args[0].(*big.Int).Int64())
				values = []interface{}{weETHFeeCoins[i]}
			case "balances":
				values = []interface{}{new(big.Int).Mul(big.NewInt(100), big.NewInt(1000000000000000000))}
			case "decimals":
				values = []interface{}{uint8(18)}
			case "stored_rates":
				values = []interface{}{[]*big.Int{big.NewInt(1000000000000000000), big.NewInt(2000000000000000000)}}
			case "dynamic_fee":
				values = []interface{}{big.NewInt(10000000)}
			case "admin_fee":
				values = []interface{}{big.NewInt(5000000000)}
			case "get_pool_asset_types":
				values = []interface{}{[]uint8{0, 1}}
			case "rewardVault":
				values = []interface{}{common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad")}
			case "legacyRewardReceiver", "getRewardsDistributor":
				values = []interface{}{boldRouteLegacy}
			case "isRewardToken":
				values = []interface{}{true}
			case "rewardRouter":
				values = []interface{}{boldRouteRouter}
			case "rewards_receiver":
				values = []interface{}{boldRouteReceiver}
			case "PROTOCOL_CONTROLLER":
				values = []interface{}{common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)}
			case "merkleDistributor":
				target := boldRouteExecutor
				if f.identityFailure {
					target = common.HexToAddress("0xdead")
				}
				values = []interface{}{target}
			case "balanceOf", "claimable_reward":
				values = []interface{}{big.NewInt(1000000000000000000)}
			case "nbIncentives":
				values = []interface{}{big.NewInt(2)}
			case "incentives":
				values = []interface{}{boldRouteGauge, boldRouteToken, big.NewInt(100), big.NewInt(1), big.NewInt(101), big.NewInt(1), boldRouteExecutor, big.NewInt(1000000000000000000), boldRouteExecutor}
			default:
				last = errors.New("unknown fixture method")
			}
			if values != nil {
				result, last = m.Outputs.Pack(values...)
			}
			break
		}
		out = append(out, multicallOutput{Success: last == nil && result != nil, ReturnData: result})
	}
	return aggregate.Methods["aggregate3"].Outputs.Pack(out)
}
func newFeeModelFixture(t *testing.T) (*StakeDAOService, *feeModelRPC, *types.Header) {
	s, f := newPoolFixture(t)
	f.feeHistory = true
	f.latest = 300000
	rpc := &feeModelRPC{fixtureRPC: f}
	s.client = rpc
	s.chainID = 1
	s.controller = common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)
	h, err := rpc.HeaderByNumber(context.Background(), big.NewInt(300000))
	if err != nil {
		t.Fatal(err)
	}
	s.snapshotTime = h.Time
	return s, rpc, h
}
func TestWeETHFeeModelUsesNetBoughtAndAdminShare(t *testing.T) {
	a, _ := abi.JSON(strings.NewReader(weETHFeeABI))
	event := a.Events["TokenExchange"]
	payload, _ := event.Inputs.NonIndexed().Pack(big.NewInt(0), big.NewInt(1000000000000000000), big.NewInt(1), big.NewInt(1000000000000000000))
	l := types.Log{Address: weETHFeePool, BlockNumber: 20, BlockHash: common.HexToHash("0x01"), TxHash: common.HexToHash("0x02"), Topics: []common.Hash{event.ID, common.HexToHash("0x03")}, Data: payload}
	r := weETHFeeReference{rates: [2]decimal.Decimal{decimal.NewFromInt(1), decimal.NewFromInt(2)}, capital: decimal.NewFromInt(100), fee: decimal.RequireFromString("0.001"), admin: decimal.RequireFromString("0.5")}
	fees, count, err := estimateWeETHSwaps([]types.Log{l, l}, a, r)
	// Independent rational calculation: 1 net weETH * 1/999 fee * 1/2 LP share * 2 ETH/weETH.
	got, _ := fees.Float64()
	if err != nil || count != 1 || math.Abs(got-1.0/999) > 1e-15 {
		t.Fatalf("wrong net/gross/admin/rate normalization: %s %d %v", fees, count, err)
	}
	variants := []types.Log{l, l, l, l, l}
	variants[0].Removed = true
	variants[1].Address = common.HexToAddress("0xdead")
	variants[2].Data = []byte{1}
	variants[3].Topics = nil
	variants[4].TxHash = common.Hash{}
	for _, bad := range variants {
		if _, _, e := estimateWeETHSwaps([]types.Log{bad}, a, r); e == nil {
			t.Fatal("malformed history accepted")
		}
	}
	conflict := l
	conflict.BlockHash = common.HexToHash("0x99")
	if _, _, e := estimateWeETHSwaps([]types.Log{l, conflict}, a, r); e == nil {
		t.Fatal("conflicting duplicate accepted")
	}
	if z, n, e := estimateWeETHSwaps(nil, a, r); e != nil || n != 0 || !z.IsZero() {
		t.Fatal("verified empty swap list not zero")
	}
}
func TestWeETHFeeHistoryIsOptionalCompleteAndSeparateFromAPY(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s, rpc, h := newFeeModelFixture(t)
		rpc.scanFailure = fail
		apr := float64(3)
		p := dto.StakeDAOPool{GaugeAddress: weETHFeeGauge, AssetAddress: weETHFeePool, ProtocolID: "0xc715e373", APR: &apr, Yield: &dto.StakeDAOYieldBreakdown{Trading: tradingUnavailable("unsupported_pool_model")}}
		s.addWeETHFeeEstimate(context.Background(), h, 7, &p)
		d := p.Diagnostics.WeETHFees
		if p.APY != nil || p.Yield.Trading.APY != nil || p.APR == nil || *p.APR != 3 {
			t.Fatal("estimate mixed into vault APR/APY")
		}
		if fail {
			if d.EstimatedAPR != nil || d.Swaps != nil || d.LogFailure == nil || d.Status != "unavailable" || d.Reasons[0] != "swap_history_unavailable" {
				t.Fatalf("partial history became zero: %+v", d)
			}
		} else {
			if d.Status != "estimate_available" || d.Swaps == nil || *d.Swaps != 0 || d.EstimatedAPR == nil || *d.EstimatedAPR != 0 || d.RangesDone != d.RangesTotal || d.RangesTotal != 34 || d.CoveredSeconds == nil || *d.CoveredSeconds != 7*86400 {
				t.Fatalf("wrong verified window: %+v", d)
			}
			sort.Slice(rpc.scans, func(i, j int) bool { return rpc.scans[i].FromBlock.Cmp(rpc.scans[j].FromBlock) < 0 })
			previous := *d.StartBlock
			for _, q := range rpc.scans {
				if q.FromBlock.Uint64() != previous+1 || q.ToBlock.Uint64()-q.FromBlock.Uint64() >= 1500 || len(q.Addresses) != 1 || q.Addresses[0] != weETHFeePool {
					t.Fatal("gap/overlap/unbounded query")
				}
				previous = q.ToBlock.Uint64()
			}
			if previous != h.Number.Uint64() {
				t.Fatal("truncated history")
			}
		}
		raw, _ := json.Marshal(p)
		if strings.Contains(string(raw), "private-key") {
			t.Fatal("RPC URL leaked")
		}
	}
	for _, days := range []int{-1, 2, 31} {
		if ValidateWeETHFeeSelection([]common.Address{weETHFeeGauge}, days) == nil {
			t.Fatal("bad window accepted")
		}
	}
	if ValidateWeETHFeeSelection([]common.Address{weETHFeeGauge, boldRouteGauge}, 7) == nil {
		t.Fatal("unbounded multi-pool scan accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, rpc, _ := newFeeModelFixture(t)
	if _, _, e := readWeETHSwaps(ctx, rpc, 1, 3000, common.Hash{}); e == nil {
		t.Fatal("cancelled scan accepted")
	}
}
func TestBoldRewardRouteExplainsZeroVaultRateWithoutInventingAPR(t *testing.T) {
	s, rpc, h := newFeeModelFixture(t)
	p := dto.StakeDAOPool{GaugeAddress: boldRouteGauge, AssetAddress: common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e"), VaultAddress: common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"), RewardReceiver: boldRouteReceiver, ProtocolID: "0xc715e373", Diagnostics: &dto.StakeDAOPoolDiagnostics{Primary: &dto.PrimaryRateDiagnostic{Locker: common.HexToAddress("0x11")}}, Rewards: []dto.StakeDAOReward{{Token: boldRouteToken, Source: "vault-extra", DistributionStatus: "not_started"}}}
	s.addBoldRewardRoute(context.Background(), h.Number, &p)
	d := p.Diagnostics.RewardRoute
	if d.Status != "configured" || d.VaultDistributionStatus != "not_started" || len(d.Campaigns) != 2 || d.Campaigns[0].Status != "expired_funding_period" || p.APR != nil || p.APY != nil {
		t.Fatalf("campaign converted to vault yield: %+v", d)
	}
	rpc.identityFailure = true
	s.addBoldRewardRoute(context.Background(), h.Number, &p)
	if p.Diagnostics.RewardRoute.Status != "unavailable" || p.Diagnostics.RewardRoute.Reasons[0] != "reward_route_identity_mismatch" {
		t.Fatal("changed router trusted")
	}
	v := []interface{}{boldRouteGauge, boldRouteToken, big.NewInt(100), big.NewInt(1), big.NewInt(101), big.NewInt(1), boldRouteExecutor, big.NewInt(1), boldRouteExecutor}
	v[6] = common.HexToAddress("0xdead")
	if _, match, valid := decodeBoldCampaign(v, 0, 200); match || !valid {
		t.Fatal("untrusted campaign counted")
	}
	v[6] = boldRouteExecutor
	v[4] = big.NewInt(102)
	if _, _, valid := decodeBoldCampaign(v, 0, 200); valid {
		t.Fatal("invalid campaign duration accepted")
	}
}

type feeLogRPCError struct {
	code    int
	message string
}

func (e feeLogRPCError) Error() string  { return e.message }
func (e feeLogRPCError) ErrorCode() int { return e.code }

func TestFeeLogFailuresPreserveSafeCauseAndDoNotGuessProviderLimits(t *testing.T) {
	cases := []struct {
		err   error
		kind  string
		limit uint64
	}{
		{feeLogRPCError{-32600, "Free tier users can query up to a 10 block range. https://rpc.invalid/private-key"}, "rpc_log_range_limit", 10},
		{feeLogRPCError{-32602, "eth_getLogs is limited to 10,000 blocks range"}, "rpc_log_range_limit", 10000},
		{feeLogRPCError{-32600, "invalid request https://rpc.invalid/private-key"}, "swap_history_unavailable", 0},
		{errors.New("block range not available"), "swap_history_unavailable", 0},
		{errors.New("block range too large"), "rpc_log_range_limit", 0},
		{errors.New("query returned more than 10000 results; try with this block range"), "rpc_log_result_limit", 0},
		{rpc.HTTPError{StatusCode: 429, Body: []byte("https://rpc.invalid/private-key")}, "rpc_rate_limited", 0},
		{rpc.HTTPError{StatusCode: 402}, "rpc_quota_exhausted", 0},
		{rpc.HTTPError{StatusCode: 403, Body: []byte("Archive requests require a personal token")}, "archive_rpc_required", 0},
		{rpc.HTTPError{StatusCode: 403}, "rpc_access_denied", 0},
		{context.DeadlineExceeded, "rpc_timeout", 0},
		{context.Canceled, "rpc_canceled", 0},
	}
	for _, tc := range cases {
		wrapped := errors.Join(tc.err, errors.New("transport context"))
		e := classifySwapHistoryFailure(wrapped, 10, 1509)
		if e.Error() != tc.kind || e.details.Method != "eth_getLogs" || e.details.FromBlock != 10 || e.details.ToBlock != 1509 {
			t.Fatalf("wrong classification: %+v", e.details)
		}
		if tc.limit == 0 && e.details.ProviderMaxBlockRange != nil || tc.limit != 0 && (e.details.ProviderMaxBlockRange == nil || *e.details.ProviderMaxBlockRange != tc.limit) {
			t.Fatalf("guessed or lost numeric limit: %+v", e.details)
		}
		if original, ok := tc.err.(feeLogRPCError); ok && (e.details.RPCCode == nil || *e.details.RPCCode != original.code) {
			t.Fatal("JSON-RPC code lost")
		}
		raw, _ := json.Marshal(e.details)
		if strings.Contains(string(raw), "private-key") || strings.Contains(string(raw), "rpc.invalid") {
			t.Fatal("raw provider message leaked")
		}
	}
}

func TestWeETHRangeFailureIsBoundedAndKeepsReferenceParameters(t *testing.T) {
	s, client, h := newFeeModelFixture(t)
	client.scanError = feeLogRPCError{-32600, "Free tier users can query up to a 10 block range https://rpc.invalid/private-key"}
	p := dto.StakeDAOPool{GaugeAddress: weETHFeeGauge, AssetAddress: weETHFeePool, ProtocolID: "0xc715e373"}
	s.addWeETHFeeEstimate(context.Background(), h, 1, &p)
	d := p.Diagnostics.WeETHFees
	if d.Status != "unavailable" || d.Swaps != nil || d.EstimatedAPR != nil || d.RangesDone != 0 || d.ReferenceCapitalETH == nil || d.DynamicFeeFraction == nil || d.AdminFeeFraction == nil {
		t.Fatalf("failure erased reference or became zero: %+v", d)
	}
	if d.LogFailure == nil || d.Reasons[0] != "rpc_log_range_limit" || d.LogFailure.ProviderMaxBlockRange == nil || *d.LogFailure.ProviderMaxBlockRange != 10 || d.LogFailure.RequiredRangesForWindow == nil || *d.LogFailure.RequiredRangesForWindow != 720 {
		t.Fatalf("range limit/budget impact lost: %+v", d)
	}
	if len(client.scans) > 3 {
		t.Fatalf("unexpected expensive automatic splitting: %d requests", len(client.scans))
	}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), "private-key") {
		t.Fatal("upstream URL leaked")
	}
}

type secondaryFeeLogsFixture struct {
	*feeModelRPC
	wrongChain        bool
	wrongHeader       bool
	headerReadsHere   int
	reorgAfterInitial bool
}

func (c *secondaryFeeLogsFixture) ChainID(context.Context) (*big.Int, error) {
	if c.wrongChain {
		return big.NewInt(137), nil
	}
	return big.NewInt(1), nil
}
func (c *secondaryFeeLogsFixture) HeaderByNumber(ctx context.Context, n *big.Int) (*types.Header, error) {
	h, e := c.fixtureRPC.HeaderByNumber(ctx, n)
	c.headerReadsHere++
	if e == nil && (c.wrongHeader || c.reorgAfterInitial && c.headerReadsHere > 2) {
		h.Extra = []byte("other-chain-history")
	}
	return h, e
}
func TestWeETHSeparateLogRPCUsesSharedBudgetAndVerifiesHistory(t *testing.T) {
	s, primary, h := newFeeModelFixture(t)
	_, second, _ := newFeeModelFixture(t)
	source := &secondaryFeeLogsFixture{feeModelRPC: second}
	budget := newSampleRPC(primary)
	s.client = budget
	s.feeLogsRPCURL = "https://rpc.invalid/private-api-key"
	s.dial = func(_ context.Context, url string) (stakeDAORPC, error) {
		if url != s.feeLogsRPCURL {
			t.Fatal("wrong configured source")
		}
		return source, nil
	}
	primary.scanError = feeLogRPCError{-32600, "up to a 10 block range"}
	p := dto.StakeDAOPool{GaugeAddress: weETHFeeGauge, AssetAddress: weETHFeePool, ProtocolID: "0xc715e373"}
	s.addWeETHFeeEstimate(context.Background(), h, 1, &p)
	d := p.Diagnostics.WeETHFees
	if d.Status != "estimate_available" || d.LogSource != "configured_logs_rpc" || !d.LogSourceVerified || len(primary.scans) != 0 || len(source.scans) != 5 || !source.closed || primary.closed || p.APY != nil {
		t.Fatalf("incorrect source routing/lifecycle: %+v", d)
	}
	if budget.remaining >= 300-5 {
		t.Fatal("secondary calls bypassed the primary budget")
	}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), "private-api-key") || strings.Contains(string(raw), "rpc.invalid") {
		t.Fatal("log-source URL leaked")
	}
}
func TestFeeLogSourceRejectsWrongChainAndSnapshotAndCloses(t *testing.T) {
	for _, mode := range []string{"chain", "header", "dial", "budget"} {
		s, primary, h := newFeeModelFixture(t)
		_, second, _ := newFeeModelFixture(t)
		source := &secondaryFeeLogsFixture{feeModelRPC: second, wrongChain: mode == "chain", wrongHeader: mode == "header"}
		budget := newSampleRPC(primary)
		s.client = budget
		s.feeLogsRPCURL = "https://rpc.invalid/private-api-key"
		if mode == "budget" {
			budget.remaining = 0
		}
		s.dial = func(context.Context, string) (stakeDAORPC, error) {
			if mode == "dial" {
				return nil, errors.New("https://rpc.invalid/private-api-key")
			}
			return source, nil
		}
		_, _, e := s.openFeeLogSource(context.Background(), h, h)
		if e == nil || len(source.scans) != 0 || mode != "dial" && !source.closed || strings.Contains(e.Error(), "private-api-key") {
			t.Fatalf("unverified source accepted/leaked for %s: %v", mode, e)
		}
	}
	s, primary, h := newFeeModelFixture(t)
	_, second, _ := newFeeModelFixture(t)
	source := &secondaryFeeLogsFixture{feeModelRPC: second, reorgAfterInitial: true}
	s.client = newSampleRPC(primary)
	s.feeLogsRPCURL = "configured"
	s.dial = func(context.Context, string) (stakeDAORPC, error) { return source, nil }
	p := dto.StakeDAOPool{GaugeAddress: weETHFeeGauge, AssetAddress: weETHFeePool, ProtocolID: "0xc715e373"}
	s.addWeETHFeeEstimate(context.Background(), h, 1, &p)
	d := p.Diagnostics.WeETHFees
	if d.EstimatedAPR != nil || d.Swaps != nil || d.Reasons[0] != "logs_rpc_snapshot_mismatch" || !source.closed {
		t.Fatalf("reorganized source became an estimate: %+v", d)
	}
}
func TestFeeLogSourceWithoutConfigurationReusesPrimaryAndNeverDials(t *testing.T) {
	s, primary, h := newFeeModelFixture(t)
	s.dial = func(context.Context, string) (stakeDAORPC, error) {
		t.Fatal("unused secondary source dialed")
		return nil, nil
	}
	got, cleanup, e := s.openFeeLogSource(context.Background(), h, h)
	if e != nil || got != primary {
		t.Fatalf("default source changed: %v", e)
	}
	cleanup()
	if primary.closed {
		t.Fatal("primary closed by optional log helper")
	}
}

// Offline research tests reuse the independently saved mainnet receipt/getters.
// They do not assert that allocation rules or a current BOLD APR were found.
func boldResearchAuditFixture(t *testing.T) (*types.Receipt, map[uint64][]interface{}) {
	t.Helper()
	raw, err := os.ReadFile("../../../examples/stakedao-bold-route-audit.json")
	if err != nil {
		t.Fatal(err)
	}
	var audit struct {
		Reads []struct {
			Method string
			Args   []string
			Values []json.RawMessage
		}
		Transaction struct {
			Hash         common.Hash
			Block        uint64
			BlockHash    common.Hash
			RelevantLogs []*types.Log
		}
	}
	if err := json.Unmarshal(raw, &audit); err != nil {
		t.Fatal(err)
	}
	receipt := &types.Receipt{Status: 1, TxHash: audit.Transaction.Hash, BlockNumber: new(big.Int).SetUint64(audit.Transaction.Block), BlockHash: audit.Transaction.BlockHash, Logs: audit.Transaction.RelevantLogs}
	if len(receipt.Logs) == 0 {
		t.Fatal("empty saved receipt")
	}
	receipt.TransactionIndex = receipt.Logs[0].TxIndex
	getters := map[uint64][]interface{}{}
	for _, read := range audit.Reads {
		if read.Method != "incentives" {
			continue
		}
		id, ok := new(big.Int).SetString(read.Args[0], 10)
		if !ok || len(read.Values) != 9 {
			t.Fatal("invalid saved getter")
		}
		v := make([]interface{}, 9)
		for i, raw := range read.Values {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				t.Fatal(err)
			}
			if i == 0 || i == 1 || i == 6 || i == 8 {
				v[i] = common.HexToAddress(text)
			} else {
				n, ok := new(big.Int).SetString(text, 10)
				if !ok {
					t.Fatal("invalid integer")
				}
				v[i] = n
			}
		}
		getters[id.Uint64()] = v
	}
	return receipt, getters
}

func TestBOLDResearchRealReceiptAndGetterBinding(t *testing.T) {
	parsed, err := abi.JSON(strings.NewReader(boldCampaignResearchABI))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Events["IncentiveAdded"].ID.Hex() != "0x6ff2feb1912bfc5e0f8e6b7e8f9c30db612efb5337419b4da8099d27d2bf8863" || parsed.Events["RewardsPulledToMerkle"].ID.Hex() != "0xc5f8cc03d66c0e8fafb19862f551957745c71666872c958050c021f785d9561d" {
		t.Fatal("ABI event signatures differ from mainnet receipt")
	}
	receipt, getters := boldResearchAuditFixture(t)
	entries := map[uint64]types.Log{}
	for _, l := range receipt.Logs {
		if l.Address != boldCampaignDistributor {
			continue
		}
		id, match, err := boldResearchCampaignEvent(*l, parsed.Events["IncentiveAdded"])
		if err != nil || !match {
			t.Fatalf("saved event rejected: %v", err)
		}
		if err := boldResearchCompareEvent(*l, parsed.Events["IncentiveAdded"], getters[id]); err != nil {
			t.Fatal(err)
		}
		entries[id] = *l
	}
	if len(entries) != 2 {
		t.Fatal("expected independently audited IDs 2593 and 2594")
	}
	if err := boldResearchVerifyFunding(receipt, entries, parsed); err != nil {
		t.Fatalf("verified transfer conservation rejected: %v", err)
	}
	for _, mode := range []string{"missing_transfer", "wrong_return", "wrong_block", "missing_campaign", "duplicate_log", "multiple_pulls", "other_gauge_pull", "removed", "wrong_getter"} {
		t.Run(mode, func(t *testing.T) {
			r, values := boldResearchAuditFixture(t)
			e := map[uint64]types.Log{2593: entries[2593], 2594: entries[2594]}
			switch mode {
			case "missing_transfer":
				r.Logs = r.Logs[:len(r.Logs)-1]
			case "wrong_return":
				r.Logs[len(r.Logs)-1].Data[31] ^= 1
			case "wrong_block":
				r.Logs[0].BlockHash = common.HexToHash("0xbad")
			case "missing_campaign":
				delete(e, 2594)
			case "duplicate_log":
				r.Logs = append(r.Logs, r.Logs[0])
			case "removed":
				r.Logs[0].Removed = true
			case "multiple_pulls", "other_gauge_pull":
				for _, l := range r.Logs {
					if l.Address == boldRouteRouter {
						copy := *l
						copy.Index = 900
						copy.Topics = append([]common.Hash(nil), l.Topics...)
						if mode == "other_gauge_pull" {
							copy.Topics[1] = common.HexToHash("0xbad")
						}
						r.Logs = append(r.Logs, &copy)
						break
					}
				}
			case "wrong_getter":
				v := values[2593]
				v[7] = big.NewInt(1)
				if err := boldResearchCompareEvent(e[2593], parsed.Events["IncentiveAdded"], v); err == nil {
					t.Fatal("getter/event mismatch accepted")
				}
				return
			}
			if err := boldResearchVerifyFunding(r, e, parsed); err == nil {
				t.Fatal("unproven funding accepted")
			}
		})
	}
}

type boldInspectorFixture struct {
	receipt     *types.Receipt
	getters     map[uint64][]interface{}
	scans       []ethereum.FilterQuery
	mode        string
	headerReads map[uint64]int
	root        common.Hash
	snapshot    uint64
}

func newBoldInspectorFixture(t *testing.T) *boldInspectorFixture {
	r, getters := boldResearchAuditFixture(t)
	f := &boldInspectorFixture{receipt: r, getters: getters, headerReads: map[uint64]int{}}
	h, _ := f.HeaderByNumber(context.Background(), r.BlockNumber)
	// Keep real event payloads; use synthetic headers so no live RPC is needed.
	r.BlockHash = h.Hash()
	for _, l := range r.Logs {
		l.BlockHash = r.BlockHash
	}
	f.headerReads = map[uint64]int{}
	return f
}
func (f *boldInspectorFixture) ChainID(context.Context) (*big.Int, error) {
	if f.mode == "wrong_chain" {
		return big.NewInt(10), nil
	}
	return big.NewInt(1), nil
}
func (f *boldInspectorFixture) HeaderByNumber(_ context.Context, n *big.Int) (*types.Header, error) {
	if n == nil {
		return nil, errors.New("test requires explicit block")
	}
	h := &types.Header{Number: new(big.Int).Set(n), Time: 1795185215}
	f.headerReads[n.Uint64()]++
	if f.mode == "reorg" && len(f.scans) > 0 {
		h.Extra = []byte("reorganized")
	}
	return h, nil
}
func (f *boldInspectorFixture) TransactionReceipt(_ context.Context, tx common.Hash) (*types.Receipt, error) {
	if tx != f.receipt.TxHash {
		return nil, errors.New("wrong transaction")
	}
	return f.receipt, nil
}
func (f *boldInspectorFixture) FilterLogs(_ context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	f.scans = append(f.scans, q)
	if f.mode == "provider_error" {
		return nil, errors.New("https://rpc.invalid/secret-key unavailable")
	}
	if f.mode == "empty" {
		return []types.Log{}, nil
	}
	logs := []types.Log{}
	for _, l := range f.receipt.Logs {
		if l.Address == boldRouteRouter || l.Address == boldCampaignDistributor {
			if l.BlockNumber >= q.FromBlock.Uint64() && l.BlockNumber <= q.ToBlock.Uint64() {
				logs = append(logs, *l)
			}
		}
	}
	return logs, nil
}
func (f *boldInspectorFixture) CallContract(_ context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	snapshot := f.snapshot
	if snapshot == 0 {
		snapshot = f.receipt.BlockNumber.Uint64() + 10
	}
	if block == nil || block.Uint64() != snapshot || msg.To == nil || *msg.To != stakeDAOMulticallAddress {
		return nil, errors.New("wrong snapshot or target")
	}
	aggregate, _ := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
	v, err := aggregate.Methods["aggregate3"].Inputs.Unpack(msg.Data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(v[0], new([]multicallInput)).(*[]multicallInput)
	results := []multicallOutput{}
	for _, call := range calls {
		found := false
		for _, raw := range []string{protocolControllerABI, rewardRouteABI, boldCampaignResearchABI, erc20DecimalsABI} {
			a, _ := abi.JSON(strings.NewReader(raw))
			method, err := a.MethodById(call.CallData[:4])
			if err != nil {
				continue
			}
			args, _ := method.Inputs.Unpack(call.CallData[4:])
			var output []interface{}
			switch method.Name {
			case "gauge":
				output = []interface{}{common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"), common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e"), boldRouteReceiver, [4]byte{0xc7, 0x15, 0xe3, 0x73}, false}
			case "rewardVault":
				output = []interface{}{common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad")}
			case "rewardRouter":
				output = []interface{}{boldRouteRouter}
			case "legacyRewardReceiver":
				output = []interface{}{boldRouteLegacy}
			case "PROTOCOL_CONTROLLER":
				output = []interface{}{common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)}
			case "merkleDistributor":
				target := boldRouteExecutor
				if f.mode == "route_changed" {
					target = common.HexToAddress("0xbad")
				}
				output = []interface{}{target}
			case "nbIncentives":
				output = []interface{}{big.NewInt(2595)}
			case "root", "ipfsHash":
				value := common.HexToHash("0x1234")
				if method.Name == "root" && f.root != (common.Hash{}) {
					value = f.root
				}
				output = []interface{}{[32]byte(value)}
			case "decimals":
				output = []interface{}{uint8(18)}
			case "incentives":
				output = f.getters[args[0].(*big.Int).Uint64()]
			}
			if output == nil {
				return nil, errors.New("unhandled getter")
			}
			encoded, err := method.Outputs.Pack(output...)
			if err != nil {
				return nil, err
			}
			results = append(results, multicallOutput{Success: true, ReturnData: encoded})
			found = true
			break
		}
		if !found {
			return nil, errors.New("unhandled selector")
		}
	}
	return aggregate.Methods["aggregate3"].Outputs.Pack(results)
}

func TestBOLDInspectorIsBoundedAndLeavesUnknownAllocationUnknown(t *testing.T) {
	for _, mode := range []string{"success", "provider_error", "route_changed", "reorg", "empty", "wrong_chain"} {
		t.Run(mode, func(t *testing.T) {
			f := newBoldInspectorFixture(t)
			f.mode = mode
			b := f.receipt.BlockNumber.Uint64()
			r, err := NewBOLDCampaignInspector(f, f, nil).Inspect(context.Background(), b, b+10)
			if mode == "provider_error" || mode == "route_changed" || mode == "reorg" || mode == "wrong_chain" {
				if err == nil || r != nil || strings.Contains(err.Error(), "secret-key") {
					t.Fatalf("failed scan leaked/became zero: %+v %v", r, err)
				}
				return
			}
			if err != nil || !r.ScanComplete || r.CurrentAPR != nil || r.BlockNumber != b+10 || r.RPCRequests > boldResearchMaxRPC || len(f.scans) != 1 {
				t.Fatalf("invalid bounded result: %+v %v", r, err)
			}
			if mode == "empty" {
				if len(r.Campaigns) != 0 || r.Status != "no_verified_campaigns_in_creation_window" {
					t.Fatal("empty creation window became zero APR")
				}
				return
			}
			if len(r.Campaigns) != 2 {
				t.Fatal("real campaign records missing")
			}
			for _, c := range r.Campaigns {
				if c.FundingStatus != "verified_router_campaign_funding" || c.AllocationStatus != "unavailable" || c.HistoricalAPR != nil || c.Reasons[0] != "allocation_rules_unverified" || c.PeriodStatus != "expired_funding_period" {
					t.Fatalf("funding became a payout/APR: %+v", c)
				}
			}
		})
	}
	f := newBoldInspectorFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewBOLDCampaignInspector(f, f, nil).Inspect(ctx, 1, 2); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel ignored")
	}
	for _, bounds := range [][2]uint64{{0, 1}, {2, 1}, {1, 240001}} {
		if _, err := NewBOLDCampaignInspector(f, f, nil).Inspect(context.Background(), bounds[0], bounds[1]); err == nil {
			t.Fatal("unbounded/invalid request accepted")
		}
	}
}

type boldAllocationResolverFunc func(context.Context, BOLDCampaignReport, BOLDCampaignRecord) (BOLDAllocationEvidence, error)

func (f boldAllocationResolverFunc) ResolveBOLDAllocation(ctx context.Context, r BOLDCampaignReport, c BOLDCampaignRecord) (BOLDAllocationEvidence, error) {
	return f(ctx, r, c)
}

func TestBOLDInspectorOptionalResolverKeepsHistoricalEstimateSeparate(t *testing.T) {
	f := newBoldInspectorFixture(t)
	resolver := boldAllocationResolverFunc(func(_ context.Context, r BOLDCampaignReport, c BOLDCampaignRecord) (BOLDAllocationEvidence, error) {
		if c.ID == 2594 {
			return BOLDAllocationEvidence{}, errors.New("https://rpc.invalid/secret-key allocation unavailable")
		}
		return BOLDAllocationEvidence{CampaignID: c.ID, SnapshotHash: r.BlockHash, Vault: r.Vault, Start: c.Start, End: c.End,
			NetRewardRaw: "1000000000000000000", RewardUnitUSD: "1", CapitalUSDSeconds: "31536000",
			RulesReference: "SYNTHETIC_TEST_ONLY_not_verified_real_allocation", ValuationReference: "test-only", EvidenceHash: common.HexToHash("0x123")}, nil
	})
	b := f.receipt.BlockNumber.Uint64()
	r, err := NewBOLDCampaignInspector(f, f, resolver).Inspect(context.Background(), b, b+10)
	if err != nil {
		t.Fatal(err)
	}
	if r.CurrentAPR != nil || r.Status != "funding_verified_with_historical_estimates" || r.Campaigns[0].HistoricalAPR == nil || *r.Campaigns[0].HistoricalAPR != "100" || r.Campaigns[1].HistoricalAPR != nil || r.Campaigns[1].Reasons[0] != "allocation_evidence_unavailable" {
		t.Fatalf("historical partial estimate mislabeled: %+v", r)
	}
	encoded, _ := json.Marshal(r)
	if strings.Contains(string(encoded), "secret-key") {
		t.Fatal("resolver error leaked")
	}
	// The resolver can perform lengthy reads; recheck the snapshot afterwards.
	f = newBoldInspectorFixture(t)
	changing := boldAllocationResolverFunc(func(ctx context.Context, r BOLDCampaignReport, c BOLDCampaignRecord) (BOLDAllocationEvidence, error) {
		f.mode = "reorg"
		return resolver(ctx, r, c)
	})
	if r, err := NewBOLDCampaignInspector(f, f, changing).Inspect(context.Background(), b, b+10); err == nil || r != nil {
		t.Fatal("snapshot changed during resolution accepted")
	}
}

func TestBOLDHistoricalAPRRequiresPeriodAttributionAndCapitalIntegral(t *testing.T) {
	r := BOLDCampaignReport{ScanComplete: true, BlockHash: common.HexToHash("0x123"), BlockTimestamp: 1000,
		Vault: common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"), Gauge: boldRouteGauge, Token: boldRouteToken, Distributor: boldCampaignDistributor}
	c := BOLDCampaignRecord{ID: 1, Start: 100, End: 200, AmountRaw: "10000000000000000000", FundingStatus: "verified_router_campaign_funding"}
	e := BOLDAllocationEvidence{CampaignID: 1, SnapshotHash: r.BlockHash, Vault: r.Vault, Start: 100, End: 200,
		NetRewardRaw: "1000000000000000000", RewardUnitUSD: "2", CapitalUSDSeconds: "63072000", RulesReference: "test-fixture-only-not-real-rules", ValuationReference: "test-only", EvidenceHash: common.HexToHash("0xabc")}
	// $2 / ($630720 average capital * 100s), annualized -> exactly 100%.
	got, err := CalculateBOLDHistoricalAPR(r, c, e)
	if err != nil || got != "100" || r.CurrentAPR != nil {
		t.Fatalf("wrong units/annualization: %s %v", got, err)
	}
	for _, mode := range []string{"missing_rules", "wrong_vault", "wrong_campaign", "wrong_block", "wrong_period", "overspend", "negative", "no_capital", "no_price", "missing_provenance", "future_period", "no_funding_proof"} {
		x, campaign, report := e, c, r
		switch mode {
		case "missing_rules":
			x.RulesReference = ""
		case "wrong_vault":
			x.Vault = boldRouteGauge
		case "wrong_campaign":
			x.CampaignID++
		case "wrong_block":
			x.SnapshotHash = common.Hash{}
		case "wrong_period":
			x.End++
		case "overspend":
			x.NetRewardRaw = "11000000000000000000"
		case "negative":
			x.NetRewardRaw = "-1"
		case "no_capital":
			x.CapitalUSDSeconds = "0"
		case "no_price":
			x.RewardUnitUSD = "0"
		case "missing_provenance":
			x.EvidenceHash = common.Hash{}
		case "future_period":
			report.BlockTimestamp = 199
		case "no_funding_proof":
			campaign.FundingStatus = "unavailable"
		}
		if _, err := CalculateBOLDHistoricalAPR(report, campaign, x); err == nil {
			t.Fatalf("invalid allocation accepted: %s", mode)
		}
	}
	e.NetRewardRaw = "0"
	if got, err := CalculateBOLDHistoricalAPR(r, c, e); err != nil || got != "0" {
		t.Fatal("verified zero allocation lost")
	}
}

func TestBOLDClaimProofUsesDoubleHashAndCannotProveCampaignAttribution(t *testing.T) {
	addressType, _ := abi.NewType("address", "", nil)
	integerType, _ := abi.NewType("uint256", "", nil)
	args := abi.Arguments{{Type: addressType}, {Type: addressType}, {Type: integerType}}
	account := common.HexToAddress("0x1234")
	amount := big.NewInt(123456)
	encoded, err := args.Pack(account, boldRouteToken, amount)
	if err != nil {
		t.Fatal(err)
	}
	leaf := crypto.Keccak256Hash(crypto.Keccak256(encoded))
	sibling := common.HexToHash("0x123")
	var root common.Hash
	if strings.Compare(leaf.Hex(), sibling.Hex()) < 0 {
		root = crypto.Keccak256Hash(leaf[:], sibling[:])
	} else {
		root = crypto.Keccak256Hash(sibling[:], leaf[:])
	}
	if !VerifyBOLDClaimProof(root, account, boldRouteToken, amount, []common.Hash{sibling}) {
		t.Fatal("valid Solidity-compatible proof rejected")
	}
	if VerifyBOLDClaimProof(root, account, boldRouteToken, big.NewInt(123457), []common.Hash{sibling}) || VerifyBOLDClaimProof(crypto.Keccak256Hash(encoded), account, boldRouteToken, amount, nil) || VerifyBOLDClaimProof(root, account, boldRouteGauge, amount, []common.Hash{sibling}) || VerifyBOLDClaimProof(root, account, boldRouteToken, new(big.Int).Lsh(big.NewInt(1), 256), nil) {
		t.Fatal("incorrect leaf/root/token/amount accepted")
	}
}

func TestBOLDReviewedPublicationVerifiesProofsAndExactPayouts(t *testing.T) {
	p, err := reviewedBOLDPublication()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Proofs) != 15 || len(p.Campaigns) != 2 || p.Campaigns[0].AllocatedRaw != "10162926236299411596" || p.Campaigns[1].AllocatedRaw != "349080457011333212616" {
		t.Fatal("reviewed allocations missing")
	}
	if _, err := decodeBOLDPublication([]byte(strings.Replace(boldReviewedPublicationJSON, "10162926236299411596", "10162926236299411597", 1))); err == nil {
		t.Fatal("modified publication accepted")
	}
}

func boldReviewedTestReport(t *testing.T) (BOLDCampaignReport, BOLDCampaignRecord) {
	t.Helper()
	p, err := reviewedBOLDPublication()
	if err != nil {
		t.Fatal(err)
	}
	r := BOLDCampaignReport{ScanComplete: true, Root: p.Root, BlockNumber: 26456929, BlockTimestamp: 1795185215, BlockHash: common.HexToHash("0x1234"), Vault: common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"), Gauge: boldRouteGauge, Token: boldRouteToken, Distributor: boldCampaignDistributor}
	item := p.Campaigns[0]
	c := BOLDCampaignRecord{ID: item.ID, Start: item.Start, End: item.End, AmountRaw: item.FundedRaw, FundingStatus: "verified_router_campaign_funding"}
	return r, c
}

func TestBOLDPublishedResolverUsesHistoricalCapitalAndKeepsVerifiedAllocationOnFailure(t *testing.T) {
	r, c := boldReviewedTestReport(t)
	point := func(_ context.Context, ts uint64) (BOLDHistoricalSample, error) {
		tvl := decimal.NewFromInt(100).Add(decimal.NewFromInt(200).Mul(decimal.NewFromInt(int64(ts - c.Start))).Div(decimal.NewFromInt(int64(c.End - c.Start))))
		return BOLDHistoricalSample{RequestedTimestamp: ts, TVLUSD: tvl.String(), RewardUSD: "2", LPUSD: "1", AssetAmount: tvl.String()}, nil
	}
	resolver := &boldPublishedResolver{sampleOverride: point}
	e, err := resolver.ResolveBOLDAllocation(context.Background(), r, c)
	if err != nil {
		t.Fatal(err)
	}
	got, err := CalculateBOLDHistoricalAPR(r, c, e)
	if err != nil {
		t.Fatal(err)
	}
	n, _ := new(big.Int).SetString(e.NetRewardRaw, 10)
	numerator := new(big.Int).Mul(n, big.NewInt(2*31536000*100))
	denominator := new(big.Int).Mul(big.NewInt(1000000000000000000), big.NewInt(int64(200*(c.End-c.Start))))
	want := new(big.Rat).SetFrac(numerator, denominator)
	actual, ok := new(big.Rat).SetString(got)
	if !ok {
		t.Fatal(got)
	}
	difference, _ := new(big.Rat).Sub(actual, want).Float64()
	if math.Abs(difference) > 1e-12 || e.Valuation == nil || *e.Valuation.AverageCapitalUSD != "200" || *e.Valuation.AverageRewardUSD != "2" || e.Publication.Recipients != 5 || r.CurrentAPR != nil {
		t.Fatalf("incorrect historical annualization: %s %+v", got, e)
	}
	for _, mode := range []string{"unknown_root", "wrong_period", "unavailable_price"} {
		report, campaign := r, c
		res := &boldPublishedResolver{sampleOverride: point}
		switch mode {
		case "unknown_root":
			report.Root = common.HexToHash("0xbad")
		case "wrong_period":
			campaign.End++
		case "unavailable_price":
			res.sampleOverride = func(context.Context, uint64) (BOLDHistoricalSample, error) {
				return BOLDHistoricalSample{}, boldCampaignFailure("historical_price_unavailable")
			}
		}
		partial, err := res.ResolveBOLDAllocation(context.Background(), report, campaign)
		if err == nil {
			t.Fatalf("invalid historical evidence accepted: %s", mode)
		}
		if mode == "unavailable_price" && (partial.Publication == nil || partial.NetRewardRaw == "" || partial.CapitalUSDSeconds != "" || partial.Valuation.Status != "unavailable") {
			t.Fatal("price failure erased verified allocation or fabricated capital")
		}
	}
	points := []BOLDHistoricalSample{{RequestedTimestamp: 10, TVLUSD: "0", RewardUSD: "1"}, {RequestedTimestamp: 20, TVLUSD: "0", RewardUSD: "1"}, {RequestedTimestamp: 30, TVLUSD: "0", RewardUSD: "1"}}
	if _, _, err := integrateBOLDHistory(points, 10, 30); err == nil {
		t.Fatal("zero capital produced APR")
	}
	points[0].TVLUSD, points[1].TVLUSD, points[2].TVLUSD = "100", "100", "1000"
	capital, _, err := integrateBOLDHistory(points, 10, 30)
	if err != nil || capital.String() != "6500" {
		t.Fatal("trapezoidal capital weights were replaced by a simple sample average")
	}
}

const boldHistoricalFixtureABI = `[
{"type":"function","name":"asset","inputs":[],"outputs":[{"type":"address"}]},
{"type":"function","name":"totalAssets","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"decimals","inputs":[],"outputs":[{"type":"uint8"}]},
{"type":"function","name":"N_COINS","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"is_meta","inputs":[{"type":"address"}],"outputs":[{"type":"bool"}]},
{"type":"function","name":"totalSupply","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"coins","inputs":[{"type":"uint256"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"balances","inputs":[{"type":"uint256"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"stored_rates","inputs":[],"outputs":[{"type":"uint256[]"}]},
{"type":"function","name":"ma_exp_time","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"ma_last_time","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"price_oracle","inputs":[{"type":"uint256"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"getFeed","inputs":[{"type":"address"},{"type":"address"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"latestRoundData","inputs":[],"outputs":[{"type":"uint80"},{"type":"int256"},{"type":"uint256"},{"type":"uint256"},{"type":"uint80"}]}
]`

type boldHistoricalRPCFixture struct {
	*boldInspectorFixture
	historicalCalls []uint64
	priceMode       string
}

func (f *boldHistoricalRPCFixture) HeaderByNumber(_ context.Context, n *big.Int) (*types.Header, error) {
	if n == nil || !n.IsUint64() || n.Uint64() > 26456929 {
		return nil, errors.New("unexpected header")
	}
	h := &types.Header{Number: new(big.Int).Set(n), Time: 1795185215 - (26456929-n.Uint64())*12}
	if f.priceMode == "reorganized" {
		h.Extra = []byte("changed")
	}
	return h, nil
}
func (f *boldHistoricalRPCFixture) CallContract(ctx context.Context, msg ethereum.CallMsg, b *big.Int) ([]byte, error) {
	if b == nil || msg.To == nil {
		return nil, errors.New("explicit block/target required")
	}
	a, _ := abi.JSON(strings.NewReader(boldHistoricalFixtureABI))
	evaluate := func(target common.Address, data []byte) ([]byte, error) {
		m, err := a.MethodById(data[:4])
		if err != nil {
			return nil, err
		}
		args, _ := m.Inputs.Unpack(data[4:])
		h, _ := f.HeaderByNumber(ctx, b)
		var values []interface{}
		lp := common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e")
		usdc := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
		feed := common.HexToAddress("0x123")
		units := func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), big.NewInt(1000000000000000000)) }
		switch m.Name {
		case "asset":
			values = []interface{}{lp}
		case "totalAssets":
			values = []interface{}{units(100)}
		case "decimals":
			d := uint8(18)
			if target == usdc {
				d = 6
			}
			if target == feed {
				d = 8
			}
			values = []interface{}{d}
		case "is_meta":
			values = []interface{}{false}
		case "N_COINS":
			values = []interface{}{big.NewInt(2)}
		case "totalSupply":
			values = []interface{}{units(1100000)}
		case "coins":
			coin := boldRouteToken
			if args[0].(*big.Int).Sign() != 0 {
				coin = usdc
			}
			values = []interface{}{coin}
		case "balances":
			n := units(600000)
			if args[0].(*big.Int).Sign() != 0 {
				n = big.NewInt(500000000000)
			}
			values = []interface{}{n}
		case "stored_rates":
			values = []interface{}{[]*big.Int{units(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)}}
		case "ma_exp_time":
			values = []interface{}{big.NewInt(600)}
		case "ma_last_time":
			ts := h.Time
			if f.priceMode == "stale" {
				ts -= 3 * 3600
			}
			values = []interface{}{new(big.Int).SetUint64(ts)}
		case "price_oracle":
			values = []interface{}{units(1)}
		case "getFeed":
			values = []interface{}{feed}
		case "latestRoundData":
			ts := new(big.Int).SetUint64(h.Time)
			values = []interface{}{big.NewInt(1), big.NewInt(100000000), ts, ts, big.NewInt(1)}
		default:
			return nil, errors.New("unexpected historical getter")
		}
		return m.Outputs.Pack(values...)
	}
	if *msg.To != stakeDAOMulticallAddress {
		f.historicalCalls = append(f.historicalCalls, b.Uint64())
		return evaluate(*msg.To, msg.Data)
	}
	aggregate, _ := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
	values, err := aggregate.Methods["aggregate3"].Inputs.Unpack(msg.Data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(values[0], new([]multicallInput)).(*[]multicallInput)
	if len(calls) != 3 {
		return f.boldInspectorFixture.CallContract(ctx, msg, b)
	}
	outputs := []multicallOutput{}
	for _, c := range calls {
		encoded, err := evaluate(c.Target, c.CallData)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, multicallOutput{Success: true, ReturnData: encoded})
	}
	f.historicalCalls = append(f.historicalCalls, b.Uint64())
	return aggregate.Methods["aggregate3"].Outputs.Pack(outputs)
}

func TestBOLDHistoricalSamplerReadsRealPriceServiceAtPastBlock(t *testing.T) {
	base := newBoldInspectorFixture(t)
	f := &boldHistoricalRPCFixture{boldInspectorFixture: base}
	r, c := boldReviewedTestReport(t)
	run := &boldResearchRun{owner: NewBOLDCampaignInspector(f, base, nil), report: &r}
	resolver := &boldPublishedResolver{run: run, headers: map[uint64]*types.Header{}, samples: map[uint64]BOLDHistoricalSample{}, anchors: map[uint64]common.Hash{}}
	point, err := resolver.sample(context.Background(), c.Start)
	if err != nil {
		t.Fatal(err)
	}
	if point.RewardUSD != "1" || point.LPUSD != "1" || point.TVLUSD != "100" || point.BlockNumber >= r.BlockNumber || point.BlockTimestamp > c.Start || c.Start-point.BlockTimestamp > 12 || point.LPPriceSource != "curve_stableswap_ng_nav" || point.RewardPriceSource != "curve_ema_usd" {
		t.Fatalf("not a historical contract valuation: %+v", point)
	}
	for _, b := range f.historicalCalls {
		if b != point.BlockNumber {
			t.Fatal("historical price call used another block")
		}
	}
	before := r.RPCRequests
	if _, err := resolver.sample(context.Background(), c.Start); err != nil || r.RPCRequests != before {
		t.Fatal("immutable sample cache missed")
	}
	if err := resolver.verifyAnchors(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.priceMode = "reorganized"
	if err := resolver.verifyAnchors(context.Background()); err == nil {
		t.Fatal("historical reorg accepted")
	}
	// Leave reserve capacity for rechecking hashes even after pricing is cut off.
	r.RPCRequests = boldHistoricalReadLimit
	if _, err := (boldHistoricalCaller{run, point.BlockNumber}).CallContract(context.Background(), ethereum.CallMsg{}, big.NewInt(int64(point.BlockNumber))); err == nil || r.RPCRequests != boldHistoricalReadLimit {
		t.Fatal("historical reads consumed verification reserve")
	}
}

func TestBOLDPublishedInspectorEndToEndHistoricalEstimates(t *testing.T) {
	base := newBoldInspectorFixture(t)
	p, err := reviewedBOLDPublication()
	if err != nil {
		t.Fatal(err)
	}
	base.root = p.Root
	base.snapshot = 26456929
	f := &boldHistoricalRPCFixture{boldInspectorFixture: base}
	h, _ := f.HeaderByNumber(context.Background(), base.receipt.BlockNumber)
	base.receipt.BlockHash = h.Hash()
	for _, l := range base.receipt.Logs {
		l.BlockHash = h.Hash()
	}
	report, err := NewBOLDPublishedCampaignInspector(f, f).Inspect(context.Background(), 26450710, 26456929)
	if err != nil {
		t.Fatal(err)
	}
	if report.CurrentAPR != nil || report.Status != "funding_verified_with_historical_estimates" || len(report.Campaigns) != 2 || report.RPCRequests > 300 {
		t.Fatalf("invalid integrated estimate: %+v", report)
	}
	for _, c := range report.Campaigns {
		if c.HistoricalAPR == nil || c.Allocation == nil || c.Allocation.Publication == nil || c.Allocation.Valuation.Status != "estimate_available" || len(c.Allocation.Valuation.Samples) != 3 || *c.Allocation.Valuation.AverageCapitalUSD != "100" {
			t.Fatalf("historical evidence incomplete: %+v", c)
		}
	}
}
