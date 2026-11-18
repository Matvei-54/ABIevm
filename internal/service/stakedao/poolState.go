package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
)

// ProtocolController interface confirmed by verified RewardVault sources;
// no pause flag is inferred from APR, reward periods, price availability or age.
const controllerStateABI = `[{"type":"function","name":"isPaused","inputs":[{"type":"bytes4"}],"outputs":[{"type":"bool"}],"stateMutability":"view"}]`

const curveGaugeControllerEthereum = "0x2F50D538606Fa9EDD2B11E2446BEb18C9D5846bB"

func stateBool(value bool) *bool { return &value }

func stateIssue(state *dto.StakeDAOPoolState, field, code string, contract common.Address, method string) {
	state.Issues = append(state.Issues, dto.StakeDAOStateIssue{Field: field, Code: code, Contract: contract, Method: method})
}

func readStateBool(state *dto.StakeDAOPoolState, value snapshotResult, field string, contract common.Address, method string) *bool {
	if value.err == nil && len(value.values) == 1 {
		if b, ok := value.values[0].(bool); ok {
			return stateBool(b)
		}
	}
	stateIssue(state, field, "getter_unavailable", contract, method)
	return nil
}

func setAssetState(state *dto.StakeDAOPoolState, vault common.Address, result snapshotResult) {
	value, err := result.unsigned()
	if err == nil {
		// Presence of raw units is independent of decimals, USD price and TVL.
		state.HasAssets = stateBool(value.Sign() > 0)
		return
	}
	stateIssue(state, "hasAssets", "asset_amount_unavailable", vault, "totalAssets")
}

// Registration and shutdown are already proven by loadPoolAt at this block.
func (s *StakeDAOService) loadPoolState(ctx context.Context, info StakeDAOVaultInfo, block *big.Int, pool *dto.StakeDAOPool) {
	state := &dto.StakeDAOPoolState{BlockNumber: block.Uint64(), Registered: stateBool(true), Shutdown: stateBool(false)}
	// The request batcher reuses loadPool's totalAssets read at the same block.
	assets := s.snapshotCalls(ctx, block, readContract(info.Vault, rewardVaultERC4626ABI, "totalAssets"))
	setAssetState(state, info.Vault, assets[0])
	s.loadContractState(ctx, info, block, state)
	pool.State = state
}

// Every flag is an independent observation. A killed/zero-weight gauge can
// still have liquidity, trading fees or other rewards, so no active flag is derived.
func (s *StakeDAOService) loadContractState(ctx context.Context, info StakeDAOVaultInfo, block *big.Int, state *dto.StakeDAOPoolState) {
	reads := []snapshotRead{readContract(s.controller, controllerStateABI, "isPaused", info.ProtocolID)}
	curve := s.chainID == 1 && info.ProtocolID == [4]byte{0xc7, 0x15, 0xe3, 0x73}
	controller := common.HexToAddress(curveGaugeControllerEthereum)
	if curve {
		reads = append(reads,
			readContract(info.Gauge, curveGaugeABI, "is_killed"),
			readContract(controller, curveControllerABI, "gauge_relative_weight", info.Gauge, new(big.Int).SetUint64(s.snapshotTime)),
		)
	}
	data := s.snapshotCalls(ctx, block, reads...)
	state.ProtocolPaused = readStateBool(state, data[0], "protocolPaused", s.controller, "isPaused")
	if !curve {
		stateIssue(state, "gaugeKilled", "unsupported_protocol", info.Gauge, "is_killed")
		stateIssue(state, "crvRelativeWeight", "unsupported_protocol", controller, "gauge_relative_weight")
		return
	}
	state.GaugeKilled = readStateBool(state, data[1], "gaugeKilled", info.Gauge, "is_killed")
	weight, err := data[2].unsigned()
	if err != nil {
		stateIssue(state, "crvRelativeWeight", "getter_unavailable", controller, "gauge_relative_weight")
		return
	}
	if weight.Cmp(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) > 0 {
		stateIssue(state, "crvRelativeWeight", "invalid_weight", controller, "gauge_relative_weight")
		return
	}
	value := decimal.NewFromBigInt(weight, -18).String()
	state.CRVRelativeWeight, state.HasCRVWeight = &value, stateBool(weight.Sign() > 0)
}

// GetGaugeStatesAt bypasses pool discovery, prices and reward calculations.
// Unlike the production list it returns shutdown and unregistered gauges too.
func (s *StakeDAOService) GetGaugeStatesAt(ctx context.Context, network string, gauges []common.Address, block uint64) (*dto.StakeDAOPoolStatesResponse, error) {
	if len(gauges) < 1 || len(gauges) > MaxSamplePools {
		return nil, errors.New("select 1–5 gauge addresses")
	}
	seen := map[common.Address]bool{}
	for _, gauge := range gauges {
		if gauge == (common.Address{}) || seen[gauge] {
			return nil, errors.New("state gauges must be nonzero and unique")
		}
		seen[gauge] = true
	}
	r, err := s.getPoolsWithSelection(ctx, network, &poolSample{gauges: append([]common.Address(nil), gauges...), limit: len(gauges), diagnostics: true, blockNumber: block, stateOnly: true})
	if err != nil {
		return nil, err
	}
	return &dto.StakeDAOPoolStatesResponse{Network: r.Network, BlockNumber: r.BlockNumber, BlockTimestamp: r.BlockTimestamp, BlockHash: r.BlockHash, RPCAttempts: r.RPCAttempts, Gauges: r.StateRecords}, nil
}

func (s *StakeDAOService) stateSnapshot(ctx context.Context, network string, block *big.Int, header *types.Header, gauges []common.Address) (*dto.StakeDAOPoolsResponse, error) {
	progressFrom(ctx).setStage("pool_state")
	worker := *s
	worker.batches = newSnapshotBatcher(ctx, s.client, block)
	defer worker.batches.close()
	reads := make([]snapshotRead, 0, len(gauges))
	for _, gauge := range gauges {
		reads = append(reads, readContract(s.controller, protocolControllerABI, "gauge", gauge))
	}
	registration := worker.snapshotCalls(ctx, block, reads...)
	rows := make([]dto.StakeDAOGaugeState, 0, len(gauges))
	for i, gauge := range gauges {
		row := dto.StakeDAOGaugeState{GaugeAddress: gauge, State: dto.StakeDAOPoolState{BlockNumber: block.Uint64()}}
		result := registration[i]
		if result.err != nil || len(result.values) != 5 {
			stateIssue(&row.State, "registered", "registration_unavailable", s.controller, "gauge")
			rows = append(rows, row)
			continue
		}
		vault, vOK := result.values[0].(common.Address)
		asset, aOK := result.values[1].(common.Address)
		receiver, rOK := result.values[2].(common.Address)
		protocol, pOK := result.values[3].([4]byte)
		shutdown, sOK := result.values[4].(bool)
		if !vOK || !aOK || !rOK || !pOK || !sOK || vault != (common.Address{}) && asset == (common.Address{}) {
			stateIssue(&row.State, "registered", "invalid_registration", s.controller, "gauge")
			rows = append(rows, row)
			continue
		}
		row.State.Registered = stateBool(vault != (common.Address{}))
		if vault == (common.Address{}) {
			rows = append(rows, row) // Other flags do not apply to an absent registration.
			continue
		}
		row.VaultAddress, row.State.Shutdown = &vault, stateBool(shutdown)
		info := StakeDAOVaultInfo{Gauge: gauge, Vault: vault, Asset: asset, RewardReceiver: receiver, ProtocolID: protocol}
		data := worker.snapshotCalls(ctx, block, readContract(vault, rewardVaultERC4626ABI, "totalAssets"), readContract(asset, erc20DecimalsABI, "decimals"))
		amount, amountErr := data[0].unsigned()
		decimals, decimalsErr := data[1].unsigned()
		if amountErr == nil && decimalsErr == nil && decimals.IsUint64() && decimals.Uint64() <= 255 {
			value := decimal.NewFromBigInt(amount, -int32(decimals.Uint64())).String()
			row.AssetAmount = &value
		}
		setAssetState(&row.State, vault, data[0])
		worker.loadContractState(ctx, info, block, &row.State)
		rows = append(rows, row)
	}
	if err := worker.batches.err(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	progressFrom(ctx).setStage("verify_snapshot")
	verified, err := s.client.HeaderByNumber(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("%w: verify state snapshot: %w", ErrRPCUnavailable, err)
	}
	if verified == nil || verified.Hash() != header.Hash() {
		return nil, errors.New("StakeDAO state snapshot block was reorganized; retry the request")
	}
	hash := header.Hash()
	r := &dto.StakeDAOPoolsResponse{Network: network, BlockNumber: block.Uint64(), BlockTimestamp: header.Time, BlockHash: &hash, StateRecords: rows}
	if client, ok := s.client.(*sampleRPC); ok {
		r.RPCAttempts = 300 - client.remaining
	}
	return r, nil
}
