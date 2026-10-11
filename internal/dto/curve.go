package dto

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type CurveCoin struct {
	Address  common.Address
	Decimals uint8
	Balance  *big.Int
	BalanceF *big.Float
}

type CurvePool struct {
	Pool  common.Address
	Coins []CurveCoin
}

type DataCurvePool struct {
	Network      string
	PlatformName string
	Name         string
	//	apy          *big.Float
	Apr decimal.Decimal
	Tvl decimal.Decimal
}
