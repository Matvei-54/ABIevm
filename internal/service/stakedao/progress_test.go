package stakedao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type waitingDiscoveryRPC struct{ *fixtureRPC }

func (c *waitingDiscoveryRPC) FilterLogs(ctx context.Context, _ ethereum.FilterQuery) ([]types.Log, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func captureLoadLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buffer bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buffer, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buffer
}

func loadLogRecords(t *testing.T, buffer *bytes.Buffer) []map[string]interface{} {
	t.Helper()
	var records []map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(buffer.String()), "\n") {
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func TestRequestDeadlineReportsDiscoveryStageAndStopsRPCWorkers(t *testing.T) {
	buffer := captureLoadLogs(t)
	s, fixture := newPoolFixture(t)
	s.dial = func(context.Context, string) (stakeDAORPC, error) { return &waitingDiscoveryRPC{fixture}, nil }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	result, err := s.GetPools(ctx, "ethereum")
	if !errors.Is(err, context.DeadlineExceeded) || result != nil || !fixture.closed {
		t.Fatalf("deadline lost or client left open: result=%v err=%v closed=%v", result, err, fixture.closed)
	}
	records := loadLogRecords(t, buffer)
	last := records[len(records)-1]
	if last["event"] != "request_deadline" || last["failure"] != "request_timeout" || last["stage"] != "discovery" || last["ranges_total"] != float64(3) || last["ranges_done"] != float64(0) || last["rpc_active"] != float64(0) || last["rpc_logs"].(float64) < 1 {
		t.Fatalf("wrong deadline diagnostics: %+v", last)
	}
}

func TestUpstreamTimeoutAndHTTPFailureAreDistinguishedWithoutSecrets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		failure string
	}{
		{"timeout", &url.Error{Op: "Post", URL: "https://provider.example/private-api-key", Err: context.DeadlineExceeded}, "rpc_timeout"},
		{"quota", rpc.HTTPError{StatusCode: 429, Status: "429 Too Many Requests", Body: []byte("private-api-key")}, "load_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buffer := captureLoadLogs(t)
			s, fixture := newPoolFixture(t)
			fixture.filterFailure = tc.err
			if result, err := s.GetPools(context.Background(), "ethereum"); err == nil || result != nil {
				t.Fatal("failed discovery returned a successful response")
			}
			if strings.Contains(buffer.String(), "private-api-key") || strings.Contains(buffer.String(), "provider.example") {
				t.Fatalf("credentials exposed in logs: %s", buffer)
			}
			records := loadLogRecords(t, buffer)
			last := records[len(records)-1]
			if last["event"] != "failed" || last["failure"] != tc.failure || last["stage"] != "discovery" || last["rpc_errors"].(float64) < 1 || last["rpc_active"] != float64(0) {
				t.Fatalf("RPC failure misclassified: %+v", last)
			}
			if tc.name == "quota" && (last["rpc_http_status"] != float64(429) || last["first_rpc_http_status"] != float64(429)) {
				t.Fatalf("quota status lost: %+v", last)
			}
		})
	}
}

func TestSuccessfulLoadReportsCompletedStagesAndCounts(t *testing.T) {
	buffer := captureLoadLogs(t)
	s, _ := newPoolFixture(t)
	if _, err := s.GetPools(context.Background(), "ethereum"); err != nil {
		t.Fatal(err)
	}
	records := loadLogRecords(t, buffer)
	stages := make(map[string]bool)
	for _, record := range records {
		stages[record["stage"].(string)] = true
	}
	last := records[len(records)-1]
	if !stages["pools"] || !stages["prices"] || last["event"] != "completed" || last["stage"] != "verify_snapshot" || last["ranges_done"] != float64(3) || last["pools_loaded"] != float64(1) || last["rpc_active"] != float64(0) || last["rpc_calls"].(float64) < 1 {
		t.Fatalf("incomplete progress counters: %+v", last)
	}
}
