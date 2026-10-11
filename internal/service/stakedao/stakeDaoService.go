package stakedao

import (
	//crv "ABIevm/pkg/uipool/curve/registry"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	dto "ABIevm/internal/service/stakedao/dto"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	//"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	//vaultControl "ABIevm/pkg/uipool/stakedao"
)

const (
	ETH_RPC                     = "https://eth-mainnet.g.alchemy.com/v2/..."
	SECOND_PER_YEAR             = 31556925
	MULTICALL_ADDRESS           = "0xcA11bde05977b3631167028862bE2a173976CA11"
	PROTOCOL_CONTROLLER_ETH_ARB = "0x2d8BcE1FaE00a959354aCD9eBf9174337A64d4fb"
	FACTORY                     = "0xEf9bef9AB7b578eb0654F0cD2C75519c9A3f7FE1"
)

var BlockChainClient *ethclient.Client

var MainRewardTokenByGauge map[common.Address]common.Address

var PriceCache sync.Map

var gaugesByChain = map[string][]common.Address{
	"ethereum": make([]common.Address, 0),
	"arbitrum": make([]common.Address, 0),
	"base":     make([]common.Address, 0),
}

var gaugesMainFromController = []string{
	"0x3285987Bbd0b695Ac48DC8571F519065B50aBA07",
	"0xbE6228125448E40F5DCBB2DbdF7fF3F5721000B4",
	"0x326b6d1a26Fd04afAa349340022076b5a2eb0c7C",
	"0xb352c96ceb4189b9801817243EB4887aDF241905",
	"0x5a332717D779267B883A1bB1B5910494D73dd9C0",
	"0x2a2F4Cf8cB8452e5Db8A2eFF286aF66C6dc213a9",
	"0xF7F4b8bFb6dE08435adc37eaAd626a22ED730A92",
	"0xD6F025f87De29B71dE36Ad4C93499E9C8B2D8507",
	"0x6B43af6A06Aa0d06C0Cc9c87A49f0F7d2E1CE044",
	"0xa5F5DBdFc6baFc69bb4107AF97Ca79519304Dbd7",
	"0xF429AeC167C92aCA16cD77aef54F196B1988cBA3",
	"0x07a01471fA544D9C6531B631E6A96A79a9AD05E9",
	"0x14fBe18CAB1F9Cc8b1B6bD5416Be225A43fbd995",
	"0xbD6d1A33A6FA37379eCc89b6CFf7c7d7F8C305eC",
	"0xbB05Ca38069d14eD34930Cb37F5C04438Df8C3f4",
	"0x5beC8A3C2d316a05FA231bDdDD23c7E24CB421a6",
	"0x8F4e42343AF1CBF4f364cE448252d801E3f73937",
	"0x30dD71C08c45FBFbff6637Af5afA1b845806d54D",
	"0xF814283bfd37dC24B2fB5515982F5C0708E508b6",
	"0xB069D59fDDFE8e2e6E624951e306D4FdC056fA82",
	"0x83f12B0506a115E610c3e76292885192731e9b53",
	"0x4B891340b51889f438a03DC0e8aAAFB0Bc89e7A6",
	"0x6732ED78BF1D8afA3ef03F1ead302d21b0B2B1FF",
	"0xDbD4dD2d5E0b927022C2be8D493F20e79e60C2e7",
	"0x5241B61045AFB7bc7E031bFe4931Ef4BD614d5eC",
}

//var seedGauges []common.Address

type StakeDAOService struct {
}

func NewStakeDAOService() *StakeDAOService {
	return &StakeDAOService{}
}

func (stkDAO StakeDAOService) PoolStakeDAOPools() {

	client, err := ethclient.Dial(ETH_RPC)
	if err != nil {
		log.Fatal("Failed to connect to Ethereum client")
		log.Fatal(err)
		return
	}

	ctx := context.Background()

	//allSeedGauges := make([]common.Address, 0, 1000)
	//
	//allSeedGauges = gaugesByChain["ethereum"]
	BlockChainClient = client

	latestBlock, err := BlockChainClient.BlockNumber(ctx)
	if err != nil {
		log.Fatal(err)
	}

	vaults, err := DiscoverStakeDAOVaults(ctx, BlockChainClient, common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB), latestBlock)
	if err != nil {
		log.Fatal(err)
	}

	for _, vaultInfo := range vaults {

		gaugeAddr := common.HexToAddress(vaultInfo.Gauge)
		reward,err := stkDAO.
	}

	pools, err := stkDAO.GetPools(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range pools {
		fmt.Println(v.Name, v.APY, v.TVL)
	}

	defer BlockChainClient.Close()
}

func (stkDAO *StakeDAOService) GetPools(ctx context.Context) ([]dto.StakeDAOPool, error) {
	out := make([]dto.StakeDAOPool, 0, len(gaugesMainFromController))

	for _, gauge := range gaugesMainFromController {
		pool, err := stkDAO.loadPool(ctx, common.HexToAddress(gauge))
		if err != nil {
			// Для production можно логировать и continue.
			continue
		}
		out = append(out, pool)
	}

	return out, nil
}

func (stkDAO *StakeDAOService) loadPool(ctx context.Context, gauge common.Address) (dto.StakeDAOPool, error) {
	var pool dto.StakeDAOPool

	vaultAddr, err := stkDAO.controllerVault(ctx, gauge)
	if err != nil {
		return pool, fmt.Errorf("controller.vault(%stkDAO): %w", gauge.Hex(), err)
	}
	if vaultAddr == (common.Address{}) {
		return pool, fmt.Errorf("gauge %stkDAO is not registered in controller", gauge.Hex())
	}

	assetAddr, err := stkDAO.controllerAsset(ctx, gauge)
	if err != nil {
		return pool, fmt.Errorf("controller.asset(%stkDAO): %w", gauge.Hex(), err)
	}

	name, _ := stkDAO.readERC20String(ctx, vaultAddr, erc20NameABI, "name")
	symbol, _ := stkDAO.readERC20String(ctx, vaultAddr, erc20SymbolABI, "symbol")
	assetSymbol, _ := stkDAO.readERC20String(ctx, assetAddr, erc20SymbolABI, "symbol")

	tvl, err := stkDAO.readVaultTVL(ctx, vaultAddr, assetAddr)
	if err != nil {
		return pool, fmt.Errorf("read tvl: %w", err)
	}

	apy, rewardToken, _ := stkDAO.readCurrentAPY(ctx, gauge, tvl)

	pool = dto.StakeDAOPool{
		GaugeAddress: gauge,
		VaultAddress: vaultAddr,
		AssetAddress: assetAddr,
		Name:         nonEmpty(name, symbol, gauge.Hex()),
		Symbol:       symbol,
		AssetSymbol:  assetSymbol,
		TVL:          tvl,
		APY:          apy,
		RewardToken:  rewardToken,
		Source:       "onchain",
		UpdatedAt:    time.Now().UTC(),
	}

	return pool, nil
}

func (stkDAO *StakeDAOService) controllerVault(ctx context.Context, gauge common.Address) (common.Address, error) {
	return stkDAO.readAddressCall(
		ctx,
		common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB),
		protocolControllerABI,
		"vault",
		gauge,
	)
}

func (stkDAO *StakeDAOService) controllerAsset(ctx context.Context, gauge common.Address) (common.Address, error) {
	return stkDAO.readAddressCall(
		ctx,
		common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB),
		protocolControllerABI,
		"asset",
		gauge,
	)
}

// TVL
func (stkDAO *StakeDAOService) readVaultTVL(ctx context.Context, vaultAddr, assetAddr common.Address) (float64, error) {
	assetDecimals, err := stkDAO.readERC20Decimals(ctx, assetAddr)
	if err != nil {
		assetDecimals = 18
	}

	// 1) Предпочтительно ERC4626 totalAssets()
	totalAssets, err := stkDAO.readUint256Call(ctx, vaultAddr, rewardVaultERC4626ABI, "totalAssets")
	if err == nil && totalAssets != nil {
		return toFloat(totalAssets, int(assetDecimals)), nil
	}

	// 2) fallback: totalSupply() * convertToAssets(1 share)
	totalSupply, errSupply := stkDAO.readUint256Call(ctx, vaultAddr, erc20TotalSupplyABI, "totalSupply")
	if errSupply != nil || totalSupply == nil {
		return 0, fmt.Errorf("totalAssets and totalSupply unavailable")
	}

	oneShare := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(assetDecimals)), nil)

	perShareAssets, errConvert := stkDAO.readUint256Call(ctx, vaultAddr, rewardVaultERC4626ABI, "convertToAssets", oneShare)
	if errConvert == nil && perShareAssets != nil {
		// tvl = totalSupply * perShareAssets / 10^decimals
		n := new(big.Int).Mul(totalSupply, perShareAssets)
		n = n.Div(n, oneShare)
		return toFloat(n, int(assetDecimals)), nil
	}

	// 3) StakeDAO docs говорят assets == shares для RewardVault.
	// Тогда totalSupply ~= totalAssets.
	return toFloat(totalSupply, int(assetDecimals)), nil
}

// APY
func (stkDAO *StakeDAOService) readCurrentAPY(ctx context.Context, gauge common.Address, tvl float64) (float64, common.Address, error) {
	if tvl <= 0 {
		return 0, common.Address{}, nil
	}

	rewardToken, ok := MainRewardTokenByGauge[gauge]
	if !ok || rewardToken == (common.Address{}) {
		return 0, common.Address{}, fmt.Errorf("main reward token is not configured for gauge %stkDAO", gauge.Hex())
	}

	rewardDecimals, err := stkDAO.readERC20Decimals(ctx, rewardToken)
	if err != nil {
		rewardDecimals = 18
	}

	// В docs StakeDAO current APR для strategies берется из gauge.reward_data(...).rate / period_finish.
	// Здесь считаем current APR по rate.
	rd, err := stkDAO.readRewardData(ctx, gauge, rewardToken)
	if err != nil {
		return 0, rewardToken, err
	}

	if rd.Rate == nil || rd.Rate.Sign() == 0 {
		return 0, rewardToken, nil
	}

	//price, err := stkDAO.priceOracle.GetTokenPrice(ctx, rewardToken, stkDAO.chainID)
	dynamicRaw, ok := PriceCache.Load(rewardToken)

	price := dynamicRaw.(float64)
	if err != nil || price <= 0 || !ok {
		return 0, rewardToken, fmt.Errorf("reward price unavailable: %w", err)
	}
	// reward per second * price * year / tvl * 100
	rewardPerSecond := toFloat(rd.Rate, int(rewardDecimals))
	apr := (rewardPerSecond * price * SECOND_PER_YEAR / tvl) * 100

	// В absence compounding model будем возвращать APR как APY.
	// Если у вас в приложении есть глобальная compounding policy, можно заменить здесь.
	return apr, rewardToken, nil
}

type GaugeRewardData struct {
	Token        common.Address
	Distributor  common.Address
	PeriodFinish *big.Int
	Rate         *big.Int
	LastUpdate   *big.Int
	Integral     *big.Int
}

func (stkDAO *StakeDAOService) readRewardData(ctx context.Context, gauge, rewardToken common.Address) (*GaugeRewardData, error) {
	parsed, err := abi.JSON(strings.NewReader(curveLikeGaugeRewardDataABI))
	if err != nil {
		return nil, err
	}

	data, err := parsed.Pack("reward_data", rewardToken)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{To: &gauge, Data: data}

	raw, err := BlockChainClient.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}

	out, err := parsed.Unpack("reward_data", raw)
	if err != nil {
		return nil, err
	}
	if len(out) != 6 {
		return nil, fmt.Errorf("unexpected reward_data tuple length: %d", len(out))
	}

	return &GaugeRewardData{
		Token:        out[0].(common.Address),
		Distributor:  out[1].(common.Address),
		PeriodFinish: out[2].(*big.Int),
		Rate:         out[3].(*big.Int),
		LastUpdate:   out[4].(*big.Int),
		Integral:     out[5].(*big.Int),
	}, nil
}

// ABI
func (stkDAO *StakeDAOService) readAddressCall(ctx context.Context, contract common.Address, abiJSON, method string, args ...interface{}) (common.Address, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return common.Address{}, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return common.Address{}, err
	}

	msg := ethereum.CallMsg{To: &contract, Data: data}

	raw, err := BlockChainClient.CallContract(ctx, msg, nil)
	if err != nil {
		return common.Address{}, err
	}

	out, err := parsed.Unpack(method, raw)
	if err != nil {
		return common.Address{}, err
	}
	if len(out) != 1 {
		return common.Address{}, fmt.Errorf("unexpected outputs len: %d", len(out))
	}

	addr, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, errors.New("output is not address")
	}
	return addr, nil
}

func (stkDAO *StakeDAOService) readUint256Call(ctx context.Context, contract common.Address, abiJSON, method string, args ...interface{}) (*big.Int, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}
	data, err := parsed.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{To: &contract, Data: data}

	raw, err := BlockChainClient.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}

	out, err := parsed.Unpack(method, raw)
	if err != nil {
		return nil, err
	}
	if len(out) != 1 {
		return nil, fmt.Errorf("unexpected outputs len: %d", len(out))
	}

	val, ok := out[0].(*big.Int)
	if !ok {
		return nil, errors.New("output is not *big.Int")
	}
	return val, nil
}

func (stkDAO *StakeDAOService) readERC20Decimals(ctx context.Context, token common.Address) (uint8, error) {
	parsed, err := abi.JSON(strings.NewReader(erc20DecimalsABI))
	if err != nil {
		return 0, err
	}
	data, err := parsed.Pack("decimals")
	if err != nil {
		return 0, err
	}

	msg := ethereum.CallMsg{To: &token, Data: data}

	raw, err := BlockChainClient.CallContract(ctx, msg, nil)
	if err != nil {
		return 0, err
	}

	out, err := parsed.Unpack("decimals", raw)
	if err != nil {
		return 0, err
	}
	if len(out) != 1 {
		return 0, errors.New("invalid decimals output")
	}

	switch v := out[0].(type) {
	case uint8:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected decimals type %T", v)
	}
}

func (stkDAO *StakeDAOService) readERC20String(ctx context.Context, token common.Address, abiJSON, method string) (string, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", err
	}
	data, err := parsed.Pack(method)
	if err != nil {
		return "", err
	}

	msg := ethereum.CallMsg{To: &token, Data: data}

	raw, err := BlockChainClient.CallContract(ctx, msg, nil)
	if err != nil {
		return "", err
	}

	out, err := parsed.Unpack(method, raw)
	if err != nil {
		return "", err
	}
	if len(out) != 1 {
		return "", errors.New("invalid string output")
	}

	val, ok := out[0].(string)
	if !ok {
		return "", fmt.Errorf("unexpected string type %T", out[0])
	}
	return val, nil
}

func toFloat(v *big.Int, decimals int) float64 {
	if v == nil {
		return 0
	}
	f := new(big.Float).SetInt(v)
	div := new(big.Float).SetFloat64(math.Pow10(decimals))
	res, _ := new(big.Float).Quo(f, div).Float64()
	return res
}

func nonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type PoolStaticMeta struct {
	Address common.Address
	//Gauge              common.Address
	//GaugeContract      *crv.LiquidityGauge
	Coins [8]common.Address
	Name  string
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

// функция получения цены токена для пулов, потом вынести в отдельный файл для всех платформ и кэшировать
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

//// функция получения списка адресов gauge в Ethereum
//func loadSliceGaugesByEventFromBlockchain() {
//
//	//parsedABI, _ := abi.JSON(strings.NewReader(controllerABI))
//	//
//	//parsedABI.Unpack()
//
//	sliceGauges := make([]common.Address, 0, 100)
//
//	ctx := context.Background()
//	latestBlock, err := BlockChainClient.BlockNumber(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	//var gaugeEventSig = crypto.Keccak256Hash([]byte(
//	//	"GaugeCreated(address,address)",
//	//))
//
//	//eventVaultCreated := crypto.Keccak256Hash(
//	//	[]byte("PoolAdded(address)"),
//	//)
//	//
//	//eventGaugeCreated := crypto.Keccak256Hash(
//	//	[]byte("PoolAdded(address)"),
//	//)
//	//
//	//eventStrategyAdded := crypto.Keccak256Hash(
//	//	[]byte("PoolAdded(address)"),
//	//)
//	//
//	//eventRegistered := crypto.Keccak256Hash(
//	//	[]byte("PoolAdded(address)"),
//	//)
//
//	query := ethereum.FilterQuery{
//		Addresses: []common.Address{
//			common.HexToAddress(FACTORY),
//		},
//		FromBlock: big.NewInt(0),
//		ToBlock:   big.NewInt(int64(latestBlock)), // latest
//	}
//
//	logs, err := BlockChainClient.FilterLogs(ctx, query)
//	if err != nil {
//		log.Println(" --Filter logs loadGaugesEvents", err)
//	}
//
//	for _, val := range logs {
//		sliceGauges = append(sliceGauges, val.Address)
//	}
//
//}

const controllerABI = ``

// -----------------------------
// Minimal ABIs
// -----------------------------

const protocolControllerABI = `[
	{"inputs":[{"internalType":"address","name":"gauge","type":"address"}],"name":"vault","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"address","name":"gauge","type":"address"}],"name":"asset","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
]`

const rewardVaultERC4626ABI = `[
	{"inputs":[],"name":"totalAssets","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"shares","type":"uint256"}],"name":"convertToAssets","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]`

const erc20TotalSupplyABI = `[
	{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]`

const erc20DecimalsABI = `[
	{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
]`

const erc20NameABI = `[
	{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}
]`

const erc20SymbolABI = `[
	{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}
]`

// Типичный Curve-like reward_data(token)
// Подходит для gauge, если конкретная интеграция StakeDAO опирается на такой интерфейс.
const curveLikeGaugeRewardDataABI = `[
	{
		"inputs":[{"internalType":"address","name":"token","type":"address"}],
		"name":"reward_data",
		"outputs":[
			{"internalType":"address","name":"token","type":"address"},
			{"internalType":"address","name":"distributor","type":"address"},
			{"internalType":"uint256","name":"period_finish","type":"uint256"},
			{"internalType":"uint256","name":"rate","type":"uint256"},
			{"internalType":"uint256","name":"last_update","type":"uint256"},
			{"internalType":"uint256","name":"integral","type":"uint256"}
		],
		"stateMutability":"view",
		"type":"function"
	}
]`

//type VaultRegistered struct {
//	Gauge common.Address
//	Vault common.Address
//	Asset common.Address
//	RewardReceiver common.Address
//	ProtocolId     [4]byte
//}

type StakeDAOVaultInfo struct {
	Gauge string
	Vault string
	Asset string
}

// Доставать информацию по хранилищая из блокчейна через события.
func DiscoverStakeDAOVaults(
	ctx context.Context,
	client *ethclient.Client,
	controller common.Address,
	fromBlock uint64,
) ([]StakeDAOVaultInfo, error) {

	const eventABI = `[
	  {
		"anonymous": false,
		"inputs": [
		  {"indexed": true, "name": "gauge", "type": "address"},
		  {"indexed": true, "name": "vault", "type": "address"},
		  {"indexed": true, "name": "asset", "type": "address"},
		  {"indexed": false, "name": "rewardReceiver", "type": "address"},
		  {"indexed": false, "name": "protocolId", "type": "bytes4"}
		],
		"name": "VaultRegistered",
		"type": "event"
	  }
	]`

	parsedABI, err := abi.JSON(strings.NewReader(eventABI))
	if err != nil {
		return nil, err
	}

	event := parsedABI.Events["VaultRegistered"]

	query := ethereum.FilterQuery{
		Addresses: []common.Address{controller},
		Topics: [][]common.Hash{
			{event.ID},
		},
		FromBlock: big.NewInt(int64(fromBlock)),
	}

	logs, err := client.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}

	var result []StakeDAOVaultInfo

	for _, vLog := range logs {

		var decoded struct {
			RewardReceiver common.Address
			ProtocolId     [4]byte
		}

		err := parsedABI.UnpackIntoInterface(
			&decoded,
			"VaultRegistered",
			vLog.Data,
		)
		if err != nil {
			continue
		}

		info := StakeDAOVaultInfo{
			Gauge: vLog.Topics[1].Hex(),
			Vault: vLog.Topics[2].Hex(),
			Asset: vLog.Topics[3].Hex(),
		}

		result = append(result, info)
	}

	return result, nil
}
