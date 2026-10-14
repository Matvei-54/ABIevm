package internal

import (
	"ABIevm/internal/service/prices"
	"ABIevm/internal/service/stakedao"
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
			contractABI := tokenABI
			if call.To == controller {
				contractABI = controllerABI
			}
			method, err := contractABI.MethodById(input[:4])
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			var value interface{}
			switch method.Name {
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
				t.Errorf("unexpected method: %s", method.Name)
				w.WriteHeader(400)
				return
			}
			encoded, err := method.Outputs.Pack(value)
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
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum", nil))
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
	response := httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum", nil))
	if response.Code != http.StatusBadGateway || strings.Contains(response.Body.String(), "private-api-key") {
		t.Fatalf("unsafe error response: %d %s", response.Code, response.Body.String())
	}
}

func newStakeDAOTestServer(priceService prices.Provider) *Server {
	server := &Server{e: echo.New()}
	handler := NewHandler()
	handler.stakeDao = stakedao.NewStakeDAOService(priceService)
	server.e.GET("/reservesStakedao/:network", handler.HandleGetStakeDaoReserves)
	return server
}
