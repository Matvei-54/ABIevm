package stakedao

import (
	"context"
	"errors"
	"math/big"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type memoryRegistry struct {
	cursor    RegistryCursor
	points    []RegistryCheckpoint
	events    []RegistryEvent
	failFrom  uint64
	savedFrom []uint64
	closed    int
}

func (m *memoryRegistry) Acquire(_ context.Context, _ RegistryKey, start uint64) (RegistrySession, error) {
	if m.cursor.StartBlock == 0 {
		m.cursor.StartBlock = start
	}
	return m, nil
}
func (m *memoryRegistry) Cursor(context.Context) (RegistryCursor, error) { return m.cursor, nil }
func (m *memoryRegistry) PreviousCheckpoint(_ context.Context, before uint64) (*RegistryCheckpoint, error) {
	for i := len(m.points) - 1; i >= 0; i-- {
		if m.points[i].Block < before {
			p := m.points[i]
			return &p, nil
		}
	}
	return nil, nil
}
func (m *memoryRegistry) SaveRange(_ context.Context, from uint64, p RegistryCheckpoint, events []RegistryEvent) error {
	if m.failFrom == from {
		return errors.New("simulated storage failure")
	}
	expected := m.cursor.StartBlock
	if m.cursor.Last != nil {
		expected = m.cursor.Last.Block + 1
	}
	if from != expected {
		return errors.New("noncontiguous commit")
	}
	m.events = append(m.events, events...)
	m.points = append(m.points, p)
	m.savedFrom = append(m.savedFrom, from)
	m.cursor.Last = &p
	return nil
}
func (m *memoryRegistry) Rewind(_ context.Context, p RegistryCheckpoint) error {
	var events []RegistryEvent
	for _, e := range m.events {
		if e.Block <= p.Block {
			events = append(events, e)
		}
	}
	m.events = events
	var points []RegistryCheckpoint
	for _, point := range m.points {
		if point.Block <= p.Block {
			points = append(points, point)
		}
	}
	m.points = points
	m.cursor.Last = &p
	return nil
}
func (m *memoryRegistry) Registrations(_ context.Context, through uint64) ([]StakeDAOVaultInfo, error) {
	latest := map[common.Address]StakeDAOVaultInfo{}
	for _, e := range m.events {
		if e.Block <= through && e.Kind == RegistryVaultRegistered {
			latest[e.Info.Gauge] = e.Info
		}
	}
	var all []StakeDAOVaultInfo
	for _, info := range latest {
		all = append(all, info)
	}
	return all, nil
}
func (m *memoryRegistry) Close() error { m.closed++; return nil }

type registryChain struct {
	mu              sync.Mutex
	finalized       uint64
	logs            []types.Log
	queries         []ethereum.FilterQuery
	revisions       map[uint64]byte
	rejectFinalized bool
	beforeRead      func(ethereum.FilterQuery)
}

func (c *registryChain) header(block uint64) *types.Header {
	return &types.Header{Number: new(big.Int).SetUint64(block), Time: block + 100, Extra: []byte{c.revisions[block]}}
}
func (c *registryChain) HeaderByNumber(_ context.Context, number *big.Int) (*types.Header, error) {
	if number.Int64() == int64(rpc.FinalizedBlockNumber) {
		if c.rejectFinalized {
			return nil, errors.New("finalized unsupported")
		}
		return c.header(c.finalized), nil
	}
	return c.header(number.Uint64()), nil
}
func (c *registryChain) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	c.mu.Lock()
	c.queries = append(c.queries, q)
	c.mu.Unlock()
	if c.beforeRead != nil {
		c.beforeRead(q)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var logs []types.Log
	for _, e := range c.logs {
		match := false
		for _, topic := range q.Topics[0] {
			if topic == e.Topics[0] {
				match = true
			}
		}
		if match && e.BlockNumber >= q.FromBlock.Uint64() && e.BlockNumber <= q.ToBlock.Uint64() {
			logs = append(logs, e)
		}
	}
	return logs, nil
}
func registryTestKey() RegistryKey {
	return RegistryKey{ChainID: 1, Platform: "stakedao", Controller: common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)}
}
func registryLog(t *testing.T, c *registryChain, gauge, vault, asset common.Address, block uint64, index uint) types.Log {
	e := registration(t, registryTestKey().Controller, gauge, vault, asset, block, index)
	e.BlockHash = c.header(block).Hash()
	e.TxHash = common.BigToHash(new(big.Int).SetUint64(block + uint64(index) + 1))
	return e
}

func TestRegistryPersistsEmptyRangesAndReadsOnlyNewFinalizedBlocks(t *testing.T) {
	chain := &registryChain{finalized: 3000}
	store := &memoryRegistry{}
	service := NewRegistryService(store)
	a, b := common.HexToAddress("0xa1"), common.HexToAddress("0xb1")
	chain.logs = []types.Log{registryLog(t, chain, a, common.HexToAddress("0x11"), common.HexToAddress("0x22"), 100, 0)}
	all, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 3000)
	if err != nil || len(all) != 1 || store.cursor.Last.Block != 3000 || !reflect.DeepEqual(store.savedFrom, []uint64{1, 1501}) {
		t.Fatalf("initial sync: all=%+v cursor=%+v err=%v", all, store.cursor, err)
	}
	chain.queries = nil
	if _, err = service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 3000); err != nil || len(chain.queries) != 0 {
		t.Fatalf("unchanged finalized head rescanned history: %v %+v", err, chain.queries)
	}
	chain.finalized = 3100
	chain.logs = append(chain.logs, registryLog(t, chain, b, common.HexToAddress("0x33"), common.HexToAddress("0x44"), 3050, 0))
	all, err = service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 3100)
	if err != nil || len(all) != 2 || len(chain.queries) != 1 || chain.queries[0].FromBlock.Uint64() != 3001 || store.cursor.Last.Block != 3100 {
		t.Fatalf("incremental sync: %+v %v", chain.queries, err)
	}
	if store.closed != 3 {
		t.Fatal("registry session was not released")
	}
}

func TestRegistryResumesCommittedRangesAfterStorageFailure(t *testing.T) {
	t.Setenv("STAKEDAO_DISCOVERY_WORKERS", "1")
	chain := &registryChain{finalized: 4500}
	store := &memoryRegistry{failFrom: 1501}
	service := NewRegistryService(store)
	if result, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 4500); err == nil || result != nil || store.cursor.Last.Block != 1500 {
		t.Fatalf("lost committed progress: cursor=%+v err=%v", store.cursor, err)
	}
	chain.queries = nil
	store.failFrom = 0
	if _, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 4500); err != nil {
		t.Fatal(err)
	}
	for _, q := range chain.queries {
		if q.FromBlock.Uint64() < 1501 {
			t.Fatal("resumed from deployment instead of saved cursor")
		}
	}
	if store.cursor.Last.Block != 4500 {
		t.Fatal("resume did not finish")
	}
}

func TestRegistryCommitsConcurrentRangesInChainOrder(t *testing.T) {
	t.Setenv("STAKEDAO_DISCOVERY_WORKERS", "3")
	second := make(chan struct{})
	var once sync.Once
	chain := &registryChain{finalized: 4500, beforeRead: func(q ethereum.FilterQuery) {
		if q.FromBlock.Uint64() == 1501 {
			once.Do(func() { close(second) })
		}
		if q.FromBlock.Uint64() == 1 {
			<-second
		}
	}}
	store := &memoryRegistry{}
	if _, err := NewRegistryService(store).SyncAndList(context.Background(), chain, registryTestKey(), 1, 4500); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.savedFrom, []uint64{1, 1501, 3001}) {
		t.Fatalf("cursor advanced out of order: %v", store.savedFrom)
	}
}

func TestRegistryDoesNotPersistUnfinalizedTail(t *testing.T) {
	chain := &registryChain{finalized: 1500}
	store := &memoryRegistry{}
	service := NewRegistryService(store)
	gauge := common.HexToAddress("0xaa")
	asset := common.HexToAddress("0xbb")
	chain.logs = []types.Log{registryLog(t, chain, gauge, common.HexToAddress("0x11"), asset, 100, 0), registryLog(t, chain, gauge, common.HexToAddress("0x22"), asset, 1510, 0)}
	all, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 1520)
	if err != nil || len(all) != 1 || all[0].Vault != common.HexToAddress("0x22") || store.cursor.Last.Block != 1500 || len(store.events) != 1 {
		t.Fatalf("tail persisted or ignored: %+v %v", all, err)
	}
	chain.logs = chain.logs[:1] // the recent registration was orphaned
	all, err = service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 1520)
	if err != nil || all[0].Vault != common.HexToAddress("0x11") {
		t.Fatalf("orphan tail remained cached: %+v %v", all, err)
	}
}

func TestRegistryReorgRewindsEventsAndCursorTogether(t *testing.T) {
	chain := &registryChain{finalized: 3000, revisions: map[uint64]byte{}}
	store := &memoryRegistry{}
	service := NewRegistryService(store)
	gauge := common.HexToAddress("0xaa")
	asset := common.HexToAddress("0xbb")
	chain.logs = []types.Log{registryLog(t, chain, gauge, common.HexToAddress("0x11"), asset, 100, 0), registryLog(t, chain, gauge, common.HexToAddress("0x22"), asset, 2000, 0)}
	if _, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 3000); err != nil {
		t.Fatal(err)
	}
	chain.revisions[3000] = 1
	chain.logs = chain.logs[:1]
	chain.queries = nil
	all, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 3000)
	if err != nil || len(all) != 1 || all[0].Vault != common.HexToAddress("0x11") || len(store.events) != 1 || store.cursor.Last.Hash != chain.header(3000).Hash() {
		t.Fatalf("orphaned registration survived: %+v %v", all, err)
	}
	if chain.queries[0].FromBlock.Uint64() != 1501 {
		t.Fatal("reorg did not resume from common checkpoint")
	}
	chain.revisions[1500] = 2
	chain.revisions[3000] = 2
	if _, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 3000); !errors.Is(err, ErrRegistryReorg) {
		t.Fatalf("deep reorg silently reset state: %v", err)
	}
}

func TestRegistryRejectsLaggingFinalityAndMissingFinalizedSupport(t *testing.T) {
	chain := &registryChain{finalized: 1500}
	store := &memoryRegistry{}
	service := NewRegistryService(store)
	if _, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 1500); err != nil {
		t.Fatal(err)
	}
	chain.finalized = 1400
	chain.queries = nil
	if _, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 1500); err == nil || len(chain.queries) != 0 || store.cursor.Last.Block != 1500 {
		t.Fatal("lagging node altered persisted state")
	}
	chain.rejectFinalized = true
	if _, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 1500); err == nil {
		t.Fatal("unsupported finalized tag fell back to a full scan")
	}
}

func TestRegistryStatusEventsMatchABIAndAreStored(t *testing.T) {
	chain := &registryChain{finalized: 1500}
	store := &memoryRegistry{}
	parsed, err := abi.JSON(strings.NewReader(registryStatusEventsABI))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../../contracts/stakedao/CreateVault.json")
	if err != nil {
		t.Fatal(err)
	}
	full, err := abi.JSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	for name, event := range parsed.Events {
		if event.Sig != full.Events[name].Sig || !event.Inputs[0].Indexed {
			t.Fatal("status event ABI mismatch")
		}
	}
	chain.logs = []types.Log{registryLog(t, chain, common.HexToAddress("0xaa"), common.HexToAddress("0xbb"), common.HexToAddress("0xcc"), 100, 0)}
	for i, name := range []string{"GaugeShutdown", "GaugeUnshutdown"} {
		block := uint64(200 + i)
		e := types.Log{Address: registryTestKey().Controller, Topics: []common.Hash{parsed.Events[name].ID, common.BytesToHash(common.HexToAddress("0xaa").Bytes())}, BlockNumber: block, BlockHash: chain.header(block).Hash(), TxHash: common.BigToHash(new(big.Int).SetUint64(block)), Index: 0}
		chain.logs = append(chain.logs, e)
	}
	if _, err := NewRegistryService(store).SyncAndList(context.Background(), chain, registryTestKey(), 1, 1500); err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, e := range store.events {
		kinds = append(kinds, e.Kind)
	}
	if !reflect.DeepEqual(kinds, []string{RegistryVaultRegistered, RegistryGaugeShutdown, RegistryGaugeUnshutdown}) {
		t.Fatalf("status history missing: %v", kinds)
	}
}

func TestRegistryServesOlderRequestSnapshotWithoutRewindingNewerCursor(t *testing.T) {
	chain := &registryChain{finalized: 3000}
	store := &memoryRegistry{}
	service := NewRegistryService(store)
	gauge := common.HexToAddress("0xaa")
	asset := common.HexToAddress("0xbb")
	chain.logs = []types.Log{registryLog(t, chain, gauge, common.HexToAddress("0x11"), asset, 100, 0), registryLog(t, chain, gauge, common.HexToAddress("0x22"), asset, 2000, 0)}
	if _, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 3000); err != nil {
		t.Fatal(err)
	}
	chain.queries = nil
	// Another request may finish indexing while this one waits for the stream lock.
	all, err := service.SyncAndList(context.Background(), chain, registryTestKey(), 1, 1500)
	if err != nil || len(all) != 1 || all[0].Vault != common.HexToAddress("0x11") || store.cursor.Last.Block != 3000 || len(chain.queries) != 0 {
		t.Fatalf("older snapshot corrupted cursor or reread history: %+v %v", all, err)
	}
}
