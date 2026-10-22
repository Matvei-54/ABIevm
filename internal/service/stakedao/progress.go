package stakedao

import (
	"context"
	"errors"
	"log/slog"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type progressContextKey struct{}

var stakeDAORequestSequence atomic.Uint64

// Request-local diagnostics. Never retain/log endpoints, calldata or raw errors:
// provider errors can contain credentials in URLs or response bodies.
type loadProgress struct {
	mu             sync.Mutex
	logger         *slog.Logger
	started        time.Time
	stage          string
	ranges         uint64
	read           uint64
	lastFrom       uint64
	lastTo         uint64
	workers        int
	visited        int
	loaded         int
	pools          int
	gauge          common.Address
	methods        map[string]uint64
	active         int
	failures       uint64
	lastRPC        string
	lastTime       time.Duration
	firstErr       *rpcFailure
	registrySaved  uint64
	registryTarget uint64
}

type rpcFailure struct {
	method, stage, kind string
	httpStatus          int
	code                *int
}

func newLoadProgress(logger *slog.Logger, network string, sample bool) *loadProgress {
	return &loadProgress{
		logger:  logger.With("component", "stakedao", "request", stakeDAORequestSequence.Add(1), "network", network, "sample", sample),
		started: time.Now(), methods: make(map[string]uint64),
	}
}

func progressFrom(ctx context.Context) *loadProgress {
	p, _ := ctx.Value(progressContextKey{}).(*loadProgress)
	return p
}

func (p *loadProgress) setStage(stage string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.stage = stage
	p.mu.Unlock()
	p.report("stage", nil)
}

func (p *loadProgress) discoveryStarted(total uint64, workers int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.ranges, p.workers = total, workers
	p.mu.Unlock()
	p.report("discovery_start", nil)
}

func (p *loadProgress) rangeRead(interval discoveryRange) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.read++
	p.lastFrom, p.lastTo = interval.from, interval.to
	p.mu.Unlock()
}

func (p *loadProgress) poolProgress(visited, loaded, total int, gauge common.Address) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.visited, p.loaded, p.pools, p.gauge = visited, loaded, total, gauge
	p.mu.Unlock()
}

func (p *loadProgress) registryProgress(point *RegistryCheckpoint, target uint64) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.registryTarget = target
	p.registrySaved = 0
	if point != nil {
		p.registrySaved = point.Block
	}
}

func (p *loadProgress) report(event string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	attrs := []any{
		"event", event, "stage", p.stage, "elapsed", time.Since(p.started).Round(time.Millisecond),
		"ranges_done", p.read, "ranges_total", p.ranges, "last_range_from", p.lastFrom, "last_range_to", p.lastTo, "workers", p.workers,
		"pools_visited", p.visited, "pools_total", p.pools, "pools_loaded", p.loaded,
		"rpc_attempts", p.methods["eth_chainId"] + p.methods["eth_blockNumber"] + p.methods["eth_getBlockByNumber"] + p.methods["eth_getCode"] + p.methods["eth_getLogs"] + p.methods["eth_call"],
		"rpc_logs", p.methods["eth_getLogs"], "rpc_calls", p.methods["eth_call"], "rpc_active", p.active, "rpc_errors", p.failures,
		"last_rpc", p.lastRPC, "last_rpc_elapsed", p.lastTime.Round(time.Millisecond),
	}
	if p.gauge != (common.Address{}) {
		attrs = append(attrs, "gauge", p.gauge.Hex())
	}
	if p.registryTarget != 0 {
		attrs = append(attrs, "registry_saved_block", p.registrySaved, "registry_target_block", p.registryTarget)
	}
	if failure := p.firstErr; failure != nil {
		attrs = append(attrs, "first_rpc_failure", failure.kind, "first_rpc_failure_method", failure.method, "first_rpc_failure_stage", failure.stage)
		if failure.httpStatus != 0 {
			attrs = append(attrs, "first_rpc_http_status", failure.httpStatus)
		}
		if failure.code != nil {
			attrs = append(attrs, "first_rpc_error_code", *failure.code)
		}
	}
	if err != nil {
		failure := safeFailure(err)
		if event == "request_deadline" {
			failure = "request_timeout"
		}
		attrs = append(attrs, "failure", failure)
		var httpErr rpc.HTTPError
		if errors.As(err, &httpErr) {
			attrs = append(attrs, "rpc_http_status", httpErr.StatusCode)
		}
		var rpcErr rpc.Error
		if errors.As(err, &rpcErr) {
			attrs = append(attrs, "rpc_error_code", rpcErr.ErrorCode())
		}
		p.logger.Error("StakeDAO load", attrs...)
		return
	}
	p.logger.Info("StakeDAO load", attrs...)
}

func safeFailure(err error) string {
	switch {
	case errors.Is(err, ErrRegistryUnavailable):
		return "registry_storage"
	case errors.Is(err, ErrRegistryReorg):
		return "registry_reorg"
	case errors.Is(err, context.DeadlineExceeded):
		return "rpc_timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	}
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return "rpc_timeout"
	}
	return "load_error"
}

// Stop and join the heartbeat before logging the final outcome; no goroutine
// survives its request. The caller's context separately identifies its deadline.
func (p *loadProgress) watch(ctx context.Context, interval time.Duration) func(error) {
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.report("progress", nil)
			}
		}
	}()
	return func(err error) {
		close(stop)
		<-done
		event := "completed"
		if err != nil {
			event = "failed"
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				event = "request_deadline"
			}
		}
		p.report(event, err)
	}
}

func (p *loadProgress) beginRPC(method string) func(error) {
	started := time.Now()
	p.mu.Lock()
	p.methods[method]++
	p.active++
	p.mu.Unlock()
	return func(err error) {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.active--
		p.lastRPC, p.lastTime = method, time.Since(started)
		if err != nil {
			p.failures++
			if p.firstErr == nil {
				failure := &rpcFailure{method: method, stage: p.stage, kind: safeFailure(err)}
				var httpErr rpc.HTTPError
				if errors.As(err, &httpErr) {
					failure.httpStatus = httpErr.StatusCode
				}
				var rpcErr rpc.Error
				if errors.As(err, &rpcErr) {
					code := rpcErr.ErrorCode()
					failure.code = &code
				}
				p.firstErr = failure
			}
		}
	}
}

type observedRPC struct {
	stakeDAORPC
	progress *loadProgress
}

func (c *observedRPC) ChainID(ctx context.Context) (value *big.Int, err error) {
	end := c.progress.beginRPC("eth_chainId")
	defer func() { end(err) }()
	return c.stakeDAORPC.ChainID(ctx)
}

func (c *observedRPC) BlockNumber(ctx context.Context) (value uint64, err error) {
	end := c.progress.beginRPC("eth_blockNumber")
	defer func() { end(err) }()
	return c.stakeDAORPC.BlockNumber(ctx)
}

func (c *observedRPC) HeaderByNumber(ctx context.Context, block *big.Int) (value *types.Header, err error) {
	end := c.progress.beginRPC("eth_getBlockByNumber")
	defer func() { end(err) }()
	return c.stakeDAORPC.HeaderByNumber(ctx, block)
}

func (c *observedRPC) CodeAt(ctx context.Context, address common.Address, block *big.Int) (value []byte, err error) {
	end := c.progress.beginRPC("eth_getCode")
	defer func() { end(err) }()
	return c.stakeDAORPC.CodeAt(ctx, address, block)
}

func (c *observedRPC) FilterLogs(ctx context.Context, query ethereum.FilterQuery) (value []types.Log, err error) {
	end := c.progress.beginRPC("eth_getLogs")
	defer func() { end(err) }()
	return c.stakeDAORPC.FilterLogs(ctx, query)
}

func (c *observedRPC) CallContract(ctx context.Context, call ethereum.CallMsg, block *big.Int) (value []byte, err error) {
	end := c.progress.beginRPC("eth_call")
	defer func() { end(err) }()
	return c.stakeDAORPC.CallContract(ctx, call, block)
}
