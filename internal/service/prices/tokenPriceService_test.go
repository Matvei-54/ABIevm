package prices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type sourceFunc func(context.Context, int64, []common.Address) (map[common.Address]Quote, error)

func (f sourceFunc) Fetch(ctx context.Context, chain int64, tokens []common.Address) (map[common.Address]Quote, error) {
	return f(ctx, chain, tokens)
}

func TestCacheSeparatesChainsAndRejectsExpiredPrices(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	now := time.Now().UTC()
	token := common.HexToAddress("0x11")
	for chain, price := range map[int64]string{1: "1.25", 8453: "3.5"} {
		if err := store.Set(ctx, TokenKey{ChainID: chain, Address: token}, Quote{USD: decimal.RequireFromString(price), UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	service := NewTokenPriceService(store, nil)
	service.now = func() time.Time { return now }
	for chain, want := range map[int64]string{1: "1.25", 8453: "3.5"} {
		got, err := service.GetTokenPrice(ctx, token, chain)
		if err != nil || got.String() != want {
			t.Fatalf("chain %d: %s %v", chain, got, err)
		}
	}
	service.now = func() time.Time { return now.Add(16 * time.Minute) }
	if _, err := service.GetTokenPrice(ctx, token, 1); !errors.Is(err, ErrPriceUnavailable) {
		t.Fatalf("stale price accepted: %v", err)
	}
	if err := store.Set(ctx, TokenKey{ChainID: 1, Address: token}, Quote{USD: decimal.Zero, UpdatedAt: now}); err == nil {
		t.Fatal("zero price accepted")
	}
}

func TestConcurrentRefreshFetchesOnceAndReadsUseOnlyMemory(t *testing.T) {
	ctx := context.Background()
	token := common.HexToAddress("0x11")
	var fetches atomic.Int32
	service := NewTokenPriceService(NewMemoryStore(), sourceFunc(func(_ context.Context, chain int64, tokens []common.Address) (map[common.Address]Quote, error) {
		fetches.Add(1)
		if chain != 1 || len(tokens) != 1 || tokens[0] != token {
			t.Error("tokens were not deduplicated")
		}
		return map[common.Address]Quote{token: {USD: decimal.NewFromInt(2), UpdatedAt: time.Now()}}, nil
	}))
	var group sync.WaitGroup
	for i := 0; i < 25; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := service.EnsurePrices(ctx, 1, []common.Address{token, token, {}}); err != nil {
				t.Error(err)
			}
			if price, err := service.GetTokenPrice(ctx, token, 1); err != nil || !price.Equal(decimal.NewFromInt(2)) {
				t.Errorf("price: %s %v", price, err)
			}
		}()
	}
	group.Wait()
	if fetches.Load() != 1 {
		t.Fatalf("refreshes: %d", fetches.Load())
	}
	for i := 0; i < 10; i++ {
		_, _ = service.GetTokenPrice(ctx, token, 1)
	}
	if fetches.Load() != 1 {
		t.Fatal("cache reads made external requests")
	}
}

func TestMissingQuoteKeepsOtherQuotesAndDoesNotInventLPPrice(t *testing.T) {
	asset, reward := common.HexToAddress("0xaa"), common.HexToAddress("0xbb")
	service := NewTokenPriceService(NewMemoryStore(), sourceFunc(func(context.Context, int64, []common.Address) (map[common.Address]Quote, error) {
		return map[common.Address]Quote{reward: {USD: decimal.NewFromInt(2), UpdatedAt: time.Now()}}, nil
	}))
	if err := service.EnsurePrices(context.Background(), 1, []common.Address{asset, reward}); !errors.Is(err, ErrPriceUnavailable) {
		t.Fatalf("missing LP price: %v", err)
	}
	if _, err := service.GetTokenPrice(context.Background(), asset, 1); !errors.Is(err, ErrPriceUnavailable) {
		t.Fatal("LP price invented")
	}
	if price, err := service.GetTokenPrice(context.Background(), reward, 1); err != nil || !price.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("reward quote lost: %s %v", price, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.EnsurePrices(ctx, 1, []common.Address{asset}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
}

func TestLlamaSourceBatchesRequestsAndUsesSourceTimestamps(t *testing.T) {
	var requests atomic.Int32
	timestamp := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		keys := strings.Split(strings.TrimPrefix(r.URL.Path, "/prices/current/"), ",")
		if len(keys) > 50 {
			t.Errorf("batch size: %d", len(keys))
		}
		coins := map[string]interface{}{}
		for _, key := range keys {
			coins[key] = map[string]interface{}{"price": 1.23456789, "timestamp": timestamp}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"coins": coins})
	}))
	defer server.Close()
	source := NewLlamaSource(server.Client())
	source.baseURL = server.URL
	service := NewTokenPriceService(NewMemoryStore(), source)
	var tokens []common.Address
	for i := 1; i <= 51; i++ {
		tokens = append(tokens, common.HexToAddress(fmt.Sprintf("0x%x", i)))
	}
	if err := service.EnsurePrices(context.Background(), 1, tokens); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("requests: %d", requests.Load())
	}
	quote, ok, err := service.store.Get(context.Background(), TokenKey{ChainID: 1, Address: tokens[0]})
	if err != nil || !ok || quote.UpdatedAt.Unix() != timestamp || quote.USD.String() != "1.23456789" {
		t.Fatalf("wrong quote: %+v %v", quote, err)
	}
	if err := service.EnsurePrices(context.Background(), 1, tokens); err != nil || requests.Load() != 2 {
		t.Fatal("fresh quotes were fetched again")
	}
}

func TestLlamaHTTPFailureAndBadQuotesRemainUnavailable(t *testing.T) {
	for _, response := range []string{"{", `{"coins":{"ethereum:0x0000000000000000000000000000000000000011":{"price":-1,"timestamp":1}}}`, `{"coins":{"ethereum:0x0000000000000000000000000000000000000011":{"price":2}}}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) }))
		source := NewLlamaSource(server.Client())
		source.baseURL = server.URL
		service := NewTokenPriceService(NewMemoryStore(), source)
		if err := service.EnsurePrices(context.Background(), 1, []common.Address{common.HexToAddress("0x11")}); err == nil {
			t.Fatal("bad response produced a price")
		}
		server.Close()
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }))
	defer server.Close()
	source := NewLlamaSource(server.Client())
	source.baseURL = server.URL
	if _, err := source.Fetch(context.Background(), 1, []common.Address{common.HexToAddress("0x11")}); err == nil {
		t.Fatal("rate limit ignored")
	}
}

func TestFailedLaterBatchPreservesAlreadyFetchedPrices(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) > 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		coins := map[string]interface{}{}
		for _, key := range strings.Split(strings.TrimPrefix(r.URL.Path, "/prices/current/"), ",") {
			coins[key] = map[string]interface{}{"price": 2, "timestamp": time.Now().Unix()}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"coins": coins})
	}))
	defer server.Close()
	source := NewLlamaSource(server.Client())
	source.baseURL = server.URL
	service := NewTokenPriceService(NewMemoryStore(), source)
	var tokens []common.Address
	for i := 1; i <= 51; i++ {
		tokens = append(tokens, common.HexToAddress(fmt.Sprintf("0x%x", i)))
	}
	if err := service.EnsurePrices(context.Background(), 1, tokens); err == nil {
		t.Fatal("rate limit was ignored")
	}
	if price, err := service.GetTokenPrice(context.Background(), tokens[0], 1); err != nil || !price.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("first batch was lost: %s %v", price, err)
	}
	if _, err := service.GetTokenPrice(context.Background(), tokens[50], 1); !errors.Is(err, ErrPriceUnavailable) {
		t.Fatal("failed batch produced a price")
	}
}
