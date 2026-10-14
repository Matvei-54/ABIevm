package prices

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

var ErrPriceUnavailable = errors.New("token USD price unavailable")

type TokenKey struct {
	ChainID int64
	Address common.Address
}

type Quote struct {
	USD       decimal.Decimal
	UpdatedAt time.Time
}

// Store is the boundary for replacing the in-memory map with Redis later.
type Store interface {
	Get(context.Context, TokenKey) (Quote, bool, error)
	Set(context.Context, TokenKey, Quote) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	quotes map[TokenKey]Quote
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{quotes: make(map[TokenKey]Quote)} }

func (m *MemoryStore) Get(ctx context.Context, key TokenKey) (Quote, bool, error) {
	if err := ctx.Err(); err != nil {
		return Quote{}, false, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	quote, ok := m.quotes[key]
	return quote, ok, nil
}

func (m *MemoryStore) Set(ctx context.Context, key TokenKey, quote Quote) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if key.ChainID <= 0 || key.Address == (common.Address{}) || !quote.USD.IsPositive() || quote.UpdatedAt.IsZero() {
		return errors.New("invalid token price quote")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quotes[key] = quote
	return nil
}

type Source interface {
	Fetch(context.Context, int64, []common.Address) (map[common.Address]Quote, error)
}

type Provider interface {
	EnsurePrices(context.Context, int64, []common.Address) error
	GetTokenPrice(context.Context, common.Address, int64) (decimal.Decimal, error)
}

// TokenPriceService is shared between platform services for the application's lifetime.
// Reads use the store only. EnsurePrices batches the missing or expired quotes.
type TokenPriceService struct {
	store   Store
	source  Source
	maxAge  time.Duration
	now     func() time.Time
	refresh chan struct{}
}

func NewTokenPriceService(store Store, source Source) *TokenPriceService {
	return &TokenPriceService{store: store, source: source, maxAge: 15 * time.Minute, now: time.Now, refresh: make(chan struct{}, 1)}
}

func (s *TokenPriceService) GetTokenPrice(ctx context.Context, token common.Address, chainID int64) (decimal.Decimal, error) {
	quote, found, err := s.store.Get(ctx, TokenKey{ChainID: chainID, Address: token})
	if err != nil {
		return decimal.Zero, err
	}
	age := s.now().Sub(quote.UpdatedAt)
	if !found || !quote.USD.IsPositive() || age > s.maxAge || age < -time.Minute {
		return decimal.Zero, fmt.Errorf("%w: chain %d token %s", ErrPriceUnavailable, chainID, token.Hex())
	}
	return quote.USD, nil
}

func (s *TokenPriceService) EnsurePrices(ctx context.Context, chainID int64, tokens []common.Address) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chainID <= 0 {
		return errors.New("invalid price chain ID")
	}
	select {
	case s.refresh <- struct{}{}:
		defer func() { <-s.refresh }()
	case <-ctx.Done():
		return ctx.Err()
	}
	missing := make([]common.Address, 0, len(tokens))
	seen := make(map[common.Address]bool)
	for _, token := range tokens {
		if token == (common.Address{}) || seen[token] {
			continue
		}
		seen[token] = true
		if _, err := s.GetTokenPrice(ctx, token, chainID); err != nil {
			if !errors.Is(err, ErrPriceUnavailable) {
				return err
			}
			missing = append(missing, token)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if s.source == nil {
		return ErrPriceUnavailable
	}
	quotes, fetchErr := s.source.Fetch(ctx, chainID, missing)
	for _, token := range missing {
		if quote, ok := quotes[token]; ok {
			if err := s.store.Set(ctx, TokenKey{ChainID: chainID, Address: token}, quote); err != nil {
				return err
			}
		}
	}
	if fetchErr != nil {
		return fetchErr
	}
	for _, token := range missing {
		if _, err := s.GetTokenPrice(ctx, token, chainID); err != nil {
			return err
		}
	}
	return nil
}
