package prices

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type LlamaSource struct {
	client  *http.Client
	baseURL string
}

func NewLlamaSource(client *http.Client) *LlamaSource {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &LlamaSource{client: client, baseURL: "https://coins.llama.fi"}
}

func (s *LlamaSource) Fetch(ctx context.Context, chainID int64, tokens []common.Address) (map[common.Address]Quote, error) {
	chain, ok := map[int64]string{1: "ethereum", 42161: "arbitrum", 8453: "base", 10: "optimism", 137: "polygon"}[chainID]
	if !ok {
		return nil, errors.New("price source does not support this chain")
	}
	result := make(map[common.Address]Quote)
	for start := 0; start < len(tokens); start += 50 {
		end := start + 50
		if end > len(tokens) {
			end = len(tokens)
		}
		keys := make([]string, 0, end-start)
		for _, token := range tokens[start:end] {
			keys = append(keys, chain+":"+strings.ToLower(token.Hex()))
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/prices/current/"+strings.Join(keys, ","), nil)
		if err != nil {
			return result, err
		}
		response, err := s.client.Do(request)
		if err != nil {
			return result, err
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			return result, fmt.Errorf("price source returned HTTP %d", response.StatusCode)
		}
		var body struct {
			Coins map[string]struct {
				Price     decimal.Decimal `json:"price"`
				Timestamp int64           `json:"timestamp"`
			} `json:"coins"`
		}
		err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&body)
		response.Body.Close()
		if err != nil {
			return result, fmt.Errorf("decode prices: %w", err)
		}
		for i, key := range keys {
			coin, found := body.Coins[key]
			if found && coin.Price.IsPositive() && coin.Timestamp > 0 {
				result[tokens[start+i]] = Quote{USD: coin.Price, UpdatedAt: time.Unix(coin.Timestamp, 0).UTC()}
			}
		}
	}
	return result, nil
}
