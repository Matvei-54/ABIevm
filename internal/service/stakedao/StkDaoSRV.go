package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"sort"
	"strings"
	"time"

	"ABIevm/internal/service/prices"
	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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
	rpcURL     string
	chainID    int64
	controller common.Address
}

// StakeDAOService owns its RPC client; concurrent HTTP requests use separate instances.
type StakeDAOService struct {
	client       stakeDAORPC
	controller   common.Address
	chainID      int64
	prices       prices.Provider
	snapshotTime uint64
	networks     map[string]stakeDAONetwork
	dial         func(context.Context, string) (stakeDAORPC, error)
}

func NewStakeDAOService(priceService prices.Provider) *StakeDAOService {
	rpcURL := os.Getenv("STAKEDAO_ETHEREUM_RPC_URL")
	if rpcURL == "" {
		rpcURL = "https://eth.llamarpc.com"
	}
	return &StakeDAOService{
		prices: priceService,
		networks: map[string]stakeDAONetwork{
			"ethereum": {rpcURL: rpcURL, chainID: 1, controller: common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)},
		},
		dial: func(ctx context.Context, endpoint string) (stakeDAORPC, error) {
			return ethclient.DialContext(ctx, endpoint)
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
	worker := &StakeDAOService{client: client, controller: config.controller, chainID: config.chainID, prices: s.prices}
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
			fromBlock, err = s.controllerDeploymentBlock(ctx, toBlock)
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
		vault, err := s.snapshotAddress(ctx, block, s.controller, protocolControllerABI, "vault", info.Gauge)
		if err != nil {
			return nil, fmt.Errorf("controller.vault(%s): %w", info.Gauge.Hex(), err)
		}
		if vault == (common.Address{}) {
			continue // Registration has since been disabled.
		}
		asset, err := s.snapshotAddress(ctx, block, s.controller, protocolControllerABI, "asset", info.Gauge)
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
		if err := s.prices.EnsurePrices(ctx, s.chainID, tokens); err != nil && ctx.Err() != nil {
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

// Locate the first block containing Controller code, rather than inventing a deployment block.
// Historical eth_getCode must be supported by the configured RPC provider.
func (s *StakeDAOService) controllerDeploymentBlock(ctx context.Context, toBlock uint64) (uint64, error) {
	code, err := s.client.CodeAt(ctx, s.controller, new(big.Int).SetUint64(toBlock))
	if err != nil {
		return 0, fmt.Errorf("%w: Controller code: %w", ErrRPCUnavailable, err)
	}
	if len(code) == 0 {
		return 0, errors.New("StakeDAO Controller has no code on the selected network")
	}
	low, high := uint64(0), toBlock
	for low < high {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		mid := low + (high-low)/2
		code, err := s.client.CodeAt(ctx, s.controller, new(big.Int).SetUint64(mid))
		if err != nil {
			return 0, fmt.Errorf("%w: historical Controller code at block %d: %w", ErrRPCUnavailable, mid, err)
		}
		if len(code) == 0 {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low, nil
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
	registrations := make(map[common.Address]StakeDAOVaultInfo)
	for start := fromBlock; ; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := toBlock
		if toBlock-start >= 2000 {
			end = start + 1999
		}
		logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
			Addresses: []common.Address{controller}, Topics: [][]common.Hash{{event.ID}},
			FromBlock: new(big.Int).SetUint64(start), ToBlock: new(big.Int).SetUint64(end),
		})
		if err != nil {
			return nil, fmt.Errorf("%w: VaultRegistered blocks %d-%d: %w", ErrRPCUnavailable, start, end, err)
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
		for _, entry := range logs {
			if entry.Removed {
				continue
			}
			if entry.Address != controller || entry.BlockNumber < start || entry.BlockNumber > end || len(entry.Topics) != 4 || entry.Topics[0] != event.ID {
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
		if end == toBlock {
			break
		}
		start = end + 1
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
	if err != nil {
		return common.Address{}, err
	}
	if len(out) != 1 {
		return common.Address{}, errors.New("invalid address output")
	}
	address, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, errors.New("contract output is not an address")
	}
	return address, nil
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
	readString := func(contract common.Address, abiJSON, method, field string) string {
		out, err := s.snapshotCall(ctx, block, contract, abiJSON, method)
		if err == nil && len(out) == 1 {
			if value, ok := out[0].(string); ok && strings.TrimSpace(value) != "" {
				return value
			}
		}
		issues = append(issues, poolIssue(info.Gauge, field, method+"() could not be read or decoded"))
		return ""
	}
	pool.Name = readString(info.Vault, erc20NameABI, "name", "name")
	pool.Symbol = readString(info.Vault, erc20SymbolABI, "symbol", "symbol")
	pool.AssetSymbol = readString(info.Asset, erc20SymbolABI, "symbol", "assetSymbol")
	pool.Name = nonEmpty(pool.Name, pool.Symbol, info.Gauge.Hex())
	decimalsOut, decimalsErr := s.snapshotCall(ctx, block, info.Asset, erc20DecimalsABI, "decimals")
	assetsOut, assetsErr := s.snapshotCall(ctx, block, info.Vault, rewardVaultERC4626ABI, "totalAssets")
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
	pool.Source = "onchain+external-prices"
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
	accountant, err := s.snapshotAddress(ctx, block, info.Vault, rewardVaultAccountantABI, "ACCOUNTANT")
	if err != nil || accountant == (common.Address{}) {
		return []dto.StakeDAOPoolError{poolIssue(info.Gauge, "apr", "RewardVault Accountant could not be read or decoded")}
	}
	pool.AccountantAddress = &accountant
	current, err := s.snapshotAddress(ctx, block, s.controller, protocolControllerABI, "accountant", info.ProtocolID)
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
	if err != nil {
		return nil, err
	}
	if len(out) != 1 {
		return nil, errors.New("invalid unsigned output")
	}
	switch value := out[0].(type) {
	case *big.Int:
		if value != nil && value.Sign() >= 0 {
			return value, nil
		}
	case uint8:
		return new(big.Int).SetUint64(uint64(value)), nil
	case uint32:
		return new(big.Int).SetUint64(uint64(value)), nil
	}
	return nil, errors.New("invalid unsigned output type")
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
	for _, token := range tokens {
		if token == (common.Address{}) || seen[token] {
			issues = append(issues, poolIssue(info.Gauge, "apr", "Invalid or duplicate extra reward token"))
			continue
		}
		seen[token] = true
		reward := dto.StakeDAOReward{Token: token, Source: "vault-extra"}
		out, err := s.snapshotCall(ctx, block, info.Vault, vaultV2ABI, "rewardData", token)
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
				} else if decimals, decErr := s.unsignedCall(ctx, block, token, erc20DecimalsABI, "decimals"); decErr == nil && decimals.Uint64() <= 255 {
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
	strategyAccountant, err := s.snapshotAddress(ctx, block, strategy, strategyV2ABI, "ACCOUNTANT")
	if err != nil || strategyAccountant != *pool.AccountantAddress {
		return decimal.Zero, errors.New("strategy Accountant mismatch")
	}
	physical, err := s.unsignedCall(ctx, block, strategy, strategyV2ABI, "balanceOf", info.Gauge)
	if err != nil || pool.AssetAmount == nil || pool.AssetDecimals == nil {
		return decimal.Zero, errors.New("physical allocation unavailable")
	}
	assets, err := decimal.NewFromString(*pool.AssetAmount)
	if err != nil || assets.Shift(int32(*pool.AssetDecimals)).BigInt().Cmp(physical) != 0 {
		return decimal.Zero, errors.New("vault supply differs from physical allocation")
	}
	strategyToken, err := s.snapshotAddress(ctx, block, strategy, strategyV2ABI, "REWARD_TOKEN")
	if err != nil || strategyToken != *pool.RewardToken {
		return decimal.Zero, errors.New("strategy reward token mismatch")
	}
	locker, err := s.snapshotAddress(ctx, block, strategy, strategyV2ABI, "LOCKER")
	if err != nil || locker == (common.Address{}) {
		return decimal.Zero, errors.New("locker unavailable")
	}
	allocator, err := s.snapshotAddress(ctx, block, info.Vault, vaultV2ABI, "allocator")
	if err != nil {
		return decimal.Zero, err
	}
	targets, err := s.addressList(ctx, block, allocator, allocatorV2ABI, "getAllocationTargets", info.Gauge)
	if err != nil {
		return decimal.Zero, err
	}
	protocolFee, err := s.unsignedCall(ctx, block, *pool.AccountantAddress, accountantV2ABI, "getProtocolFeePercent")
	if err != nil {
		return decimal.Zero, err
	}
	harvestFee, err := s.unsignedCall(ctx, block, *pool.AccountantAddress, accountantV2ABI, "getHarvestFeePercent")
	if err != nil {
		return decimal.Zero, err
	}
	policy, err := s.unsignedCall(ctx, block, info.Vault, vaultV2ABI, "POLICY")
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
			inflation, err := s.unsignedCall(ctx, block, info.Gauge, curveGaugeABI, "inflation_rate")
			if err != nil {
				return decimal.Zero, err
			}
			weight, err := s.unsignedCall(ctx, block, common.HexToAddress("0x2F50D538606Fa9EDD2B11E2446BEb18C9D5846bB"), curveControllerABI, "gauge_relative_weight", info.Gauge, new(big.Int).SetUint64(s.snapshotTime))
			if err != nil {
				return decimal.Zero, err
			}
			working, err := s.unsignedCall(ctx, block, info.Gauge, curveGaugeABI, "working_balances", locker)
			if err != nil {
				return decimal.Zero, err
			}
			supply, err := s.unsignedCall(ctx, block, info.Gauge, curveGaugeABI, "working_supply")
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
			sidecarAccountant, err := s.snapshotAddress(ctx, block, target, sidecarV2ABI, "ACCOUNTANT")
			if err != nil || sidecarAccountant != *pool.AccountantAddress {
				return decimal.Zero, errors.New("sidecar Accountant mismatch")
			}
			token, err := s.snapshotAddress(ctx, block, target, sidecarV2ABI, "REWARD_TOKEN")
			if err != nil || token != *pool.RewardToken {
				return decimal.Zero, errors.New("unsupported sidecar")
			}
			gauge, err := s.snapshotAddress(ctx, block, target, sidecarV2ABI, "gauge")
			if err != nil || gauge != info.Gauge {
				return decimal.Zero, errors.New("sidecar gauge mismatch")
			}
			rewardPool, err := s.snapshotAddress(ctx, block, target, sidecarV2ABI, "baseRewardPool")
			if err != nil {
				return decimal.Zero, err
			}
			token, err = s.snapshotAddress(ctx, block, rewardPool, convexPoolABI, "rewardToken")
			if err != nil || token != *pool.RewardToken {
				return decimal.Zero, errors.New("Convex reward token mismatch")
			}
			finish, err := s.unsignedCall(ctx, block, rewardPool, convexPoolABI, "periodFinish")
			if err != nil || !finish.IsUint64() {
				return decimal.Zero, errors.New("invalid reward period")
			}
			if finish.Uint64() <= s.snapshotTime {
				continue
			}
			balance, err := s.unsignedCall(ctx, block, target, sidecarV2ABI, "balanceOf")
			if err != nil {
				return decimal.Zero, err
			}
			if balance.Sign() == 0 {
				continue
			}
			supply, err := s.unsignedCall(ctx, block, rewardPool, convexPoolABI, "totalSupply")
			if err != nil || supply == nil || supply.Sign() == 0 || balance.Cmp(supply) > 0 {
				return decimal.Zero, errors.New("invalid Convex supply")
			}
			rate, err := s.unsignedCall(ctx, block, rewardPool, convexPoolABI, "rewardRate")
			if err != nil {
				return decimal.Zero, err
			}
			sidecarRate = sidecarRate.Add(decimal.NewFromBigInt(rate, 0).Mul(decimal.NewFromBigInt(balance, 0)).DivRound(decimal.NewFromBigInt(supply, 0), 128))
		}
	}
	decimals, err := s.unsignedCall(ctx, block, *pool.RewardToken, erc20DecimalsABI, "decimals")
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
