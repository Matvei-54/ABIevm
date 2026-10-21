package stakedao

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type discoveryReader func(context.Context, ethereum.FilterQuery) ([]types.Log, error)

func (f discoveryReader) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	return f(ctx, q)
}

func TestDiscoveryUsesBoundedConcurrentWorkers(t *testing.T) {
	t.Setenv("STAKEDAO_DISCOVERY_WORKERS", "3")
	started := make(chan struct{}, 3)
	release := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var active atomic.Int32
	reader := discoveryReader(func(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
		n := active.Add(1)
		defer active.Add(-1)
		if n > 3 {
			t.Error("worker bound exceeded")
		}
		if q.ToBlock.Uint64()-q.FromBlock.Uint64()+1 != 1500 {
			t.Error("wrong query size")
		}
		started <- struct{}{}
		select {
		case <-release:
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	done := make(chan error, 1)
	go func() {
		_, err := DiscoverStakeDAOVaults(ctx, reader, common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB), 0, 4499)
		done <- err
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("RPC reads did not overlap")
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if active.Load() != 0 {
		t.Fatal("worker did not exit")
	}
}

func TestDiscoveryFailureCancelsOtherWorkersAndReturnsNoPartialData(t *testing.T) {
	t.Setenv("STAKEDAO_DISCOVERY_WORKERS", "4")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var active atomic.Int32
	reader := discoveryReader(func(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
		active.Add(1)
		defer active.Add(-1)
		if q.FromBlock.Uint64() == 0 {
			return nil, errors.New("provider range error")
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	result, err := DiscoverStakeDAOVaults(ctx, reader, common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB), 0, 1000000000)
	if err == nil || !strings.Contains(err.Error(), "provider range error") || result != nil || active.Load() != 0 {
		t.Fatalf("failure did not stop workers: %+v %v active=%d", result, err, active.Load())
	}
}

func TestDiscoveryRejectsLogFromWrongChunk(t *testing.T) {
	t.Setenv("STAKEDAO_DISCOVERY_WORKERS", "1")
	controller := common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)
	entry := registration(t, controller, common.HexToAddress("0x11"), common.HexToAddress("0x22"), common.HexToAddress("0x33"), 2000, 0)
	reader := discoveryReader(func(context.Context, ethereum.FilterQuery) ([]types.Log, error) { return []types.Log{entry}, nil })
	if _, err := DiscoverStakeDAOVaults(context.Background(), reader, controller, 0, 2999); err == nil {
		t.Fatal("accepted a log from outside its own query range")
	}
}

func TestDiscoveryLatestRegistrationDoesNotDependOnCompletionOrder(t *testing.T) {
	t.Setenv("STAKEDAO_DISCOVERY_WORKERS", "2")
	controller := common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)
	gauge, asset := common.HexToAddress("0x11"), common.HexToAddress("0x22")
	old := registration(t, controller, gauge, common.HexToAddress("0x33"), asset, 0, 0)
	latest := registration(t, controller, gauge, common.HexToAddress("0x44"), asset, 2999, 0)
	newerReturned := make(chan struct{})
	reader := discoveryReader(func(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
		if q.FromBlock.Uint64() == 0 {
			select {
			case <-newerReturned:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return []types.Log{old}, nil
		}
		close(newerReturned)
		return []types.Log{latest}, nil
	})
	result, err := DiscoverStakeDAOVaults(context.Background(), reader, controller, 0, 2999)
	if err != nil || len(result) != 1 || result[0].Vault != common.HexToAddress("0x44") {
		t.Fatalf("wrong registration %+v %v", result, err)
	}
}

func TestDiscoveryInvalidWorkerConfigDoesNotReadRPC(t *testing.T) {
	t.Setenv("STAKEDAO_DISCOVERY_WORKERS", "0")
	reader := discoveryReader(func(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
		t.Fatal("invalid worker config made RPC request")
		return nil, nil
	})
	if _, err := DiscoverStakeDAOVaults(context.Background(), reader, common.Address{}, 0, 100); err == nil {
		t.Fatal("invalid worker count accepted")
	}
}

func TestControllerStartBlockUsesConfiguredNetworkWithoutArchiveReads(t *testing.T) {
	s, f := newPoolFixture(t)
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil || len(result.Pools) != 1 || f.codeReads != 0 {
		t.Fatalf("archive state still used: code=%d err=%v", f.codeReads, err)
	}
	s.deploymentBlock = 100
	if _, err := s.controllerStartBlock(99); err == nil {
		t.Fatal("deployment after snapshot accepted")
	}
	s.deploymentBlock = 0
	if _, err := s.controllerStartBlock(100); err == nil {
		t.Fatal("missing deployment accepted")
	}
}

func TestConfiguredStartBlockAndInvalidOverride(t *testing.T) {
	t.Setenv("STAKEDAO_CONTROLLER_DEPLOYMENT_BLOCK", "123")
	s := NewStakeDAOService(nil)
	if s.networks["ethereum"].deploymentBlock != 123 {
		t.Fatal("start block override ignored")
	}
	t.Setenv("STAKEDAO_CONTROLLER_DEPLOYMENT_BLOCK", "invalid")
	s = NewStakeDAOService(nil)
	s.dial = func(context.Context, string) (stakeDAORPC, error) { t.Fatal("bad config dialed RPC"); return nil, nil }
	if _, err := s.GetPools(context.Background(), "ethereum"); err == nil {
		t.Fatal("bad deployment configuration accepted")
	}
}

// Make the huge-range arithmetic case explicit: range generation is streamed,
// and cancellation must work without allocating one object per chain chunk.
func TestDiscoveryHugeRangeHonorsAlreadyCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := discoveryReader(func(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
		t.Fatal("cancelled discovery read RPC")
		return nil, nil
	})
	_, err := DiscoverStakeDAOVaults(ctx, reader, common.Address{}, 0, ^uint64(0))
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
