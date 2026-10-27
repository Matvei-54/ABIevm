package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Ethereum Mainnet Multicall3. Calls are simulated by eth_call; no transaction is sent.
var stakeDAOMulticallAddress = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// Subset of the ABI already checked into contracts/curve/Multicall3Curve.json.
const stakeDAOMulticallABI = `[{"type":"function","name":"aggregate3","stateMutability":"payable","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"allowFailure","type":"bool"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

const stakeDAOMulticallBatchSize = 24

type multicallInput struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

type multicallOutput struct {
	Success    bool
	ReturnData []byte
}

type snapshotRead struct {
	target  common.Address
	abiJSON string
	method  string
	args    []interface{}
}

func readContract(target common.Address, abiJSON, method string, args ...interface{}) snapshotRead {
	return snapshotRead{target: target, abiJSON: abiJSON, method: method, args: args}
}

type snapshotResult struct {
	values []interface{}
	err    error
}

func (r snapshotResult) address() (common.Address, error) {
	if r.err != nil {
		return common.Address{}, r.err
	}
	if len(r.values) == 1 {
		if address, ok := r.values[0].(common.Address); ok {
			return address, nil
		}
	}
	return common.Address{}, errors.New("invalid address output")
}

func (r snapshotResult) unsigned() (*big.Int, error) {
	if r.err != nil {
		return nil, r.err
	}
	if len(r.values) == 1 {
		switch value := r.values[0].(type) {
		case *big.Int:
			if value != nil && value.Sign() >= 0 {
				return value, nil
			}
		case uint8:
			return new(big.Int).SetUint64(uint64(value)), nil
		case uint32:
			return new(big.Int).SetUint64(uint64(value)), nil
		}
	}
	return nil, errors.New("invalid unsigned output")
}

// Only explicitly selected, caller-independent getters belong here: subcalls see
// Multicall3 as msg.sender. Dependent reads remain in separate stages. Results
// retain their input order and failures; a failed batch never triggers direct retries.
func (s *StakeDAOService) snapshotCalls(ctx context.Context, block *big.Int, reads ...snapshotRead) []snapshotResult {
	if s.batches != nil {
		return s.batches.read(ctx, block, reads)
	}
	results := make([]snapshotResult, len(reads))
	failFrom := func(start int, err error) []snapshotResult {
		for i := start; i < len(results); i++ {
			results[i].err = err
		}
		return results
	}
	if len(reads) == 0 {
		return results
	}
	if block == nil || block.Sign() < 0 {
		return failFrom(0, errors.New("Multicall requires a fixed nonnegative snapshot block"))
	}
	for start := 0; start < len(reads); start += stakeDAOMulticallBatchSize {
		if err := ctx.Err(); err != nil {
			return failFrom(start, err)
		}
		end := min(start+stakeDAOMulticallBatchSize, len(reads))
		calls := make([]multicallInput, end-start)
		parsers := make([]abi.ABI, end-start)
		for j, read := range reads[start:end] {
			parsed, err := abi.JSON(strings.NewReader(read.abiJSON))
			if err != nil {
				return failFrom(start, err)
			}
			data, err := parsed.Pack(read.method, read.args...)
			if err != nil {
				return failFrom(start, err)
			}
			calls[j] = multicallInput{Target: read.target, AllowFailure: true, CallData: data}
			parsers[j] = parsed
		}
		decoded, err := executeMulticall(ctx, s.client, block, calls)
		if err != nil {
			return failFrom(start, err)
		}
		for j, result := range decoded {
			if !result.Success {
				results[start+j].err = fmt.Errorf("%s reverted in Multicall", reads[start+j].method)
				continue
			}
			results[start+j].values, results[start+j].err = parsers[j].Unpack(reads[start+j].method, result.ReturnData)
		}
	}
	return results
}

func executeMulticall(ctx context.Context, client stakeDAORPC, block *big.Int, calls []multicallInput) ([]multicallOutput, error) {
	aggregate, err := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
	if err != nil {
		return nil, err
	}
	data, err := aggregate.Pack("aggregate3", calls)
	if err != nil {
		return nil, err
	}
	raw, err := client.CallContract(ctx, ethereum.CallMsg{To: &stakeDAOMulticallAddress, Data: data}, block)
	if err != nil {
		return nil, fmt.Errorf("%w: Multicall at block %s: %w", ErrRPCUnavailable, block, err)
	}
	out, err := aggregate.Unpack("aggregate3", raw)
	if err != nil || len(out) != 1 {
		return nil, fmt.Errorf("%w: invalid Multicall response", ErrRPCUnavailable)
	}
	decoded := *abi.ConvertType(out[0], new([]multicallOutput)).(*[]multicallOutput)
	if len(decoded) != len(calls) {
		return nil, fmt.Errorf("%w: Multicall result count mismatch", ErrRPCUnavailable)
	}
	return decoded, nil
}

type snapshotKey struct {
	target common.Address
	data   string
}

type snapshotFuture struct {
	call multicallInput
	done chan struct{}
	raw  []byte
	err  error
}

// A request-local scheduler for caller-independent getters at exactly one block.
// Only its dispatcher touches the RPC client: pool workers share batches and
// in-flight reads, not mutable decoded ABI values. Nothing survives the request.
type snapshotBatcher struct {
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	queue   chan *snapshotFuture
	client  stakeDAORPC
	block   *big.Int
	mu      sync.Mutex
	entries map[snapshotKey]*snapshotFuture
	parsers map[string]abi.ABI
	failure error
}

func newSnapshotBatcher(ctx context.Context, client stakeDAORPC, block *big.Int) *snapshotBatcher {
	ctx, cancel := context.WithCancel(ctx)
	b := &snapshotBatcher{
		ctx: ctx, cancel: cancel, done: make(chan struct{}), client: client,
		block: new(big.Int).Set(block), queue: make(chan *snapshotFuture, stakeDAOMulticallBatchSize*stakeDAOPoolWorkers),
		entries: make(map[snapshotKey]*snapshotFuture), parsers: make(map[string]abi.ABI),
	}
	go b.run()
	return b
}

func (b *snapshotBatcher) close() {
	b.cancel()
	<-b.done
}

func (b *snapshotBatcher) err() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.failure
}

func (b *snapshotBatcher) read(ctx context.Context, block *big.Int, reads []snapshotRead) []snapshotResult {
	results := make([]snapshotResult, len(reads))
	if err := ctx.Err(); err != nil {
		for i := range results {
			results[i].err = err
		}
		return results
	}
	futures := make([]*snapshotFuture, len(reads))
	parsers := make([]abi.ABI, len(reads))
	for i, read := range reads {
		if block == nil || block.Cmp(b.block) != 0 {
			results[i].err = errors.New("Multicall scheduler snapshot mismatch")
			continue
		}
		b.mu.Lock()
		parsed, found := b.parsers[read.abiJSON]
		if !found {
			var err error
			parsed, err = abi.JSON(strings.NewReader(read.abiJSON))
			if err != nil {
				b.mu.Unlock()
				results[i].err = err
				continue
			}
			b.parsers[read.abiJSON] = parsed
		}
		b.mu.Unlock()
		data, err := parsed.Pack(read.method, read.args...)
		if err != nil {
			results[i].err = err
			continue
		}
		key := snapshotKey{target: read.target, data: string(data)}
		b.mu.Lock()
		future, found := b.entries[key]
		if !found {
			future = &snapshotFuture{call: multicallInput{Target: read.target, AllowFailure: true, CallData: data}, done: make(chan struct{})}
			b.entries[key] = future
		}
		b.mu.Unlock()
		futures[i], parsers[i] = future, parsed
		if !found {
			select {
			case b.queue <- future:
			case <-b.ctx.Done():
			case <-ctx.Done():
				// This future was never enqueued; don't strand other waiters or
				// cache a single caller's cancellation as a contract result.
				b.mu.Lock()
				delete(b.entries, key)
				b.mu.Unlock()
				future.err = ctx.Err()
				close(future.done)
			}
		}
	}
	for i, future := range futures {
		if future == nil {
			continue
		}
		select {
		case <-future.done:
			if future.err != nil {
				results[i].err = future.err
			} else {
				// Decode separately for each consumer: *big.Int/slices are mutable.
				results[i].values, results[i].err = parsers[i].Unpack(reads[i].method, future.raw)
			}
		case <-b.ctx.Done():
			results[i].err = b.ctx.Err()
		case <-ctx.Done():
			results[i].err = ctx.Err()
		}
		if err := b.err(); err != nil {
			results[i] = snapshotResult{err: err}
		}
	}
	return results
}

func (b *snapshotBatcher) run() {
	defer close(b.done)
	for {
		var first *snapshotFuture
		select {
		case <-b.ctx.Done():
			return
		case first = <-b.queue:
		}
		pending := []*snapshotFuture{first}
		// A short coalescing window lets other pool workers submit independent
		// getters; batches and the queue stay bounded even with a slow provider.
		timer := time.NewTimer(time.Millisecond)
	collect:
		for len(pending) < stakeDAOMulticallBatchSize {
			select {
			case <-b.ctx.Done():
				timer.Stop()
				return
			case next := <-b.queue:
				pending = append(pending, next)
			case <-timer.C:
				break collect
			}
		}
		timer.Stop()
		if b.ctx.Err() != nil {
			return
		}
		calls := make([]multicallInput, len(pending))
		for i, future := range pending {
			calls[i] = future.call
		}
		out, err := executeMulticall(b.ctx, b.client, b.block, calls)
		if err != nil {
			b.mu.Lock()
			b.failure = err
			b.mu.Unlock()
			b.cancel()
			return
		}
		for i, future := range pending {
			if out[i].Success {
				future.raw = out[i].ReturnData
			} else {
				future.err = errors.New("contract getter reverted in Multicall")
			}
			close(future.done)
		}
	}
}
