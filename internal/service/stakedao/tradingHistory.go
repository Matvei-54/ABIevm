package stakedao

import (
	"context"
	"errors"
	"math"
	"math/big"
	"strings"
	"sync"

	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
)

// Reviewed plain fixed-rate pools and standard-asset NG pool. This is a fee
// growth model, never a generic virtual-price APY for yield-bearing/rebasing LPs.
// Provenance, assumptions and unsupported models: TRADING_HISTORY_RU.md.
type feePoolRoute struct {
	pool  common.Address
	coins [2]common.Address
	ng    bool
}

type feeSnapshot struct {
	virtualPrice, initialA, futureA *big.Int
	initialTime, futureTime         uint64
}

type feeSnapshotKey struct {
	pool common.Address
	hash common.Hash
}

// Shared bounded cache of immutable, hash-keyed reads only. No API, database or
// current-price cache participates. Boundary headers are always resolved anew.
type TradingHistoryService struct {
	mu        sync.Mutex
	snapshots map[feeSnapshotKey]feeSnapshot
	order     []feeSnapshotKey
	routes    map[common.Address]feePoolRoute
}

func newTradingHistoryService() *TradingHistoryService {
	usdc, usdt := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	crvUSD, weth := common.HexToAddress("0xf939E0A03FB07F59A73314E73794Be0E57ac1b4E"), common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	routes := map[common.Address]feePoolRoute{}
	add := func(pool string, coins [2]common.Address, ng bool) {
		a := common.HexToAddress(pool)
		routes[a] = feePoolRoute{a, coins, ng}
	}
	add("0x4dece678ceceb27446b35c672dc7d61f30bad69e", [2]common.Address{usdc, crvUSD}, false)
	add("0x390f3595bca2df7d23783dfd126427cceb997bf4", [2]common.Address{usdt, crvUSD}, false)
	add("0x9c3b46c0ceb5b9e304fcd6d88fc50f7dd24b31bc", [2]common.Address{weth, common.HexToAddress("0x5E8422345238F34275888049021821E8E08CAa1f")}, false)
	add("0xefc6516323fbd28e80b85a497b65a86243a54b3e", [2]common.Address{common.HexToAddress("0x6440f144b7e50D6a8439336510312d2F54beB01D"), usdc}, true)
	return &TradingHistoryService{snapshots: map[feeSnapshotKey]feeSnapshot{}, routes: routes}
}

const feePoolABI = `[
{"type":"function","name":"get_virtual_price","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"initial_A","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"future_A","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"initial_A_time","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"future_A_time","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"coins","inputs":[{"type":"uint256"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"stored_rates","inputs":[],"outputs":[{"type":"uint256[]"}]},
{"type":"function","name":"get_pool_asset_types","inputs":[{"type":"address"}],"outputs":[{"type":"uint8[]"}]}
]`

func tradingUnavailable(reason string) dto.StakeDAOTradingYield {
	days := 30
	return dto.StakeDAOTradingYield{Status: "unavailable", Source: "curve_onchain_fee_growth", Scope: "fixed_rate_lp_fee_growth", WindowDays: &days, Reasons: []string{reason}}
}

func (s *StakeDAOService) addTradingHistory(ctx context.Context, end *types.Header, pools []dto.StakeDAOPool) {
	headers := map[uint64]*types.Header{end.Number.Uint64(): end}
	boundaries := map[int]*types.Header{}
	boundaryErrors := map[int]error{}
	for i := range pools {
		p := &pools[i]
		if p.Yield == nil {
			continue
		}
		p.APY = nil
		p.Yield.Trading = tradingUnavailable("unsupported_pool_model")
		if s.trading == nil || s.chainID != 1 {
			continue
		}
		route, known := s.trading.routes[p.AssetAddress]
		if !known || p.ProtocolID != "0xc715e373" {
			continue
		}
		p.Yield.Trading.Pool = &route.pool
		endPoint, err := s.trading.snapshot(ctx, s, route, end)
		if err != nil {
			p.Yield.Trading.Reasons = []string{err.Error()}
			continue
		}
		available := 0
		for _, days := range []int{7, 30} {
			w := dto.StakeDAOTradingWindow{WindowDays: days, Status: "unavailable", EndBlock: end.Number.Uint64(), EndTimestamp: end.Time, EndBlockHash: end.Hash()}
			vp := endPoint.virtualPrice.String()
			w.EndVirtualPrice = &vp
			start, found := boundaries[days]
			if !found && boundaryErrors[days] == nil {
				start, err = findFeeBoundary(ctx, s.client, end, days, headers)
				if err != nil {
					boundaryErrors[days] = err
				} else {
					boundaries[days] = start
				}
			}
			if boundaryErrors[days] != nil {
				w.Reasons = []string{boundaryErrors[days].Error()}
			} else {
				n, t, h := start.Number.Uint64(), start.Time, start.Hash()
				seconds := end.Time - start.Time
				w.StartBlock, w.StartTimestamp, w.StartBlockHash, w.CoveredSeconds = &n, &t, &h, &seconds
				beginPoint, readErr := s.trading.snapshot(ctx, s, route, start)
				if readErr != nil {
					w.Reasons = []string{readErr.Error()}
				} else {
					v := beginPoint.virtualPrice.String()
					w.StartVirtualPrice = &v
					w.APY, w.ObservedReturnPercent, readErr = feeGrowth(beginPoint, endPoint, start.Time, end.Time)
					if readErr != nil {
						w.Reasons = []string{readErr.Error()}
					} else {
						verified, verifyErr := s.client.HeaderByNumber(ctx, start.Number)
						if verifyErr != nil || verified == nil || verified.Hash() != h {
							w.APY, w.ObservedReturnPercent = nil, nil
							w.Reasons = []string{"history_snapshot_changed_or_unavailable"}
						} else {
							w.Status = "available"
							available++
						}
					}
				}
			}
			p.Yield.Trading.Windows = append(p.Yield.Trading.Windows, w)
			if days == 30 {
				p.Yield.Trading.APY = w.APY
				p.Yield.Trading.Reasons = w.Reasons
				p.APY = w.APY
			}
		}
		if available == 2 {
			p.Yield.Trading.Status = "available"
		} else if available > 0 {
			p.Yield.Trading.Status = "partial"
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (h *TradingHistoryService) snapshot(ctx context.Context, s *StakeDAOService, route feePoolRoute, header *types.Header) (feeSnapshot, error) {
	key := feeSnapshotKey{route.pool, header.Hash()}
	h.mu.Lock()
	saved, found := h.snapshots[key]
	h.mu.Unlock()
	if found {
		return saved, nil
	}
	reads := []snapshotRead{}
	for _, method := range []string{"get_virtual_price", "initial_A", "future_A", "initial_A_time", "future_A_time"} {
		reads = append(reads, readContract(route.pool, feePoolABI, method))
	}
	for i := 0; i < 2; i++ {
		reads = append(reads, readContract(route.pool, feePoolABI, "coins", big.NewInt(int64(i))))
	}
	if route.ng {
		reads = append(reads, readContract(common.HexToAddress("0x6A8cbed756804B16E05E741eDaBd5cB544AE21bf"), feePoolABI, "get_pool_asset_types", route.pool), readContract(route.pool, feePoolABI, "stored_rates"))
	}
	worker := *s
	worker.batches = nil // Historical blocks must never use the end-block batcher.
	data := worker.snapshotCalls(ctx, header.Number, reads...)
	for _, r := range data {
		if r.err != nil {
			message := strings.ToLower(r.err.Error())
			if strings.Contains(message, "archive requests require") || strings.Contains(message, "archive state unavailable") || strings.Contains(message, "historical state") && strings.Contains(message, "not available") {
				return feeSnapshot{}, errors.New("archive_rpc_required")
			}
			return feeSnapshot{}, errors.New("historical_state_unavailable_or_incompatible")
		}
	}
	values := make([]*big.Int, 5)
	for i := range values {
		v, err := data[i].unsigned()
		if err != nil {
			return feeSnapshot{}, errors.New("invalid_fee_snapshot")
		}
		values[i] = v
	}
	if values[0].Sign() <= 0 || values[1].Sign() <= 0 || values[2].Sign() <= 0 || !values[3].IsUint64() || !values[4].IsUint64() || values[3].Uint64() > header.Time || values[4].Cmp(values[3]) < 0 {
		return feeSnapshot{}, errors.New("invalid_fee_snapshot")
	}
	for i, coin := range route.coins {
		actual, err := data[5+i].address()
		if err != nil || actual != coin {
			return feeSnapshot{}, errors.New("fee_pool_identity_mismatch")
		}
	}
	if route.ng {
		if len(data[7].values) != 1 || len(data[8].values) != 1 {
			return feeSnapshot{}, errors.New("unsupported_rate_model")
		}
		types, tOK := data[7].values[0].([]uint8)
		rates, rOK := data[8].values[0].([]*big.Int)
		if !tOK || !rOK || len(types) != 2 || len(rates) != 2 || types[0] != 0 || types[1] != 0 || rates[0] == nil || rates[1] == nil || rates[0].Cmp(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) != 0 || rates[1].Cmp(new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)) != 0 {
			return feeSnapshot{}, errors.New("unsupported_rate_model")
		}
	}
	result := feeSnapshot{values[0], values[1], values[2], values[3].Uint64(), values[4].Uint64()}
	h.mu.Lock()
	if _, exists := h.snapshots[key]; !exists {
		if len(h.order) >= 256 {
			delete(h.snapshots, h.order[0])
			h.order = h.order[1:]
		}
		h.snapshots[key] = result
		h.order = append(h.order, key)
	}
	h.mu.Unlock()
	return result, nil
}

func findFeeBoundary(ctx context.Context, client stakeDAORPC, end *types.Header, days int, cache map[uint64]*types.Header) (*types.Header, error) {
	if ctx.Err() != nil {
		return nil, errors.New("historical_header_unavailable")
	}
	span := uint64(days) * 86400
	if days <= 0 || end.Time < span {
		return nil, errors.New("insufficient_history")
	}
	target := end.Time - span
	last := end.Number.Uint64()
	read := func(n uint64) (*types.Header, error) {
		if h, ok := cache[n]; ok {
			return h, nil
		}
		h, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(n))
		if err != nil || h == nil || h.Number == nil || h.Number.Uint64() != n || h.Time > end.Time {
			return nil, errors.New("historical_header_unavailable")
		}
		cache[n] = h
		return h, nil
	}
	// Approximation only brackets a search; timestamps determine the actual block.
	estimate := uint64(0)
	if last > span/12 {
		estimate = last - span/12
	}
	low := uint64(0)
	if estimate > 7200 {
		low = estimate - 7200
	}
	high := min(last, estimate+7200)
	lh, err := read(low)
	if err != nil {
		return nil, err
	}
	hh, err := read(high)
	if err != nil {
		return nil, err
	}
	if lh.Time > target || hh.Time <= target {
		low, high = 0, last
		lh, err = read(low)
		if err != nil {
			return nil, err
		}
		if lh.Time > target {
			return nil, errors.New("insufficient_history")
		}
	}
	for high-low > 1 {
		if ctx.Err() != nil {
			return nil, errors.New("historical_header_unavailable")
		}
		mid := low + (high-low)/2
		mh, err := read(mid)
		if err != nil {
			return nil, err
		}
		if mh.Time <= target {
			low, lh = mid, mh
		} else {
			high = mid
		}
	}
	if target < lh.Time || target-lh.Time > 120 {
		return nil, errors.New("history_boundary_gap")
	}
	return lh, nil
}

func feeGrowth(start, end feeSnapshot, startTime, endTime uint64) (*float64, *float64, error) {
	if endTime <= startTime || start.virtualPrice == nil || end.virtualPrice == nil || start.virtualPrice.Sign() <= 0 || end.virtualPrice.Sign() <= 0 {
		return nil, nil, errors.New("invalid_fee_snapshot")
	}
	// Latest ramp timestamps cannot be backdated by the reviewed implementations.
	// Reject any parameter ramp affecting the window, even if endpoints look equal.
	if start.futureTime > startTime || end.futureTime > startTime || end.initialTime > startTime || start.futureA == nil || end.futureA == nil || start.futureA.Cmp(end.futureA) != 0 {
		return nil, nil, errors.New("amplification_changed_in_window")
	}
	if end.virtualPrice.Cmp(start.virtualPrice) < 0 {
		return nil, nil, errors.New("virtual_price_decreased")
	}
	growth := decimal.NewFromBigInt(end.virtualPrice, 0).DivRound(decimal.NewFromBigInt(start.virtualPrice, 0), 36).Sub(decimal.NewFromInt(1))
	f, _ := growth.Float64()
	observed, _ := growth.Mul(decimal.NewFromInt(100)).Float64()
	apy := math.Expm1(math.Log1p(f)*31536000/float64(endTime-startTime)) * 100
	if math.IsNaN(apy) || math.IsInf(apy, 0) || apy > 1e6 {
		return nil, nil, errors.New("invalid_annualized_return")
	}
	return &apy, &observed, nil
}
