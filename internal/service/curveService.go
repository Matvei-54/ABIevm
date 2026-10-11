package internal

import (
	"ABIevm/internal/dto"
	curve "ABIevm/pkg/uipool/curve"
	crv "ABIevm/pkg/uipool/curve/registry"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	CurveAddressProvider   = "0x0000000022D53366457F9d5E68Ec105046FC4383"
	ETH_RPC                = "https://eth-mainnet.g.alchemy.com/v2/---"
	ETH_RPC_SECOND         = "https://eth.llamarpc.com"
	GaugeControllerAddress = "0x2F50D538606Fa9EDD2B11E2446BEb18C9D5846bB"
	MulticallCurve         = "0xcA11bde05977b3631167028862bE2a173976CA11"
	SECOND_PER_YEAR        = 31556925
)

var factoryStartBlocks = map[common.Address]uint64{
	common.HexToAddress("0x90E00ACe148ca3b23Ac1bC8C240C2a7Dd9c2d7f5"): 10647875,
	common.HexToAddress("0x8F942C20D02bEfc377D41445793068908E2250D0"): 13986752,
	common.HexToAddress("0xF98B45FA17DE75FB1aD0e7aFD971b0ca00e379fC"): 13986752,
}

var MulticallAddresses = map[string]string{
	"ethereum": "0xcA11bde05977b3631167028862bE2a173976CA11",
	"arbitrum": "0xcA11bde05977b3631167028862bE2a173976CA11",
	"polygon":  "0xcA11bde05977b3631167028862bE2a173976CA11",
	"optimism": "0xcA11bde05977b3631167028862bE2a173976CA11",
}

var PriceCache sync.Map

var GlobalCurvePoolMap sync.Map

//var CurvePoolFeeMap sync.Map

//var CurveDayFeeMap map[string]float64

var CurveDay24HFeeMap map[string]float64

var CurvePoolStaticMeta = map[string]*PoolStaticMeta{}

var CurvePoolDynamicMeta sync.Map

//var CurvePoolsMap sync.Map

var BlockchainClientS *ethclient.Client

var RegistryPoolMap = map[common.Address]common.Address{}

var TokenSymbolCache sync.Map

type CurveService struct {
}

func NewCurveService() *CurveService {
	return &CurveService{}
}

func (c CurveService) PullCurvePools() {

	//clear(CurvePoolStaticMeta)
	GlobalCurvePoolMap = sync.Map{}
	CurvePoolDynamicMeta = sync.Map{}
	clear(CurveDay24HFeeMap)

	fmt.Println(" - from PullCurvePools, before all")
	timeBegin := time.Now()

	client, err := ethclient.Dial(ETH_RPC)
	if err != nil {
		log.Fatal("Failed to connect to Ethereum client")
	}

	BlockchainClientS = client

	defer BlockchainClientS.Close()

	opts := &bind.CallOpts{Context: context.Background()}

	ap, err := curve.NewAddressProvider(common.HexToAddress(CurveAddressProvider), BlockchainClientS)
	if err != nil {
		log.Println("Curve NewPool ", err.Error())
		//return nil, err
	}

	gaugeController, err := crv.NewGaugeController(common.HexToAddress(GaugeControllerAddress), BlockchainClientS)
	if err != nil {
		log.Println("Curve Gauge Controller failed ", err.Error())
	}

	crvPrice := getCRVPrice(BlockchainClientS)

	stableSwapReg, _ := ap.GetAddress(opts, big.NewInt(0))
	factoryReg, _ := ap.GetAddress(opts, big.NewInt(5))
	cryptoReg, _ := ap.GetAddress(opts, big.NewInt(7))

	registryAddrs := []common.Address{
		stableSwapReg,
		factoryReg,
		cryptoReg,
	}

	allPools := make([]common.Address, 0, 1000)

	for _, addr := range registryAddrs {
		if (addr == common.Address{}) {
			continue
		}

		registry, err := crv.NewUipool(addr, BlockchainClientS)
		if err != nil {
			log.Println("Error NewUipool ", err.Error())
			return
		}
		poolsFromReg, _ := getCountPoolAndList(registry)
		allPools = append(allPools, poolsFromReg...)

		for _, v := range poolsFromReg {
			RegistryPoolMap[v] = addr
		}
	}

	//allPools, err = loadSliceAllPools(registryAddrs)
	//if err != nil {
	//	log.Println(" -- loadSliceAllPools ", err.Error())
	//	return
	//}

	log.Println(" -- After allPools")

	if len(allPools) == 0 {
		log.Println(" -- All pools is empty")
		return
	}

	errf := filterPoolAndGetMetaDataPool(allPools)
	if errf != nil {
		log.Println(" -- Error filterPoolAndGetMetaDataPool log - ", errf.Error())
		return
	}

	log.Println(" -- allPools count:", len(allPools))
	log.Println(" -- CurvePoolStaticMeta count:", len(CurvePoolStaticMeta))

	//registryABI := LoadRegistryABI()
	//resultMulticall, err := multicallBatchGetPoolData(allPools, registryABI)

	tokens := collectTokens(CurvePoolStaticMeta)
	if errLTP := loadTokenPrices(tokens); errLTP != nil {
		log.Println(" -- Error loadTokenPrices", err.Error())
		return
	}
	//getTokenPriceUSD(tokens, nil)
	//loadTokenPrices(tokens)

	errLV24 := load24hVolumeUSD(CurvePoolStaticMeta)
	if errLV24 != nil {
		log.Println(" -- Error load24hVolumeUSD log", errLV24.Error())
		return
	}

	res, err := addPoolToResultSlice(CurvePoolStaticMeta, gaugeController, crvPrice)
	if err != nil {
		log.Println(" --Error addPoolToResultSlice ", err.Error())
		return
	}

	log.Println(" -- After addPoolToResultSlice")

	if len(res) == 0 {
		log.Println(" -- There is no pool available")
	}

	for _, res := range res {
		fmt.Printf(
			"POOL: %-30s | APY: %8s%% | TVL: %12s$\n",
			res.Name,
			res.Apr.StringFixed(2),
			res.Tvl.StringFixed(0),
		)
	}

	fmt.Println(time.Now().Sub(timeBegin))

	fmt.Println(" - after Curve CallContract in CurveService")
}

func filterPoolAndGetMetaDataPool(pools []common.Address) error {
	opts := &bind.CallOpts{Context: context.Background()}
	var skipped int

	for _, pool := range pools {
		if _, exists := GlobalCurvePoolMap.Load(pool); exists {
			continue
		}

		registry, err := crv.NewUipool(RegistryPoolMap[pool], BlockchainClientS)
		if err != nil {
			log.Println("Error NewUipool ", err.Error())
		}

		gauge, err := registry.GetGauge(opts, pool)
		if err != nil || gauge == (common.Address{}) {
			skipped++
			continue
		}

		gaugeContract, err := crv.NewLiquidityGauge(gauge, BlockchainClientS)
		if err != nil {
			skipped++
			return err
		}

		killed, err := gaugeContract.IsKilled(opts)
		if err == nil && killed {
			skipped++
			continue
		}

		coins, err := registry.GetCoins(opts, pool)
		if err != nil {
			//пока написал continue, но можно сделать log с адресом и чекнуть функцию через обозреватель
			//return err
			skipped++
			continue
		}

		decimals, err := registry.GetDecimals(opts, pool)
		if err != nil {
			skipped++
			//return err
			continue
		}

		balances, err := registry.GetBalances(opts, pool)
		if err != nil {
			skipped++
			//return err
			continue
		}

		name, err := registry.GetPoolName(opts, pool)
		if err != nil || name == "" {
			skipped++
			//return err
			continue
		}

		lp, err := registry.GetLpToken(opts, pool)
		if err != nil {
			//return err
			skipped++
			continue
		}

		virtualPrice, err := registry.GetVirtualPriceFromLpToken(opts, lp)
		if err != nil {
			//return err
			skipped++
			continue
		}

		underlyingCoins, err := registry.GetUnderlyingCoins(opts, pool)
		if err != nil {
			skipped++
			continue
		}

		underlyingDecimals, err := registry.GetUnderlyingDecimals(opts, pool)
		if err != nil {
			skipped++
			continue
		}

		CurvePoolStaticMeta[strings.ToLower(pool.Hex())] = &PoolStaticMeta{
			Address:            pool,
			Gauge:              gauge,
			GaugeContract:      gaugeContract,
			Coins:              coins,
			Decimals:           decimals,
			UnderlyingCoins:    underlyingCoins,
			UnderlyingDecimals: underlyingDecimals,
			Name:               name,
			Registry:           registry,
		}

		CurvePoolDynamicMeta.Store(strings.ToLower(pool.Hex()), &PoolDynamicMeta{
			Balances:     balances,
			VirtualPrice: virtualPrice,
		})

		GlobalCurvePoolMap.Store(pool, struct{}{})
	}
	log.Println(" -- [filter] done. skipped:", skipped)
	return nil
}

func getCountPoolAndList(reg *crv.Uipool) ([]common.Address, error) {

	opts := &bind.CallOpts{Context: context.Background()}

	count, err := reg.PoolCount(opts)
	if err != nil {

		return nil, fmt.Errorf(" -- Pool Count error: %s", err.Error())
	}
	pools := make([]common.Address, 0, count.Int64())

	limit := count.Int64() // убрать limit, делал для тестов
	if limit > 150 {
		limit = 150
	}

	for i := int64(0); i < limit; i++ {
		addr, err := reg.PoolList(opts, big.NewInt(i))
		if err != nil {
			continue
		}
		pools = append(pools, addr) // узнали адреса все пулов из всех 3 регистров
	}
	return pools, nil
}

func addPoolToResultSlice(
	//poolsAddr []common.Address,
	poolsStaticMap map[string]*PoolStaticMeta, //	registry *crv.Uipool,
	gaugeController *crv.GaugeController,
	crvPrice float64) ([]dto.DataCurvePool, error) {

	opts := &bind.CallOpts{Context: context.Background()}

	var wg sync.WaitGroup

	semaphore := make(chan struct{}, 3)
	resultChan := make(chan dto.DataCurvePool, len(poolsStaticMap))
	//minePercentAPR := decimal.NewFromFloat(1.0)

	// мягкий global rate limit
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Println(" -- [DEBUG] addPoolToResultSlice input pools:", len(poolsStaticMap))

	for _, value := range poolsStaticMap {

		<-ticker.C

		semaphore <- struct{}{}
		wg.Add(1)

		go func(pool *PoolStaticMeta) {

			address := strings.ToLower(pool.Address.Hex())

			defer wg.Done()
			defer func() { <-semaphore }()

			inflationRate, err := value.GaugeContract.InflationRate(opts)
			if err != nil {
				log.Println(" -- Get Inflation Rate error: ", err.Error())
				return
			}

			workingSupply, err := value.GaugeContract.WorkingSupply(opts)
			if err != nil {
				log.Println(" -- Get Working Supply error: ", err.Error())
				return
			}

			gaugeRelativeWeight, err := gaugeController.GaugeRelativeWeight(opts, value.Gauge)
			if err != nil {
				log.Println(" -- Get Gauge Relative Weight error: ", err.Error())
				return
			}

			if gaugeRelativeWeight == nil {
				log.Println(" -- GaugeRelativeWeight is nil for: ", value.Address)
				return
			}

			dynamicRaw, ok := CurvePoolDynamicMeta.Load(address)
			if !ok {
				log.Println(" -- Get Dynamic Raw error: ", address)
				return
			}

			dynamicMeta := dynamicRaw.(*PoolDynamicMeta)

			tvlUSD := calcTVL(value.Coins[:], dynamicMeta.Balances[:], value.Decimals[:], value.Registry, value.Address)
			if tvlUSD <= 0 {
				log.Println(" -- Get TVL USD zero for: ", address)
			}

			//if strings.EqualFold(value.Address.Hex(), "0x52EA46506B9CC5Ef470C5bf89f17Dc28bB35D85C") {
			//	log.Println(" -- from addPoolToResultSlice equalFold - usdt")
			//	log.Println("USDT DEBUG tvlUSD:", tvlUSD)
			//}

			//fmt.Println(" -- Registry pool address:", address)
			feesAPR := getPoolFeeARP(pool.Address, tvlUSD, value.Registry) //награда от комиссий обменов

			if feesAPR > 1000 {
				log.Println("!!! ABNORMAL APR:", value.Address.Hex(), feesAPR)
			}

			//награда от платформы Curve в их токенах
			crvAprReward := calcGaugeAPR(inflationRate, gaugeRelativeWeight, crvPrice, workingSupply, dynamicMeta.VirtualPrice)
			//fmt.Println(" - addPoolToResultSlice poolname - ", value.Name)
			//fmt.Println(" - addPoolToResultSlice CRV APR:", crvAprReward)
			//fmt.Println(" - addPoolToResultSlice Fee APR:", feesAPR)
			totalArp := crvAprReward + feesAPR

			resultChan <- dto.DataCurvePool{
				Network:      "Ethereum",
				PlatformName: "Curve",
				Name:         buildPoolDisplayName(value),
				Apr:          decimal.NewFromFloat(totalArp).Round(4),
				Tvl:          decimal.NewFromFloat(tvlUSD),
			}
		}(value)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	pools := make([]dto.DataCurvePool, 0, len(resultChan))

	for p := range resultChan {

		//временно комментирую, потом вернуть. Фильтр по пула с низким процентом.
		//if p.Apr.LessThan(minePercentAPR) {
		//	continue
		//}
		//log.Println(" -- [DEBUG][RESULT]", p.Name, "APR:", p.Apr.String(), "TVL:", p.Tvl.String())
		pools = append(pools, p)
	}

	log.Println(" -- [DEBUG] addPoolToResultSlice output pools:", len(pools))
	return pools, nil
}

func calcGaugeAPR(inflationRate *big.Int, relativeWeight *big.Int, crvPrice float64,
	workingSupply *big.Int, virtualPrice *big.Int) float64 {

	if inflationRate == nil || relativeWeight == nil {
		return 0
	}

	secondsPerYear := float64(SECOND_PER_YEAR)

	// inflation_rate в 1e18 precision
	infl := new(big.Float).Quo(
		new(big.Float).SetInt(inflationRate),
		big.NewFloat(1e18),
	)

	weight := new(big.Float).Quo(
		new(big.Float).SetInt(relativeWeight),
		big.NewFloat(1e18),
	)

	//новое
	workSup := new(big.Float).Quo(
		new(big.Float).SetInt(workingSupply),
		big.NewFloat(1e18),
	)

	vPrice := new(big.Float).Quo(
		new(big.Float).SetInt(virtualPrice),
		big.NewFloat(1e18),
	)

	totalStakedUSD := new(big.Float).Mul(workSup, vPrice)

	if totalStakedUSD.Cmp(big.NewFloat(0)) == 0 {
		return 0
	}

	inflationFloat, _ := infl.Float64()
	weightFloat, _ := weight.Float64()
	totalStakedUSDFloat, _ := totalStakedUSD.Float64()

	crvPerYear := inflationFloat * weightFloat * secondsPerYear

	yearlyRewardsUSD := crvPerYear * crvPrice

	apr := yearlyRewardsUSD / totalStakedUSDFloat

	return apr * 100
}

func getCRVPrice(client bind.ContractBackend) float64 {

	feed, err := crv.NewChainlink(
		common.HexToAddress("0xCd627aA160A6Fa45EB793D19Ef54f5062F20f33f"),
		client,
	)
	if err != nil {
		return 0
	}

	opts := &bind.CallOpts{Context: context.Background()}

	roundData, err := feed.LatestRoundData(opts)
	if err != nil {
		return 0
	}

	decimals, _ := feed.Decimals(opts)

	price := new(big.Float).Quo(
		new(big.Float).SetInt(roundData.Answer),
		new(big.Float).SetFloat64(math.Pow10(int(decimals))),
	)

	p, _ := price.Float64()

	return p
}

func calcTVL(coins []common.Address, balances []*big.Int, decimals []*big.Int, reg *crv.Uipool, addr common.Address) float64 {
	opts := &bind.CallOpts{Context: context.Background()}

	tvl := 0.0
	for i := 0; i < len(coins); i++ {
		if (coins[i] == common.Address{}) {
			continue
		}

		if balances[i] == nil || decimals[i] == nil {
			continue
		}

		amountF := new(big.Float).Quo(new(big.Float).SetInt(balances[i]),
			new(big.Float).SetFloat64(math.Pow10(int(decimals[i].Int64()))))

		amount, _ := amountF.Float64()

		isBasePoolLPToken, err := reg.GetPoolFromLpToken(opts, coins[i])
		if err != nil {
			continue
		}
		if isBasePoolLPToken != (addr) {
			tvl += recursivePools(reg, coins[i], amount)
		}

		price := getTokenPriceUSD(coins[i])

		tvl += amount * price
	}
	return tvl
}

// bulk price
// переписать, чтобы цены запрашивались вместе сразу от всех токенов, а не по одному
func getTokenPriceUSD(
	//tokens []common.Address,
	token common.Address) float64 {

	if val, ok := PriceCache.Load(token); ok {
		return val.(float64)
	}

	urlRequest := "https://coins.llama.fi/prices/current/ethereum:" + token.Hex()
	resp, err := http.Get(urlRequest)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Coins map[string]struct {
			Price float64 `json:"price"`
		} `json:"coins"`
	}
	json.Unmarshal(body, &result)
	key := "ethereum:" + token.Hex()
	price := result.Coins[key].Price
	PriceCache.Store(token, price)

	return price
}

func getPoolFeeARP(poolAddress common.Address, tvlUSD float64, reg *crv.Uipool) float64 {

	addr := strings.ToLower(poolAddress.Hex())

	if tvlUSD == 0 {
		return 0
	}

	//fmt.Println(" ---getPoolFeeARP Searching volume for:", addr, tvlUSD)

	val, ok := CurveDay24HFeeMap[addr]

	if !ok {
		fmt.Println(" ---getPoolFeeARP Volume NOT FOUND for:", addr)
		return 0
	}
	opts := &bind.CallOpts{Context: context.Background()}

	fees, err := reg.GetFees(opts, poolAddress)
	if err != nil {
		return 0
	}

	feeBig := new(big.Float).Quo(new(big.Float).SetInt(fees[0]), big.NewFloat(1e10))
	feeFloat, _ := feeBig.Float64()

	if strings.EqualFold(poolAddress.Hex(), "0x0FaFaFD3C393ead5F5129cFC7e0E12367088c473") {
		log.Println("BAOUSD DEBUG volume24h:", val)
		log.Println("BAOUSD DEBUG tvlUSD:", tvlUSD)
		log.Println("BAOUSD DEBUG raw fees[0]:", fees[0].String())
		log.Println("BAOUSD DEBUG feeFloat:", feeFloat)
		log.Println("BAOUSD DEBUG yearlyFees:", val*feeFloat*365)
		log.Println("BAOUSD DEBUG APR calc:", (val*feeFloat*365/tvlUSD)*100)

		feeBig := new(big.Float).Quo(new(big.Float).SetInt(fees[0]), big.NewFloat(1e10))
		feeFloatDebug, _ := feeBig.Float64()

		log.Println("feeFloat:", feeFloatDebug)

		yearlyFees := val * feeFloatDebug * 365
		log.Println("yearlyFees:", yearlyFees)

		if tvlUSD > 0 {
			log.Println("APR calc:", (yearlyFees/tvlUSD)*100)
		}

		log.Println("========================")

	}

	yearlyFees := val * feeFloat * 365

	return (yearlyFees / tvlUSD) * 100
}

// func loadDayFeePoolAPI() error { //заменить на load24hVolumeUSD(get24hVolumeUSD)
//
//		client := &http.Client{
//			Timeout: 15 * time.Second,
//		}
//
//		sliceHttpRequest := []string{
//			"https://api.curve.fi/api/getPools/ethereum/main",
//			"https://api.curve.fi/api/getPools/ethereum/factory",
//			"https://api.curve.fi/api/getPools/ethereum/crypto",
//		}
//		type curveResponse struct {
//			Data struct {
//				PoolData []struct {
//					Address   string  `json:"address"`
//					VolumeUSD float64 `json:"volumeUSD"`
//				} `json:"poolData"`
//			} `json:"data"`
//		}
//
//		combined := make(map[string]float64)
//
//		for _, url := range sliceHttpRequest {
//			resp, err := client.Get(url)
//			if err != nil {
//				continue
//			}
//			body, err := io.ReadAll(resp.Body)
//			resp.Body.Close()
//			if err != nil {
//				continue
//			}
//			var resultResponse curveResponse
//
//			if err := json.Unmarshal(body, &resultResponse); err != nil {
//				log.Println("Curve API unmarshal error:", url, err)
//				continue
//			}
//			for _, v := range resultResponse.Data.PoolData {
//				//fmt.Println(" -- API pool from loadDayFeePoolAPI:", v.Address)
//				addr := strings.ToUpper(v.Address)
//
//				// если вдруг пул повторяется — суммируем volume
//				if existing, ok := combined[addr]; ok {
//					combined[addr] = existing + v.VolumeUSD
//				} else {
//					combined[addr] = v.VolumeUSD
//				}
//			}
//		}
//
//		CurveDayFeeMap = combined
//
//		fmt.Println(" -- Load CurveDayFeeMap done")
//		return nil
//	}
//

// За 24 часа Ищет Swap события пула, складывает объём торгов, кладет volume в USD.
// Замена api.curve.fi/api/getPools/ethereum/main. Нужно для расчета Fees APR
func load24hVolumeUSD(poolsMap map[string]*PoolStaticMeta) error {

	const blockStep uint64 = 10
	const workers = 2

	parsedSwapABI := loadCurveSwapABI()

	// итоговая мапа volume
	combined := make(map[string]float64, len(poolsMap))

	addresses := make([]common.Address, 0, len(poolsMap))
	for addr := range poolsMap {
		addresses = append(addresses, common.HexToAddress(addr))
	}

	ctx := context.Background()
	latestBlock, err := BlockchainClientS.BlockNumber(ctx)
	if err != nil {
		return err
	}

	fromBlock, err := findBlock24hAgo()
	if err != nil {
		return err
	}

	tokenExchangeEventSig := crypto.Keccak256Hash(
		[]byte("TokenExchange(address,int128,uint256,int128,uint256)"),
	)

	tokenExchangeUnderlyingEventSig := crypto.Keccak256Hash(
		[]byte("TokenExchangeUnderlying(address,int128,uint256,int128,uint256)"),
	)

	type blockRange struct {
		start uint64
		end   uint64
	}

	ranges := make([]blockRange, 0)
	for start := fromBlock; start <= latestBlock; start += blockStep {
		end := start + blockStep - 1
		if end > latestBlock {
			end = latestBlock
		}
		ranges = append(ranges, blockRange{start: start, end: end})
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	rangeChan := make(chan blockRange, len(ranges))
	errChan := make(chan error, 1)

	totalSkippedMeta := 0
	totalSkippedByte128 := 0
	totalSkippedSanity := 0
	totalSkippedBadEvent := 0
	totalSkippedBadIndex := 0

	for _, r := range ranges {
		rangeChan <- r
	}

	close(rangeChan)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for r := range rangeChan {
				<-ticker.C
				localComd, skipMeta, skipByt128, skipSan, skipBadEv, skipBadInx, err := processVolumeLogsChunk(
					ctx,
					r.start,
					r.end,
					addresses,
					poolsMap,
					parsedSwapABI,
					tokenExchangeEventSig,
					tokenExchangeUnderlyingEventSig,
				)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}
				mu.Lock()
				for addr, vol := range localComd {
					combined[addr] += vol
				}
				totalSkippedMeta += skipMeta
				totalSkippedByte128 += skipByt128
				totalSkippedSanity += skipSan
				totalSkippedBadEvent += skipBadEv
				totalSkippedBadIndex += skipBadInx
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
	}

	//for start := fromBlock; start <= latestBlock; start += blockStep {
	//	end := start + blockStep - 1
	//	if end > latestBlock {
	//		end = latestBlock
	//	}
	//
	//	// формируем запрос логов
	//	query := ethereum.FilterQuery{
	//		FromBlock: big.NewInt(int64(start)),
	//		ToBlock:   big.NewInt(int64(end)),
	//		Addresses: addresses,
	//		Topics:    [][]common.Hash{{tokenExchangeEventSig, tokenExchangeUnderlyingEventSig}},
	//	}
	//
	//	logs, err := BlockchainClientS.FilterLogs(ctx, query)
	//	if err != nil {
	//		return err
	//	}
	//
	//	for _, vlog := range logs {
	//		poolAddr := strings.ToLower(vlog.Address.Hex())
	//
	//		//достаём metadata пула
	//		meta := poolsMap[poolAddr]
	//		if meta == nil {
	//			skippedMeta++
	//			log.Println("-- load24hVolumeUSD load meta data not found for:", poolAddr)
	//			continue
	//		}
	//
	//		if len(vlog.Data) < 128 {
	//			skippedByte128++
	//			continue
	//		}
	//
	//		decoded, isUnderlying, err := decodeCurveSwapEvent(
	//			vlog, parsedSwapABI, tokenExchangeEventSig, tokenExchangeUnderlyingEventSig)
	//		if err != nil {
	//			skippedBadEvent++
	//			continue
	//		}
	//
	//		soldID := int(decoded.SoldID.Int64())
	//		boughtID := int(decoded.BoughtID.Int64())
	//
	//		//soldID := int(new(big.Int).SetBytes(vlog.Data[0:32]).Int64())
	//		//tokenSoldRaw := new(big.Int).SetBytes(vlog.Data[32:64])
	//		//
	//		//// --- BOUGHT ---
	//		//boughtID := int(new(big.Int).SetBytes(vlog.Data[64:96]).Int64())
	//		//tokenBoughtRaw := new(big.Int).SetBytes(vlog.Data[96:128])
	//
	//		var soldToken, boughtToken common.Address
	//		var soldDecimals, boughtDecimals int
	//
	//		if isUnderlying {
	//			if soldID < 0 || soldID >= len(meta.UnderlyingCoins) || boughtID < 0 || boughtID >= len(meta.UnderlyingCoins) {
	//				skippedBadIndex++
	//				continue
	//			}
	//			if meta.UnderlyingDecimals[soldID] == nil || meta.UnderlyingDecimals[boughtID] == nil {
	//				skippedBadIndex++
	//				continue
	//			}
	//
	//			soldToken = meta.UnderlyingCoins[soldID]
	//			boughtToken = meta.UnderlyingCoins[boughtID]
	//			soldDecimals = int(meta.UnderlyingDecimals[soldID].Int64())
	//			boughtDecimals = int(meta.UnderlyingDecimals[boughtID].Int64())
	//		} else {
	//			if soldID < 0 || soldID >= len(meta.Coins) || boughtID < 0 || boughtID >= len(meta.Coins) {
	//				skippedBadIndex++
	//				continue
	//			}
	//			if meta.Decimals[soldID] == nil || meta.Decimals[boughtID] == nil {
	//				skippedBadIndex++
	//				continue
	//			}
	//
	//			soldToken = meta.Coins[soldID]
	//			boughtToken = meta.Coins[boughtID]
	//			soldDecimals = int(meta.Decimals[soldID].Int64())
	//			boughtDecimals = int(meta.Decimals[boughtID].Int64())
	//		}
	//
	//		//switch vlog.Topics[0] {
	//		//case tokenExchangeEventSig:
	//		//
	//		//	// Обычный event -> используем coins arrays
	//		//	if soldID < 0 || soldID >= len(meta.Coins) || boughtID < 0 || boughtID >= len(meta.Coins) {
	//		//		skippedBadIndex++
	//		//		continue
	//		//	}
	//		//	if meta.Decimals[soldID] == nil || meta.Decimals[boughtID] == nil {
	//		//		skippedBadIndex++
	//		//		continue
	//		//	}
	//		//
	//		//	soldToken = meta.Coins[soldID]
	//		//	boughtToken = meta.Coins[boughtID]
	//		//	soldDecimals = int(meta.Decimals[soldID].Int64())
	//		//	boughtDecimals = int(meta.Decimals[boughtID].Int64())
	//		//
	//		//case tokenExchangeUnderlyingEventSig:
	//		//
	//		//	// Underlying event -> используем underlying arrays
	//		//	if soldID < 0 || soldID >= len(meta.UnderlyingCoins) || boughtID < 0 || boughtID >= len(meta.UnderlyingCoins) {
	//		//		skippedBadIndex++
	//		//		continue
	//		//	}
	//		//	if meta.UnderlyingDecimals[soldID] == nil || meta.UnderlyingDecimals[boughtID] == nil {
	//		//		skippedBadIndex++
	//		//		continue
	//		//	}
	//		//
	//		//	// адреса токенов sold и bought
	//		//	soldToken = meta.UnderlyingCoins[soldID]
	//		//	boughtToken = meta.UnderlyingCoins[boughtID]
	//		//	// decimals
	//		//	soldDecimals = int(meta.UnderlyingDecimals[soldID].Int64())
	//		//	boughtDecimals = int(meta.UnderlyingDecimals[boughtID].Int64())
	//		//
	//		//default:
	//		//	skippedBadEvent++
	//		//	continue
	//		//}
	//
	//		if soldToken == (common.Address{}) || boughtToken == (common.Address{}) {
	//			continue
	//		}
	//
	//		//// нормализация
	//		//soldAmount := normalize(tokenSoldRaw, soldDecimals)
	//		//boughtAmount := normalize(tokenBoughtRaw, boughtDecimals)
	//
	//		// цены (⚠потом заменишь на batch)
	//		soldPrice := getTokenPriceUSD(soldToken)
	//		boughtPrice := getTokenPriceUSD(boughtToken)
	//
	//		soldAmount, soldUSD := normalizeSwapAmountWithFallback(decoded.TokensSold, soldDecimals, soldPrice)
	//		boughtAmount, boughtUSD := normalizeSwapAmountWithFallback(decoded.TokensBought, boughtDecimals, boughtPrice)
	//
	//		//// перевод в USD
	//		//soldUSD := soldAmount * soldPrice
	//		//boughtUSD := boughtAmount * boughtPrice
	//
	//		// логируем только реально плохие события
	//		if soldUSD > 1_000_000 || boughtUSD > 1_000_000 {
	//			if strings.EqualFold(poolAddr, "0x0fafafd3c393ead5f5129cfc7e0e12367088c473") {
	//				log.Println("BAOUSD BAD EVENT topic0:", vlog.Topics[0].Hex())
	//				log.Println("BAOUSD BAD EVENT soldID:", soldID, "boughtID:", boughtID)
	//				log.Println("BAOUSD BAD EVENT soldToken:", soldToken.Hex(), "boughtToken:", boughtToken.Hex())
	//				log.Println("BAOUSD BAD EVENT soldUSD:", soldUSD, "boughtUSD:", boughtUSD)
	//				log.Println("BAOUSD BAD EVENT soldAmount:", soldAmount, "boughtAmount:", boughtAmount)
	//			}
	//			continue
	//		}
	//
	//		maxUSD := math.Max(soldUSD, boughtUSD)
	//		minUSD := math.Min(soldUSD, boughtUSD)
	//
	//		if maxUSD > 0 && minUSD > 0 && maxUSD/minUSD > 50 {
	//			log.Println(" -- Sanity filter load24hVolumeUSD ", vlog.Address.Hex())
	//			skippedSanity++
	//			continue
	//		}
	//
	//		// правильный volume
	//		combined[poolAddr] += (soldUSD + boughtUSD) / 2
	//
	//	}
	//}

	log.Println("-- load24hVolumeUSD skipped Meta", totalSkippedMeta)
	log.Println("-- load24hVolumeUSD skippedByte128", totalSkippedByte128)
	log.Println(" --load24hVolumeUSD skippedSanity", totalSkippedSanity)
	log.Println(" --load24hVolumeUSD skippedBadEvent", totalSkippedBadEvent)
	log.Println(" --load24hVolumeUSD skippedBadIndex", totalSkippedBadIndex)
	log.Println(" -- Count combined:", len(combined))
	CurveDay24HFeeMap = combined
	log.Println(" -- Count CurveDay24HFeeMap:", len(CurveDay24HFeeMap))

	//err := client.Call(&lastBlock, "eth_getBlockByNumber", "latest")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//latestBlock := blockchainClient.
	return nil
}

func loadSliceAllPools(factoryPoolsAddr []common.Address) ([]common.Address, error) {

	ctx := context.Background()
	latestBlock, err := BlockchainClientS.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	const blockChunk uint64 = 20

	allPools := make([]common.Address, 0, 1024)
	seenPools := make(map[common.Address]struct{}, 1024)

	//ticker := time.NewTicker(100 * time.Millisecond)
	//defer ticker.Stop()

	eventSig := crypto.Keccak256Hash(
		[]byte("PoolAdded(address)"),
	)

	for _, factoryAddress := range factoryPoolsAddr {

		if factoryAddress == (common.Address{}) {
			continue
		}

		startBlock, ok := factoryStartBlocks[factoryAddress]
		if !ok {
			log.Printf("loadSliceAllPools: no start block for factory %s, skip", factoryAddress.Hex())
			continue
		}

		// Читаем историю factory кусками по blockChunk
		for from := startBlock; from <= latestBlock; from += blockChunk {

			//<-ticker.C

			to := from + blockChunk - 2
			if to >= latestBlock {
				to = latestBlock
			}

			query := ethereum.FilterQuery{
				FromBlock: big.NewInt(int64(from)),
				ToBlock:   big.NewInt(int64(to)),
				Addresses: []common.Address{factoryAddress},
				Topics:    [][]common.Hash{{eventSig}},
			}

			logs, err := BlockchainClientS.FilterLogs(ctx, query)
			if err != nil {
				return nil, fmt.Errorf(
					"filter logs factory=%s from=%d to=%d: %w",
					factoryAddress.Hex(), from, to, err,
				)
			}

			for _, vlog := range logs {

				// Для PoolAdded(address) адрес пула лежит в topics[1]
				if len(vlog.Topics) < 2 {
					continue
				}

				pool := common.HexToAddress(vlog.Topics[1].Hex())

				if pool == (common.Address{}) {
					continue
				}

				// Убираем дубли
				if _, exists := seenPools[pool]; exists {
					continue
				}

				seenPools[pool] = struct{}{}

				RegistryPoolMap[pool] = factoryAddress
				allPools = append(allPools, pool)
			}
		}

		//query := ethereum.FilterQuery{
		//	FromBlock: big.NewInt(int64(factoryDeployBlock)),
		//	Addresses: []common.Address{factoryAddress},
		//	Topics:    [][]common.Hash{{eventSig}},
		//}

		//logs, err := BlockchainClientS.FilterLogs(ctx, query)
		//if err != nil {
		//	return nil, err
		//}

	}
	return allPools, nil
}

func normalize(value *big.Int, decimals int) float64 { //приводим цену к нормальному виду

	v := new(big.Float).SetInt(value)

	divisor := new(big.Float).SetFloat64(math.Pow10(decimals)) //получаем степень

	result, _ := new(big.Float).Quo(v, divisor).Float64() //делим значение на степень

	return result
}

func findBlock24hAgo() (uint64, error) {

	ctx := context.Background()

	latestBlock, err := BlockchainClientS.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	latestHeader, err := BlockchainClientS.HeaderByNumber(ctx, big.NewInt(int64(latestBlock)))
	if err != nil {
		return 0, err
	}

	targetTime := latestHeader.Time - 86400

	low := uint64(0)
	high := latestBlock

	for low <= high {

		mid := (low + high) / 2

		header, err := BlockchainClientS.HeaderByNumber(ctx, big.NewInt(int64(mid)))
		if err != nil {
			return 0, err
		}

		if header.Time > targetTime {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	return low, nil
}

func recursivePools(reg *crv.Uipool, pool common.Address, amount float64) float64 {

	opts := &bind.CallOpts{Context: context.Background()}

	basePool, err := reg.GetPoolFromLpToken(opts, pool)
	if err != nil || basePool == (common.Address{}) {
		price := getTokenPriceUSD(pool)
		return amount * price
	}

	coins, err := reg.GetCoins(opts, basePool)
	if err != nil {
		return 0
	}

	balances, err := reg.GetBalances(opts, basePool)
	if err != nil {
		return 0
	}

	decimals, err := reg.GetDecimals(opts, basePool)
	if err != nil {
		return 0
	}

	tvl := 0.0

	for i := 0; i < len(coins); i++ {

		if coins[i] == (common.Address{}) {
			continue
		}

		share := new(big.Float).Quo(
			new(big.Float).SetInt(balances[i]),
			new(big.Float).SetFloat64(math.Pow10(int(decimals[i].Int64()))),
		)

		val, _ := share.Float64()

		price := getTokenPriceUSD(coins[i])

		tvl += val * price
	}

	return tvl
}

func multicallBatchGetPoolData(pools []common.Address, registryABI abi.ABI,
) (*BatchPoolData, error) {

	opts := &bind.CallOpts{Context: context.Background()}
	contract := NewMulticall(common.HexToAddress(MulticallCurve))

	calls := make([]Call3, 0, len(pools)*5)

	for _, pool := range pools {

		registryAddress, ok := RegistryPoolMap[pool]
		if !ok {
			continue
		}

		bData, err := registryABI.Pack("get_balances", pool)
		if err != nil {
			continue
		}

		calls = append(calls, Call3{
			Target:       registryAddress,
			AllowFailure: true,
			CallData:     bData,
		})

		cData, err := registryABI.Pack("get_coins", pool)
		if err != nil {
			continue
		}
		calls = append(calls, Call3{
			Target:       registryAddress,
			AllowFailure: true,
			CallData:     cData,
		})

		dData, err := registryABI.Pack("get_decimals", pool)
		if err != nil {
			continue
		}
		calls = append(calls, Call3{
			Target:       registryAddress,
			AllowFailure: true,
			CallData:     dData,
		})

		nData, err := registryABI.Pack("get_pool_name", pool)
		if err != nil {
			continue
		}
		calls = append(calls, Call3{
			Target:       registryAddress,
			AllowFailure: true,
			CallData:     nData,
		})

		lpData, err := registryABI.Pack("get_lp_token", pool)
		if err != nil {
			continue
		}
		calls = append(calls, Call3{
			Target:       registryAddress,
			AllowFailure: true,
			CallData:     lpData,
		})

	}

	var results []Aggregate3Result
	out := []any{&results}

	errCall := contract.Call(opts, &out, "aggregate3", calls)
	if errCall != nil {
		return nil, errCall
	}

	balances := make([][8]*big.Int, len(pools))
	coins := make([][8]common.Address, len(pools))
	decimals := make([][8]*big.Int, len(pools))
	names := make([]string, len(pools))
	lpToken := make([]common.Address, len(pools))

	for i := 0; i < len(pools); i++ {
		base := i * 5

		if base+4 >= len(results) {
			break
		}

		if results[base].Success {
			registryABI.UnpackIntoInterface(&balances[i], "get_balances", results[base].ReturnData)
		}

		if results[base+1].Success {
			registryABI.UnpackIntoInterface(&coins[i], "get_coins", results[base+1].ReturnData)
		}

		if results[base+2].Success {
			registryABI.UnpackIntoInterface(&decimals[i], "get_decimals", results[base+2].ReturnData)
		}

		if results[base+3].Success {
			registryABI.UnpackIntoInterface(&names[i], "get_pool_name", results[base+3].ReturnData)
		}

		if results[base+4].Success {
			registryABI.UnpackIntoInterface(&lpToken[i], "get_lp_token", results[base+4].ReturnData)
		}
	}

	return &BatchPoolData{
		Balances: balances,
		Coins:    coins,
		Decimals: decimals,
		Names:    names,
		LPTokens: lpToken,
	}, nil
}

func collectTokens(pools map[string]*PoolStaticMeta) []common.Address {

	tokenSet := map[common.Address]struct{}{}

	for _, p := range pools {

		for _, c := range p.Coins {

			if c == (common.Address{}) {
				continue
			}

			tokenSet[c] = struct{}{}
		}
	}

	tokens := []common.Address{}

	for t := range tokenSet {
		tokens = append(tokens, t)
	}

	return tokens
}

func loadTokenPrices(tokens []common.Address) error {
	if len(tokens) == 0 {
		return nil
	}

	parts := make([]string, 0, len(tokens))
	seen := make(map[string]struct{})

	for _, token := range tokens {
		if token == (common.Address{}) {
			continue
		}

		addr := strings.ToLower(token.Hex())
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}

		parts = append(parts, "ethereum:"+addr)
	}

	if len(parts) == 0 {
		return nil
	}

	urlRequest := "https://coins.llama.fi/prices/current/" + strings.Join(parts, ",")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(urlRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Coins map[string]struct {
			Price float64 `json:"price"`
		} `json:"coins"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	for key, value := range result.Coins {
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			continue
		}

		addr := common.HexToAddress(parts[1])
		PriceCache.Store(addr, value.Price)
	}

	return nil
}

func calcTVLUSD(reg *crv.Uipool, pool common.Address, totalSupply *big.Int) float64 {

	opts := &bind.CallOpts{Context: context.Background()}

	lp, err := reg.GetLpToken(opts, pool)
	if err != nil {
		return 0
	}

	vp, err := reg.GetVirtualPriceFromLpToken(opts, lp)
	if err != nil {
		return 0
	}

	supply := new(big.Float).SetInt(totalSupply)

	virtualPrice := new(big.Float).Quo(
		new(big.Float).SetInt(vp),
		big.NewFloat(1e18),
	)

	tvl := new(big.Float).Mul(supply, virtualPrice)

	tvlFloat, _ := tvl.Float64()

	return tvlFloat
}

func normalized(balance *big.Int, decimal uint8) *big.Float {
	f := new(big.Float).SetInt(balance)
	denom := new(big.Float).SetFloat64(math.Pow10(int(decimal)))

	return new(big.Float).Quo(f, denom)
}

func NewMulticall(multicallAddr common.Address) *bind.BoundContract {

	file, err := os.ReadFile("ABIevm/curve/Multicall3Curve.json")
	if err != nil {
		panic(err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(file)))
	if err != nil {
		panic(err)
	}

	contract := bind.NewBoundContract(
		multicallAddr,
		parsedABI,
		BlockchainClientS,
		BlockchainClientS,
		BlockchainClientS,
	)

	return contract
}

func LoadRegistryABI() abi.ABI {

	file, err := os.ReadFile("./ABIevm/contracts/curve/CurveMetaRegistry.json")
	if err != nil {
		panic(err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(file)))
	if err != nil {
		panic(err)
	}

	return parsedABI
}

func processVolumeLogsChunk(
	ctx context.Context,
	start uint64,
	end uint64,
	addresses []common.Address,
	poolsMap map[string]*PoolStaticMeta,
	parsedSwapABI abi.ABI,
	tokenExchangeEventSig common.Hash,
	tokenExchangeUnderlyingEventSig common.Hash,
) (map[string]float64, int, int, int, int, int, error) {

	localCombined := make(map[string]float64)

	skippedMeta := 0
	skippedByte128 := 0
	skippedSanity := 0
	skippedBadEvent := 0
	skippedBadIndex := 0

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(start)),
		ToBlock:   big.NewInt(int64(end)),
		Addresses: addresses,
		Topics:    [][]common.Hash{{tokenExchangeEventSig, tokenExchangeUnderlyingEventSig}},
	}

	logs, err := BlockchainClientS.FilterLogs(ctx, query)
	if err != nil {
		return nil, 0, 0, 0, 0, 0, err
	}

	for _, vlog := range logs {
		poolAddr := strings.ToLower(vlog.Address.Hex())

		meta := poolsMap[poolAddr]
		if meta == nil {
			skippedMeta++
			continue
		}

		if len(vlog.Data) < 128 {
			skippedByte128++
			continue
		}

		decoded, isUnderlying, err := decodeCurveSwapEvent(
			vlog,
			parsedSwapABI,
			tokenExchangeEventSig,
			tokenExchangeUnderlyingEventSig,
		)
		if err != nil {
			skippedBadEvent++
			continue
		}

		soldID := int(decoded.SoldID.Int64())
		boughtID := int(decoded.BoughtID.Int64())

		var soldToken, boughtToken common.Address
		var soldDecimals, boughtDecimals int

		if isUnderlying {
			if soldID < 0 || soldID >= len(meta.UnderlyingCoins) || boughtID < 0 || boughtID >= len(meta.UnderlyingCoins) {
				skippedBadIndex++
				continue
			}
			if meta.UnderlyingDecimals[soldID] == nil || meta.UnderlyingDecimals[boughtID] == nil {
				skippedBadIndex++
				continue
			}

			soldToken = meta.UnderlyingCoins[soldID]
			boughtToken = meta.UnderlyingCoins[boughtID]
			soldDecimals = int(meta.UnderlyingDecimals[soldID].Int64())
			boughtDecimals = int(meta.UnderlyingDecimals[boughtID].Int64())
		} else {
			if soldID < 0 || soldID >= len(meta.Coins) || boughtID < 0 || boughtID >= len(meta.Coins) {
				skippedBadIndex++
				continue
			}
			if meta.Decimals[soldID] == nil || meta.Decimals[boughtID] == nil {
				skippedBadIndex++
				continue
			}

			soldToken = meta.Coins[soldID]
			boughtToken = meta.Coins[boughtID]
			soldDecimals = int(meta.Decimals[soldID].Int64())
			boughtDecimals = int(meta.Decimals[boughtID].Int64())
		}

		if soldToken == (common.Address{}) || boughtToken == (common.Address{}) {
			continue
		}

		soldPrice := getTokenPriceUSD(soldToken)
		boughtPrice := getTokenPriceUSD(boughtToken)

		soldAmount, soldUSD := normalizeSwapAmountWithFallback(decoded.TokensSold, soldDecimals, soldPrice)
		boughtAmount, boughtUSD := normalizeSwapAmountWithFallback(decoded.TokensBought, boughtDecimals, boughtPrice)

		_ = soldAmount
		_ = boughtAmount

		if soldUSD > 1_000_000 || boughtUSD > 1_000_000 {
			continue
		}

		if soldUSD <= 0 || boughtUSD <= 0 {
			skippedSanity++
			continue
		}

		maxUSD := math.Max(soldUSD, boughtUSD)
		minUSD := math.Min(soldUSD, boughtUSD)

		if maxUSD/minUSD > 50 {
			skippedSanity++
			continue
		}

		localCombined[poolAddr] += (soldUSD + boughtUSD) / 2
	}

	return localCombined, skippedMeta, skippedByte128, skippedSanity, skippedBadEvent, skippedBadIndex, nil
}

//func debugLogShape(vlog types.Log, tokenExchangeSig, tokenExchangeUnderlyingSig common.Hash) {
//	eventType := "unknown"
//	switch vlog.Topics[0] {
//	case tokenExchangeSig:
//		eventType = "TokenExchange"
//	case tokenExchangeUnderlyingSig:
//		eventType = "TokenExchangeUnderlying"
//	}
//
//	log.Println("---- EVENT DEBUG ----")
//	log.Println("type:", eventType)
//	log.Println("address:", vlog.Address.Hex())
//	log.Println("topics len:", len(vlog.Topics))
//	log.Println("data len:", len(vlog.Data))
//	log.Println("topic0:", vlog.Topics[0].Hex())
//	log.Println("data hex:", hex.EncodeToString(vlog.Data))
//}

type PoolStaticMeta struct {
	Address            common.Address
	Gauge              common.Address
	GaugeContract      *crv.LiquidityGauge
	Coins              [8]common.Address
	Decimals           [8]*big.Int
	UnderlyingCoins    [8]common.Address
	UnderlyingDecimals [8]*big.Int
	Name               string
	Registry           *crv.Uipool
}

type PoolDynamicMeta struct {
	VirtualPrice *big.Int
	Balances     [8]*big.Int
}

type Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

type Aggregate3Result struct {
	Success    bool
	ReturnData []byte
}

type BatchPoolData struct {
	Balances [][8]*big.Int
	Coins    [][8]common.Address
	Decimals [][8]*big.Int
	Names    []string
	LPTokens []common.Address
}

type CurveSwapEvent struct {
	SoldID       *big.Int
	TokensSold   *big.Int
	BoughtID     *big.Int
	TokensBought *big.Int
}

func loadCurveSwapABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(curveSwapEventsABI))
	if err != nil {
		panic(err)
	}
	return parsed
}

func decodeCurveSwapEvent(vlog types.Log, parsed abi.ABI, tokenExchangeSig, tokenExchangeUnderlyingSig common.Hash) (*CurveSwapEvent, bool, error) {
	var eventName string
	var isUnderlying bool

	switch vlog.Topics[0] {
	case tokenExchangeSig:
		eventName = "TokenExchange"
		isUnderlying = false
	case tokenExchangeUnderlyingSig:
		eventName = "TokenExchangeUnderlying"
		isUnderlying = true
	default:
		return nil, false, fmt.Errorf("unknown event topic: %s", vlog.Topics[0].Hex())
	}

	values, err := parsed.Events[eventName].Inputs.NonIndexed().Unpack(vlog.Data)
	if err != nil {
		return nil, false, err
	}
	if len(values) != 4 {
		return nil, false, fmt.Errorf("unexpected values len: %d", len(values))
	}

	soldID, ok1 := values[0].(*big.Int)
	tokensSold, ok2 := values[1].(*big.Int)
	boughtID, ok3 := values[2].(*big.Int)
	tokensBought, ok4 := values[3].(*big.Int)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return nil, false, fmt.Errorf("bad event decode types")
	}

	return &CurveSwapEvent{
		SoldID:       soldID,
		TokensSold:   tokensSold,
		BoughtID:     boughtID,
		TokensBought: tokensBought,
	}, isUnderlying, nil
}

func normalizeSwapAmountWithFallback(raw *big.Int, tokenDecimals int, price float64) (float64, float64) {
	// Базовый вариант: обычные decimals токена
	amountNative := normalize(raw, tokenDecimals)
	usdNative := amountNative * price

	// Для проблемных metapool-событий иногда raw amount выглядит как 1e18-scaled.
	// Если нативная интерпретация даёт абсурдный USD, пробуем 18 decimals.
	if tokenDecimals < 18 && usdNative > 1_000_000 {
		amount18 := normalize(raw, 18)
		usd18 := amount18 * price

		// Берём 18-decimal интерпретацию, только если она выглядит реалистично.
		if usd18 > 0 && usd18 < 1_000_000 {
			return amount18, usd18
		}
	}

	return amountNative, usdNative
}

func getTokenSymbol(token common.Address) string {
	if token == (common.Address{}) {
		return ""
	}

	var knownTokenSymbols = map[common.Address]string{
		common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"): "ETH",
	}

	if symbol, ok := knownTokenSymbols[token]; ok {
		return symbol
	}

	if val, ok := TokenSymbolCache.Load(token); ok {
		return val.(string)
	}

	erc20ABIJson := `[
		{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(erc20ABIJson))
	if err != nil {
		return token.Hex()[:6]
	}

	contract := bind.NewBoundContract(token, parsedABI, BlockchainClientS, BlockchainClientS, BlockchainClientS)
	opts := &bind.CallOpts{Context: context.Background()}

	var out []any
	err = contract.Call(opts, &out, "symbol")
	if err != nil || len(out) == 0 {
		short := token.Hex()[:6]
		TokenSymbolCache.Store(token, short)
		return short
	}

	symbol, ok := out[0].(string)
	if !ok || symbol == "" {
		short := token.Hex()[:6]
		TokenSymbolCache.Store(token, short)
		return short
	}

	TokenSymbolCache.Store(token, symbol)
	return symbol
}
func buildPoolDisplayName(pool *PoolStaticMeta) string {
	if pool == nil {
		return ""
	}

	symbols := make([]string, 0, len(pool.Coins))
	seen := make(map[string]struct{})

	for _, token := range pool.Coins {
		if token == (common.Address{}) {
			continue
		}

		symbol := getTokenSymbol(token)
		if symbol == "" {
			continue
		}

		if _, ok := seen[symbol]; ok {
			continue
		}
		seen[symbol] = struct{}{}

		symbols = append(symbols, symbol)
	}

	if len(symbols) == 0 {
		return pool.Name
	}

	return strings.Join(symbols, "/")
}

const curveSwapEventsABI = `[
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true,  "internalType": "address", "name": "buyer",         "type": "address"},
      {"indexed": false, "internalType": "int128",  "name": "sold_id",       "type": "int128"},
      {"indexed": false, "internalType": "uint256", "name": "tokens_sold",   "type": "uint256"},
      {"indexed": false, "internalType": "int128",  "name": "bought_id",     "type": "int128"},
      {"indexed": false, "internalType": "uint256", "name": "tokens_bought", "type": "uint256"}
    ],
    "name": "TokenExchange",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true,  "internalType": "address", "name": "buyer",         "type": "address"},
      {"indexed": false, "internalType": "int128",  "name": "sold_id",       "type": "int128"},
      {"indexed": false, "internalType": "uint256", "name": "tokens_sold",   "type": "uint256"},
      {"indexed": false, "internalType": "int128",  "name": "bought_id",     "type": "int128"},
      {"indexed": false, "internalType": "uint256", "name": "tokens_bought", "type": "uint256"}
    ],
    "name": "TokenExchangeUnderlying",
    "type": "event"
  }
]`
