package internal

import (
	"ABIevm/internal/service/prices"
	"ABIevm/internal/service/stakedao"
	"context"
	"encoding/json"
	"errors"
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
		{"type":"function","name":"is_killed","inputs":[],"outputs":[{"type":"bool"}]},
		{"type":"function","name":"gauge_relative_weight","inputs":[{"type":"address"},{"type":"uint256"}],"outputs":[{"type":"uint256"}]},
		{"type":"function","name":"poolInfo","inputs":[{"type":"uint256"}],"outputs":[{"type":"address"},{"type":"address"},{"type":"address"},{"type":"address"},{"type":"address"},{"type":"bool"}]},
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
		case "isPaused", "is_killed":
			value = false
		case "gauge_relative_weight":
			value = new(big.Int)
		case "poolInfo":
			return method.Outputs.Pack(common.Address{}, common.Address{}, common.Address{}, common.Address{}, common.Address{}, false)
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
			var selected string
			if err := json.Unmarshal(request.Params[0], &selected); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			number, err := hexutil.DecodeUint64(selected)
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			result = &types.Header{Number: new(big.Int).SetUint64(number), Difficulty: big.NewInt(0), Extra: []byte{}}
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
			APRScope    string         `json:"aprScope"`
			State       struct {
				Registered *bool `json:"registered"`
				HasAssets  *bool `json:"hasAssets"`
			} `json:"state"`
			Yield struct {
				Status string `json:"status"`
				Merkl  struct {
					APR    *float64 `json:"apr"`
					Status string   `json:"status"`
				} `json:"merkl"`
				Trading struct {
					APY     *float64 `json:"apy"`
					Reasons []string `json:"reasons"`
				} `json:"trading"`
			} `json:"yieldBreakdown"`
		} `json:"pools"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.BlockNumber != 10 || len(body.Pools) != 1 || body.Pools[0].Name != "Stake DAO HTTP fixture Vault" || body.Pools[0].AssetAmount != "2.500000" || body.Pools[0].TVL == nil || *body.Pools[0].TVL != 5 || body.Pools[0].APY != nil || body.Pools[0].RewardToken != common.HexToAddress("0x66") {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
	if body.Pools[0].APRScope != "vault_rewards" || body.Pools[0].Yield.Status != "partial" || body.Pools[0].Yield.Merkl.APR != nil || body.Pools[0].Yield.Merkl.Status != "unavailable" || body.Pools[0].Yield.Trading.APY != nil || len(body.Pools[0].Yield.Trading.Reasons) != 1 {
		t.Fatalf("missing yield scopes or fabricated unknowns: %s", response.Body.String())
	}
	if body.Pools[0].State.Registered == nil || !*body.Pools[0].State.Registered || body.Pools[0].State.HasAssets == nil || !*body.Pools[0].State.HasAssets {
		t.Fatalf("state depends on unavailable primary APR: %s", response.Body.String())
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
			Gauge      common.Address `json:"gaugeAddress"`
			APRReasons []string       `json:"aprReasons"`
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
	if len(diagnostic.Pools[0].APRReasons) != 1 || diagnostic.Pools[0].APRReasons[0] != "reward_data_missing" {
		t.Fatalf("missing APR reason: %s", response.Body.String())
	}
	before = historyReads.Load()
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/diagnostics?limit=3&gauges=0x0000000000000000000000000000000000000011,0x0000000000000000000000000000000000000012,0x0000000000000000000000000000000000000013", nil))
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &diagnostic) != nil || len(diagnostic.Pools) != 3 || diagnostic.Pools[0].Gauge != common.HexToAddress("0x11") || historyReads.Load() != before {
		t.Fatalf("custom diagnostics used history or wrong gauges: %s", response.Body.String())
	}
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/diagnostics?gauges=0x0000000000000000000000000000000000000011&block=9&strictPrices=false", nil))
	var fixed struct {
		Requested *uint64 `json:"requestedBlockNumber"`
		Block     uint64  `json:"blockNumber"`
		Pools     []struct {
			State struct {
				Block uint64 `json:"blockNumber"`
			} `json:"state"`
		} `json:"pools"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &fixed) != nil || fixed.Requested == nil || *fixed.Requested != 9 || fixed.Block != 9 || len(fixed.Pools) != 1 || fixed.Pools[0].State.Block != 9 {
		t.Fatalf("HTTP requested block lost, decremented or replaced by head: %s", response.Body.String())
	}
	before = historyReads.Load()
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/state?gauges=0x0000000000000000000000000000000000000011&block=10", nil))
	var states struct {
		Gauges []struct {
			State struct {
				Registered  *bool `json:"registered"`
				Shutdown    *bool `json:"shutdown"`
				GaugeKilled *bool `json:"gaugeKilled"`
			} `json:"state"`
		} `json:"gauges"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &states) != nil || len(states.Gauges) != 1 || states.Gauges[0].State.Registered == nil || !*states.Gauges[0].State.Registered || states.Gauges[0].State.Shutdown == nil || *states.Gauges[0].State.Shutdown || historyReads.Load() != before || strings.Contains(response.Body.String(), "\"apr\"") {
		t.Fatalf("state route used history/APR or lost registration: %s", response.Body.String())
	}
	for _, query := range []string{"", "gauges=0x11", "gauges=a,b,c,d,e,f", "gauges=0x0000000000000000000000000000000000000011&block=0", "gauges=0x0000000000000000000000000000000000000011&block=latest"} {
		response = httptest.NewRecorder()
		server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/state?"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid state query accepted: %s", query)
		}
	}
	before = historyReads.Load()
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/diagnostics?limit=5", nil))
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &diagnostic) != nil || len(diagnostic.Pools) != 5 || historyReads.Load() != before {
		t.Fatalf("five-pool diagnostics used history or returned wrong selection: %s", response.Body.String())
	}
	for _, limit := range []string{"1", "6", "bad"} {
		response = httptest.NewRecorder()
		server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/diagnostics?limit="+limit, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid diagnostics limit accepted: %s", limit)
		}
	}
	for _, query := range []string{"block=0", "block=-1", "block=", "block=latest", "block=0xff", "block=18446744073709551616", "strictPrices=bad", "strictPrices=", "gauges=", "gauges=0x11", "gauges=0x0000000000000000000000000000000000000000", "gauges=0x0000000000000000000000000000000000000011,0x0000000000000000000000000000000000000011", "gauges=a,b,c,d,e,f", "weethFeeDays=7", "weethFeeDays=0&gauges=0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2", "weethFeeDays=2&gauges=0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2", "weethFeeDays=7&gauges=0x0000000000000000000000000000000000000011"} {
		response = httptest.NewRecorder()
		server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/diagnostics?"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid snapshot argument accepted: %s: %s", query, response.Body.String())
		}
	}
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/diagnostics?block=18446744073709551615", nil))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "beyond the current head") {
		t.Fatalf("future snapshot error lost: %s", response.Body.String())
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
	for _, path := range []string{"/reservesStakedao/ethereum", "/reservesStakedao/ethereum/diagnostics", "/reservesStakedao/ethereum/state?gauges=0x0000000000000000000000000000000000000011"} {
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

func TestBOLDCampaignSwaggerRouteIsSeparateAndReturnsShareableJSON(t *testing.T) {
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "https://must-not-connect.invalid")
	store := &unavailableRegistry{}
	handler := NewHandlerWithRegistry(store)
	server := &Server{e: echo.New()}
	server.routes(handler)
	swagger := httptest.NewRecorder()
	server.e.ServeHTTP(swagger, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	var spec struct {
		Paths map[string]struct {
			Get struct {
				OperationID string `json:"operationId"`
				Parameters  []struct {
					Name     string
					Required bool
					Default  interface{}
				}
			}
		}
		Definitions map[string]struct {
			Properties map[string]struct {
				Type string
				Ref  string `json:"$ref"`
			}
		}
	}
	if swagger.Code != 200 || json.Unmarshal(swagger.Body.Bytes(), &spec) != nil {
		t.Fatalf("Swagger document unavailable: %d", swagger.Code)
	}
	operation := spec.Paths["/reservesStakedao/{network}/bold-campaigns"].Get
	if operation.OperationID != "getStakeDaoBOLDCampaigns" || len(operation.Parameters) != 3 || spec.Definitions["stakedao.BOLDCampaignReport"].Properties["blockHash"].Type != "string" {
		t.Fatal("Swagger is missing the callable BOLD method or correct JSON hash type")
	}
	if spec.Definitions["stakedao.BOLDHistoricalValuation"].Properties["assetHistory"].Ref != "#/definitions/stakedao.BOLDAssetHistory" || spec.Definitions["stakedao.BOLDAssetCheckpoint"].Properties["matched"].Type != "boolean" || spec.Definitions["stakedao.BOLDAssetEvent"].Properties["transactionHash"].Type != "string" {
		t.Fatal("Swagger lost capital history schemas")
	}
	for _, p := range operation.Parameters {
		if !p.Required || p.Name != "network" && p.Default == nil {
			t.Fatal("Swagger is missing required parameters or historical example defaults")
		}
	}
	called := 0
	handler.boldCampaigns = func(_ context.Context, network string, from, block uint64) (*stakedao.BOLDCampaignReport, error) {
		called++
		if network != "ethereum" || from != 26450710 || block != 26456929 {
			t.Fatal("incorrect selection forwarded")
		}
		return &stakedao.BOLDCampaignReport{ScanComplete: true, FromBlock: from, BlockNumber: block, Status: "funding_verified_allocation_unavailable",
			Gauge: common.HexToAddress("0x07a01471fa544d9c6531b631e6a96a79a9ad05e9"),
			Campaigns: []stakedao.BOLDCampaignRecord{
				{ID: 2593, FundingStatus: "verified_router_campaign_funding", AllocationStatus: "published_allocation_verified", Reasons: []string{"historical_assets_not_reconciled"}, Allocation: &stakedao.BOLDAllocationEvidence{Valuation: &stakedao.BOLDHistoricalValuation{Status: "unavailable", AssetHistory: &stakedao.BOLDAssetHistory{Status: "unavailable", ScanComplete: true, Checkpoints: []stakedao.BOLDAssetCheckpoint{{BlockNumber: 26445000, ObservedRaw: "100", ReconstructedRaw: "200", Matched: false}}}}}},
				{ID: 2594, FundingStatus: "verified_router_campaign_funding", AllocationStatus: "unavailable", Reasons: []string{"allocation_rules_unverified"}},
			}}, nil
	}
	const path = "/reservesStakedao/ethereum/bold-campaigns?fromBlock=26450710&block=26456929"
	response := httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	var report stakedao.BOLDCampaignReport
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &report) != nil || len(report.Campaigns) != 2 || report.Campaigns[0].ID != 2593 || report.Campaigns[1].ID != 2594 || report.CurrentAPR != nil || called != 1 || store.called {
		t.Fatalf("research route did not return its isolated report: %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"historicalApr":null`) || !strings.Contains(response.Body.String(), `"currentApr":null`) || strings.Contains(response.Body.String(), `"pools"`) || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		t.Fatal("nullable research JSON was mixed with pool response")
	}
	history := report.Campaigns[0].Allocation.Valuation.AssetHistory
	if history == nil || !history.ScanComplete || len(history.Checkpoints) != 1 || history.Checkpoints[0].Matched || history.Checkpoints[0].ReconstructedRaw != "200" {
		t.Fatal("capital diagnostic evidence lost in HTTP JSON")
	}
	for _, query := range []string{"", "fromBlock=1", "block=2", "fromBlock=0&block=1", "fromBlock=-1&block=2", "fromBlock=2&block=1", "fromBlock=1&block=240001", "fromBlock=1&block=0x10", "fromBlock=1&block=latest", "fromBlock=1&block=18446744073709551616", "fromBlock=1&block=2&block=3", "fromBlock=1&fromBlock=2&block=3"} {
		response = httptest.NewRecorder()
		server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum/bold-campaigns?"+query, nil))
		if response.Code != 400 || called != 1 {
			t.Fatalf("invalid selection reached RPC: %s -> %d", query, response.Code)
		}
	}
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/reservesStakedao/base/bold-campaigns?fromBlock=1&block=2", nil))
	if response.Code != 400 || called != 1 {
		t.Fatal("unsupported network reached inspector")
	}
	for _, tc := range []struct {
		err    error
		status int
		reason string
	}{
		{errors.New("https://provider.invalid/private-token: internal RPC message"), 502, "bold_campaign_check_failed"},
		{context.DeadlineExceeded, 504, "request_timeout"},
		{context.Canceled, 408, "request_canceled"},
		{stakedao.ErrBOLDCampaignRPCConfig, 503, "bold_campaign_rpc_not_configured"},
		{stakedao.ErrBOLDCampaignInvalidRequest, 400, "bold_campaign_invalid_request"},
	} {
		handler.boldCampaigns = func(context.Context, string, uint64, uint64) (*stakedao.BOLDCampaignReport, error) {
			return nil, tc.err
		}
		response = httptest.NewRecorder()
		server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		var body map[string]string
		json.Unmarshal(response.Body.Bytes(), &body)
		if response.Code != tc.status || body["reason"] != tc.reason || strings.Contains(response.Body.String(), "private-token") || strings.Contains(response.Body.String(), "provider.invalid") {
			t.Fatalf("unsafe/incorrect failure: %d %s", response.Code, response.Body.String())
		}
	}
	handler.boldCampaigns = func(context.Context, string, uint64, uint64) (*stakedao.BOLDCampaignReport, error) {
		return &stakedao.BOLDCampaignReport{ScanComplete: true, FromBlock: 26450710, BlockNumber: 26456928}, nil
	}
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != 502 || !strings.Contains(response.Body.String(), "bold_campaign_response_mismatch") {
		t.Fatal("mismatched snapshot accepted")
	}
	if store.called {
		t.Fatal("campaign inspection touched registry")
	}
}

func TestBOLDCampaignRouteUsesRealInspectorAndRedactsRPCFailure(t *testing.T) {
	var calls atomic.Int32
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage
			Method string
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		calls.Add(1)
		if req.Method != "eth_chainId" {
			t.Errorf("unexpected method %s", req.Method)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "error": map[string]interface{}{"code": -32000, "message": "https://provider.invalid/private-token unavailable"}})
	}))
	defer rpc.Close()
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", rpc.URL+"/private-token")
	t.Setenv("STAKEDAO_ETHEREUM_LOGS_RPC_URL", "")
	store := &unavailableRegistry{}
	server := NewServerWithRegistry(store)
	const path = "/reservesStakedao/ethereum/bold-campaigns?fromBlock=26450710&block=26456929"
	response := httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != 502 || calls.Load() != 1 || store.called || !strings.Contains(response.Body.String(), "bold_campaign_chain_mismatch_or_unavailable") || strings.Contains(response.Body.String(), "private-token") {
		t.Fatalf("real inspector failure not exposed safely: %d %s", response.Code, response.Body.String())
	}
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "")
	response = httptest.NewRecorder()
	server.e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != 503 || calls.Load() != 1 || !strings.Contains(response.Body.String(), "bold_campaign_rpc_not_configured") {
		t.Fatal("missing RPC configuration triggered a request")
	}
}
