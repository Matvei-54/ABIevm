package internal

import (
	"ABIevm/internal/service/prices"
	"ABIevm/internal/service/stakedao"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

// Exercise the real Echo route and ethclient JSON-RPC transport without mainnet access.
func TestStakeDAORoute(t *testing.T) {
	t.Setenv("STAKEDAO_CONTROLLER_DEPLOYMENT_BLOCK", "3")
	controller := common.HexToAddress("0x2d8BcE1FaE00a959354aCD9eBf9174337A64d4fb")
	gauge, vault, asset := common.HexToAddress("0x11"), common.HexToAddress("0x22"), common.HexToAddress("0x33")
	data, err := os.ReadFile("../contracts/stakedao/CreateVault.json")
	if err != nil {
		t.Fatal(err)
	}
	controllerABI, err := abi.JSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	event := controllerABI.Events["VaultRegistered"]
	eventData, err := event.Inputs.NonIndexed().Pack(common.HexToAddress("0x44"), [4]byte{0xc7, 0x15, 0xe3, 0x73})
	if err != nil {
		t.Fatal(err)
	}
	entry := types.Log{
		Address: controller, BlockNumber: 4, Data: eventData,
		Topics: []common.Hash{event.ID, common.BytesToHash(gauge.Bytes()), common.BytesToHash(vault.Bytes()), common.BytesToHash(asset.Bytes())},
	}
	tokenABI, err := abi.JSON(strings.NewReader(`[
		{"type":"function","name":"strategy","inputs":[],"outputs":[{"type":"address"}]},
		{"type":"function","name":"getRewardTokens","inputs":[],"outputs":[{"type":"address[]"}]},
		{"type":"function","name":"ACCOUNTANT","inputs":[],"outputs":[{"type":"address"}]},
		{"type":"function","name":"REWARD_TOKEN","inputs":[],"outputs":[{"type":"address"}]},
		{"type":"function","name":"name","inputs":[],"outputs":[{"type":"string"}]},
		{"type":"function","name":"symbol","inputs":[],"outputs":[{"type":"string"}]},
		{"type":"function","name":"decimals","inputs":[],"outputs":[{"type":"uint8"}]},
		{"type":"function","name":"totalAssets","inputs":[],"outputs":[{"type":"uint256"}]}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	multicallData, err := os.ReadFile("../contracts/curve/Multicall3Curve.json")
	if err != nil {
		t.Fatal(err)
	}
	multicallABI, err := abi.JSON(strings.NewReader(string(multicallData)))
	if err != nil {
		t.Fatal(err)
	}
	type multicallInput struct {
		Target       common.Address
		AllowFailure bool
		CallData     []byte
	}
	type multicallOutput struct {
		Success    bool
		ReturnData []byte
	}
	var executeCall func(common.Address, []byte) ([]byte, error)
	executeCall = func(to common.Address, input []byte) ([]byte, error) {
		if len(input) < 4 {
			return nil, fmt.Errorf("missing calldata")
		}
		if to == common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11") {
			method := multicallABI.Methods["aggregate3"]
			args, err := method.Inputs.Unpack(input[4:])
			if err != nil {
				return nil, err
			}
			calls := *abi.ConvertType(args[0], new([]multicallInput)).(*[]multicallInput)
			results := make([]multicallOutput, len(calls))
			for i, call := range calls {
				data, err := executeCall(call.Target, call.CallData)
				results[i] = multicallOutput{Success: err == nil, ReturnData: data}
			}
			return method.Outputs.Pack(results)
		}
		contractABI := tokenABI
		if to == controller {
			contractABI = controllerABI
		}
		method, err := contractABI.MethodById(input[:4])
		if err != nil {
			return nil, err
		}
		var value interface{}
		switch method.Name {
		case "gauge":
			return method.Outputs.Pack(vault, asset, common.HexToAddress("0x44"), [4]byte{0xc7, 0x15, 0xe3, 0x73}, false)
		case "strategy":
			value = common.Address{}
		case "getRewardTokens":
			value = []common.Address{}
		case "ACCOUNTANT", "accountant":
			value = common.HexToAddress("0x55")
		case "REWARD_TOKEN":
			value = common.HexToAddress("0x66")
		case "vault":
			value = vault
		case "asset":
			value = asset
		case "name":
			value = "Stake DAO HTTP fixture Vault"
		case "symbol":
			value = "FIX"
		case "decimals":
			value = uint8(6)
		case "totalAssets":
			value = big.NewInt(2500000)
		default:
			return nil, fmt.Errorf("unexpected method: %s", method.Name)
		}
		return method.Outputs.Pack(value)
	}
	var historyReads atomic.Int64
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		var result interface{}
		switch request.Method {
		case "eth_chainId":
			result = "0x1"
		case "eth_blockNumber":
			result = "0xa"
		case "eth_getBlockByNumber":
			result = &types.Header{Number: big.NewInt(10), Difficulty: big.NewInt(0), Extra: []byte{}}
		case "eth_getCode":
			historyReads.Add(1)
			var number string
			if err := json.Unmarshal(request.Params[1], &number); err != nil {
				t.Error(err)
			}
			block, err := hexutil.DecodeUint64(number)
			if err != nil {
				t.Error(err)
			}
			result = "0x"
			if block >= 3 {
				result = "0x01"
			}
		case "eth_getLogs":
			historyReads.Add(1)
			result = []types.Log{entry}
		case "eth_call":
			var call struct {
				To    common.Address `json:"to"`
				Input hexutil.Bytes  `json:"input"`
				Data  hexutil.Bytes  `json:"data"`
			}
			if err := json.Unmarshal(request.Params[0], &call); err != nil {
				t.Error(err)
			}
			input := call.Input
			if len(input) == 0 {
				input = call.Data
			}
			if len(input) < 4 {
				t.Error("missing calldata")
				w.WriteHeader(400)
				return
			}
			encoded, err := executeCall(call.To, input)
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			result = hexutil.Bytes(encoded)
		default:
			t.Errorf("unexpected RPC: %s", request.Method)
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": request.ID, "result": result}); err != nil {
			t.Error(err)
		}
	}))
	defer rpcServer.Close()
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", rpcServer.URL)
	store := prices.NewMemoryStore()
	if err := store.Set(context.Background(), prices.TokenKey{ChainID: 1, Address: asset}, prices.Quote{USD: decimal.NewFromInt(2), UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	server := newStakeDAOTestServer(prices.NewTokenPriceService(store, nil))
	response := httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum?details=true", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		BlockNumber uint64 `json:"blockNumber"`
		Pools       []struct {
			Name        string         `json:"name"`
			AssetAmount string         `json:"assetAmount"`
			TVL         *float64       `json:"tvl"`
			APY         *float64       `json:"apy"`
			RewardToken common.Address `json:"rewardToken"`
		} `json:"pools"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.BlockNumber != 10 || len(body.Pools) != 1 || body.Pools[0].Name != "Stake DAO HTTP fixture Vault" || body.Pools[0].AssetAmount != "2.500000" || body.Pools[0].TVL == nil || *body.Pools[0].TVL != 5 || body.Pools[0].APY != nil || body.Pools[0].RewardToken != common.HexToAddress("0x66") {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum", nil))
	var summary struct {
		Pools []map[string]json.RawMessage `json:"pools"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &summary) != nil || len(summary.Pools) != 1 {
		t.Fatalf("invalid compact response: %s", response.Body.String())
	}
	row := summary.Pools[0]
	if len(row) != 6 || string(row["address"]) != `"`+strings.ToLower(vault.Hex())+`"` || string(row["status"]) != `"unavailable"` || string(row["apr"]) != "null" || string(row["apy"]) != "null" || string(row["symbol"]) != `"FIX"` {
		t.Fatalf("compact schema lost vault identity or unknown APR: %s", response.Body.String())
	}
	before := historyReads.Load()
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/diagnostics", nil))
	var diagnostic struct {
		Pools []struct {
			Gauge common.Address `json:"gaugeAddress"`
		} `json:"pools"`
		Errors []struct {
			Code     string         `json:"code"`
			Stage    string         `json:"stage"`
			Contract common.Address `json:"contract"`
			Method   string         `json:"method"`
		} `json:"errors"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &diagnostic) != nil || len(diagnostic.Pools) != 2 || historyReads.Load() != before {
		t.Fatalf("diagnostic route must read only two current registrations without history: %s", response.Body.String())
	}
	if diagnostic.Pools[0].Gauge != common.HexToAddress("0x95f00391cb5eebcd190eb58728b4ce23dbfa6ac1") || diagnostic.Pools[1].Gauge != common.HexToAddress("0x009acd89535dabc270c93f9b39d3232105fef453") {
		t.Fatalf("diagnostic route selected other gauges: %+v", diagnostic.Pools)
	}
	if len(diagnostic.Errors) != 2 || diagnostic.Errors[0].Code != "primary_reward_strategy" || diagnostic.Errors[0].Stage != "strategy" || diagnostic.Errors[0].Contract != vault || diagnostic.Errors[0].Method != "strategy" {
		t.Fatalf("missing structured primary diagnostics: %s", response.Body.String())
	}
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/base", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unsupported network returned %d", response.Code)
	}
}

func TestStakeDAORouteDoesNotExposeRPCSecrets(t *testing.T) {
	rpcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	defer rpcServer.Close()
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", rpcServer.URL+"/private-api-key")
	server := newStakeDAOTestServer(prices.NewTokenPriceService(prices.NewMemoryStore(), nil))
	for _, path := range []string{"/reservesStakedao/ethereum", "/reservesStakedao/ethereum/diagnostics"} {
		response := httptest.NewRecorder()
		server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadGateway || strings.Contains(response.Body.String(), "private-api-key") {
			t.Fatalf("unsafe error response: %d %s", response.Code, response.Body.String())
		}
	}
	for _, path := range []string{"/reservesStakedao/base/diagnostics", "/reservesStakedao/ethereum?details=garbage"} {
		response := httptest.NewRecorder()
		server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid selection returned %d: %s", response.Code, response.Body.String())
		}
	}

}

func newStakeDAOTestServer(priceService prices.Provider) *Server {
	server := &Server{e: echo.New()}
	handler := NewHandler()
	handler.stakeDao = stakedao.NewStakeDAOService(priceService)
	server.routes(handler)
	return server
}
