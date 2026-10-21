package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"ABIevm/internal/rpcclient"
	"ABIevm/internal/service/prices"
	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
)

var (
	ErrUnsupportedNetwork = errors.New("unsupported StakeDAO network")
	ErrRPCUnavailable     = errors.New("StakeDAO RPC unavailable")
)

const vaultRegisteredABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"gauge","type":"address"},{"indexed":true,"name":"vault","type":"address"},{"indexed":true,"name":"asset","type":"address"},{"indexed":false,"name":"rewardReceiver","type":"address"},{"indexed":false,"name":"protocolId","type":"bytes4"}],"name":"VaultRegistered","type":"event"}]`

type stakeDAORPC interface {
	ChainID(context.Context) (*big.Int, error)
	BlockNumber(context.Context) (uint64, error)
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	CodeAt(context.Context, common.Address, *big.Int) ([]byte, error)
	FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error)
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
	Close()
}

type stakeDAONetwork struct {
	rpcURL          string
	chainID         int64
	controller      common.Address
	deploymentBlock uint64
}

// StakeDAOService owns its RPC client; concurrent HTTP requests use separate instances.
type StakeDAOService struct {
	client          stakeDAORPC
	controller      common.Address
	chainID         int64
	prices          prices.Provider
	snapshotTime    uint64
	deploymentBlock uint64
	configErr       error
	networks        map[string]stakeDAONetwork
	dial            func(context.Context, string) (stakeDAORPC, error)
}

func NewStakeDAOService(priceService prices.Provider) *StakeDAOService {
	rpcURL := os.Getenv("STAKEDAO_ETHEREUM_RPC_URL")
	if rpcURL == "" {
		rpcURL = "https://eth.llamarpc.com"
	}
	deploymentBlock := uint64(23225415)
	var configErr error
	if text := os.Getenv("STAKEDAO_CONTROLLER_DEPLOYMENT_BLOCK"); text != "" {
		var err error
		deploymentBlock, err = strconv.ParseUint(text, 10, 64)
		if err != nil || deploymentBlock == 0 {
			configErr = errors.New("STAKEDAO_CONTROLLER_DEPLOYMENT_BLOCK must be a positive decimal block number")
		}
	}
	return &StakeDAOService{
		prices:    priceService,
		configErr: configErr,
		networks: map[string]stakeDAONetwork{
			"ethereum": {rpcURL: rpcURL, chainID: 1, controller: common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB), deploymentBlock: deploymentBlock},
		},
		dial: func(ctx context.Context, endpoint string) (stakeDAORPC, error) {
			return rpcclient.DialContext(ctx, endpoint)
		},
	}
}

// GetPools discovers registrations on every request; token prices are shared across requests.
func (s *StakeDAOService) GetPools(ctx context.Context, network string) (*dto.StakeDAOPoolsResponse, error) {
	return s.getPoolsWithSelection(ctx, network, nil)
}

func (s *StakeDAOService) getPoolsWithSelection(ctx context.Context, network string, sample *poolSample) (*dto.StakeDAOPoolsResponse, error) {
	config, ok := s.networks[network]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedNetwork, network)
	}
	if sample == nil && s.configErr != nil {
		return nil, s.configErr
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	client, err := s.dial(ctx, config.rpcURL)
	if err != nil {
		return nil, fmt.Errorf("%w: dial: %w", ErrRPCUnavailable, err)
	}
	defer client.Close()
	if sample != nil {
		client = newSampleRPC(client)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: chain ID: %w", ErrRPCUnavailable, err)
	}
	if chainID == nil || chainID.Cmp(big.NewInt(config.chainID)) != 0 {
		return nil, fmt.Errorf("%w: expected chain ID %d", ErrRPCUnavailable, config.chainID)
	}
	worker := &StakeDAOService{client: client, controller: config.controller, chainID: config.chainID, prices: s.prices, deploymentBlock: config.deploymentBlock}
	return worker.getPoolsSelected(ctx, network, sample)
}

func (s *StakeDAOService) getPools(ctx context.Context, network string) (*dto.StakeDAOPoolsResponse, error) {
	return s.getPoolsSelected(ctx, network, nil)
}

func (s *StakeDAOService) getPoolsSelected(ctx context.Context, network string, sample *poolSample) (*dto.StakeDAOPoolsResponse, error) {
	toBlock, err := s.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: latest block: %w", ErrRPCUnavailable, err)
	}
	block := new(big.Int).SetUint64(toBlock)
	header, err := s.client.HeaderByNumber(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("%w: snapshot header: %w", ErrRPCUnavailable, err)
	}
	if header == nil || header.Number == nil || header.Number.Cmp(block) != 0 {
		return nil, errors.New("invalid StakeDAO snapshot header")
	}
	s.snapshotTime = header.Time
	var vaults []StakeDAOVaultInfo
	if sample != nil && len(sample.gauges) > 0 {
		vaults, err = s.currentGaugeRegistrations(ctx, block, sample.gauges)
	} else {
		var fromBlock, logsToBlock uint64
		if sample == nil {
			fromBlock, err = s.controllerStartBlock(toBlock)
			logsToBlock = toBlock
			if err != nil {
				return nil, err
			}
		} else {
			if sample.toBlock > toBlock {
				return nil, errors.New("sample log range extends beyond the current block")
			}
			fromBlock, logsToBlock = sample.fromBlock, sample.toBlock
		}
		vaults, err = DiscoverStakeDAOVaults(ctx, s.client, s.controller, fromBlock, logsToBlock)
	}
	if err != nil {
		return nil, err
	}
	response := &dto.StakeDAOPoolsResponse{
		Network: network, BlockNumber: toBlock,
		Pools: make([]dto.StakeDAOPool, 0, len(vaults)),
	}
	for _, info := range vaults {
		if sample != nil && len(response.Pools) >= sample.limit {
			break
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		registration := s.snapshotCalls(ctx, block,
			readContract(s.controller, protocolControllerABI, "vault", info.Gauge),
			readContract(s.controller, protocolControllerABI, "asset", info.Gauge),
		)
		vault, err := registration[0].address()
		if err != nil {
			return nil, fmt.Errorf("controller.vault(%s): %w", info.Gauge.Hex(), err)
		}
		if vault == (common.Address{}) {
			continue // Registration has since been disabled.
		}
		asset, err := registration[1].address()
		if err != nil {
			return nil, fmt.Errorf("controller.asset(%s): %w", info.Gauge.Hex(), err)
		}
		if asset == (common.Address{}) {
			response.Errors = append(response.Errors, poolIssue(info.Gauge, "asset", "Controller returned a zero asset address"))
			continue
		}
		if vault != info.Vault || asset != info.Asset {
			// Avoid attaching rewardReceiver/protocolId from an obsolete registration.
			response.Errors = append(response.Errors, poolIssue(info.Gauge, "registration", "Registration differs from Controller state at the snapshot block"))
			continue
		}
		pool, issues := s.loadPool(ctx, info, block)
		issues = append(issues, s.loadPoolRewardMechanism(ctx, info, block, &pool)...)
		issues = append(issues, s.loadRewardStreams(ctx, info, block, &pool)...)
		response.Pools = append(response.Pools, pool)
		response.Errors = append(response.Errors, issues...)
	}
	if sample != nil && len(response.Pools) == 0 {
		if len(sample.gauges) > 0 {
			return nil, errors.New("no active vaults for the requested gauges in current Controller state; check the gauge addresses")
		}
		return nil, errors.New("no current matching vault registrations in the sample window; choose another window of up to 100 blocks")
	}
	tokens := make([]common.Address, 0, len(response.Pools)*2)
	for _, pool := range response.Pools {
		tokens = append(tokens, pool.AssetAddress)
		if pool.RewardToken != nil {
			tokens = append(tokens, *pool.RewardToken)
		}
		for _, reward := range pool.Rewards {
			tokens = append(tokens, reward.Token)
		}
	}
	if s.prices != nil {
		// Successful quotes remain usable if other tokens have no price.
		// Per-pool diagnostics below do not expose upstream transport URLs or credentials.
		var priceErr error
		if provider, ok := s.prices.(prices.SnapshotProvider); ok {
			priceErr = provider.EnsurePricesAt(ctx, s.chainID, tokens, prices.Snapshot{Caller: s.client, Block: toBlock, Timestamp: s.snapshotTime})
		} else {
			priceErr = s.prices.EnsurePrices(ctx, s.chainID, tokens)
		}
		if priceErr != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	for i := range response.Pools {
		response.Errors = append(response.Errors, s.valuePool(ctx, &response.Pools[i])...)
		response.Errors = append(response.Errors, s.valueRewards(ctx, &response.Pools[i])...)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client, ok := s.client.(*sampleRPC); ok && client.exhausted {
		return nil, errors.New("sample RPC request budget exhausted; retry with -limit 1")
	}
	finalHeader, err := s.client.HeaderByNumber(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("%w: verify snapshot: %w", ErrRPCUnavailable, err)
	}
	if finalHeader == nil || finalHeader.Hash() != header.Hash() {
		return nil, errors.New("StakeDAO snapshot block was reorganized; retry the request")
	}
	return response, nil
}

// The original live Alchemy binary search located first code at 23225415
// (reported in the user's subsequent eth_getLogs failure). Reuse that start
// instead of requiring archive state on every request. A creation receipt was
// not independently retrieved in this environment; provenance is in README.md.
func (s *StakeDAOService) controllerStartBlock(toBlock uint64) (uint64, error) {
	if s.deploymentBlock == 0 || s.deploymentBlock > toBlock {
		return 0, errors.New("Controller deployment block is unknown or beyond the snapshot")
	}
	return s.deploymentBlock, nil
}

type StakeDAOVaultInfo struct {
	Gauge          common.Address
	Vault          common.Address
	Asset          common.Address
	RewardReceiver common.Address
	ProtocolID     [4]byte
}

type vaultLogReader interface {
	FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error)
}

// DiscoverStakeDAOVaults reads a bounded history, keeping the latest registration per gauge.
func DiscoverStakeDAOVaults(ctx context.Context, client vaultLogReader, controller common.Address, fromBlock, toBlock uint64) ([]StakeDAOVaultInfo, error) {
	if fromBlock > toBlock {
		return nil, errors.New("invalid StakeDAO discovery block range")
	}
	parsed, err := abi.JSON(strings.NewReader(vaultRegisteredABI))
	if err != nil {
		return nil, err
	}
	event := parsed.Events["VaultRegistered"]
	logs, err := readRegistrationLogs(ctx, client, controller, event.ID, fromBlock, toBlock)
	if err != nil {
		return nil, err
	}
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		if logs[i].TxIndex != logs[j].TxIndex {
			return logs[i].TxIndex < logs[j].TxIndex
		}
		return logs[i].Index < logs[j].Index
	})
	registrations := make(map[common.Address]StakeDAOVaultInfo)
	for _, entry := range logs {
		if entry.Removed {
			continue
		}
		if entry.Address != controller || entry.BlockNumber < fromBlock || entry.BlockNumber > toBlock || len(entry.Topics) != 4 || entry.Topics[0] != event.ID {
			return nil, fmt.Errorf("invalid VaultRegistered log at block %d", entry.BlockNumber)
		}
		var decoded struct {
			RewardReceiver common.Address
			ProtocolId     [4]byte
		}
		if err := parsed.UnpackIntoInterface(&decoded, "VaultRegistered", entry.Data); err != nil {
			return nil, fmt.Errorf("decode VaultRegistered at block %d: %w", entry.BlockNumber, err)
		}
		for _, topic := range entry.Topics[1:] {
			if new(big.Int).SetBytes(topic[:12]).Sign() != 0 {
				return nil, errors.New("invalid address padding in VaultRegistered")
			}
		}
		info := StakeDAOVaultInfo{
			Gauge:          common.BytesToAddress(entry.Topics[1].Bytes()),
			Vault:          common.BytesToAddress(entry.Topics[2].Bytes()),
			Asset:          common.BytesToAddress(entry.Topics[3].Bytes()),
			RewardReceiver: decoded.RewardReceiver, ProtocolID: decoded.ProtocolId,
		}
		if info.Gauge == (common.Address{}) || info.Vault == (common.Address{}) || info.Asset == (common.Address{}) {
			return nil, errors.New("zero gauge, vault or asset in VaultRegistered")
		}
		registrations[info.Gauge] = info
	}
	result := make([]StakeDAOVaultInfo, 0, len(registrations))
	for _, info := range registrations {
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Gauge.Hex() < result[j].Gauge.Hex() })
	return result, nil
}

func (s *StakeDAOService) snapshotCall(ctx context.Context, block *big.Int, contract common.Address, abiJSON, method string, args ...interface{}) ([]interface{}, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	raw, err := s.client.CallContract(ctx, ethereum.CallMsg{To: &contract, Data: data}, block)
	if err != nil {
		return nil, err
	}
	return parsed.Unpack(method, raw)
}

func (s *StakeDAOService) snapshotAddress(ctx context.Context, block *big.Int, contract common.Address, abiJSON, method string, args ...interface{}) (common.Address, error) {
	out, err := s.snapshotCall(ctx, block, contract, abiJSON, method, args...)
	return (snapshotResult{values: out, err: err}).address()
}

func (s *StakeDAOService) loadPool(ctx context.Context, info StakeDAOVaultInfo, block *big.Int) (dto.StakeDAOPool, []dto.StakeDAOPoolError) {
	pool := dto.StakeDAOPool{
		GaugeAddress: info.Gauge, VaultAddress: info.Vault, AssetAddress: info.Asset,
		RewardReceiver: info.RewardReceiver, ProtocolID: fmt.Sprintf("0x%x", info.ProtocolID),
		Source: "onchain", UpdatedAt: time.Now().UTC(),
		APRStatus: "unavailable",
		APRScope:  "rewards",
		Rewards:   make([]dto.StakeDAOReward, 0),
	}
	var issues []dto.StakeDAOPoolError
	data := s.snapshotCalls(ctx, block,
		readContract(info.Vault, erc20NameABI, "name"),
		readContract(info.Vault, erc20SymbolABI, "symbol"),
		readContract(info.Asset, erc20SymbolABI, "symbol"),
		readContract(info.Asset, erc20DecimalsABI, "decimals"),
		readContract(info.Vault, rewardVaultERC4626ABI, "totalAssets"),
	)
	readString := func(result snapshotResult, method, field string) string {
		out, err := result.values, result.err
		if err == nil && len(out) == 1 {
			if value, ok := out[0].(string); ok && strings.TrimSpace(value) != "" {
				return value
			}
		}
		issues = append(issues, poolIssue(info.Gauge, field, method+"() could not be read or decoded"))
		return ""
	}
	pool.Name = readString(data[0], "name", "name")
	pool.Symbol = readString(data[1], "symbol", "symbol")
	pool.AssetSymbol = readString(data[2], "symbol", "assetSymbol")
	pool.Name = nonEmpty(pool.Name, pool.Symbol, info.Gauge.Hex())
	decimalsOut, decimalsErr := data[3].values, data[3].err
	assetsOut, assetsErr := data[4].values, data[4].err
	var decimals uint8
	var decimalsOK bool
	if decimalsErr == nil && len(decimalsOut) == 1 {
		decimals, decimalsOK = decimalsOut[0].(uint8)
	}
	if !decimalsOK {
		issues = append(issues, poolIssue(info.Gauge, "assetDecimals", "decimals() could not be read or decoded"))
	} else {
		pool.AssetDecimals = &decimals
	}
	var assets *big.Int
	if assetsErr == nil && len(assetsOut) == 1 {
		assets, _ = assetsOut[0].(*big.Int)
	}
	if assets == nil {
		issues = append(issues, poolIssue(info.Gauge, "assetAmount", "totalAssets() could not be read or decoded"))
	} else if decimalsOK {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
		amount := new(big.Rat).SetFrac(assets, divisor).FloatString(int(decimals))
		pool.AssetAmount = &amount
	}
	return pool, issues
}

func poolIssue(gauge common.Address, field, message string) dto.StakeDAOPoolError {
	return dto.StakeDAOPoolError{GaugeAddress: gauge, Field: field, Message: message}
}

func (s *StakeDAOService) valuePool(ctx context.Context, pool *dto.StakeDAOPool) []dto.StakeDAOPoolError {
	if pool.AssetAmount == nil {
		return []dto.StakeDAOPoolError{poolIssue(pool.GaugeAddress, "tvl", "Underlying asset amount is unavailable")}
	}
	if s.prices == nil {
		return []dto.StakeDAOPoolError{poolIssue(pool.GaugeAddress, "tvl", "Token price service is unavailable")}
	}
	price, err := s.prices.GetTokenPrice(ctx, pool.AssetAddress, s.chainID)
	if err != nil {
		return []dto.StakeDAOPoolError{poolIssue(pool.GaugeAddress, "tvl", "Fresh USD price for the underlying asset/LP token is unavailable")}
	}
	amount, err := decimal.NewFromString(*pool.AssetAmount)
	if err != nil || amount.IsNegative() || !price.IsPositive() {
		return []dto.StakeDAOPoolError{poolIssue(pool.GaugeAddress, "tvl", "Invalid asset amount or USD price")}
	}
	value, _ := amount.Mul(price).Float64()
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return []dto.StakeDAOPoolError{poolIssue(pool.GaugeAddress, "tvl", "USD TVL cannot be represented")}
	}
	pool.TVL = &value
	pool.Source = "onchain+price-cache"
	return nil
}

// V2 primary rewards are accounted separately from RewardVault's extra reward streams.
// Getter signatures are confirmed by the published staking-v2 audits; no guessed reward_data ABI is used.
func (s *StakeDAOService) loadPoolRewardMechanism(ctx context.Context, info StakeDAOVaultInfo, block *big.Int, pool *dto.StakeDAOPool) []dto.StakeDAOPoolError {
	protocol := fmt.Sprintf("0x%x", info.ProtocolID)
	if protocol != "0xc715e373" && protocol != "0xb774acb8" {
		pool.RewardMechanism = "unsupported"
		return []dto.StakeDAOPoolError{poolIssue(info.Gauge, "apr", "Unverified reward mechanism for this protocol ID")}
	}
	pool.RewardMechanism = "accountant"
	data := s.snapshotCalls(ctx, block,
		readContract(info.Vault, rewardVaultAccountantABI, "ACCOUNTANT"),
		readContract(s.controller, protocolControllerABI, "accountant", info.ProtocolID),
	)
	accountant, err := data[0].address()
	if err != nil || accountant == (common.Address{}) {
		return []dto.StakeDAOPoolError{poolIssue(info.Gauge, "apr", "RewardVault Accountant could not be read or decoded")}
	}
	pool.AccountantAddress = &accountant
	current, err := data[1].address()
	if err != nil || current != accountant {
		return []dto.StakeDAOPoolError{poolIssue(info.Gauge, "apr", "Vault Accountant does not match the current Controller Accountant")}
	}
	token, err := s.snapshotAddress(ctx, block, accountant, accountantRewardTokenABI, "REWARD_TOKEN")
	if err != nil || token == (common.Address{}) {
		return []dto.StakeDAOPoolError{poolIssue(info.Gauge, "apr", "Accountant reward token could not be read or decoded")}
	}
	pool.RewardToken = &token
	return nil
}

func (s *StakeDAOService) unsignedCall(ctx context.Context, block *big.Int, address common.Address, abiJSON, method string, args ...interface{}) (*big.Int, error) {
	out, err := s.snapshotCall(ctx, block, address, abiJSON, method, args...)
	return (snapshotResult{values: out, err: err}).unsigned()
}

func (s *StakeDAOService) addressList(ctx context.Context, block *big.Int, address common.Address, abiJSON, method string, args ...interface{}) ([]common.Address, error) {
	out, err := s.snapshotCall(ctx, block, address, abiJSON, method, args...)
	if err != nil {
		return nil, err
	}
	if len(out) != 1 {
		return nil, errors.New("invalid address list")
	}
	values, ok := out[0].([]common.Address)
	if !ok || len(values) > 16 {
		return nil, errors.New("unsupported address list")
	}
	return values, nil
}

func (s *StakeDAOService) loadRewardStreams(ctx context.Context, info StakeDAOVaultInfo, block *big.Int, pool *dto.StakeDAOPool) []dto.StakeDAOPoolError {
	if pool.RewardMechanism != "accountant" {
		return nil
	}
	var issues []dto.StakeDAOPoolError
	if pool.RewardToken != nil {
		rate, err := s.curvePrimaryRate(ctx, info, block, pool)
		if err != nil {
			issues = append(issues, poolIssue(info.Gauge, "apr", "Primary reward-rate/fee/allocation data is unavailable or unsupported"))
		} else {
			value := rate.String()
			pool.Rewards = append(pool.Rewards, dto.StakeDAOReward{Token: *pool.RewardToken, Source: "curve-primary-net", RatePerSecond: &value})
		}
	}
	tokens, err := s.addressList(ctx, block, info.Vault, vaultV2ABI, "getRewardTokens")
	if err != nil {
		return append(issues, poolIssue(info.Gauge, "apr", "RewardVault extra reward list is unavailable"))
	}
	seen := map[common.Address]bool{}
	var reads []snapshotRead
	for _, token := range tokens {
		if token != (common.Address{}) && !seen[token] {
			reads = append(reads, readContract(info.Vault, vaultV2ABI, "rewardData", token), readContract(token, erc20DecimalsABI, "decimals"))
			seen[token] = true
		}
	}
	extra := s.snapshotCalls(ctx, block, reads...)
	clear(seen)
	index := 0
	for _, token := range tokens {
		if token == (common.Address{}) || seen[token] {
			issues = append(issues, poolIssue(info.Gauge, "apr", "Invalid or duplicate extra reward token"))
			continue
		}
		seen[token] = true
		reward := dto.StakeDAOReward{Token: token, Source: "vault-extra"}
		out, err := extra[index].values, extra[index].err
		decimals, decErr := extra[index+1].unsigned()
		index += 2
		if err == nil && len(out) == 5 {
			finish, okFinish := out[2].(uint32)
			raw, okRate := out[3].(*big.Int)
			distributor, okDistributor := out[0].(common.Address)
			if okFinish && okRate && raw != nil && okDistributor && distributor != (common.Address{}) {
				period := uint64(finish)
				reward.PeriodFinish = &period
				value := "0"
				if period <= s.snapshotTime || raw.Sign() == 0 {
					reward.RatePerSecond = &value
				} else if decErr == nil && decimals.IsUint64() && decimals.Uint64() <= 255 {
					// Verified RewardVault rate is scaled by 1e18 in addition to ERC20 decimals.
					value = decimal.NewFromBigInt(raw, -18-int32(decimals.Uint64())).String()
					reward.RatePerSecond = &value
				}
			}
		}
		if reward.RatePerSecond == nil {
			issues = append(issues, poolIssue(info.Gauge, "apr", "Extra reward rate or decimals could not be read"))
		}
		pool.Rewards = append(pool.Rewards, reward)
	}
	pool.RewardsComplete = len(issues) == 0 && pool.RewardToken != nil
	return issues
}

// Snapshot projection of Curve emissions, after Accountant fees. Convex primary
// CRV is fee-exempt from the protocol fee; CHECKPOINT harvest fees apply to both targets.
func (s *StakeDAOService) curvePrimaryRate(ctx context.Context, info StakeDAOVaultInfo, block *big.Int, pool *dto.StakeDAOPool) (decimal.Decimal, error) {
	if s.chainID != 1 || info.ProtocolID != [4]byte{0xc7, 0x15, 0xe3, 0x73} || pool.AccountantAddress == nil || pool.RewardToken == nil {
		return decimal.Zero, errors.New("unsupported primary model")
	}
	strategy, err := s.snapshotAddress(ctx, block, info.Vault, vaultV2ABI, "strategy")
	if err != nil || strategy == (common.Address{}) {
		return decimal.Zero, errors.New("strategy unavailable")
	}
	data := s.snapshotCalls(ctx, block,
		readContract(strategy, strategyV2ABI, "ACCOUNTANT"),
		readContract(strategy, strategyV2ABI, "balanceOf", info.Gauge),
		readContract(strategy, strategyV2ABI, "REWARD_TOKEN"),
		readContract(strategy, strategyV2ABI, "LOCKER"),
		readContract(info.Vault, vaultV2ABI, "allocator"),
		readContract(*pool.AccountantAddress, accountantV2ABI, "getProtocolFeePercent"),
		readContract(*pool.AccountantAddress, accountantV2ABI, "getHarvestFeePercent"),
		readContract(info.Vault, vaultV2ABI, "POLICY"),
		readContract(*pool.RewardToken, erc20DecimalsABI, "decimals"),
	)
	strategyAccountant, err := data[0].address()
	if err != nil || strategyAccountant != *pool.AccountantAddress {
		return decimal.Zero, errors.New("strategy Accountant mismatch")
	}
	physical, err := data[1].unsigned()
	if err != nil || pool.AssetAmount == nil || pool.AssetDecimals == nil {
		return decimal.Zero, errors.New("physical allocation unavailable")
	}
	assets, err := decimal.NewFromString(*pool.AssetAmount)
	if err != nil || assets.Shift(int32(*pool.AssetDecimals)).BigInt().Cmp(physical) != 0 {
		return decimal.Zero, errors.New("vault supply differs from physical allocation")
	}
	strategyToken, err := data[2].address()
	if err != nil || strategyToken != *pool.RewardToken {
		return decimal.Zero, errors.New("strategy reward token mismatch")
	}
	locker, err := data[3].address()
	if err != nil || locker == (common.Address{}) {
		return decimal.Zero, errors.New("locker unavailable")
	}
	allocator, err := data[4].address()
	if err != nil {
		return decimal.Zero, err
	}
	targets, err := s.addressList(ctx, block, allocator, allocatorV2ABI, "getAllocationTargets", info.Gauge)
	if err != nil {
		return decimal.Zero, err
	}
	protocolFee, err := data[5].unsigned()
	if err != nil {
		return decimal.Zero, err
	}
	harvestFee, err := data[6].unsigned()
	if err != nil {
		return decimal.Zero, err
	}
	policy, err := data[7].unsigned()
	if err != nil || !policy.IsUint64() || policy.Uint64() > 1 {
		return decimal.Zero, errors.New("unsupported harvest policy")
	}
	if policy.Uint64() == 1 {
		harvestFee = new(big.Int)
	}
	pf, hf := decimal.NewFromBigInt(protocolFee, -18), decimal.NewFromBigInt(harvestFee, -18)
	if pf.IsNegative() || hf.IsNegative() || pf.Add(hf).GreaterThan(decimal.RequireFromString("0.4")) {
		return decimal.Zero, errors.New("invalid fee fractions")
	}
	lockerRate, sidecarRate := decimal.Zero, decimal.Zero
	seen := map[common.Address]bool{}
	for _, target := range targets {
		if target == (common.Address{}) || seen[target] {
			return decimal.Zero, errors.New("invalid allocation target")
		}
		seen[target] = true
		if target == locker {
			out, err := s.snapshotCall(ctx, block, info.Gauge, curveGaugeABI, "is_killed")
			if err != nil || len(out) != 1 {
				return decimal.Zero, errors.New("gauge status unavailable")
			}
			killed, ok := out[0].(bool)
			if !ok {
				return decimal.Zero, errors.New("invalid gauge status")
			}
			if killed {
				continue
			}
			gaugeData := s.snapshotCalls(ctx, block,
				readContract(info.Gauge, curveGaugeABI, "inflation_rate"),
				readContract(common.HexToAddress("0x2F50D538606Fa9EDD2B11E2446BEb18C9D5846bB"), curveControllerABI, "gauge_relative_weight", info.Gauge, new(big.Int).SetUint64(s.snapshotTime)),
				readContract(info.Gauge, curveGaugeABI, "working_balances", locker),
				readContract(info.Gauge, curveGaugeABI, "working_supply"),
			)
			inflation, err := gaugeData[0].unsigned()
			if err != nil {
				return decimal.Zero, err
			}
			weight, err := gaugeData[1].unsigned()
			if err != nil {
				return decimal.Zero, err
			}
			working, err := gaugeData[2].unsigned()
			if err != nil {
				return decimal.Zero, err
			}
			supply, err := gaugeData[3].unsigned()
			if err != nil {
				return decimal.Zero, err
			}
			if working.Sign() == 0 {
				continue
			}
			if supply.Sign() == 0 || working.Cmp(supply) > 0 || weight.Cmp(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) > 0 {
				return decimal.Zero, errors.New("invalid gauge working supply/weight")
			}
			lockerRate = decimal.NewFromBigInt(inflation, 0).Mul(decimal.NewFromBigInt(weight, -18)).Mul(decimal.NewFromBigInt(working, 0)).DivRound(decimal.NewFromBigInt(supply, 0), 128)
		} else {
			sidecarData := s.snapshotCalls(ctx, block,
				readContract(target, sidecarV2ABI, "ACCOUNTANT"),
				readContract(target, sidecarV2ABI, "REWARD_TOKEN"),
				readContract(target, sidecarV2ABI, "gauge"),
				readContract(target, sidecarV2ABI, "baseRewardPool"),
			)
			sidecarAccountant, err := sidecarData[0].address()
			if err != nil || sidecarAccountant != *pool.AccountantAddress {
				return decimal.Zero, errors.New("sidecar Accountant mismatch")
			}
			token, err := sidecarData[1].address()
			if err != nil || token != *pool.RewardToken {
				return decimal.Zero, errors.New("unsupported sidecar")
			}
			gauge, err := sidecarData[2].address()
			if err != nil || gauge != info.Gauge {
				return decimal.Zero, errors.New("sidecar gauge mismatch")
			}
			rewardPool, err := sidecarData[3].address()
			if err != nil {
				return decimal.Zero, err
			}
			rewardData := s.snapshotCalls(ctx, block,
				readContract(rewardPool, convexPoolABI, "rewardToken"),
				readContract(rewardPool, convexPoolABI, "periodFinish"),
			)
			token, err = rewardData[0].address()
			if err != nil || token != *pool.RewardToken {
				return decimal.Zero, errors.New("Convex reward token mismatch")
			}
			finish, err := rewardData[1].unsigned()
			if err != nil || !finish.IsUint64() {
				return decimal.Zero, errors.New("invalid reward period")
			}
			if finish.Uint64() <= s.snapshotTime {
				continue
			}
			allocation := s.snapshotCalls(ctx, block,
				readContract(target, sidecarV2ABI, "balanceOf"),
				readContract(rewardPool, convexPoolABI, "totalSupply"),
				readContract(rewardPool, convexPoolABI, "rewardRate"),
			)
			balance, err := allocation[0].unsigned()
			if err != nil {
				return decimal.Zero, err
			}
			if balance.Sign() == 0 {
				continue
			}
			supply, err := allocation[1].unsigned()
			if err != nil || supply == nil || supply.Sign() == 0 || balance.Cmp(supply) > 0 {
				return decimal.Zero, errors.New("invalid Convex supply")
			}
			rate, err := allocation[2].unsigned()
			if err != nil {
				return decimal.Zero, err
			}
			sidecarRate = sidecarRate.Add(decimal.NewFromBigInt(rate, 0).Mul(decimal.NewFromBigInt(balance, 0)).DivRound(decimal.NewFromBigInt(supply, 0), 128))
		}
	}
	decimals, err := data[8].unsigned()
	if err != nil || decimals.Uint64() > 255 {
		return decimal.Zero, errors.New("reward decimals unavailable")
	}
	net := netPrimaryRate(lockerRate, sidecarRate, pf, hf)
	return net.Shift(-int32(decimals.Uint64())), nil
}

func netPrimaryRate(locker, sidecar, protocolFee, harvestFee decimal.Decimal) decimal.Decimal {
	return locker.Add(sidecar).Sub(locker.Mul(protocolFee)).Sub(locker.Add(sidecar).Mul(harvestFee))
}

func (s *StakeDAOService) valueRewards(ctx context.Context, pool *dto.StakeDAOPool) []dto.StakeDAOPoolError {
	if pool.TVL == nil || *pool.TVL <= 0 {
		return []dto.StakeDAOPoolError{poolIssue(pool.GaugeAddress, "apr", "Positive USD TVL is required for APR")}
	}
	complete := pool.RewardsComplete
	primaryFound := false
	total := decimal.Zero
	valued := 0
	var issues []dto.StakeDAOPoolError
	for i := range pool.Rewards {
		reward := &pool.Rewards[i]
		if reward.Source == "curve-primary-net" {
			primaryFound = true
		}
		if reward.RatePerSecond == nil {
			complete = false
			continue
		}
		rate, err := decimal.NewFromString(*reward.RatePerSecond)
		if err != nil || rate.IsNegative() {
			complete = false
			continue
		}
		apr := decimal.Zero
		if rate.IsPositive() {
			if s.prices == nil {
				complete = false
				continue
			}
			price, err := s.prices.GetTokenPrice(ctx, reward.Token, s.chainID)
			if err != nil || !price.IsPositive() {
				complete = false
				issues = append(issues, poolIssue(pool.GaugeAddress, "apr", "Active reward token USD price is unavailable"))
				continue
			}
			apr = rate.Mul(price).Mul(decimal.NewFromInt(31536000)).DivRound(decimal.NewFromFloat(*pool.TVL), 128).Mul(decimal.NewFromInt(100))
		}
		value, _ := apr.Float64()
		if math.IsInf(value, 0) || math.IsNaN(value) {
			complete = false
			continue
		}
		reward.APR = &value
		total = total.Add(apr)
		valued++
	}
	if valued > 0 {
		value, _ := total.Float64()
		if !math.IsInf(value, 0) && !math.IsNaN(value) {
			pool.APR = &value
			pool.APRStatus = "partial"
			if complete && primaryFound {
				pool.APRStatus = "available"
			}
		}
	}
	return issues
}
