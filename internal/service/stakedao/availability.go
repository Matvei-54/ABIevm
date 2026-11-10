package stakedao

import (
	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/shopspring/decimal"
)

// APR completeness is independent of pool demand or activity. Reasons describe
// calculation prerequisites; detailed field errors retain the actual failures.
func setAPRReasons(pool *dto.StakeDAOPool, issues []dto.StakeDAOPoolError) {
	pool.APRReasons = nil
	if pool.APRStatus == "available" {
		return
	}
	amountKnown := false
	if pool.AssetAmount != nil {
		amount, err := decimal.NewFromString(*pool.AssetAmount)
		amountKnown = err == nil && !amount.IsNegative()
		if amountKnown && amount.IsZero() {
			pool.APRReasons = []string{"empty_vault"}
			return
		}
	}
	if !amountKnown {
		pool.APRReasons = append(pool.APRReasons, "asset_data_missing")
	} else if pool.TVL == nil {
		pool.APRReasons = append(pool.APRReasons, "asset_price_missing")
	}
	unsupported := pool.RewardMechanism == "unsupported"
	rewardPriceMissing := false
	for _, issue := range issues {
		if issue.GaugeAddress != pool.GaugeAddress {
			continue
		}
		switch issue.Code {
		case "unsupported_model", "primary_reward_model", "primary_reward_allocation_model":
			unsupported = true
		case "reward_price_missing":
			rewardPriceMissing = true
		}
	}
	if unsupported {
		pool.APRReasons = append(pool.APRReasons, "unsupported_model")
	} else if pool.Allocation != nil && pool.Allocation.Status == "deficit" {
		pool.APRReasons = append(pool.APRReasons, "accounting_mismatch")
	} else if !pool.RewardsComplete {
		pool.APRReasons = append(pool.APRReasons, "reward_data_missing")
	}
	if rewardPriceMissing {
		pool.APRReasons = append(pool.APRReasons, "reward_price_missing")
	}
	if len(pool.APRReasons) == 0 {
		pool.APRReasons = []string{"reward_data_missing"}
	}
}
