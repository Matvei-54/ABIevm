package stakedao

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"

	service "ABIevm/internal/service/stakedao"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type integrationChain struct {
	mu        sync.Mutex
	finalized uint64
	logs      []types.Log
	queries   []ethereum.FilterQuery
}

func (c *integrationChain) header(block uint64) *types.Header {
	return &types.Header{Number: new(big.Int).SetUint64(block), Time: block + 1}
}
func (c *integrationChain) HeaderByNumber(_ context.Context, block *big.Int) (*types.Header, error) {
	if block.Int64() == int64(rpc.FinalizedBlockNumber) {
		return c.header(c.finalized), nil
	}
	return c.header(block.Uint64()), nil
}
func (c *integrationChain) FilterLogs(_ context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queries = append(c.queries, q)
	var logs []types.Log
	for _, entry := range c.logs {
		if entry.BlockNumber >= q.FromBlock.Uint64() && entry.BlockNumber <= q.ToBlock.Uint64() {
			logs = append(logs, entry)
		}
	}
	return logs, nil
}
func integrationRegistration(t *testing.T, chain *integrationChain, block uint64, vault string) types.Log {
	t.Helper()
	data, err := os.ReadFile("../../../contracts/stakedao/CreateVault.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := abi.JSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	event := parsed.Events["VaultRegistered"]
	body, err := event.Inputs.NonIndexed().Pack(common.HexToAddress("0x44"), [4]byte{0xc7, 0x15, 0xe3, 0x73})
	if err != nil {
		t.Fatal(err)
	}
	return types.Log{Address: testKey().Controller, BlockNumber: block, BlockHash: chain.header(block).Hash(), TxHash: common.HexToHash("0x1234"), Data: body, Topics: []common.Hash{event.ID, common.BytesToHash(common.HexToAddress("0xaa").Bytes()), common.BytesToHash(common.HexToAddress(vault).Bytes()), common.BytesToHash(common.HexToAddress("0xbb").Bytes())}}
}

func TestRegistryServiceWithRealPostgresResumesFromAnotherPool(t *testing.T) {
	r, dsn := postgresFixture(t)
	ctx := context.Background()
	chain := &integrationChain{finalized: 1500}
	chain.logs = []types.Log{integrationRegistration(t, chain, 100, "0x11")}
	first, err := service.NewRegistryService(r).SyncAndList(ctx, chain, testKey(), 1, 1500)
	if err != nil || len(first) != 1 || first[0].Vault != common.HexToAddress("0x11") {
		t.Fatalf("initial durable sync failed: %+v %v", first, err)
	}
	reopened, err := OpenRegistryPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	chain.queries = nil
	chain.finalized = 1510
	chain.logs = append(chain.logs, integrationRegistration(t, chain, 1505, "0x22"))
	second, err := service.NewRegistryService(reopened).SyncAndList(ctx, chain, testKey(), 1, 1510)
	if err != nil || len(second) != 1 || second[0].Vault != common.HexToAddress("0x22") {
		t.Fatalf("incremental durable sync failed: %+v %v", second, err)
	}
	if len(chain.queries) != 1 || chain.queries[0].FromBlock.Uint64() != 1501 || chain.queries[0].ToBlock.Uint64() != 1510 {
		t.Fatalf("new pool reread historical blocks: %+v", chain.queries)
	}
}

type swapIntegrationChain struct {
	*integrationChain
	epoch    bool
	failFrom uint64
}

func (c *swapIntegrationChain) HeaderByNumber(ctx context.Context, n *big.Int) (*types.Header, error) {
	h, e := c.integrationChain.HeaderByNumber(ctx, n)
	if e == nil && c.epoch && h.Number.Uint64() >= 90 {
		h.Extra = []byte("new canonical chain")
	}
	return h, e
}
func (c *swapIntegrationChain) FilterLogs(_ context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queries = append(c.queries, q)
	if c.failFrom != 0 && q.FromBlock.Uint64() >= c.failFrom {
		return nil, errors.New("simulated log transport failure")
	}
	out := []types.Log{}
	for _, entry := range c.logs {
		if entry.BlockNumber >= q.FromBlock.Uint64() && entry.BlockNumber <= q.ToBlock.Uint64() {
			if c.epoch {
				h := c.header(entry.BlockNumber)
				h.Extra = []byte("new canonical chain")
				entry.BlockHash = h.Hash()
			}
			out = append(out, entry)
		}
	}
	return out, nil
}
func TestSwapServiceWithPostgresBackfillTailHistoryAndReorg(t *testing.T) {
	db, _ := postgresFixture(t)
	ctx := context.Background()
	key := testSwapKey()
	srv := service.NewRegistryService(db)
	chain := &swapIntegrationChain{integrationChain: &integrationChain{finalized: 120, logs: []types.Log{testSwap(105), testSwap(123), testSwap(129)}}}
	first, a, e := srv.SyncSwapEvents(ctx, chain, key, 100, 125)
	if e != nil || len(first) != 2 || a.LastSavedBlock == nil || *a.LastSavedBlock != 120 {
		t.Fatalf("first sync: %+v %v", a, e)
	}
	chain.queries = nil
	again, b, e := srv.SyncSwapEvents(ctx, chain, key, 100, 125)
	if e != nil || len(again) != 2 || b.CachedEvents != 1 || len(chain.queries) != 1 || chain.queries[0].FromBlock.Uint64() != 121 {
		t.Fatalf("cached history rescanned or tail persisted: %+v %+v %v", b, chain.queries, e)
	}
	chain.finalized = 125
	chain.queries = nil
	result, c, e := srv.SyncSwapEvents(ctx, chain, key, 100, 130)
	if e != nil || len(result) != 3 || *c.LastSavedBlock != 125 {
		t.Fatalf("incremental sync: %+v %v", c, e)
	}
	for _, q := range chain.queries {
		if q.FromBlock.Uint64() < 121 {
			t.Fatal("old finalized logs fetched again")
		}
	}
	chain.finalized = 130
	chain.queries = nil
	result, d, e := srv.SyncSwapEvents(ctx, chain, key, 90, 130)
	if e != nil || len(result) != 3 || *d.LastSavedBlock != 130 {
		t.Fatalf("backfill failed: %+v %v", d, e)
	}
	for _, q := range chain.queries {
		if q.FromBlock.Uint64() >= 100 && q.FromBlock.Uint64() <= 125 {
			t.Fatal("backfill reread covered middle")
		}
	}
	chain.queries = nil
	old, _, e := srv.SyncSwapEvents(ctx, chain, key, 100, 110)
	if e != nil || len(old) != 1 || len(chain.queries) != 0 {
		t.Fatal("explicit past snapshot rewound or rescanned cache")
	}
	chain.epoch = true
	chain.queries = nil
	replaced, f, e := srv.SyncSwapEvents(ctx, chain, key, 100, 110)
	if e != nil || len(replaced) != 1 || replaced[0].BlockHash == old[0].BlockHash || *f.LastSavedBlock != 110 {
		t.Fatalf("stale coverage not invalidated: %+v %v", f, e)
	}
	chain.epoch = false
	chain.failFrom = 100
	_, failed, e := srv.SyncSwapEvents(ctx, chain, key, 100, 110)
	if e == nil || failed.LastSavedBlock != nil {
		t.Fatalf("invalidated cursor still reported after failed recovery: %+v %v", failed, e)
	}
}
func TestSwapServicePostgresResumesCommittedBatchAfterFailure(t *testing.T) {
	db, _ := postgresFixture(t)
	ctx := context.Background()
	srv := service.NewRegistryService(db)
	key := testSwapKey()
	chain := &swapIntegrationChain{integrationChain: &integrationChain{finalized: 30000}, failFrom: 24001}
	_, a, e := srv.SyncSwapEvents(ctx, chain, key, 1, 30000)
	if e == nil || a.LastSavedBlock == nil || *a.LastSavedBlock != 24000 {
		t.Fatalf("successful batch lost: %+v %v", a, e)
	}
	chain.failFrom = 0
	chain.queries = nil
	logs, b, e := srv.SyncSwapEvents(ctx, chain, key, 1, 30000)
	if e != nil || len(logs) != 0 || *b.LastSavedBlock != 30000 {
		t.Fatalf("resume failed: %+v %v", b, e)
	}
	for _, q := range chain.queries {
		if q.FromBlock.Uint64() < 24001 {
			t.Fatal("committed empty range fetched again")
		}
	}
}
