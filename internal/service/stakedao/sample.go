package stakedao

import (
	"context"
	"errors"
	"math/big"
	"time"

	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type poolSample struct {
	fromBlock, toBlock uint64
	limit              int
}

// GetPoolsSample checks at most three current vaults registered in one small log
// window. It deliberately avoids deployment discovery and historical eth_getCode.
// An older registration is used only when its vault and asset still match Controller.
func (s *StakeDAOService) GetPoolsSample(ctx context.Context, network string, fromBlock, toBlock uint64, limit int) (*dto.StakeDAOPoolsResponse, error) {
	if limit < 1 || limit > 3 {
		return nil, errors.New("sample limit must be between 1 and 3")
	}
	if fromBlock > toBlock || toBlock-fromBlock >= 100 {
		return nil, errors.New("sample log window must contain between 1 and 100 blocks")
	}
	return s.getPoolsWithSelection(ctx, network, &poolSample{fromBlock: fromBlock, toBlock: toBlock, limit: limit})
}

// The sample path is sequential: at most five RPC requests per second and 300
// attempts in total. This is a request budget, not a provider CU estimate.
type sampleRPC struct {
	stakeDAORPC
	remaining int
	next      time.Time
	exhausted bool
}

func newSampleRPC(client stakeDAORPC) *sampleRPC {
	return &sampleRPC{stakeDAORPC: client, remaining: 300}
}

func (c *sampleRPC) before(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.remaining == 0 {
		c.exhausted = true
		return errors.New("sample RPC request budget exhausted")
	}
	if delay := time.Until(c.next); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	c.remaining--
	c.next = time.Now().Add(200 * time.Millisecond)
	return nil
}

func (c *sampleRPC) ChainID(ctx context.Context) (*big.Int, error) {
	if err := c.before(ctx); err != nil {
		return nil, err
	}
	return c.stakeDAORPC.ChainID(ctx)
}

func (c *sampleRPC) BlockNumber(ctx context.Context) (uint64, error) {
	if err := c.before(ctx); err != nil {
		return 0, err
	}
	return c.stakeDAORPC.BlockNumber(ctx)
}

func (c *sampleRPC) HeaderByNumber(ctx context.Context, block *big.Int) (*types.Header, error) {
	if err := c.before(ctx); err != nil {
		return nil, err
	}
	return c.stakeDAORPC.HeaderByNumber(ctx, block)
}

func (c *sampleRPC) CodeAt(ctx context.Context, address common.Address, block *big.Int) ([]byte, error) {
	if err := c.before(ctx); err != nil {
		return nil, err
	}
	return c.stakeDAORPC.CodeAt(ctx, address, block)
}

func (c *sampleRPC) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	if err := c.before(ctx); err != nil {
		return nil, err
	}
	return c.stakeDAORPC.FilterLogs(ctx, query)
}

func (c *sampleRPC) CallContract(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	if err := c.before(ctx); err != nil {
		return nil, err
	}
	return c.stakeDAORPC.CallContract(ctx, call, block)
}
