package stakedao

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestControllerGaugeGetterMatchesRepositoryABI(t *testing.T) {
	data, err := os.ReadFile("../../../contracts/stakedao/CreateVault.json")
	if err != nil {
		t.Fatal(err)
	}
	full, err := abi.JSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	subset, err := abi.JSON(strings.NewReader(protocolControllerABI))
	if err != nil {
		t.Fatal(err)
	}
	want, got := full.Methods["gauge"], subset.Methods["gauge"]
	if got.Sig != want.Sig || len(got.Outputs) != len(want.Outputs) {
		t.Fatal("Controller.gauge ABI signature mismatch")
	}
	for i := range want.Outputs {
		if got.Outputs[i].Type.String() != want.Outputs[i].Type.String() {
			t.Fatalf("Controller.gauge output %d mismatch", i)
		}
	}
}

func TestExplicitGaugesReadCurrentRegistrationWithoutLogsOrCode(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary = true
	gauge := common.BytesToAddress(f.logs[0].Topics[1].Bytes())
	// No usable historical registrations are available at all.
	f.logs = nil
	result, err := s.GetPoolsForGauges(context.Background(), "ethereum", []common.Address{gauge}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.queries) != 0 || f.codeReads != 0 || !f.closed || len(result.Pools) != 1 {
		t.Fatalf("explicit sample used history: queries=%+v code=%d result=%+v", f.queries, f.codeReads, result)
	}
	pool := result.Pools[0]
	if pool.APRStatus != "available" || pool.TVL == nil || pool.APR == nil || pool.ProtocolID != "0xc715e373" || pool.VaultAddress != f.vaults[gauge] {
		t.Fatalf("did not use current onchain registration: %+v issues=%+v", pool, result.Errors)
	}
	for _, block := range f.callBlocks {
		if block != f.latest {
			t.Fatal("explicit sample used historical state")
		}
	}
}

func TestExplicitGaugesSkipUnregisteredAndShutdown(t *testing.T) {
	s, f := newPoolFixture(t)
	gauge := common.BytesToAddress(f.logs[0].Topics[1].Bytes())
	shutdown, unregistered := common.HexToAddress("0x11"), common.HexToAddress("0x12")
	f.vaults[shutdown], f.assets[shutdown] = f.vaults[gauge], f.assets[gauge]
	f.shutdown = map[common.Address]bool{shutdown: true}
	s.client, s.controller, s.chainID = f, f.controller, 1
	result, err := s.getPoolsSelected(context.Background(), "ethereum", &poolSample{limit: 3, gauges: []common.Address{shutdown, unregistered, gauge}})
	if err != nil || len(result.Pools) != 1 || result.Pools[0].GaugeAddress != gauge {
		t.Fatalf("selected inactive gauge: result=%+v err=%v", result, err)
	}
	_, err = s.getPoolsSelected(context.Background(), "ethereum", &poolSample{limit: 1, gauges: []common.Address{unregistered}})
	if err == nil || !strings.Contains(err.Error(), "no active vaults") {
		t.Fatalf("unregistered gauge accepted: %v", err)
	}
}

func TestInvalidExplicitGaugesRejectBeforeDial(t *testing.T) {
	s := NewStakeDAOService(nil)
	s.dial = func(context.Context, string) (stakeDAORPC, error) {
		t.Fatal("invalid candidates caused an RPC connection")
		return nil, nil
	}
	a, b := common.HexToAddress("0x11"), common.HexToAddress("0x12")
	for _, gauges := range [][]common.Address{nil, {{}}, {a, a}, {a, b, a, b}} {
		if _, err := s.GetPoolsForGauges(context.Background(), "ethereum", gauges, 1); err == nil {
			t.Fatal("invalid gauges accepted")
		}
	}
}

func TestSampleBoundsRejectBeforeDial(t *testing.T) {
	s := NewStakeDAOService(nil)
	s.dial = func(context.Context, string) (stakeDAORPC, error) {
		t.Fatal("invalid sample must not connect")
		return nil, nil
	}
	for _, tc := range []struct {
		fromBlock, toBlock uint64
		limit              int
	}{{1, 100, 0}, {1, 100, 4}, {2, 1, 1}, {1, 101, 1}, {0, ^uint64(0), 1}} {
		if _, err := s.GetPoolsSample(context.Background(), "ethereum", tc.fromBlock, tc.toBlock, tc.limit); err == nil {
			t.Fatalf("accepted invalid bounds: %+v", tc)
		}
	}
}

func TestSampleUsesCurrentStateWithoutHistoricalCode(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary = true
	result, err := s.GetPoolsSample(context.Background(), "ethereum", 123, 222, 1)
	if err != nil {
		t.Fatal(err)
	}
	if f.codeReads != 0 || !f.closed || len(f.queries) != 1 || f.queries[0].FromBlock.Uint64() != 123 || f.queries[0].ToBlock.Uint64() != 222 {
		t.Fatalf("unexpected discovery: code=%d, closed=%v, logs=%+v", f.codeReads, f.closed, f.queries)
	}
	if len(result.Pools) != 1 || result.Pools[0].APRStatus != "available" || result.Pools[0].TVL == nil || result.Pools[0].APR == nil || result.Pools[0].APY != nil {
		t.Fatalf("sample did not reuse pricing/rewards: %+v", result)
	}
	for _, b := range f.callBlocks {
		if b != f.latest {
			t.Fatalf("read historical state at %d instead of current snapshot %d", b, f.latest)
		}
	}
}

func TestSampleLimitsActivePoolsAndSkipsObsoleteRegistrations(t *testing.T) {
	for _, limit := range []int{1, 3} {
		s, f := newPoolFixture(t)
		// Keep the original active Curve vault and add three others, including
		// an obsolete registration that must never be attached to current state.
		for i := int64(1); i <= 4; i++ {
			gauge := common.BigToAddress(big.NewInt(1000 + i))
			vault := common.BigToAddress(big.NewInt(2000 + i))
			asset := common.BytesToAddress(f.logs[0].Topics[3].Bytes())
			f.logs = append(f.logs, registration(t, f.controller, gauge, vault, asset, 126, uint(i)))
			f.vaults[gauge], f.assets[gauge] = vault, asset
			if i == 1 {
				f.vaults[gauge] = common.HexToAddress("0xffff")
			}
		}
		s.client, s.controller, s.chainID = f, f.controller, 1
		result, err := s.getPoolsSelected(context.Background(), "ethereum", &poolSample{fromBlock: 123, toBlock: 222, limit: limit})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Pools) != limit || f.codeReads != 0 || len(f.queries) != 1 {
			t.Fatalf("limit %d: unexpected sample %+v", limit, result)
		}
		for _, pool := range result.Pools {
			if pool.GaugeAddress == common.BigToAddress(big.NewInt(1001)) {
				t.Fatal("included obsolete registration")
			}
		}
	}
}

func TestSampleRejectsEmptyWindowFutureWindowAndReorg(t *testing.T) {
	for _, kind := range []string{"empty", "future", "reorg"} {
		t.Run(kind, func(t *testing.T) {
			s, f := newPoolFixture(t)
			s.client, s.controller, s.chainID = f, f.controller, 1
			selection := &poolSample{fromBlock: 123, toBlock: 222, limit: 1}
			if kind == "empty" {
				f.logs = nil
			}
			if kind == "future" {
				selection.fromBlock, selection.toBlock = f.latest, f.latest+1
			}
			f.reorg = kind == "reorg"
			if _, err := s.getPoolsSelected(context.Background(), "ethereum", selection); err == nil {
				t.Fatal("sample falsely succeeded")
			}
		})
	}
}

func TestSampleRPCBudgetAndCancellation(t *testing.T) {
	f := &fixtureRPC{latest: 123}
	c := newSampleRPC(f)
	for i := 0; i < 300; i++ {
		c.next = time.Time{} // Exercise the real budget without waiting a minute.
		if _, err := c.BlockNumber(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.BlockNumber(context.Background()); err == nil || !strings.Contains(err.Error(), "budget") || !c.exhausted {
		t.Fatalf("budget not enforced: %v", err)
	}
	c = newSampleRPC(f)
	c.next = time.Now().Add(time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.BlockNumber(ctx); !errors.Is(err, context.Canceled) || c.remaining != 300 {
		t.Fatalf("cancelled call consumed budget: %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := c.BlockNumber(ctx); !errors.Is(err, context.DeadlineExceeded) || c.remaining != 300 {
		t.Fatalf("waiting for throttle ignored deadline: %v", err)
	}
}

func TestSampleDoesNotReturnSuccessWhenBudgetRunsOutDuringMetadata(t *testing.T) {
	s, f := newPoolFixture(t)
	c := newSampleRPC(f)
	c.remaining = 6
	s.client, s.controller, s.chainID = c, f.controller, 1
	result, err := s.getPoolsSelected(context.Background(), "ethereum", &poolSample{fromBlock: 123, toBlock: 222, limit: 1})
	if err == nil || result != nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("exhausted sample returned success: result=%+v err=%v", result, err)
	}
}
