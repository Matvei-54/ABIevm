package internal

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ABIevm/internal/service/stakedao"
	"github.com/ethereum/go-ethereum/core/types"
)

type unavailableRegistry struct{ called bool }

func (s *unavailableRegistry) Acquire(context.Context, stakedao.RegistryKey, uint64) (stakedao.RegistrySession, error) {
	s.called = true
	return nil, errors.Join(stakedao.ErrRegistryUnavailable, errors.New("private-db-password"))
}

func TestEchoInjectsRegistryAndReturnsSafe503(t *testing.T) {
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
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
		default:
			t.Errorf("unexpected RPC before registry acquire: %s", request.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": request.ID, "result": result}); err != nil {
			t.Error(err)
		}
	}))
	defer rpc.Close()
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", rpc.URL)
	t.Setenv("STAKEDAO_CONTROLLER_DEPLOYMENT_BLOCK", "1")
	store := &unavailableRegistry{}
	server := NewServerWithRegistry(store)
	recorder := httptest.NewRecorder()
	server.e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/reservesStakedao/ethereum", nil))
	if !store.called || recorder.Code != http.StatusServiceUnavailable || strings.Contains(recorder.Body.String(), "private-db-password") {
		t.Fatalf("registry injection/503 failed: %d %s called=%v", recorder.Code, recorder.Body.String(), store.called)
	}
}
