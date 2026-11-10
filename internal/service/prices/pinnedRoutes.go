package prices

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type pinnedRouteKind uint8

const (
	pinnedChainlinkETH pinnedRouteKind = iota
	pinnedCurveEMA
	pinnedCurveNG
	pinnedCurveMeta
)

type pinnedPriceRoute struct {
	kind        pinnedRouteKind
	source      common.Address
	quoteToken  common.Address
	targetIndex int64
	indexedEMA  bool
	description string
}

// Ethereum-only reviewed routes, not cached prices. Provenance, oracle units,
// liquidity/freshness policy and mainnet results: ORACLE_ROUTES_RU.md.
// An unavailable pinned source never falls back to a peg or an unreviewed route.
var ethereumPinnedPriceRoutes = map[common.Address]pinnedPriceRoute{
	common.HexToAddress("0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"): {
		kind: pinnedChainlinkETH, source: common.HexToAddress("0x5c9C449BbC9a6075A2c061dF312a35fd1E05fF22"), quoteToken: common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
		description: "weETH / ETH",
	},
	common.HexToAddress("0x5E8422345238F34275888049021821E8E08CAa1f"): {
		kind: pinnedCurveEMA, source: common.HexToAddress("0x9c3b46c0ceb5b9e304fcd6d88fc50f7dd24b31bc"), quoteToken: common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), targetIndex: 1,
	},
	common.HexToAddress("0x6440f144b7e50D6a8439336510312d2F54beB01D"): {
		kind: pinnedCurveEMA, source: common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e"), quoteToken: common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), targetIndex: 0, indexedEMA: true,
	},
	common.HexToAddress("0xdb74dfdd3bb46be8ce6c33dc9d82777bcfc3ded5"): {kind: pinnedCurveNG},
	common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e"): {kind: pinnedCurveNG},
	common.HexToAddress("0x9c3b46c0ceb5b9e304fcd6d88fc50f7dd24b31bc"): {kind: pinnedCurveMeta, source: common.HexToAddress("0x9c3b46c0ceb5b9e304fcd6d88fc50f7dd24b31bc")},
}

func (r *contractResolver) pinnedPrice(token common.Address, depth int, route pinnedPriceRoute) (Quote, error) {
	switch route.kind {
	case pinnedChainlinkETH:
		desc, err := r.call(route.source, "description")
		if err != nil {
			return Quote{}, err
		}
		if len(desc) != 1 || route.description == "" || desc[0] != route.description {
			return Quote{}, errors.New("pinned Chainlink pair description mismatch")
		}
		q, err := r.chainlinkQuote(route.source)
		if err != nil {
			return Quote{}, err
		}
		base, err := r.price(route.quoteToken, depth+1)
		if err != nil {
			return Quote{}, err
		}
		q.USD = q.USD.Mul(base.USD)
		q.Source, q.Components = "chainlink_cross_usd", []common.Address{route.quoteToken}
		if deadline := quoteDeadline(base); deadline.Before(q.ValidUntil) {
			q.ValidUntil = deadline
		}
		return q, nil
	case pinnedCurveEMA:
		return r.curveEMA(token, depth, route)
	case pinnedCurveNG, pinnedCurveMeta:
		if depth != 0 {
			return Quote{}, errors.New("nested Curve LP pricing is not supported")
		}
		if route.kind == pinnedCurveNG {
			return r.pool(token)
		}
		return r.metaPool(token, route.source)
	default:
		return Quote{}, errors.New("unknown pinned price route")
	}
}

func (r *contractResolver) curveEMA(token common.Address, depth int, route pinnedPriceRoute) (Quote, error) {
	// Both reviewed pools expose coin 1 in coin 0 units, scaled by 1e18.
	// BOLD is coin 0 (invert USDC/BOLD); frxETH is coin 1 (WETH/frxETH).
	coins := [2]common.Address{}
	for i := range coins {
		coin, err := r.address(route.source, "coins", big.NewInt(int64(i)))
		if err != nil {
			return Quote{}, err
		}
		coins[i] = coin
	}
	if route.targetIndex < 0 || route.targetIndex > 1 || coins[route.targetIndex] != token || coins[1-route.targetIndex] != route.quoteToken {
		return Quote{}, errors.New("pinned EMA pool coin identity mismatch")
	}
	if route.indexedEMA {
		n, err := r.unsigned(route.source, "N_COINS")
		if err != nil {
			return Quote{}, err
		}
		if n.Cmp(big.NewInt(2)) != 0 {
			return Quote{}, errors.New("pinned EMA pool must have two coins")
		}
		// NG EMA is expressed in normalized xp units. Only standard, fixed
		// normalization rates are allowed for this BOLD/USDC route.
		out, err := r.call(route.source, "stored_rates")
		if err != nil {
			return Quote{}, err
		}
		if len(out) != 1 {
			return Quote{}, errors.New("invalid EMA normalization rates")
		}
		rates, ok := out[0].([]*big.Int)
		if !ok || len(rates) != 2 {
			return Quote{}, errors.New("invalid EMA normalization rates")
		}
		for i, coin := range coins {
			d, err := r.tokenDecimals(coin)
			if err != nil {
				return Quote{}, err
			}
			expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(36-int64(d)), nil)
			if rates[i] == nil || rates[i].Cmp(expected) != 0 {
				return Quote{}, errors.New("EMA uses unsupported dynamic normalization rates")
			}
		}
	}
	window, err := r.unsigned(route.source, "ma_exp_time")
	if err != nil {
		return Quote{}, err
	}
	if !window.IsUint64() || window.Uint64() < 300 || window.Uint64() > 86400 {
		return Quote{}, errors.New("EMA averaging window outside reviewed bounds")
	}
	packed, err := r.unsigned(route.source, "ma_last_time")
	if err != nil {
		return Quote{}, err
	}
	mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	observed := new(big.Int).And(packed, mask)
	const maxAge = uint64(2 * time.Hour / time.Second)
	if !observed.IsUint64() || observed.Sign() == 0 || observed.Uint64() > r.snapshot.Timestamp || r.snapshot.Timestamp-observed.Uint64() >= maxAge {
		return Quote{}, errors.New("invalid or stale EMA observation")
	}
	method := "price_oracle"
	var args []interface{}
	if route.indexedEMA {
		method = "price_oracle0"
		args = []interface{}{big.NewInt(0)}
	}
	raw, err := r.unsigned(route.source, method, args...)
	if err != nil {
		return Quote{}, err
	}
	if raw.Sign() <= 0 {
		return Quote{}, errors.New("EMA price must be positive")
	}
	base, err := r.price(route.quoteToken, depth+1)
	if err != nil {
		return Quote{}, err
	}
	price := decimal.NewFromBigInt(raw, -18)
	if route.targetIndex == 0 {
		price = decimal.NewFromInt(1).DivRound(price, 128)
	}
	price = price.Mul(base.USD)
	// Require independently USD-anchored quote liquidity as well as target
	// liquidity. The threshold is a route policy, not an oracle guarantee.
	for i, coin := range coins {
		balance, err := r.unsigned(route.source, "balances", big.NewInt(int64(i)))
		if err != nil {
			return Quote{}, err
		}
		d, err := r.tokenDecimals(coin)
		if err != nil {
			return Quote{}, err
		}
		usd := base.USD
		if coin == token {
			usd = price
		}
		if decimal.NewFromBigInt(balance, -int32(d)).Mul(usd).LessThan(decimal.NewFromInt(100000)) {
			return Quote{}, errors.New("EMA pool liquidity below 100000 USD per side")
		}
	}
	deadline := time.Unix(int64(observed.Uint64()+maxAge), 0)
	if other := quoteDeadline(base); other.Before(deadline) {
		deadline = other
	}
	return Quote{USD: price, UpdatedAt: time.Unix(int64(r.snapshot.Timestamp), 0), ValidUntil: deadline, Source: "curve_ema_usd", SourceContract: route.source, BlockNumber: r.snapshot.Block, Components: []common.Address{route.quoteToken}}, nil
}
