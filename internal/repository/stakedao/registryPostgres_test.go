package stakedao

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	service "ABIevm/internal/service/stakedao"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration tests opt in to a dedicated DB, and isolate all DDL in a fresh
// schema. DATABASE_URL is deliberately never used by tests.
func postgresFixture(t *testing.T) (*RegistryPostgres, string) {
	t.Helper()
	dsn := os.Getenv("STAKEDAO_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set STAKEDAO_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("test Postgres config unavailable")
	}
	var random [8]byte
	if _, err = rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "registry_test_" + hex.EncodeToString(random[:])
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+identifier); err != nil {
		admin.Close()
		t.Fatal("cannot create isolated test schema")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	params := parsed.Query()
	params.Set("search_path", schema)
	parsed.RawQuery = params.Encode()
	schemaURL := parsed.String()
	config, err := pgxpool.ParseConfig(schemaURL)
	if err != nil {
		t.Fatal("test schema config unavailable")
	}
	config.MaxConns = 5
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	r := &RegistryPostgres{pool: pool}
	t.Cleanup(func() {
		r.Close()
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := admin.Exec(cleanup, `DROP SCHEMA `+identifier+` CASCADE`)
		admin.Close()
		if err != nil {
			t.Error("test schema cleanup failed")
		}
	})
	for _, name := range []string{"000001_registry.up.sql", "000002_swap_history.up.sql"} {
		migration, e := os.ReadFile("../../../migrations/" + name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = pool.Exec(ctx, string(migration)); e != nil {
			t.Fatalf("migration %s failed: %v", name, e)
		}
	}
	return r, schemaURL
}
func testKey() service.RegistryKey {
	return service.RegistryKey{ChainID: 1, Platform: "stakedao", Controller: common.HexToAddress("0x1234")}
}
func testEvent(block uint64, vault string) service.RegistryEvent {
	return service.RegistryEvent{Kind: service.RegistryVaultRegistered, Info: service.StakeDAOVaultInfo{Gauge: common.HexToAddress("0xaa"), Vault: common.HexToAddress(vault), Asset: common.HexToAddress("0xbb"), RewardReceiver: common.HexToAddress("0xcc"), ProtocolID: [4]byte{0xc7, 0x15, 0xe3, 0x73}}, Block: block, BlockHash: common.HexToHash("0x01"), TxHash: common.HexToHash("0x02")}
}
func acquire(t *testing.T, r *RegistryPostgres, key service.RegistryKey) service.RegistrySession {
	t.Helper()
	s, err := r.Acquire(context.Background(), key, 1)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPostgresEventsCursorAndRewind(t *testing.T) {
	r, _ := postgresFixture(t)
	s := acquire(t, r, testKey())
	defer s.Close()
	ctx := context.Background()
	cursor, err := s.Cursor(ctx)
	if err != nil || cursor.Last != nil {
		t.Fatal("new registry already has a cursor")
	}
	first := service.RegistryCheckpoint{Block: 1500, Hash: common.HexToHash("0x10")}
	if err = s.SaveRange(ctx, 1, first, []service.RegistryEvent{testEvent(100, "0x11")}); err != nil {
		t.Fatal(err)
	}
	next := service.RegistryCheckpoint{Block: 3000, Hash: common.HexToHash("0x20")}
	if err = s.SaveRange(ctx, 1501, next, []service.RegistryEvent{testEvent(2000, "0x22")}); err != nil {
		t.Fatal(err)
	}
	all, err := s.Registrations(ctx, 3000)
	if err != nil || len(all) != 1 || all[0].Vault != common.HexToAddress("0x22") {
		t.Fatalf("latest version: %+v %v", all, err)
	}
	old, err := s.Registrations(ctx, 1500)
	if err != nil || old[0].Vault != common.HexToAddress("0x11") {
		t.Fatal("historical version lost")
	}
	prev, err := s.PreviousCheckpoint(ctx, 3000)
	if err != nil || prev.Block != 1500 || prev.Hash != first.Hash {
		t.Fatal("checkpoint history lost")
	}
	if err = s.Rewind(ctx, first); err != nil {
		t.Fatal(err)
	}
	all, err = s.Registrations(ctx, 3000)
	if err != nil || len(all) != 1 || all[0].Vault != common.HexToAddress("0x11") {
		t.Fatal("orphaned registration survived rewind")
	}
	cursor, err = s.Cursor(ctx)
	if err != nil || cursor.Last.Block != 1500 || cursor.Last.Hash != first.Hash {
		t.Fatal("cursor did not rewind with events")
	}
	if err = s.SaveRange(ctx, 1501, next, nil); err != nil {
		t.Fatal(err)
	} // empty ranges are durable too
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	again := acquire(t, r, testKey())
	defer again.Close()
	cursor, err = again.Cursor(ctx)
	if err != nil || cursor.Last.Block != 3000 {
		t.Fatal("cursor did not survive a new DB session")
	}
}

func TestPostgresRollsBackEventsWhenCheckpointWriteFails(t *testing.T) {
	r, _ := postgresFixture(t)
	s := acquire(t, r, testKey())
	defer s.Close()
	ctx := context.Background()
	_, err := r.pool.Exec(ctx, `CREATE FUNCTION fail_checkpoint() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated checkpoint failure'; END $$;
 CREATE TRIGGER fail_checkpoint BEFORE INSERT ON registry_checkpoints FOR EACH ROW EXECUTE FUNCTION fail_checkpoint();`)
	if err != nil {
		t.Fatal(err)
	}
	err = s.SaveRange(ctx, 1, service.RegistryCheckpoint{Block: 1500, Hash: common.HexToHash("0x10")}, []service.RegistryEvent{testEvent(100, "0x11")})
	if !errors.Is(err, service.ErrRegistryUnavailable) {
		t.Fatalf("checkpoint error not surfaced safely: %v", err)
	}
	cursor, err := s.Cursor(ctx)
	if err != nil || cursor.Last != nil {
		t.Fatal("failed transaction advanced cursor")
	}
	var count int
	if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM registry_events`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed transaction left partial history")
	}
}

func TestPostgresLocksCancelAndDoNotLeakIntoPool(t *testing.T) {
	r, _ := postgresFixture(t)
	first := acquire(t, r, testKey())
	defer first.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if second, err := r.Acquire(ctx, testKey(), 1); err == nil {
		second.Close()
		t.Fatal("two writers acquired the same stream")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	retry, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	third, err := r.Acquire(retry, testKey(), 1)
	if err != nil {
		t.Fatalf("canceled session leaked an advisory lock: %v", err)
	}
	third.Close()
}

func TestPostgresSeparatesNetworksAndRejectsChangedStart(t *testing.T) {
	r, _ := postgresFixture(t)
	a := acquire(t, r, testKey())
	defer a.Close()
	ctx := context.Background()
	if err := a.SaveRange(ctx, 1, service.RegistryCheckpoint{Block: 1500, Hash: common.HexToHash("0x10")}, []service.RegistryEvent{testEvent(100, "0x11")}); err != nil {
		t.Fatal(err)
	}
	key := testKey()
	key.ChainID = 10
	b := acquire(t, r, key)
	defer b.Close()
	if all, err := b.Registrations(ctx, 1500); err != nil || len(all) != 0 {
		t.Fatal("registrations leaked between chain IDs")
	}
	a.Close()
	if other, err := r.Acquire(ctx, testKey(), 2); err == nil {
		other.Close()
		t.Fatal("changed deployment block silently reused old registry")
	}
}

func TestPostgresUpDownAndStartupValidation(t *testing.T) {
	r, dsn := postgresFixture(t)
	ctx := context.Background()
	ready, err := OpenRegistryPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	ready.Close()
	down, err := os.ReadFile("../../../migrations/000001_registry.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("down migration failed: %v", err)
	}
	if opened, err := OpenRegistryPostgres(ctx, dsn); err == nil {
		opened.Close()
		t.Fatal("missing schema accepted at startup")
	}
	up, err := os.ReadFile("../../../migrations/000001_registry.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.pool.Exec(ctx, string(up)); err != nil {
		t.Fatal("up after down failed")
	}
}

func TestPostgresConnectionErrorsDoNotExposeCredentials(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := OpenRegistryPostgres(ctx, "postgres://user:private-key@127.0.0.1:1/db?sslmode=disable")
	if err == nil || strings.Contains(err.Error(), "private-key") || strings.Contains(err.Error(), "postgres://") {
		t.Fatal("unsafe connection error")
	}
}

func TestPostgresBoundsCheckpointsWithoutDroppingRegistrationHistory(t *testing.T) {
	r, _ := postgresFixture(t)
	s := acquire(t, r, testKey())
	defer s.Close()
	ctx := context.Background()
	for i := uint64(0); i < 24; i++ {
		var events []service.RegistryEvent
		if i == 0 {
			events = []service.RegistryEvent{testEvent(100, "0x11")}
		}
		if err := s.SaveRange(ctx, i*1500+1, service.RegistryCheckpoint{Block: (i + 1) * 1500, Hash: common.HexToHash("0x10")}, events); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM registry_checkpoints`).Scan(&count); err != nil || count != service.RegistryCheckpointWindow {
		t.Fatalf("checkpoint retention failed: %d %v", count, err)
	}
	all, err := s.Registrations(ctx, 36000)
	if err != nil || len(all) != 1 || all[0].Vault != common.HexToAddress("0x11") {
		t.Fatal("checkpoint pruning removed registrations")
	}
}

func testSwapKey() service.SwapKey {
	return service.SwapKey{ChainID: 1, Platform: "stakedao", Pool: common.HexToAddress("0xaa"), Event: crypto.Keccak256Hash([]byte("TokenExchange(address,int128,uint256,int128,uint256)"))}
}
func testSwap(block uint64) types.Log {
	data := make([]byte, 128)
	data[63] = 10
	data[95] = 1
	data[127] = 9
	return types.Log{Address: testSwapKey().Pool, BlockNumber: block, BlockHash: (&types.Header{Number: new(big.Int).SetUint64(block), Time: block + 1}).Hash(), TxHash: common.BigToHash(new(big.Int).SetUint64(block + 123)), Topics: []common.Hash{testSwapKey().Event, common.HexToHash("0x12")}, Data: data}
}
func testSwapPoint(block uint64) service.RegistryCheckpoint {
	return service.RegistryCheckpoint{Block: block, Hash: (&types.Header{Number: new(big.Int).SetUint64(block), Time: block + 1}).Hash()}
}
func TestPostgresSwapHistoryCoverageIdempotencyAndIsolation(t *testing.T) {
	db, _ := postgresFixture(t)
	ctx := context.Background()
	key := testSwapKey()
	session, e := db.AcquireSwaps(ctx, key)
	if e != nil {
		t.Fatal(e)
	}
	defer session.Close()
	if e = session.SaveSwapRange(ctx, 100, testSwapPoint(109), []types.Log{testSwap(105)}); e != nil {
		t.Fatal(e)
	}
	if e = session.SaveSwapRange(ctx, 105, testSwapPoint(120), []types.Log{testSwap(105), testSwap(118)}); e != nil {
		t.Fatal(e)
	}
	if e = session.SaveSwapRange(ctx, 90, testSwapPoint(99), nil); e != nil {
		t.Fatal(e)
	}
	coverage, e := session.Coverage(ctx, 1, 200)
	if e != nil || len(coverage) != 1 || coverage[0].From != 90 || coverage[0].Through.Block != 120 {
		t.Fatalf("overlap/backfill failed: %+v %v", coverage, e)
	}
	logs, e := session.Swaps(ctx, 100, 110)
	if e != nil || len(logs) != 1 || !bytes.Equal(logs[0].Data, testSwap(105).Data) || len(logs[0].Topics) != 2 || logs[0].Topics[1] != testSwap(105).Topics[1] {
		t.Fatalf("round-trip or historic slice failed: %+v %v", logs, e)
	}
	bad := testSwap(105)
	bad.Data = append([]byte(nil), bad.Data...)
	bad.Data[127] = 8
	if e = session.SaveSwapRange(ctx, 100, testSwapPoint(120), []types.Log{bad}); e == nil {
		t.Fatal("conflicting cached event accepted")
	}
	logs, e = session.Swaps(ctx, 1, 200)
	if e != nil || len(logs) != 2 || logs[0].Data[127] != 9 {
		t.Fatal("failed range changed durable data")
	}
	if e = session.SaveSwapRange(ctx, 200, testSwapPoint(209), nil); e != nil {
		t.Fatal(e)
	}
	point, e := session.Cursor(ctx)
	if e != nil || point == nil || point.Block != 209 {
		t.Fatal("empty-range cursor not saved")
	}
	other := key
	other.ChainID = 137
	second, e := db.AcquireSwaps(ctx, other)
	if e != nil {
		t.Fatal(e)
	}
	got, e := second.Swaps(ctx, 1, 300)
	_ = second.Close()
	if e != nil || len(got) != 0 {
		t.Fatal("cross-chain cache collision")
	}
	if e = session.DropSwapRange(ctx, coverage[0]); e != nil {
		t.Fatal(e)
	}
	logs, e = session.Swaps(ctx, 1, 300)
	if e != nil || len(logs) != 0 {
		t.Fatal("orphaned events retained")
	}
	point, e = session.Cursor(ctx)
	if e != nil || point.Block != 209 {
		t.Fatal("unrelated coverage lost")
	}
	rest, e := session.Coverage(ctx, 1, 300)
	if e != nil || len(rest) != 1 {
		t.Fatal(e)
	}
	if e = session.DropSwapRange(ctx, rest[0]); e != nil {
		t.Fatal(e)
	}
	point, e = session.Cursor(ctx)
	if e != nil || point != nil {
		t.Fatal("invalidated cursor not reset")
	}
}
func TestPostgresSwapLocksAndReopen(t *testing.T) {
	db, dsn := postgresFixture(t)
	ctx := context.Background()
	key := testSwapKey()
	first, e := db.AcquireSwaps(ctx, key)
	if e != nil {
		t.Fatal(e)
	}
	if e = first.SaveSwapRange(ctx, 10, testSwapPoint(20), []types.Log{testSwap(15)}); e != nil {
		t.Fatal(e)
	}
	reopened, e := OpenRegistryPostgres(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	blocked, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	if session, e := reopened.AcquireSwaps(blocked, key); e == nil {
		session.Close()
		t.Fatal("stream lock not exclusive across pools")
	}
	_ = first.Close()
	second, e := reopened.AcquireSwaps(ctx, key)
	if e != nil {
		t.Fatal(e)
	}
	defer second.Close()
	point, e := second.Cursor(ctx)
	if e != nil || point == nil || point.Block != 20 {
		t.Fatal("new pool lost cursor")
	}
	logs, e := second.Swaps(ctx, 10, 20)
	if e != nil || len(logs) != 1 {
		t.Fatal("new pool lost logs")
	}
}

func TestSwapMigrationRequiredWithoutResettingRegistry(t *testing.T) {
	db, dsn := postgresFixture(t)
	ctx := context.Background()
	session, e := db.Acquire(ctx, testKey(), 1)
	if e != nil {
		t.Fatal(e)
	}
	_ = session.Close()
	down, e := os.ReadFile("../../../migrations/000002_swap_history.down.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.pool.Exec(ctx, string(down)); e != nil {
		t.Fatal(e)
	}
	if opened, e := OpenRegistryPostgres(ctx, dsn); e == nil {
		opened.Close()
		t.Fatal("missing swap schema accepted")
	} else if !strings.Contains(e.Error(), "000002_swap_history.up.sql") || strings.Contains(e.Error(), "000001") {
		t.Fatalf("existing DB received ambiguous migration instruction: %v", e)
	}
	// Existing Controller history remains available after rolling back only swaps.
	session, e = db.Acquire(ctx, testKey(), 1)
	if e != nil {
		t.Fatal("registry affected by swap migration")
	}
	_ = session.Close()
	up, e := os.ReadFile("../../../migrations/000002_swap_history.up.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.pool.Exec(ctx, string(up)); e != nil {
		t.Fatal(e)
	}
	ready, e := OpenRegistryPostgres(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	ready.Close()
}
