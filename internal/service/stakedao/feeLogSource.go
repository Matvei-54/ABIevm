package stakedao

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// The optional log transport shares the PRIMARY request's reservations. Creating
// a second sampleRPC would incorrectly double both the RPS limit and RPC budget.
type sharedFeeLogRPC struct {
	stakeDAORPC
	budget *sampleRPC
}

func (c *sharedFeeLogRPC) ChainID(ctx context.Context) (*big.Int, error) {
	if e := c.budget.before(ctx); e != nil {
		return nil, e
	}
	return c.stakeDAORPC.ChainID(ctx)
}
func (c *sharedFeeLogRPC) BlockNumber(ctx context.Context) (uint64, error) {
	if e := c.budget.before(ctx); e != nil {
		return 0, e
	}
	return c.stakeDAORPC.BlockNumber(ctx)
}
func (c *sharedFeeLogRPC) HeaderByNumber(ctx context.Context, n *big.Int) (*types.Header, error) {
	if e := c.budget.before(ctx); e != nil {
		return nil, e
	}
	return c.stakeDAORPC.HeaderByNumber(ctx, n)
}
func (c *sharedFeeLogRPC) CodeAt(ctx context.Context, a common.Address, n *big.Int) ([]byte, error) {
	if e := c.budget.before(ctx); e != nil {
		return nil, e
	}
	return c.stakeDAORPC.CodeAt(ctx, a, n)
}
func (c *sharedFeeLogRPC) CallContract(ctx context.Context, m ethereum.CallMsg, n *big.Int) ([]byte, error) {
	if e := c.budget.before(ctx); e != nil {
		return nil, e
	}
	return c.stakeDAORPC.CallContract(ctx, m, n)
}
func (c *sharedFeeLogRPC) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	if e := c.budget.before(ctx); e != nil {
		return nil, e
	}
	return c.stakeDAORPC.FilterLogs(ctx, q)
}

func verifyFeeLogHeader(ctx context.Context, source stakeDAORPC, want *types.Header) error {
	got, err := source.HeaderByNumber(ctx, want.Number)
	if err != nil || got == nil || got.Number == nil || got.Number.Cmp(want.Number) != 0 {
		return errors.New("logs_rpc_header_unavailable")
	}
	if got.Hash() != want.Hash() {
		return errors.New("logs_rpc_snapshot_mismatch")
	}
	return nil
}

// Only used by the explicit weETH fee-history diagnostic. All prices, contract
// calls, state endpoints and registry discovery continue to use the primary RPC.
func (s *StakeDAOService) openFeeLogSource(ctx context.Context, start, end *types.Header) (stakeDAORPC, func(), error) {
	noop := func() {}
	if s.feeLogsRPCURL == "" {
		return s.client, noop, nil
	}
	budget, ok := s.client.(*sampleRPC)
	if !ok || s.dial == nil {
		return nil, noop, errors.New("logs_rpc_shared_budget_unavailable")
	}
	source, err := s.dial(ctx, s.feeLogsRPCURL)
	if err != nil || source == nil {
		return nil, noop, errors.New("logs_rpc_unavailable")
	}
	success := false
	defer func() {
		if !success {
			source.Close()
		}
	}()
	// Real HTTP requests have a progress observer. Unit fixtures need no logger.
	var observed stakeDAORPC = source
	if progress, ok := ctx.Value(progressContextKey{}).(*loadProgress); ok {
		observed = &observedRPC{stakeDAORPC: source, progress: progress}
	}
	shared := &sharedFeeLogRPC{stakeDAORPC: observed, budget: budget}
	chain, err := shared.ChainID(ctx)
	if err != nil {
		return nil, noop, errors.New("logs_rpc_chain_unavailable")
	}
	if chain == nil || !chain.IsInt64() || chain.Int64() != s.chainID {
		return nil, noop, errors.New("logs_rpc_chain_mismatch")
	}
	if err = verifyFeeLogHeader(ctx, shared, end); err != nil {
		return nil, noop, err
	}
	if err = verifyFeeLogHeader(ctx, shared, start); err != nil {
		return nil, noop, err
	}
	success = true
	return shared, source.Close, nil
}
