// Package rpcclient shares one HTTP JSON-RPC rate limit across application clients.
package rpcclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var shared struct {
	once    sync.Once
	limiter *limiter
	err     error
}

// DialContext limits all HTTP RPC methods, including generated bindings.
// The quota is shared by Aave, Curve and StakeDAO in this application process.
// Requests from other programs/instances are not covered.
func DialContext(ctx context.Context, endpoint string) (*ethclient.Client, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errors.New("shared RPC client requires an HTTP(S) endpoint")
	}
	shared.once.Do(func() {
		rps := 18 // Margin below the user's free 20 RPS quota.
		if text := os.Getenv("RPC_MAX_RPS"); text != "" {
			var err error
			rps, err = strconv.Atoi(text)
			if err != nil || rps < 1 || rps > 10000 {
				shared.err = errors.New("RPC_MAX_RPS must be an integer between 1 and 10000")
				return
			}
		}
		shared.limiter = newLimiter(rps)
	})
	if shared.err != nil {
		return nil, shared.err
	}
	client := &http.Client{Transport: &limitedTransport{base: http.DefaultTransport, limit: shared.limiter}, Timeout: 45 * time.Second}
	connection, err := rpc.DialOptions(ctx, endpoint, rpc.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return ethclient.NewClient(connection), nil
}

type limiter struct {
	gate     chan struct{}
	interval time.Duration
	next     time.Time
	rps      int
}

func newLimiter(rps int) *limiter {
	return &limiter{gate: make(chan struct{}, 1), interval: (time.Second + time.Duration(rps) - 1) / time.Duration(rps), rps: rps}
}

func (l *limiter) wait(ctx context.Context, cost int) error {
	if cost < 1 || cost > l.rps {
		return errors.New("RPC batch exceeds the configured per-second limit; split the batch")
	}
	select {
	case l.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-l.gate }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay := time.Until(l.next); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	l.next = time.Now().Add(l.interval * time.Duration(cost))
	return nil
}

type limitedTransport struct {
	base  http.RoundTripper
	limit *limiter
}

func (t *limitedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	delegated := false
	defer func() {
		if !delegated && request.Body != nil {
			request.Body.Close()
		}
	}()
	// A JSON-RPC batch is charged per method. A Multicall eth_call is one RPC
	// method, with a separate provider compute/gas cost. Preserve the body.
	var data []byte
	if request.GetBody != nil {
		body, err := request.GetBody()
		if err != nil {
			return nil, err
		}
		data, err = io.ReadAll(body)
		body.Close()
		if err != nil {
			return nil, err
		}
	} else if request.Body != nil {
		var err error
		data, err = io.ReadAll(request.Body)
		request.Body.Close()
		if err != nil {
			return nil, err
		}
		request.Body = io.NopCloser(bytes.NewReader(data))
	}
	cost := 1
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(trimmed, &batch); err != nil {
			return nil, errors.New("invalid JSON-RPC batch")
		}
		if len(batch) > 0 {
			cost = len(batch)
		}
	}
	if err := t.limit.wait(request.Context(), cost); err != nil {
		return nil, err
	}
	delegated = true
	return t.base.RoundTrip(request)
}
