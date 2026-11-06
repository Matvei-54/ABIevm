package stakedao

import (
	"context"
	"math"
	"math/big"
	"strconv"

	"ABIevm/internal/service/prices"
	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

const convexPoolUtilitiesEthereum = "0x5Fba69a794F395184b5760DAf1134028608e5Cd1"

// Official PoolUtilities, verified source and mainnet probe provenance:
// CONVEX_REFERENCE_REVIEW_RU.md. Diagnostic use only; not an additional reward.
const convexPoolUtilitiesABI = `[{"name":"rewardRates","type":"function","stateMutability":"view","inputs":[{"type":"uint256"}],"outputs":[{"type":"address[]"},{"type":"uint256[]"}]}]`

type quoteProvider interface {
	GetTokenQuote(context.Context, common.Address, int64) (prices.Quote, error)
}

func (s *StakeDAOService) valuationPrice(ctx context.Context, pool *dto.StakeDAOPool, token common.Address) (decimal.Decimal, error) {
	if s.diagnostics {
		if provider, ok := s.prices.(quoteProvider); ok {
			q, err := provider.GetTokenQuote(ctx, token, s.chainID)
			if err != nil {
				return decimal.Zero, err
			}
			if pool.Diagnostics == nil {
				pool.Diagnostics = &dto.StakeDAOPoolDiagnostics{}
			}
			// Retain the quote actually used, even if another request refreshes
			// the shared cache before this report is assembled.
			for _, old := range pool.Diagnostics.Prices {
				if old.Token == token {
					return decimal.RequireFromString(old.USD), nil
				}
			}
			pool.Diagnostics.Prices = append(pool.Diagnostics.Prices, priceDiagnostic(token, q))
			return q.USD, nil
		}
	}
	return s.prices.GetTokenPrice(ctx, token, s.chainID)
}

func priceDiagnostic(token common.Address, quote prices.Quote) dto.PriceDiagnostic {
	return dto.PriceDiagnostic{
		Token: token, USD: quote.USD.String(), Source: quote.Source,
		SourceContract: quote.SourceContract, BlockNumber: quote.BlockNumber,
		UpdatedAt: quote.UpdatedAt, ValidUntil: quote.ValidUntil, Components: quote.Components,
	}
}

func (s *StakeDAOService) addPoolDiagnostics(ctx context.Context, block *big.Int, pool *dto.StakeDAOPool) {
	if pool.Diagnostics == nil {
		pool.Diagnostics = &dto.StakeDAOPoolDiagnostics{}
	}
	// Prices used in valuation are retained above. Add cached component
	// provenance, with explicit individual block/time labels and no extra RPC.
	if provider, ok := s.prices.(quoteProvider); ok {
		seen := map[common.Address]bool{}
		used := map[common.Address]dto.PriceDiagnostic{}
		for _, q := range pool.Diagnostics.Prices {
			used[q.Token] = q
		}
		var visit func(common.Address, int)
		visit = func(token common.Address, depth int) {
			if depth > 4 || seen[token] {
				return
			}
			seen[token] = true
			entry, found := used[token]
			if !found {
				quote, err := provider.GetTokenQuote(ctx, token, s.chainID)
				if err != nil {
					return
				}
				entry = priceDiagnostic(token, quote)
				pool.Diagnostics.Prices = append(pool.Diagnostics.Prices, entry)
			}
			for _, component := range entry.Components {
				visit(component, depth+1)
			}
		}
		visit(pool.AssetAddress, 0)
		for _, reward := range pool.Rewards {
			visit(reward.Token, 0)
		}
	}
	s.addConvexDiagnostic(ctx, block, pool)
}

func (s *StakeDAOService) addConvexDiagnostic(ctx context.Context, block *big.Int, pool *dto.StakeDAOPool) {
	if s.chainID != 1 {
		return
	}
	var pid *big.Int
	var own *dto.ConvexPlacement
	if p := pool.Diagnostics.Primary; p != nil && len(p.Sidecars) == 1 {
		own = &p.Sidecars[0]
		pid, _ = new(big.Int).SetString(own.PID, 10)
	}
	// These references also permit a comparison for FXS/ETH, whose actual
	// StakeDAO position has no Sidecar. Verify both identities before use.
	if pid == nil {
		switch pool.GaugeAddress {
		case common.HexToAddress("0x95f00391cb5eebcd190eb58728b4ce23dbfa6ac1"):
			pid = big.NewInt(182)
		case common.HexToAddress("0x009acd89535dabc270c93f9b39d3232105fef453"):
			pid = big.NewInt(69)
		default:
			return
		}
	}
	d := &dto.ConvexRateDiagnostic{Utility: common.HexToAddress(convexPoolUtilitiesEthereum), PID: pid.String(), Status: "unavailable"}
	pool.Diagnostics.Convex = d
	d.Message = "Canonical Convex LP/gauge/reward-pool identity could not be verified"
	identity, err := s.snapshotCall(ctx, block, common.HexToAddress(convexBoosterEthereum), convexBoosterABI, "poolInfo", pid)
	if err != nil || len(identity) != 6 {
		return
	}
	lp, lpOK := identity[0].(common.Address)
	gauge, gaugeOK := identity[2].(common.Address)
	rrp, rrpOK := identity[3].(common.Address)
	if !lpOK || !gaugeOK || !rrpOK || lp != pool.AssetAddress || gauge != pool.GaugeAddress || rrp == (common.Address{}) || (own != nil && own.RewardPool != rrp) {
		return
	}
	d.RewardPool = rrp
	reads := []snapshotRead{
		readContract(d.Utility, convexPoolUtilitiesABI, "rewardRates", pid),
		readContract(rrp, convexPoolABI, "rewardToken"),
		readContract(rrp, convexPoolABI, "rewardRate"),
		readContract(rrp, convexPoolABI, "totalSupply"),
		readContract(rrp, convexPoolABI, "periodFinish"),
	}
	if own != nil {
		reads = append(reads, readContract(own.Address, sidecarV2ABI, "balanceOf"))
	}
	results := s.snapshotCalls(ctx, block, reads...)
	d.Message = "PoolUtilities or BaseRewardPool reference data could not be read or decoded"
	if results[0].err != nil || len(results[0].values) != 2 {
		return
	}
	tokens, okT := results[0].values[0].([]common.Address)
	rates, okR := results[0].values[1].([]*big.Int)
	if !okT || !okR || len(tokens) < 2 || len(tokens) != len(rates) || len(tokens) > 64 {
		return
	}
	crv := common.HexToAddress("0xD533a949740bb3306d119CC777fa900bA034cd52")
	token, errT := results[1].address()
	raw, errR := results[2].unsigned()
	supply, errS := results[3].unsigned()
	finish, errF := results[4].unsigned()
	if errT != nil || token != crv || tokens[0] != crv || errR != nil || errS != nil || errF != nil || !finish.IsUint64() {
		return
	}
	for i, rate := range rates {
		if rate == nil || rate.Sign() < 0 {
			return
		}
		source := "convex_extra:" + strconv.Itoa(i-2)
		if i == 0 {
			source = "crv_base"
		}
		if i == 1 {
			source = "cvx_mint_projection"
		}
		d.Slots = append(d.Slots, dto.ConvexRewardSlot{Token: tokens[i], Source: source, RateRaw: rate.String()})
	}
	rawText, supplyText, helperText, period := raw.String(), supply.String(), rates[0].String(), finish.Uint64()
	d.RewardRateRaw, d.StakedSupplyRaw, d.HelperCrvRateRaw, d.PeriodFinish = &rawText, &supplyText, &helperText, &period
	if supply.Sign() == 0 {
		d.Status, d.Message = "zero_supply", "No per-LP comparison: PoolUtilities skips normalization when staked supply is zero"
		return
	}
	expected := helperCRVRate(raw, supply, period, s.snapshotTime)
	expectedText := expected.String()
	d.ExpectedCrvRateRaw = &expectedText
	d.Status, d.Message = "matched", ""
	if expected.Cmp(rates[0]) != 0 {
		d.Status, d.Message = "mismatch", "Helper CRV rate differs from floor(active rewardRate * 1e18 / supply)"
	}
	if own == nil {
		if d.Message == "" {
			d.Message = "Convex pool reference only; no verified single Sidecar placement for this vault"
		}
		return
	}
	balance, err := results[5].unsigned()
	if err != nil || balance.Cmp(supply) > 0 {
		d.Status, d.Message = "unavailable", "Sidecar balance could not be read or exceeds Convex supply"
		return
	}
	balanceText := balance.String()
	d.SidecarBalanceRaw = &balanceText
	// CRV has 18 decimals. First /1e18 undoes helper normalization, second
	// converts raw CRV to tokens. Helper floors before allocation, so retain
	// its bounded rounding difference instead of requiring exact equality.
	fromHelper := decimal.NewFromBigInt(rates[0], -36).Mul(decimal.NewFromBigInt(balance, 0))
	if period <= s.snapshotTime {
		fromHelper = decimal.Zero
	}
	actual, err := decimal.NewFromString(pool.Diagnostics.Primary.ConvexGrossRate)
	if err != nil {
		return
	}
	helperOwn, actualOwn, difference := fromHelper.String(), actual.String(), actual.Sub(fromHelper).String()
	d.HelperOwnSidecarRate, d.OwnSidecarRate, d.RoundingDifference = &helperOwn, &actualOwn, &difference
	bound := decimal.NewFromBigInt(balance, -36)
	if actual.Sub(fromHelper).Abs().GreaterThan(bound.Add(decimal.New(1, -100))) {
		d.Status, d.Message = "mismatch", "Own Sidecar rate differs beyond helper integer-rounding tolerance"
	}
}

func helperCRVRate(raw, supply *big.Int, finish, timestamp uint64) *big.Int {
	if supply.Sign() <= 0 || finish < timestamp {
		return new(big.Int)
	}
	return new(big.Int).Quo(new(big.Int).Mul(raw, new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)), supply)
}

// Public @stake-dao/reader@0.5.52 getOnlyboostData uses the whole stored
// BaseRewardPool rate divided by Sidecar TVL, then caps at maxApr. Published
// Hub samples satisfy CRV APR = minApr * weighted boost * (1 - protocolFee).
// Keep this reference out of actual rewards and label the private-Hub gap.
func addReaderModel(block uint64, pool *dto.StakeDAOPool) {
	d := &dto.ReaderModelDiagnostic{
		Status: "unavailable", Source: "@stake-dao/reader@0.5.52 + observed published Hub fee identity",
		Scope:   "CRV only; public-reader reference, not a block-verified live UI value",
		Message: "Complete CRV inputs and prices from the selected block are required",
	}
	pool.Diagnostics.ReaderModel = d
	p := pool.Diagnostics.Primary
	crv := common.HexToAddress("0xD533a949740bb3306d119CC777fa900bA034cd52")
	if p == nil || p.LockerInputs == nil || pool.RewardToken == nil || *pool.RewardToken != crv || pool.AssetDecimals == nil || *pool.AssetDecimals != 18 || pool.AssetAmount == nil {
		return
	}
	var lpPrice, crvPrice decimal.Decimal
	for _, q := range pool.Diagnostics.Prices {
		if q.BlockNumber != block {
			return
		}
		if q.Token == pool.AssetAddress {
			lpPrice, _ = decimal.NewFromString(q.USD)
		}
		if q.Token == crv {
			crvPrice, _ = decimal.NewFromString(q.USD)
		}
	}
	assets, err := decimal.NewFromString(*pool.AssetAmount)
	if err != nil || !assets.IsPositive() || !lpPrice.IsPositive() || !crvPrice.IsPositive() {
		return
	}
	locker, errL := decimal.NewFromString(p.LockerGrossRate)
	own, errO := decimal.NewFromString(p.ConvexGrossRate)
	pf, errP := decimal.NewFromString(p.ProtocolFeeFraction)
	hf, errH := decimal.NewFromString(p.HarvestFeeFraction)
	if errL != nil || errO != nil || errP != nil || errH != nil {
		return
	}
	unitMinRate := decimal.Zero
	if !p.LockerInputs.Killed {
		i, errI := decimal.NewFromString(p.LockerInputs.InflationRaw)
		w, errW := decimal.NewFromString(p.LockerInputs.RelativeWeightRaw)
		s, errS := decimal.NewFromString(p.LockerInputs.WorkingSupplyRaw)
		if errI != nil || errW != nil || errS != nil || !s.IsPositive() {
			return
		}
		unitMinRate = i.Mul(w.Shift(-18)).Mul(decimal.RequireFromString("0.4")).DivRound(s, 128)
	}
	tvl := assets.Mul(lpPrice)
	annualFactor := crvPrice.Mul(decimal.NewFromInt(31536000)).Mul(decimal.NewFromInt(100)).DivRound(tvl, 128)
	global, candidate, sidecarAssets := decimal.Zero, decimal.Zero, decimal.Zero
	if len(p.Sidecars) != 0 {
		c := pool.Diagnostics.Convex
		if len(p.Sidecars) != 1 || c == nil || c.Status != "matched" || c.RewardRateRaw == nil || c.SidecarBalanceRaw == nil || pool.Allocation == nil || pool.Allocation.Status != "covered" {
			return
		}
		raw, errR := decimal.NewFromString(*c.RewardRateRaw)
		balance, errB := decimal.NewFromString(*c.SidecarBalanceRaw)
		if errR != nil || errB != nil || balance.IsNegative() {
			return
		}
		global = raw.Shift(-18)
		sidecarAssets = balance.Shift(-18)
		if sidecarAssets.GreaterThan(assets) {
			return
		}
		candidate = decimal.Min(global, unitMinRate.Mul(decimal.RequireFromString("2.5")).Mul(sidecarAssets))
	}
	// Separate the policy differences instead of silently changing our APR.
	actual := netPrimaryRate(locker, own, pf, hf).Mul(annualFactor)
	noHarvest := locker.Add(own).Sub(locker.Mul(pf)).Mul(annualFactor)
	globalFee := locker.Add(own).Mul(decimal.NewFromInt(1).Sub(pf)).Mul(annualFactor)
	reference := locker.Add(candidate).Mul(decimal.NewFromInt(1).Sub(pf)).Mul(annualFactor)
	minAPR := unitMinRate.Mul(crvPrice).Mul(decimal.NewFromInt(3153600000)).DivRound(lpPrice, 128)
	boost := decimal.RequireFromString("2.5")
	if unitMinRate.IsPositive() {
		boost = locker.Add(candidate).DivRound(assets.Mul(unitMinRate), 128)
	}
	f := func(v decimal.Decimal) *float64 {
		x, _ := v.Float64()
		if math.IsInf(x, 0) || math.IsNaN(x) {
			return nil
		}
		return &x
	}
	actualOwn, globalText, candidateText := own.String(), global.String(), candidate.String()
	d.Status, d.Message = "available", "Private Hub formula and live UI block are not verified; reference values are not included in pool APR"
	d.GrossMinAPR, d.WeightedBoost, d.ActualCRVAPR = f(minAPR), f(boost), f(actual)
	d.WithoutHarvestFeeAPR, d.GlobalProtocolFeeAPR, d.ReferenceAPR = f(noHarvest), f(globalFee), f(reference)
	d.DifferencePercentagePoints = f(reference.Sub(actual))
	d.OwnSidecarCRVRate, d.GlobalPoolCRVRate, d.CappedReferenceSidecarCRVRate = &actualOwn, &globalText, &candidateText
	if d.GrossMinAPR == nil || d.WeightedBoost == nil || d.ActualCRVAPR == nil || d.WithoutHarvestFeeAPR == nil || d.GlobalProtocolFeeAPR == nil || d.ReferenceAPR == nil || d.DifferencePercentagePoints == nil {
		d.Status, d.Message = "unavailable", "Reference arithmetic cannot be represented as finite JSON numbers"
	}
}
