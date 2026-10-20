package internal

import (
	_ "ABIevm/docs"
	"ABIevm/internal/service/prices"
	"ABIevm/internal/service/stakedao"

	//"ABIevm/internal/dto"
	"ABIevm/internal/service"
	"context"
	"errors"
	//"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	//"strings"

	//decoder "ABIevm/internal/util"

	//"github.com/ethereum/go-ethereum/accounts/abi"

	//"github.com/ethereum/go-ethereum"
	//"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	evmService     *PlatformService
	abiLoadFetcher *AbiLoadFetcher
	curveService   *internal.CurveService
	stakeDao       *stakedao.StakeDAOService
}

func NewHandler() *Handler {
	priceStore := prices.NewMemoryStore()
	tokenPrices := prices.NewTokenPriceService(priceStore, prices.NewContractSource(priceStore))
	return &Handler{
		evmService:     NewPlatformService(),
		abiLoadFetcher: NewAbiLoadFetcher(),
		curveService:   internal.NewCurveService(),
		stakeDao:       stakedao.NewStakeDAOService(tokenPrices),
	}
}

//type evmClientAdapter struct {
//	evm *evm.EVMClient
//}
//
//func (a *evmClientAdapter) Call(to common.Address, data []byte) ([]byte, error) {
//	return a.evm.CallContract(to, data)
//}

// @Summary      Получить данные всех резервов Aave V3
// @Description  Возвращает полный список резервов с параметрами (LTV, ликвидность, цены и т.д.)
// @Tags         Aave V3
// @Accept       json
// @Produce      json
// @Param        network  path      string  true  "Название сети: ethereum | arbitrum | polygon | optimism | base"
// @Success      200      {object}  interface{}
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /reservesAave/{network} [get]
func (h *Handler) HandleGetAaveReserves(c echo.Context) error {
	provider := c.Param("network")

	if provider == "" {
		log.Println("provider is empty")
	}

	network := Networks[provider]

	// build calldata and call

	raw, err := h.evmService.CallContract(common.HexToAddress(network.contractUIProvider), network.rpcUrl)
	if err != nil {
		log.Println(" - evm call CallContract ", err.Error())
	}

	fmt.Println(" - after evmService.CallContract")

	//poolsSlice := make([]dto.Pool, 0, 100)

	//fmt.Println(" - stepen const: ", new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	//fmt.Println(" - after Unpack GetReservesData")

	return c.JSON(http.StatusOK, raw)
}

// @Summary      Получить данные всех резервов Curve V3
// @Description  Возвращает полный список резервов с параметрами (LTV, ликвидность, цены и т.д.)
// @Tags         Curve V3
// @Accept       json
// @Produce      json
// @Param        network  path      string  true  "Название сети: ethereum | arbitrum | polygon | optimism | base"
// @Success      200      {object}  interface{}
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /reservesCurve/{network} [get]
func (h *Handler) HandleGetCurveReserves(c echo.Context) error {
	paramNetwork := c.Param("network")

	if paramNetwork == "" {
		log.Println("provider is empty")
	}
	fmt.Println(" - from HandleGetCurveReserves, before PullCurvePools ")

	h.curveService.PullCurvePools()

	//addressProviderCurve := "0x5ffe7FB82894076ECB99A30D6A32e969e6e35E98"

	//StableSwapRegistry := "0xF98B45FA17DE75FB1aD0e7aFD971b0ca00e379fC"
	//Factory := "0x8F942C20D02bEfc377D41445793068908E2250D0"
	//metaRegistry := "0xF98B45FA17DE75FB1aD0e7aFD971b0ca00e379fC"

	//blockchainClient, err := ethclient.Dial(network.rpcUrl)
	//
	//provider, err := curve.NewUipool(common.HexToAddress(metaRegistry), blockchainClient)
	//if err != nil {
	//	log.Println("Curve NewPool ", err.Error())
	//	return c.JSON(http.StatusInternalServerError, err.Error())
	//}

	//DAI_USDC_USDT := "0xbEbc44782C7dB0a1A60Cb6fe97d0b483032FF1C7"
	//
	//coins, err := provider.GetCoins(&bind.CallOpts{Context: context.Background()}, common.HexToAddress(DAI_USDC_USDT))
	//if err != nil {
	//	log.Println("Curve NewPool ", err.Error())
	//	return err
	//}
	//
	//decimals, err := provider.GetDecimals(&bind.CallOpts{Context: context.Background()}, common.HexToAddress(DAI_USDC_USDT))
	//if err != nil {
	//	log.Println("Curve NewPool ", err.Error())
	//	return err
	//}
	//
	//balances, err := provider.GetBalances(&bind.CallOpts{Context: context.Background()}, common.HexToAddress(DAI_USDC_USDT))
	//if err != nil {
	//	log.Println("Curve NewPool ", err.Error())
	//	return err
	//}
	//if len(coins) != len(decimals) || len(coins) != len(balances) {
	//	log.Println("Curve NewPool ", err.Error())
	//	return errors.New("curve registry invariant violated")
	//}

	//pools := make([]dto.CurveCoin, len(coins))

	//for i := 0; i < len(coins); i++ {
	//	if coins[i] == (common.Address{}) {
	//		continue
	//	}
	//
	//	balanceF := normalized(balances[i], decimals[i])
	//
	//	pools = append(pools, dto.CurveCoin{
	//		Address:  coins[i],
	//		Decimals: decimals[i],
	//		Balance:  balances[i],
	//		BalanceF: balanceF,
	//	})
	//
	//}

	//msg := ethereum.CallMsg{
	//	To:   &to,
	//	Data: data,
	//}

	//result, err := blockchainClient.CallContract(context.Background(), msg, nil)

	//raw, err := h.evmService.CallContract(common.HexToAddress(addressProviderCurve), network.rpcUrl)
	//if err != nil {
	//	log.Println(" - evm call CallContract ", err.Error())
	//}

	fmt.Println(" - after Curve CallContract")

	return c.JSON(http.StatusOK, nil)
}

// @Summary      Получить пулы StakeDAO из блокчейна
// @Description  Возвращает TVL и оценку APR наград для Curve V2; aprStatus показывает полноту. APY пока недоступен.
// @Tags         StakeDao
// @Accept       json
// @Produce      json
// @Param        network  path      string  true  "Поддерживаемая сеть: ethereum"
// @Success      200      {object}  interface{}
// @Failure      400      {object}  map[string]string
// @Failure      502      {object}  map[string]string
// @Failure      504      {object}  map[string]string
// @Router       /reservesStakedao/{network} [get]
func (h *Handler) HandleGetStakeDaoReserves(c echo.Context) error {
	result, err := h.stakeDao.GetPools(c.Request().Context(), c.Param("network"))
	if err != nil {
		switch {
		case errors.Is(err, stakedao.ErrUnsupportedNetwork):
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "StakeDAO currently supports only ethereum"})
		case errors.Is(err, context.DeadlineExceeded):
			return c.JSON(http.StatusGatewayTimeout, map[string]string{"error": "StakeDAO RPC request timed out"})
		default:
			// RPC errors may contain the endpoint's API key; never expose them in HTTP responses.
			return c.JSON(http.StatusBadGateway, map[string]string{"error": "StakeDAO blockchain data could not be loaded"})
		}
	}
	return c.JSON(http.StatusOK, result)
}

func normalized(balance *big.Int, decimal *big.Int) *big.Float {
	base := new(big.Int).Exp(big.NewInt(10), decimal, nil)
	return new(big.Float).Quo(
		new(big.Float).SetInt(balance),
		new(big.Float).SetInt(base),
	)
}
