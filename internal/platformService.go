package internal

import (
	"ABIevm/internal/dto"
	"ABIevm/internal/rpcclient"
	"ABIevm/internal/util"
	"context"
	"fmt"
	"math/big"

	"ABIevm/pkg/uipool/aave"
	//"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type PlatformService struct {
	// Client *ethclient.Client
}

// AAVE логика
func NewPlatformService() *PlatformService {
	return &PlatformService{}
}

func (e *PlatformService) CallContract(to common.Address, rpc string) ([]dto.Pool, error) {

	blockchainClient, err := rpcclient.DialContext(context.Background(), rpc)

	if err != nil {
		return nil, err
	}
	defer blockchainClient.Close()

	provider, err := aave.NewUipool(to, blockchainClient)
	if err != nil {
		return nil, err
	}

	var reserves []aave.IUiPoolDataProviderV3AggregatedReserveData

	reserves, _, err = provider.GetReservesData(
		&bind.CallOpts{Context: context.Background()},
		common.HexToAddress("0x2f39d218133AFaB8F2B819B1066c7E434Ad94E9e"),
	)

	if err != nil {
		return nil, err
	}

	poolsSlice := make([]dto.Pool, 0, 100)

	//result := NewReservesResponse(reserves, base)

	//for _, value := range reserves {
	//	fmt.Println(" - ", value.Symbol, convertToAPY(value.LiquidityRate).Text('f', 6), "%",
	//		"availableLiquidity:", normalize(value.AvailableLiquidity, value.Decimals).Text('f', 6))
	//
	//	poolsSlice = append(poolsSlice, *dto.NewPool(value.Symbol, convertToAPY(value.LiquidityRate)))
	//}

	//filterSlice := stream.OfSlice(reserves).Filter(func(p uipool.IUiPoolDataProviderV3AggregatedReserveData) bool {
	//	return p.LiquidityRate.Cmp(big.NewInt(5)) > 0
	//}).ToSlice()

	newReserves := util.FilterPools(reserves, func(p aave.IUiPoolDataProviderV3AggregatedReserveData) bool {
		return convertToAPY(p.LiquidityRate).Cmp(big.NewFloat(3)) > 0
	})

	for _, value := range newReserves {

		fmt.Println(" - ", value.Symbol, convertToAPY(value.LiquidityRate).Text('f', 6), "%",
			"availableLiquidity:", normalize(value.AvailableLiquidity, value.Decimals).Text('f', 6))
	}

	//for _, value := range filterSlice {
	//	fmt.Println(" - ", value.Symbol, convertToAPY(value.LiquidityRate).Text('f', 6), "%",
	//		"availableLiquidity:", normalize(value.AvailableLiquidity, value.Decimals).Text('f', 6))
	//}

	//msg := ethereum.CallMsg{
	//	To:   &to,
	//	Data: data,
	//}
	//fmt.Println(" - PlatformService CallContract, msg: ", msg)
	//return blockchainClient.CallContract(context.Background(), msg, nil)

	return poolsSlice, nil
}

type ReservesResponse struct {
	Reserves []aave.IUiPoolDataProviderV3AggregatedReserveData
	Base     aave.IUiPoolDataProviderV3BaseCurrencyInfo
}

func NewReservesResponse(reserves []aave.IUiPoolDataProviderV3AggregatedReserveData,
	base aave.IUiPoolDataProviderV3BaseCurrencyInfo) *ReservesResponse {
	return &ReservesResponse{
		Reserves: reserves,
		Base:     base,
	}
}

func convertToAPY(initial *big.Int) *big.Float {
	//resultPow := new(big.Int)
	//resultPow.Exp(big.NewInt(10), big.NewInt(18), nil) //вынести в константу

	//return new(big.Int).Div(initial, resultPow) //вынести в константу

	//var ten18 = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) //const
	apy := new(big.Float).Quo(new(big.Float).SetInt(initial), new(big.Float).SetFloat64(1e27))
	return new(big.Float).Mul(apy, big.NewFloat(100))
}

func normalize(amount *big.Int, decimals *big.Int) *big.Float {
	base := new(big.Int).Exp(big.NewInt(10), decimals, nil)
	return new(big.Float).Quo(
		new(big.Float).SetInt(amount),
		new(big.Float).SetInt(base),
	)
}
