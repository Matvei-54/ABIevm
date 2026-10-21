package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const discoveryBlockRange uint64 = 1500

type discoveryRange struct{ from, to uint64 }
type discoveryResult struct {
	interval discoveryRange
	logs     []types.Log
	err      error
}

// Only RPC reads run concurrently. The caller sorts and decodes the merged logs
// in canonical chain order before choosing the latest registration per gauge.
func readRegistrationLogs(ctx context.Context, client vaultLogReader, controller common.Address, event common.Hash, fromBlock, toBlock uint64) ([]types.Log, error) {
	workers := 8
	if text := os.Getenv("STAKEDAO_DISCOVERY_WORKERS"); text != "" {
		var err error
		workers, err = strconv.Atoi(text)
		if err != nil || workers < 1 || workers > 64 {
			return nil, errors.New("STAKEDAO_DISCOVERY_WORKERS must be between 1 and 64")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	count := (toBlock-fromBlock)/discoveryBlockRange + 1
	if count < uint64(workers) {
		workers = int(count)
	}
	progress := progressFrom(ctx)
	progress.discoveryStarted(count, workers)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan discoveryRange)
	results := make(chan discoveryResult, workers)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)
		for start := fromBlock; ; {
			end := toBlock
			if toBlock-start >= discoveryBlockRange {
				end = start + discoveryBlockRange - 1
			}
			select {
			case <-runCtx.Done():
				return
			case jobs <- discoveryRange{start, end}:
			}
			if end == toBlock {
				return
			}
			start = end + 1
		}
	}()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				var interval discoveryRange
				select {
				case <-runCtx.Done():
					return
				case next, ok := <-jobs:
					if !ok {
						return
					}
					interval = next
				}
				logs, err := client.FilterLogs(runCtx, ethereum.FilterQuery{Addresses: []common.Address{controller}, Topics: [][]common.Hash{{event}}, FromBlock: new(big.Int).SetUint64(interval.from), ToBlock: new(big.Int).SetUint64(interval.to)})
				select {
				case <-runCtx.Done():
					return
				case results <- discoveryResult{interval, logs, err}:
				}
				if err != nil {
					return
				}
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	var entries []types.Log
	var firstErr error
	for result := range results {
		if firstErr != nil {
			continue
		}
		if result.err != nil {
			firstErr = fmt.Errorf("%w: VaultRegistered blocks %d-%d: %w", ErrRPCUnavailable, result.interval.from, result.interval.to, result.err)
			cancel()
			continue
		}
		for _, entry := range result.logs {
			if entry.Removed {
				continue
			}
			if entry.Address != controller || entry.BlockNumber < result.interval.from || entry.BlockNumber > result.interval.to || len(entry.Topics) != 4 || entry.Topics[0] != event {
				firstErr = fmt.Errorf("invalid VaultRegistered log at block %d", entry.BlockNumber)
				cancel()
				break
			}
		}
		if firstErr == nil {
			entries = append(entries, result.logs...)
			progress.rangeRead(result.interval)
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
