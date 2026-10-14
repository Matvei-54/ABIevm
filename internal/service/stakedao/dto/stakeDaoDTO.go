package dto

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type StakeDAOPool struct {
	GaugeAddress   common.Address `json:"gaugeAddress"`
	VaultAddress   common.Address `json:"vaultAddress"`
	AssetAddress   common.Address `json:"assetAddress"`
	RewardReceiver common.Address `json:"rewardReceiver"`
	ProtocolID     string         `json:"protocolId"`

	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	AssetSymbol string `json:"assetSymbol"`

	// AssetAmount is an exact decimal quantity of the underlying token, not USD TVL.
	AssetAmount   *string  `json:"assetAmount"`
	AssetDecimals *uint8   `json:"assetDecimals"`
	TVL           *float64 `json:"tvl"`
	APR           *float64 `json:"apr"`
	APRStatus     string   `json:"aprStatus"`
	APY           *float64 `json:"apy"`

	RewardMechanism   string           `json:"rewardMechanism"`
	AccountantAddress *common.Address  `json:"accountantAddress"`
	RewardToken       *common.Address  `json:"rewardToken"`
	APRScope          string           `json:"aprScope"`
	Rewards           []StakeDAOReward `json:"rewards"`
	RewardsComplete   bool             `json:"-"`
	//	ChainID     int64          `json:"chainId"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type StakeDAOReward struct {
	Token         common.Address `json:"token"`
	Source        string         `json:"source"`
	RatePerSecond *string        `json:"ratePerSecond"`
	PeriodFinish  *uint64        `json:"periodFinish"`
	APR           *float64       `json:"apr"`
}

type StakeDAOPoolsResponse struct {
	Network     string              `json:"network"`
	BlockNumber uint64              `json:"blockNumber"`
	Pools       []StakeDAOPool      `json:"pools"`
	Errors      []StakeDAOPoolError `json:"errors,omitempty"`
}

type StakeDAOPoolError struct {
	GaugeAddress common.Address `json:"gaugeAddress"`
	Field        string         `json:"field"`
	Message      string         `json:"message"`
}
