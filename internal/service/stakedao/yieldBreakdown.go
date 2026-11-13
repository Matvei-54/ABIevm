package stakedao

import (
	"context"
	"math"
	"math/big"

	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

// Curve factory gauge getters: contracts/LiquidityGauge.vy in curvefi/curve-factory.
// Reward is (token, distributor, period_finish, rate, last_update, integral).
const curveGaugeExtrasABI = `[
{"type":"function","name":"lp_token","inputs":[],"outputs":[{"type":"address"}],"stateMutability":"view"},
{"type":"function","name":"totalSupply","inputs":[],"outputs":[{"type":"uint256"}],"stateMutability":"view"},
{"type":"function","name":"reward_count","inputs":[],"outputs":[{"type":"uint256"}],"stateMutability":"view"},
{"type":"function","name":"reward_tokens","inputs":[{"type":"uint256"}],"outputs":[{"type":"address"}],"stateMutability":"view"},
{"type":"function","name":"reward_data","inputs":[{"type":"address"}],"outputs":[{"type":"address"},{"type":"address"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"}],"stateMutability":"view"}
]`

func aprComponent(source, scope, reason string) dto.StakeDAOAPRComponent {
	return dto.StakeDAOAPRComponent{Status: "unavailable", Source: source, Scope: scope, Reasons: []string{reason}}
}

func (s *StakeDAOService) loadYieldBreakdown(ctx context.Context, info StakeDAOVaultInfo, block *big.Int, pool *dto.StakeDAOPool) {
	pool.Yield = &dto.StakeDAOYieldBreakdown{
		Status: "partial", IncludedInAPR: []string{"crv", "vaultExtra"},
		CRV:        aprComponent("curve-primary-net", "net_vault_stream", "reward_data_missing"),
		VaultExtra: aprComponent("vault-extra", "active_vault_distribution", "reward_data_missing"),
		GaugeExtra: dto.StakeDAOGaugeYield{
			StakeDAOAPRComponent: aprComponent("curve_gauge", "gross_per_staked_lp_projection", "gauge_data_missing"),
			Contract:             info.Gauge, Rewards: []dto.StakeDAOGaugeReward{},
		},
		Merkl:   aprComponent("merkl_incentives", "campaign_projection", "incentive_registry_required"),
		Trading: dto.StakeDAOTradingYield{Status: "unavailable", Source: "historical_pool_fees", Reasons: []string{"historical_window_required"}},
	}
	g := &pool.Yield.GaugeExtra
	if s.chainID != 1 || info.ProtocolID != [4]byte{0xc7, 0x15, 0xe3, 0x73} {
		g.Reasons = []string{"unsupported_model"}
		return
	}
	reads := s.snapshotCalls(ctx, block,
		readContract(info.Gauge, curveGaugeExtrasABI, "lp_token"),
		readContract(info.Gauge, curveGaugeExtrasABI, "totalSupply"),
		readContract(info.Gauge, curveGaugeExtrasABI, "reward_count"),
	)
	lp, lpErr := reads[0].address()
	supply, supplyErr := reads[1].unsigned()
	count, countErr := reads[2].unsigned()
	if lpErr != nil || supplyErr != nil || countErr != nil {
		return // A missing interface is not an empty reward list.
	}
	if lp != info.Asset {
		g.Reasons = []string{"gauge_asset_mismatch"}
		return
	}
	if !count.IsUint64() || count.Uint64() > 8 {
		g.Reasons = []string{"unsupported_reward_count"}
		return
	}
	if pool.AssetDecimals == nil || *pool.AssetDecimals > 36 {
		g.Reasons = []string{"asset_data_missing"}
		return
	}
	quantity := decimal.NewFromBigInt(supply, -int32(*pool.AssetDecimals)).String()
	g.TotalSupply = &quantity
	if count.Sign() == 0 {
		zero := float64(0)
		g.APR, g.Status, g.Reasons = &zero, "available", nil
		return
	}
	var tokenReads []snapshotRead
	for i := uint64(0); i < count.Uint64(); i++ {
		tokenReads = append(tokenReads, readContract(info.Gauge, curveGaugeExtrasABI, "reward_tokens", new(big.Int).SetUint64(i)))
	}
	tokens := s.snapshotCalls(ctx, block, tokenReads...)
	seen := map[common.Address]bool{}
	var rewardReads []snapshotRead
	for _, tokenResult := range tokens {
		token, err := tokenResult.address()
		if err != nil || token == (common.Address{}) || seen[token] {
			g.Reasons = []string{"invalid_reward_list"}
			g.Rewards = []dto.StakeDAOGaugeReward{}
			return
		}
		seen[token] = true
		g.Rewards = append(g.Rewards, dto.StakeDAOGaugeReward{Token: token, DistributionStatus: "unavailable", Reason: "reward_data_missing"})
		rewardReads = append(rewardReads, readContract(info.Gauge, curveGaugeExtrasABI, "reward_data", token), readContract(token, erc20DecimalsABI, "decimals"))
	}
	data := s.snapshotCalls(ctx, block, rewardReads...)
	g.Reasons = []string{"reward_data_missing"}
	for i := range g.Rewards {
		reward := &g.Rewards[i]
		out, err := data[2*i].values, data[2*i].err
		if err != nil || len(out) != 6 {
			continue
		}
		distributor, ok := out[1].(common.Address)
		finish, finishOK := out[2].(*big.Int)
		rate, rateOK := out[3].(*big.Int)
		updated, updatedOK := out[4].(*big.Int)
		if !ok || distributor == (common.Address{}) || !finishOK || finish == nil || !finish.IsUint64() || !rateOK || rate == nil || rate.Sign() < 0 || !updatedOK || updated == nil || !updated.IsUint64() || updated.Uint64() > s.snapshotTime {
			continue
		}
		period, value := finish.Uint64(), "0"
		reward.PeriodFinish, reward.Reason = &period, ""
		switch {
		case period == 0 && rate.Sign() == 0:
			reward.DistributionStatus = "not_started"
		case period <= s.snapshotTime:
			reward.DistributionStatus = "expired"
		case rate.Sign() == 0:
			reward.DistributionStatus = "zero_rate"
		default:
			d, err := data[2*i+1].unsigned()
			if err != nil || !d.IsUint64() || d.Uint64() > 36 {
				reward.Reason = "reward_decimals_missing"
				continue
			}
			value = decimal.NewFromBigInt(rate, -int32(d.Uint64())).String()
			reward.DistributionStatus = "active"
		}
		reward.RatePerSecond = &value
	}
}

func (s *StakeDAOService) valueYieldBreakdown(ctx context.Context, pool *dto.StakeDAOPool) {
	if pool.Yield == nil {
		return
	}
	y := pool.Yield
	y.CRV = streamComponent(pool, "curve-primary-net", "net_vault_stream", true)
	y.VaultExtra = streamComponent(pool, "vault-extra", "active_vault_distribution", false)
	g := &y.GaugeExtra
	if g.Status == "available" || g.TotalSupply == nil || len(g.Rewards) == 0 {
		return
	}
	supply, err := decimal.NewFromString(*g.TotalSupply)
	if err != nil || !supply.IsPositive() {
		g.Reasons = []string{"gauge_supply_zero"}
		return
	}
	total, valued := decimal.Zero, 0
	complete := true
	g.Reasons = []string{"reward_data_or_price_missing"}
	for i := range g.Rewards {
		r := &g.Rewards[i]
		if r.RatePerSecond == nil {
			complete = false
			continue
		}
		rate, err := decimal.NewFromString(*r.RatePerSecond)
		if err != nil || rate.IsNegative() {
			complete = false
			continue
		}
		apr := decimal.Zero
		if rate.IsPositive() {
			if s.prices == nil {
				r.Reason, complete = "price_service_unavailable", false
				continue
			}
			lp, lpErr := s.valuationPrice(ctx, pool, pool.AssetAddress)
			price, priceErr := s.valuationPrice(ctx, pool, r.Token)
			if lpErr != nil || priceErr != nil || !lp.IsPositive() || !price.IsPositive() {
				r.Reason, complete = "price_missing", false
				continue
			}
			apr = rate.Mul(price).Mul(decimal.NewFromInt(31536000)).Mul(decimal.NewFromInt(100)).DivRound(supply.Mul(lp), 128)
		}
		value, ok := finiteFloat(apr)
		if !ok {
			r.Reason, complete = "invalid_apr", false
			continue
		}
		r.APR, r.Reason = &value, ""
		total, valued = total.Add(apr), valued+1
	}
	if valued > 0 {
		value, ok := finiteFloat(total)
		if ok {
			g.APR, g.Status, g.Reasons = &value, "partial", []string{"reward_data_or_price_missing"}
			if complete {
				g.Status, g.Reasons = "available", nil
			}
		}
	}
}

// Aggregate only already-valued vault streams; never add upstream projections.
func streamComponent(pool *dto.StakeDAOPool, source, scope string, required bool) dto.StakeDAOAPRComponent {
	c := aprComponent(source, scope, "reward_data_or_price_missing")
	total, found, valued, complete := decimal.Zero, false, false, true
	for _, r := range pool.Rewards {
		if r.Source != source {
			continue
		}
		found = true
		if r.APR == nil || math.IsNaN(*r.APR) || math.IsInf(*r.APR, 0) {
			complete = false
			continue
		}
		total, valued = total.Add(decimal.NewFromFloat(*r.APR)), true
	}
	if !found {
		if required || !pool.VaultExtrasComplete {
			return c
		}
		if pool.TVL == nil || *pool.TVL <= 0 {
			c.Reasons = []string{"positive_vault_tvl_required"}
			return c
		}
		valued = true // A successfully read empty vault-extra list confirms zero.
	}
	if valued {
		value, ok := finiteFloat(total)
		if ok {
			c.APR, c.Status, c.Reasons = &value, "partial", []string{"reward_data_or_price_missing"}
			if complete {
				c.Status, c.Reasons = "available", nil
			}
		}
	}
	return c
}

func finiteFloat(value decimal.Decimal) (float64, bool) {
	f, _ := value.Float64()
	return f, !math.IsNaN(f) && !math.IsInf(f, 0)
}
