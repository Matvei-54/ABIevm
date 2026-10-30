package dto

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type StakeDAOPool struct {
	GaugeAddress   common.Address `json:"gaugeAddress" swaggertype:"string"`
	VaultAddress   common.Address `json:"vaultAddress" swaggertype:"string"`
	AssetAddress   common.Address `json:"assetAddress" swaggertype:"string"`
	RewardReceiver common.Address `json:"rewardReceiver" swaggertype:"string"`
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
	AccountantAddress *common.Address  `json:"accountantAddress" swaggertype:"string"`
	RewardToken       *common.Address  `json:"rewardToken" swaggertype:"string"`
	APRScope          string           `json:"aprScope"`
	Rewards           []StakeDAOReward `json:"rewards"`
	RewardsComplete   bool             `json:"-"`
	//	ChainID     int64          `json:"chainId"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type StakeDAOReward struct {
	Token         common.Address `json:"token" swaggertype:"string"`
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
	GaugeAddress common.Address  `json:"gaugeAddress" swaggertype:"string"`
	Field        string          `json:"field"`
	Message      string          `json:"message"`
	Code         string          `json:"code,omitempty"`
	Stage        string          `json:"stage,omitempty"`
	Contract     *common.Address `json:"contract,omitempty" swaggertype:"string"`
	Method       string          `json:"method,omitempty"`
}

// The public list keeps calculation internals in the optional detailed view.
// Address is the StakeDAO vault; Status describes APR completeness, not activity.
type StakeDAOPoolSummary struct {
	Name    string         `json:"name"`
	APR     *float64       `json:"apr"`
	Symbol  string         `json:"symbol"`
	Address common.Address `json:"address" swaggertype:"string"`
	APY     *float64       `json:"apy"`
	Status  string         `json:"status"`
}

type StakeDAOPoolsSummaryResponse struct {
	Network     string                `json:"network"`
	BlockNumber uint64                `json:"blockNumber"`
	Pools       []StakeDAOPoolSummary `json:"pools"`
}

func (r *StakeDAOPoolsResponse) Summary() StakeDAOPoolsSummaryResponse {
	result := StakeDAOPoolsSummaryResponse{Network: r.Network, BlockNumber: r.BlockNumber, Pools: make([]StakeDAOPoolSummary, 0, len(r.Pools))}
	for _, pool := range r.Pools {
		result.Pools = append(result.Pools, StakeDAOPoolSummary{
			Name: pool.Name, APR: pool.APR, Symbol: pool.Symbol,
			Address: pool.VaultAddress, APY: pool.APY, Status: pool.APRStatus,
		})
	}
	return result
}
