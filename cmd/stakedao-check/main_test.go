package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"ABIevm/internal/service/stakedao"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestInvalidCheckDoesNotMakeRequests(t *testing.T) {
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "")
	if err := run(context.Background(), nil, io.Discard, io.Discard); err == nil {
		t.Fatal("missing endpoint was accepted")
	}
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "https://must-not-connect.invalid")
	for _, args := range [][]string{{"-limit", "6"}, {"-from", "1", "-to", "101"}, {"-from", "2", "-to", "1"}, {"-from", "1"}, {"-gauges", "invalid"}, {"-gauges", "0x0000000000000000000000000000000000000000"}, {"unexpected"}} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
	for _, args := range [][]string{{"-diagnostics"}, {"-diagnostics", "-limit", "3"}, {"-diagnostics", "-limit", "5", "-gauges", "invalid"}, {"-diagnostics", "-limit", "2", "-from", "1", "-to", "2"}} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid diagnostics selection accepted: %v", args)
		}
	}
	for _, args := range [][]string{{"-block", "1"}, {"-strict-prices"}, {"-diagnostics", "-limit", "2", "-block", "0"}, {"-diagnostics", "-limit", "2", "-block", "-1"}} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid snapshot arguments accepted: %v", args)
		}
	}
}

func TestFileIsSavedOnlyAfterSuccessAndCanBeReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.json")
	want := []byte("{\"pools\": []}\n")
	if err := saveResult(path, want, io.Discard); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "")
	if err := run(context.Background(), []string{"-output", path}, io.Discard, io.Discard); err == nil {
		t.Fatal("missing endpoint was accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("failed check modified previous output: %s %v", got, err)
	}
	want = []byte("{\"pools\": [], \"network\": \"ethereum\"}\n")
	if err := saveResult(path, want, io.Discard); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("could not replace output: %s %v", got, err)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".stakedao-check-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatal("temporary result files were left behind")
	}
}

func TestBOLDCampaignFlagsRejectInvalidRequestsBeforeRPC(t *testing.T) {
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "https://must-not-connect.invalid")
	t.Setenv("STAKEDAO_ETHEREUM_LOGS_RPC_URL", "https://must-not-connect.invalid")
	for _, args := range [][]string{
		{"-bold-campaigns"},
		{"-bold-campaigns", "-from", "1"},
		{"-bold-campaigns", "-block", "2"},
		{"-bold-campaigns", "-from", "2", "-block", "1"},
		{"-bold-campaigns", "-from", "1", "-block", "240001"},
		{"-bold-campaigns", "-from", "1", "-block", "2", "-to", "2"},
		{"-bold-campaigns", "-from", "1", "-block", "2", "-diagnostics"},
		{"-bold-campaigns", "-from", "1", "-block", "2", "-limit", "1"},
		{"-bold-campaigns", "-from", "1", "-block", "2", "unexpected"},
	} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid BOLD request accepted: %v", args)
		}
	}
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "")
	if err := run(context.Background(), []string{"-bold-campaigns", "-from", "1", "-block", "2"}, io.Discard, io.Discard); err == nil {
		t.Fatal("missing primary RPC accepted")
	}
}

// Exercise the real CLI -> ethclient -> isolated inspector path, never mainnet.
func TestBOLDCampaignCommandUsesTwoRPCsAndSavesOnlySuccessfulJSON(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/curve/Multicall3Curve.json")
	if err != nil {
		t.Fatal(err)
	}
	multicall, err := abi.JSON(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	addressType, _ := abi.NewType("address", "", nil)
	uintType, _ := abi.NewType("uint256", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	bytes4Type, _ := abi.NewType("bytes4", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	boolType, _ := abi.NewType("bool", "", nil)
	encode := func(types []abi.Type, values ...interface{}) []byte {
		args := make(abi.Arguments, len(types))
		for i, typ := range types {
			args[i] = abi.Argument{Type: typ}
		}
		b, err := args.Pack(values...)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	selectID := func(signature string) string { return hexutil.Encode(crypto.Keccak256([]byte(signature))[:4]) }
	addresses := map[string]common.Address{
		selectID("rewardVault()"):          common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"),
		selectID("rewardRouter()"):         common.HexToAddress("0x9a207a85e372fcdac3014f945a65868f2a05ba12"),
		selectID("legacyRewardReceiver()"): common.HexToAddress("0x8e980f6d681a02a701d4ce5ef68f4786351a3d8f"),
		selectID("PROTOCOL_CONTROLLER()"):  common.HexToAddress("0x2d8BcE1FaE00a959354aCD9eBf9174337A64d4fb"),
		selectID("merkleDistributor()"):    common.HexToAddress("0xdbd24b092f686b12650ec1450e3a7138f714506c"),
	}
	type input struct {
		Target       common.Address
		AllowFailure bool
		CallData     []byte
	}
	type output struct {
		Success    bool
		ReturnData []byte
	}
	var contractCalls, logCalls atomic.Int64
	var failState atomic.Bool
	server := func(role string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				ID     json.RawMessage
				Method string
				Params []json.RawMessage
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			var result interface{}
			switch req.Method {
			case "eth_chainId":
				if role == "state" && failState.Load() {
					json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "error": map[string]interface{}{"code": -32000, "message": "https://provider.invalid/private-state-key failed"}})
					return
				}
				result = "0x1"
			case "eth_getBlockByNumber":
				var number string
				json.Unmarshal(req.Params[0], &number)
				n, err := hexutil.DecodeBig(number)
				if err != nil {
					t.Error("snapshot must be explicit")
					w.WriteHeader(400)
					return
				}
				result = &types.Header{Number: n, Time: 1795185215, Difficulty: big.NewInt(0), GasLimit: 30000000, Extra: []byte{}}
			case "eth_getLogs":
				if role != "logs" {
					t.Error("logs used primary RPC")
				}
				logCalls.Add(1)
				result = []types.Log{}
			case "eth_call":
				if role != "state" {
					t.Error("state used logs RPC")
				}
				contractCalls.Add(1)
				var call struct {
					Data  hexutil.Bytes
					Input hexutil.Bytes
				}
				json.Unmarshal(req.Params[0], &call)
				data := call.Data
				if len(data) == 0 {
					data = call.Input
				}
				values, err := multicall.Methods["aggregate3"].Inputs.Unpack(data[4:])
				if err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				calls := *abi.ConvertType(values[0], new([]input)).(*[]input)
				results := make([]output, 0, len(calls))
				for _, c := range calls {
					id := hexutil.Encode(c.CallData[:4])
					var encoded []byte
					if address, ok := addresses[id]; ok {
						encoded = encode([]abi.Type{addressType}, address)
					} else {
						switch id {
						case selectID("gauge(address)"):
							encoded = encode([]abi.Type{addressType, addressType, addressType, bytes4Type, boolType}, common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"), common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e"), common.HexToAddress("0xaee4d726c1ccdb82e79e7cf9922fa6f5b89320d3"), [4]byte{0xc7, 0x15, 0xe3, 0x73}, false)
						case selectID("nbIncentives()"):
							encoded = encode([]abi.Type{uintType}, big.NewInt(0))
						case selectID("root()"), selectID("ipfsHash()"):
							encoded = encode([]abi.Type{bytes32Type}, [32]byte{})
						case selectID("decimals()"):
							encoded = encode([]abi.Type{uint8Type}, uint8(18))
						default:
							t.Errorf("unexpected getter %s", id)
						}
					}
					results = append(results, output{Success: encoded != nil, ReturnData: encoded})
				}
				encoded, err := multicall.Methods["aggregate3"].Outputs.Pack(results)
				if err != nil {
					t.Error(err)
				}
				result = hexutil.Bytes(encoded)
			default:
				t.Errorf("unexpected method %s", req.Method)
				w.WriteHeader(400)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result})
		}))
	}
	state, logs := server("state"), server("logs")
	defer state.Close()
	defer logs.Close()
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", state.URL+"/private-state-key")
	t.Setenv("STAKEDAO_ETHEREUM_LOGS_RPC_URL", logs.URL+"/private-logs-key")
	path := filepath.Join(t.TempDir(), "bold.json")
	args := []string{"-bold-campaigns", "-from", "100", "-block", "101", "-output", path}
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report stakedao.BOLDCampaignReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if !report.ScanComplete || report.BlockNumber != 101 || report.CurrentAPR != nil || report.Status != "no_verified_campaigns_in_creation_window" || contractCalls.Load() != 1 || logCalls.Load() != 1 || stdout.Len() != 0 {
		t.Fatalf("wrong CLI result: %+v", report)
	}
	if strings.Contains(stderr.String(), "private-") || strings.Contains(string(data), "private-") {
		t.Fatal("RPC credentials leaked")
	}
	failState.Store(true)
	if err := run(context.Background(), args, &stdout, &stderr); err == nil || strings.Contains(err.Error(), "private-") {
		t.Fatalf("failure not propagated safely: %v", err)
	}
	previous, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(previous, data) {
		t.Fatal("failed check overwrote previous JSON")
	}
}
