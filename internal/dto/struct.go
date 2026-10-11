package dto

import (
	"fmt"
	"math/big"
)

type Pool struct {
	name        string
	yearPercent *big.Float
}

func NewPool(name string, yearPercent *big.Float) *Pool {
	return &Pool{
		name:        name,
		yearPercent: yearPercent,
	}
}

func (p *Pool) String() string {
	return fmt.Sprintf("Pool {%s, %s}", p.name, p.yearPercent)
}

//type ReservesResponse struct {
//	Reserves []uipool.IUiPoolDataProviderV3AggregatedReserveData
//	Base     uipool.IUiPoolDataProviderV3BaseCurrencyInfo
//}
//
//func NewReservesResponse(reserves []uipool.IUiPoolDataProviderV3AggregatedReserveData,
//	base uipool.IUiPoolDataProviderV3BaseCurrencyInfo) *ReservesResponse {
//	return &ReservesResponse{
//		Reserves: reserves,
//		Base:     base,
//	}
//}

type DataPool struct {
	network      string
	platformName string
	name         string
	apy          *big.Float
	apr          *big.Float
	tvl          *big.Int
}

func NewDataPool(network, platformName, name string, apy, apr *big.Float) *DataPool {
	return &DataPool{
		network:      network,
		platformName: platformName,
		name:         name,
		apy:          apy,
		apr:          apr,
	}
}
