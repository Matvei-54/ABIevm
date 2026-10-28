package prices

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
)

type ContractCaller interface {
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
}

type Snapshot struct {
	Caller    ContractCaller
	Block     uint64
	Timestamp uint64
	// Workers is opt-in token concurrency (0/1 is sequential, maximum 8).
	// Callers must remain safe for concurrent direct eth_call requests.
	Workers int
}

// ErrPriceRPCUnavailable distinguishes provider failures from a token without a price.
var ErrPriceRPCUnavailable = errors.New("contract price RPC unavailable")

type SnapshotSource interface {
	FetchAt(context.Context, int64, []common.Address, Snapshot) (map[common.Address]Quote, error)
}

type SnapshotProvider interface {
	EnsurePricesAt(context.Context, int64, []common.Address, Snapshot) error
}

var (
	feedRegistry    = common.HexToAddress("0x47Fb2585D2C56Fe188D0E6ec628a38b74fCeeeDf")
	usdDenomination = common.BigToAddress(big.NewInt(840))
	curveNGFactory  = common.HexToAddress("0x6A8cbed756804B16E05E741eDaBd5cB544AE21bf")
	errNoUSDFeed    = errors.New("no Chainlink USD feed")
)

// Subsets of the official interfaces; provenance is in README.md.
const contractPriceABI = `[
{"type":"function","name":"getFeed","inputs":[{"type":"address"},{"type":"address"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"latestRoundData","inputs":[],"outputs":[{"type":"uint80"},{"type":"int256"},{"type":"uint256"},{"type":"uint256"},{"type":"uint80"}]},
{"type":"function","name":"decimals","inputs":[],"outputs":[{"type":"uint8"}]},
{"type":"function","name":"asset","inputs":[],"outputs":[{"type":"address"}]},
{"type":"function","name":"convertToAssets","inputs":[{"type":"uint256"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"get_implementation_address","inputs":[{"type":"address"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"is_meta","inputs":[{"type":"address"}],"outputs":[{"type":"bool"}]},
{"type":"function","name":"N_COINS","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"coins","inputs":[{"type":"uint256"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"balances","inputs":[{"type":"uint256"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"totalSupply","inputs":[],"outputs":[{"type":"uint256"}]}
]`

// ContractSource resolves USD prices from Ethereum contracts only. It supports
// Chainlink USD feeds, ERC4626 shares, and registered plain Curve StableSwap NG
// pools. Missing or stale USD anchors remain unavailable; no $1/virtual-price
// substitution, HTTP price fallback, or recursive LP valuation is performed.
type ContractSource struct {
	store   Store
	methods abi.ABI
}

func NewContractSource(store Store) *ContractSource {
	methods, err := abi.JSON(strings.NewReader(contractPriceABI))
	if err != nil {
		panic(err)
	} // Compile-time ABI constant, never provider input.
	return &ContractSource{store: store, methods: methods}
}

func (s *ContractSource) Fetch(context.Context, int64, []common.Address) (map[common.Address]Quote, error) {
	return nil, errors.New("contract prices require EnsurePricesAt with an RPC snapshot")
}

func (s *ContractSource) FetchAt(ctx context.Context, chainID int64, tokens []common.Address, snapshot Snapshot) (map[common.Address]Quote, error) {
	if chainID != 1 || snapshot.Caller == nil || snapshot.Block == 0 || snapshot.Timestamp == 0 || snapshot.Timestamp > uint64(time.Now().Add(time.Minute).Unix()) {
		return nil, errors.New("contract prices require a valid Ethereum snapshot")
	}
	if snapshot.Workers < 0 || snapshot.Workers > 8 {
		return nil, errors.New("price workers must be between 0 and 8")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	calls := &contractCallCache{ctx: ctx, cancel: cancel, caller: snapshot.Caller, block: snapshot.Block, entries: make(map[contractCallKey]*contractCallResult)}
	snapshot.Caller = calls
	result := make(map[common.Address]Quote)
	issues := make([]error, len(tokens))
	type tokenResult struct {
		index int
		quote Quote
		err   error
	}
	workers := min(max(snapshot.Workers, 1), len(tokens))
	jobs := make(chan int)
	results := make(chan tokenResult, workers)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)
		for i := range tokens {
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Dependency stacks and decoded values belong to one worker. Only
			// immutable RPC bytes and completed Store quotes are shared.
			r := &contractResolver{source: s, ctx: ctx, snapshot: snapshot, memo: make(map[common.Address]Quote), visiting: make(map[common.Address]bool), decimals: make(map[common.Address]uint8)}
			for index := range jobs {
				if ctx.Err() != nil {
					return
				}
				q, err := r.price(tokens[index], 0)
				results <- tokenResult{index: index, quote: q, err: err}
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	for item := range results {
		token := tokens[item.index]
		if item.err != nil {
			issues[item.index] = fmt.Errorf("token %s: %w", token.Hex(), item.err)
		} else {
			result[token] = item.quote
		}
	}
	if err := calls.terminalError(); err != nil {
		return result, fmt.Errorf("%w: %w", ErrPriceRPCUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, errors.Join(issues...)
}

type contractCallKey struct {
	target common.Address
	data   string
}

type contractCallResult struct {
	done chan struct{}
	raw  []byte
	err  error
}

// This cache belongs to one FetchAt and one fixed block. Both successful calls
// and failed capability probes are deduplicated, including in-flight requests.
// Reads stay direct: moving oracle calls through Multicall could change access
// checks based on msg.sender/tx.origin. The caller's shared RPS limit still applies.
type contractCallCache struct {
	ctx     context.Context
	cancel  context.CancelFunc
	caller  ContractCaller
	block   uint64
	mu      sync.Mutex
	entries map[contractCallKey]*contractCallResult
	failure error
}

func (c *contractCallCache) terminalError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failure
}

func (c *contractCallCache) CallContract(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if call.To == nil || block == nil || !block.IsUint64() || block.Uint64() != c.block {
		return nil, errors.New("price call differs from its fixed snapshot")
	}
	key := contractCallKey{target: *call.To, data: string(call.Data)}
	c.mu.Lock()
	if c.failure != nil {
		err := c.failure
		c.mu.Unlock()
		return nil, err
	}
	entry, found := c.entries[key]
	if !found {
		entry = &contractCallResult{done: make(chan struct{})}
		c.entries[key] = entry
	}
	c.mu.Unlock()
	if !found {
		entry.raw, entry.err = c.caller.CallContract(c.ctx, call, block)
		var httpErr rpc.HTTPError
		var netErr net.Error
		if errors.As(entry.err, &httpErr) || errors.As(entry.err, &netErr) {
			c.mu.Lock()
			if c.failure == nil {
				c.failure = entry.err
			}
			c.mu.Unlock()
			c.cancel()
		}
		close(entry.done)
	}
	select {
	case <-entry.done:
		// ABI decoders/consumers must not share a mutable RPC response buffer.
		return bytes.Clone(entry.raw), entry.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	}
}

type contractResolver struct {
	source   *ContractSource
	ctx      context.Context
	snapshot Snapshot
	memo     map[common.Address]Quote
	visiting map[common.Address]bool
	decimals map[common.Address]uint8
}

func (r *contractResolver) call(target common.Address, method string, args ...interface{}) ([]interface{}, error) {
	data, err := r.source.methods.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	raw, err := r.snapshot.Caller.CallContract(r.ctx, ethereum.CallMsg{To: &target, Data: data}, new(big.Int).SetUint64(r.snapshot.Block))
	if err != nil {
		return nil, err
	}
	return r.source.methods.Unpack(method, raw)
}

func (r *contractResolver) address(target common.Address, method string, args ...interface{}) (common.Address, error) {
	out, err := r.call(target, method, args...)
	if err != nil {
		return common.Address{}, err
	}
	return out[0].(common.Address), nil
}

func (r *contractResolver) unsigned(target common.Address, method string, args ...interface{}) (*big.Int, error) {
	out, err := r.call(target, method, args...)
	if err != nil {
		return nil, err
	}
	v := out[0].(*big.Int)
	if v.Sign() < 0 {
		return nil, errors.New("negative contract quantity")
	}
	return v, nil
}

func (r *contractResolver) tokenDecimals(token common.Address) (uint8, error) {
	if d, ok := r.decimals[token]; ok {
		return d, nil
	}
	out, err := r.call(token, "decimals")
	if err != nil {
		return 0, err
	}
	d := out[0].(uint8)
	if d > 36 {
		return 0, errors.New("unsupported decimals above 36")
	}
	r.decimals[token] = d
	return d, nil
}

func (r *contractResolver) price(token common.Address, depth int) (Quote, error) {
	if token == (common.Address{}) || depth > 3 || r.visiting[token] {
		return Quote{}, errors.New("invalid or cyclic token dependency")
	}
	if q, ok := r.memo[token]; ok {
		return q, nil
	}
	now := time.Unix(int64(r.snapshot.Timestamp), 0)
	if r.source.store != nil {
		q, found, err := r.source.store.Get(r.ctx, TokenKey{ChainID: 1, Address: token})
		if err != nil {
			return Quote{}, err
		}
		age := now.Sub(q.UpdatedAt)
		if found && q.USD.IsPositive() && age >= -time.Minute && age <= 15*time.Minute && (q.ValidUntil.IsZero() || now.Before(q.ValidUntil)) {
			r.memo[token] = q
			return q, nil
		}
	}
	r.visiting[token] = true
	defer delete(r.visiting, token)
	q, err := r.directUSD(token)
	if errors.Is(err, errNoUSDFeed) {
		var implementation common.Address
		implementation, err = r.address(curveNGFactory, "get_implementation_address", token)
		if err == nil {
			if implementation != (common.Address{}) {
				if depth != 0 {
					return Quote{}, errors.New("nested Curve LP pricing is not supported")
				}
				q, err = r.pool(token)
			} else {
				q, err = r.vault(token, depth)
			}
		}
	}
	if err != nil {
		return Quote{}, err
	}
	if !q.USD.IsPositive() {
		return Quote{}, ErrPriceUnavailable
	}
	r.memo[token] = q
	if r.source.store != nil {
		if err := r.source.store.Set(r.ctx, TokenKey{ChainID: 1, Address: token}, q); err != nil {
			return Quote{}, err
		}
	}
	return q, nil
}

func (r *contractResolver) directUSD(token common.Address) (Quote, error) {
	feed, err := r.address(feedRegistry, "getFeed", token, usdDenomination)
	if err != nil {
		if data, ok := ethclient.RevertErrorData(err); ok {
			if reason, unpackErr := abi.UnpackRevert(data); unpackErr == nil && reason == "Feed not found" {
				return Quote{}, errNoUSDFeed
			}
		}
		if strings.Contains(err.Error(), "Feed not found") {
			return Quote{}, errNoUSDFeed
		}
		return Quote{}, err
	}
	if feed == (common.Address{}) {
		return Quote{}, errNoUSDFeed
	}
	d, err := r.tokenDecimals(feed)
	if err != nil {
		return Quote{}, err
	}
	out, err := r.call(feed, "latestRoundData")
	if err != nil {
		return Quote{}, err
	}
	round, answer, started, updated, answered := out[0].(*big.Int), out[1].(*big.Int), out[2].(*big.Int), out[3].(*big.Int), out[4].(*big.Int)
	// 24-hour heartbeat plus one hour of grace is a conservative, explicit
	// policy for the feeds used here. Cache lifetime never extends this deadline.
	const maxAge = uint64((25 * time.Hour) / time.Second)
	if round.Sign() <= 0 || answer.Sign() <= 0 || answered.Cmp(round) < 0 || !updated.IsUint64() || updated.Sign() <= 0 || started.Sign() <= 0 || started.Cmp(updated) > 0 || updated.Uint64() > r.snapshot.Timestamp || r.snapshot.Timestamp-updated.Uint64() >= maxAge {
		return Quote{}, errors.New("invalid or stale Chainlink USD round")
	}
	return Quote{USD: decimal.NewFromBigInt(answer, -int32(d)), UpdatedAt: time.Unix(int64(r.snapshot.Timestamp), 0), ValidUntil: time.Unix(int64(updated.Uint64()+maxAge), 0)}, nil
}

func (r *contractResolver) vault(token common.Address, depth int) (Quote, error) {
	underlying, err := r.address(token, "asset")
	if err != nil {
		return Quote{}, fmt.Errorf("%w: no USD feed or readable ERC4626 asset", ErrPriceUnavailable)
	}
	if underlying == (common.Address{}) || underlying == token {
		return Quote{}, errors.New("invalid ERC4626 underlying asset")
	}
	d, err := r.tokenDecimals(token)
	if err != nil {
		return Quote{}, err
	}
	unit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d)), nil)
	assets, err := r.unsigned(token, "convertToAssets", unit)
	if err != nil || assets.Sign() <= 0 {
		return Quote{}, fmt.Errorf("%w: invalid ERC4626 conversion", ErrPriceUnavailable)
	}
	underlyingDecimals, err := r.tokenDecimals(underlying)
	if err != nil {
		return Quote{}, err
	}
	base, err := r.price(underlying, depth+1)
	if err != nil {
		return Quote{}, err
	}
	return Quote{USD: decimal.NewFromBigInt(assets, -int32(underlyingDecimals)).Mul(base.USD), UpdatedAt: time.Unix(int64(r.snapshot.Timestamp), 0), ValidUntil: quoteDeadline(base)}, nil
}

func (r *contractResolver) pool(token common.Address) (Quote, error) {
	out, err := r.call(curveNGFactory, "is_meta", token)
	if err != nil {
		return Quote{}, err
	}
	if out[0].(bool) {
		return Quote{}, errors.New("Curve metapool pricing is not supported")
	}
	n, err := r.unsigned(token, "N_COINS")
	if err != nil {
		return Quote{}, err
	}
	if !n.IsUint64() || n.Uint64() < 2 || n.Uint64() > 8 {
		return Quote{}, errors.New("invalid Curve NG coin count")
	}
	supply, err := r.unsigned(token, "totalSupply")
	if err != nil {
		return Quote{}, err
	}
	if supply.Sign() <= 0 {
		return Quote{}, errors.New("Curve NG LP supply is zero")
	}
	d, err := r.tokenDecimals(token)
	if err != nil {
		return Quote{}, err
	}
	value := decimal.Zero
	var deadline time.Time
	seen := make(map[common.Address]bool)
	for i := uint64(0); i < n.Uint64(); i++ {
		coin, err := r.address(token, "coins", new(big.Int).SetUint64(i))
		if err != nil {
			return Quote{}, err
		}
		if coin == (common.Address{}) || seen[coin] {
			return Quote{}, errors.New("invalid Curve NG coin list")
		}
		seen[coin] = true
		balance, err := r.unsigned(token, "balances", new(big.Int).SetUint64(i))
		if err != nil {
			return Quote{}, err
		}
		coinDecimals, err := r.tokenDecimals(coin)
		if err != nil {
			return Quote{}, err
		}
		price, err := r.price(coin, 1)
		if err != nil {
			return Quote{}, fmt.Errorf("Curve component %s: %w", coin.Hex(), err)
		}
		value = value.Add(decimal.NewFromBigInt(balance, -int32(coinDecimals)).Mul(price.USD))
		expires := quoteDeadline(price)
		if deadline.IsZero() || expires.Before(deadline) {
			deadline = expires
		}
	}
	return Quote{USD: value.DivRound(decimal.NewFromBigInt(supply, -int32(d)), 36), UpdatedAt: time.Unix(int64(r.snapshot.Timestamp), 0), ValidUntil: deadline}, nil
}

func quoteDeadline(q Quote) time.Time {
	deadline := q.UpdatedAt.Add(15 * time.Minute)
	if !q.ValidUntil.IsZero() && q.ValidUntil.Before(deadline) {
		return q.ValidUntil
	}
	return deadline
}
