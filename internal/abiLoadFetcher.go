package internal

import (
	"bytes"
	"log"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const EthApiKEY = "---1PV---"

// const EthUrlAbi = "https://api.etherscan.io/api?module=contract&action=getabi&address=%s&apikey=%s"
// const EthApiGithub = "https://github.com/aave/aave-utilities/blob/master/abi/UiPoolDataProviderV3.json"
const ABI_PATH = "contracts/uipool_abi.json"

type AbiLoadFetcher struct {
	cache map[string]string
	mu    sync.RWMutex
}

func NewAbiLoadFetcher() *AbiLoadFetcher {
	return &AbiLoadFetcher{
		cache: make(map[string]string),
	}
}

//type etherscanResp struct {
//	Status  string `json:"status"`
//	Message string `json:"message"`
//	Result  string `json:"result"`
//}

//func (f *AbiLoadFetcher) GetABI(contract string) (string, error) {
//	f.mu.RLock()
//	if v, ok := f.cache[contract]; ok {
//		f.mu.RUnlock()
//		return v, nil
//	}
//	f.mu.RUnlock()
//
//	// Etherscan API (mainnet)
//	//url := fmt.Sprintf(EthUrlAbi, contract, EthApiKEY)
//	fmt.Println(" - abiLoaderFetcher GetABI, before http.Get url ")
//	resp, err := http.Get(EthApiGithub)
//	if err != nil {
//		log.Println(" - abiLoaderFetcher GetABI, http.Get Github- " + " err - " + err.Error())
//		return "", err
//	}
//
//	fmt.Println(resp)
//	if resp.Body == nil {
//		log.Println(" - abiLoaderFetcher GetABI, http.Get - " + " resp == nil")
//	}
//
//	defer resp.Body.Close()
//	body, _ := io.ReadAll(resp.Body)
//
//	//parsed, err := abi.JSON(bytes.NewReader(body))
//	if err != nil {
//		log.Println(" - abiLoaderFetcher GetABI, ioutil.ReadAll err - " + err.Error())
//	}
//	var r etherscanResp
//	if err := json.Unmarshal(body, &r); err != nil {
//		log.Println(" - abiLoaderFetcher GetABI, json.Unmarshal " + err.Error())
//		return "", err
//	}
//	if r.Status != "1" && r.Result == "" {
//		return "", errors.New("etherscan returned empty ABI: " + r.Result)
//	}
//
//	// store
//	f.mu.Lock()
//	f.cache[contract] = r.Result
//	f.mu.Unlock()
//
//	return r.Result, nil
//	//return parsed, nil
//}

func (f *AbiLoadFetcher) LoadABI() (abi.ABI, error) {
	data, err := os.ReadFile(ABI_PATH)
	if err != nil {
		log.Fatalf("ABI read error: %v", err)
	}
	parsed, err := abi.JSON(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("ABI parse error: %v", err)
	}
	return parsed, nil
}
