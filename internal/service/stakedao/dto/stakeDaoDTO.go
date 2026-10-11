package dto

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type StakeDAOPool struct {
	GaugeAddress common.Address `json:"gaugeAddress"`
	VaultAddress common.Address `json:"vaultAddress"`
	AssetAddress common.Address `json:"assetAddress"`

	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	AssetSymbol string `json:"assetSymbol"`

	TVL float64 `json:"tvl"`
	APY float64 `json:"apy"`

	RewardToken common.Address `json:"rewardToken"`
	//	ChainID     int64          `json:"chainId"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type StakeDAOConfig struct {
	RPCURL             string
	ChainID            int64
	ProtocolController common.Address
	SeedGauges         []common.Address

	// Для расчёта APY нужен основной reward token по gauge.
	// Так как StakeDAO controller не даёт enumerate/discovery reward token-ов,
	// это задаётся в конфиге приложения.
	MainRewardTokenByGauge map[common.Address]common.Address

	// Цена reward token в USD/ETH/etc — сюда подставь ваш общий ценовой сервис.
	// StakeDAO.go только вызывает этот интерфейс.
	PriceOracle TokenPriceOracle
}

type TokenPriceOracle interface {
	// Возвращает цену 1 токена в "денежной" единице приложения, например USD.
	// Значение должно быть уже нормализовано, например 2.35 = $2.35
	GetTokenPrice(ctx context.Context, token common.Address, chainID int64) (float64, error)
}
