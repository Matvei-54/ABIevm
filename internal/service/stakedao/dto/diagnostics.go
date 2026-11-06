package dto

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Optional diagnostics never change the compact pool schema or calculated APR.
type StakeDAOPoolDiagnostics struct {
	Primary     *PrimaryRateDiagnostic `json:"primary,omitempty"`
	Convex      *ConvexRateDiagnostic  `json:"convex,omitempty"`
	Prices      []PriceDiagnostic      `json:"prices,omitempty"`
	ReaderModel *ReaderModelDiagnostic `json:"readerModel,omitempty"`
}

// A public-reader reference and an observed Hub fee identity, not a claim that
// the private Hub implementation or a live UI snapshot was obtained.
type ReaderModelDiagnostic struct {
	Status                        string   `json:"status"`
	Source                        string   `json:"source"`
	Scope                         string   `json:"scope"`
	Message                       string   `json:"message,omitempty"`
	GrossMinAPR                   *float64 `json:"grossMinApr"`
	WeightedBoost                 *float64 `json:"weightedBoost"`
	ActualCRVAPR                  *float64 `json:"actualCrvApr"`
	WithoutHarvestFeeAPR          *float64 `json:"withoutHarvestFeeApr"`
	GlobalProtocolFeeAPR          *float64 `json:"globalProtocolFeeApr"`
	ReferenceAPR                  *float64 `json:"referenceApr"`
	DifferencePercentagePoints    *float64 `json:"differencePercentagePoints"`
	OwnSidecarCRVRate             *string  `json:"ownSidecarCrvRate"`
	GlobalPoolCRVRate             *string  `json:"globalPoolCrvRate"`
	CappedReferenceSidecarCRVRate *string  `json:"cappedReferenceSidecarCrvRate"`
}

type PrimaryRateDiagnostic struct {
	Unit                string                `json:"unit"`
	Locker              common.Address        `json:"locker" swaggertype:"string"`
	LockerInputs        *LockerRateDiagnostic `json:"lockerInputs,omitempty"`
	LockerGrossRate     string                `json:"lockerGrossRate"`
	ConvexGrossRate     string                `json:"convexGrossRate"`
	ProtocolFeeFraction string                `json:"protocolFeeFraction"`
	HarvestFeeFraction  string                `json:"harvestFeeFraction"`
	ProtocolFeeRate     string                `json:"protocolFeeRate"`
	HarvestFeeRate      string                `json:"harvestFeeRate"`
	NetRate             string                `json:"netRate"`
	Sidecars            []ConvexPlacement     `json:"sidecars"`
}

type LockerRateDiagnostic struct {
	Killed            bool   `json:"killed"`
	InflationRaw      string `json:"inflationRaw,omitempty"`
	RelativeWeightRaw string `json:"relativeWeightRaw,omitempty"`
	WorkingBalanceRaw string `json:"workingBalanceRaw,omitempty"`
	WorkingSupplyRaw  string `json:"workingSupplyRaw,omitempty"`
}

type ConvexPlacement struct {
	Address    common.Address `json:"address" swaggertype:"string"`
	PID        string         `json:"pid"`
	RewardPool common.Address `json:"rewardPool" swaggertype:"string"`
}

type ConvexRateDiagnostic struct {
	Utility            common.Address `json:"utility" swaggertype:"string"`
	PID                string         `json:"pid"`
	Status             string         `json:"status"`
	Message            string         `json:"message,omitempty"`
	RewardPool         common.Address `json:"rewardPool" swaggertype:"string"`
	RewardRateRaw      *string        `json:"rewardRateRaw"`
	StakedSupplyRaw    *string        `json:"stakedSupplyRaw"`
	PeriodFinish       *uint64        `json:"periodFinish"`
	ExpectedCrvRateRaw *string        `json:"expectedCrvRateRaw"`
	HelperCrvRateRaw   *string        `json:"helperCrvRateRaw"`
	// This helper rate is raw reward units per 1e18 raw LP units per second.
	Slots                []ConvexRewardSlot `json:"slots,omitempty"`
	OwnSidecarRate       *string            `json:"ownSidecarRate"`
	HelperOwnSidecarRate *string            `json:"helperOwnSidecarRate"`
	RoundingDifference   *string            `json:"roundingDifference"`
	SidecarBalanceRaw    *string            `json:"sidecarBalanceRaw"`
}

type ConvexRewardSlot struct {
	Token   common.Address `json:"token" swaggertype:"string"`
	Source  string         `json:"source"`
	RateRaw string         `json:"rateRaw"`
}

type PriceDiagnostic struct {
	Token          common.Address   `json:"token" swaggertype:"string"`
	USD            string           `json:"usd"`
	Source         string           `json:"source"`
	SourceContract common.Address   `json:"sourceContract" swaggertype:"string"`
	BlockNumber    uint64           `json:"blockNumber"`
	UpdatedAt      time.Time        `json:"updatedAt"`
	ValidUntil     time.Time        `json:"validUntil"`
	Components     []common.Address `json:"components,omitempty" swaggertype:"array,string"`
}
