package rpcclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDifferentRPCClientsShareTheProcessLimit(t *testing.T) {
	t.Setenv("RPC_MAX_RPS", "20")
	var mu sync.Mutex
	var arrivals []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		arrivals = append(arrivals, time.Now())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": request.ID, "result": "0x1"})
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a, err := DialContext(ctx, server.URL+"/aave")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := DialContext(ctx, server.URL+"/stakedao")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_, err := a.ChainID(ctx)
				if err != nil {
					t.Error(err)
				}
			} else {
				_, err := b.BlockNumber(ctx)
				if err != nil {
					t.Error(err)
				}
			}
		}(i)
	}
	wg.Wait()
	if len(arrivals) != 8 {
		t.Fatalf("got %d RPC requests", len(arrivals))
	}
	// Permit scheduling/network jitter while catching a separate limit per client.
	if elapsed := arrivals[len(arrivals)-1].Sub(arrivals[0]); elapsed < 7*shared.limiter.interval*8/10 {
		t.Fatalf("clients bypassed shared pacing: %s", elapsed)
	}
}

func TestLimiterCancellationDoesNotReserveFutureSlots(t *testing.T) {
	l := newLimiter(20)
	l.next = time.Now().Add(time.Hour)
	previous := l.next
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := l.wait(ctx, 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("wait did not honor deadline", err)
	}
	if l.next != previous {
		t.Fatal("cancelled waiter consumed future quota")
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if err := l.wait(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled request was allowed", err)
	}
}

type recordingTransport struct{ bodies []string }

func (r *recordingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	r.bodies = append(r.bodies, string(data))
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
}

func TestJSONRPCBatchCountsMethodsAndPreservesBody(t *testing.T) {
	base := &recordingTransport{}
	l := newLimiter(20)
	transport := &limitedTransport{base: base, limit: l}
	batch := `[{"jsonrpc":"2.0","id":1,"method":"eth_chainId"},{"jsonrpc":"2.0","id":2,"method":"eth_blockNumber"}]`
	req, _ := http.NewRequest(http.MethodPost, "http://rpc.invalid", strings.NewReader(batch))
	r, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	started := time.Now()
	if err := l.wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 2*l.interval*9/10 {
		t.Fatal("batch counted as only one method")
	}
	if len(base.bodies) != 1 || base.bodies[0] != batch {
		t.Fatal("RPC body was modified")
	}
	if err := l.wait(context.Background(), 21); err == nil {
		t.Fatal("oversized batch was accepted")
	}
}

func TestCancelledTransportNeverSendsRPC(t *testing.T) {
	base := &recordingTransport{}
	transport := &limitedTransport{base: base, limit: newLimiter(20)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://rpc.invalid", strings.NewReader(`{"method":"eth_chainId"}`))
	if _, err := transport.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatal("request cancellation lost", err)
	}
	if len(base.bodies) != 0 {
		t.Fatal("cancelled RPC was sent")
	}
}
