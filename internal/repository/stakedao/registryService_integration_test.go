package stakedao

import (
	"context"
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
