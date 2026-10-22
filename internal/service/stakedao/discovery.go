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

func readRegistrationLogs(ctx context.Context, client vaultLogReader, controller common.Address, event common.Hash, fromBlock, toBlock uint64) ([]types.Log, error) {
	var entries []types.Log
	err := readRegistryRanges(ctx, client, controller, map[common.Hash]int{event: 4}, fromBlock, toBlock, func(_ discoveryRange, logs []types.Log) error { entries = append(entries, logs...); return nil })
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// RPC reads are parallel, callbacks are strictly ordered. Credits are released
// only after committing a range, bounding both in-flight work and reordered data.
func readRegistryRanges(ctx context.Context, client vaultLogReader, controller common.Address, events map[common.Hash]int, fromBlock, toBlock uint64, consume func(discoveryRange, []types.Log) error) error {
	if fromBlock > toBlock {
		return errors.New("invalid StakeDAO discovery block range")
	}
	workers := 8
	if text := os.Getenv("STAKEDAO_DISCOVERY_WORKERS"); text != "" {
		var err error
		workers, err = strconv.Atoi(text)
		if err != nil || workers < 1 || workers > 64 {
			return errors.New("STAKEDAO_DISCOVERY_WORKERS must be between 1 and 64")
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	count := (toBlock-fromBlock)/discoveryBlockRange + 1
	if count < uint64(workers) {
		workers = int(count)
	}
	var topics []common.Hash
	for id := range events {
		topics = append(topics, id)
	}
	if len(topics) == 0 {
		return errors.New("registry event topics are empty")
	}
	progress := progressFrom(ctx)
	progress.discoveryStarted(count, workers)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan discoveryRange)
	results := make(chan discoveryResult, workers)
	credits := make(chan struct{}, workers)
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
			case credits <- struct{}{}:
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
				logs, err := client.FilterLogs(runCtx, ethereum.FilterQuery{Addresses: []common.Address{controller}, Topics: [][]common.Hash{topics}, FromBlock: new(big.Int).SetUint64(interval.from), ToBlock: new(big.Int).SetUint64(interval.to)})
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
	pending := make(map[uint64]discoveryResult)
	next := fromBlock
	var firstErr error
	for result := range results {
		if firstErr != nil {
			continue
		}
		if result.err != nil {
			firstErr = fmt.Errorf("%w: registry events blocks %d-%d: %w", ErrRPCUnavailable, result.interval.from, result.interval.to, result.err)
			cancel()
			continue
		}
		for _, entry := range result.logs {
			if entry.Removed {
				continue
			}
			expected := 0
			if len(entry.Topics) > 0 {
				expected = events[entry.Topics[0]]
			}
			if entry.Address != controller || entry.BlockNumber < result.interval.from || entry.BlockNumber > result.interval.to || expected == 0 || len(entry.Topics) != expected {
				firstErr = fmt.Errorf("invalid registry log at block %d", entry.BlockNumber)
				cancel()
				break
			}
		}
		if firstErr != nil {
			continue
		}
		pending[result.interval.from] = result
		for {
			ready, ok := pending[next]
			if !ok {
				break
			}
			if err := ctx.Err(); err != nil {
				firstErr = err
				cancel()
				break
			}
			if err := consume(ready.interval, ready.logs); err != nil {
				firstErr = err
				cancel()
				break
			}
			delete(pending, next)
			<-credits
			progress.rangeRead(ready.interval)
			if ready.interval.to == toBlock {
				break
			}
			next = ready.interval.to + 1
		}
	}
	if firstErr != nil {
		return firstErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
