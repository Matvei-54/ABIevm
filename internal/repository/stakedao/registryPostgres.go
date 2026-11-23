// Package stakedao implements PostgreSQL persistence; blockchain reads stay in the service.
package stakedao

import (
	"bytes"
	"context"
	"errors"
	"math"
	"time"

	service "ABIevm/internal/service/stakedao"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegistryPostgres struct{ pool *pgxpool.Pool }

// OpenRegistryPostgres validates connectivity/schema at startup. It never runs DDL.
func OpenRegistryPostgres(ctx context.Context, url string) (*RegistryPostgres, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("invalid DATABASE_URL")
	}
	config.MaxConns, config.MinConns = 5, 0
	config.MaxConnLifetime, config.MaxConnIdleTime = 30*time.Minute, 5*time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, service.ErrRegistryUnavailable
	}
	if err = pool.Ping(ctx); err == nil {
		// Reference required columns, not just table existence; missing migrations fail early.
		_, err = pool.Exec(ctx, `SELECT s.decoder_version, s.last_scanned_hash, c.block_hash, e.event_kind
   FROM registry_sync_state s LEFT JOIN registry_checkpoints c USING (chain_id,platform,registry_address)
   LEFT JOIN registry_events e USING (chain_id,platform,registry_address) LIMIT 0`)
	}
	missingSwapSchema := false
	if err == nil {
		_, err = pool.Exec(ctx, `SELECT s.decoder_version,s.last_saved_block,s.last_saved_hash,r.from_block,r.through_hash,e.buyer_address,e.event_data
 FROM swap_streams s LEFT JOIN swap_scan_ranges r USING(chain_id,platform,pool_address,event_topic)
 LEFT JOIN swap_events e USING(chain_id,platform,pool_address,event_topic) LIMIT 0`)
		missingSwapSchema = err != nil
	}
	if err != nil {
		pool.Close()
		if missingSwapSchema {
			return nil, errors.New("Postgres swap history schema unavailable; apply migrations/000002_swap_history.up.sql to the existing DATABASE_URL database")
		}
		return nil, errors.New("Postgres connection/schema unavailable; check DATABASE_URL and apply migrations/000001_registry.up.sql and 000002_swap_history.up.sql")
	}
	return &RegistryPostgres{pool: pool}, nil
}

func (r *RegistryPostgres) Close() { r.pool.Close() }

func (r *RegistryPostgres) Acquire(ctx context.Context, key service.RegistryKey, start uint64) (service.RegistrySession, error) {
	if err := key.Validate(); err != nil {
		return nil, err
	}
	if start == 0 || start > math.MaxInt64 {
		return nil, errors.New("invalid registry start block")
	}
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, storageError(err)
	}
	s := &registrySession{conn: conn, key: key, start: start}
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1, 0))`, key.String()); err != nil {
		// Cancellation can race lock acquisition. Discard the physical session so
		// an acquired lock can never return to the connection pool accidentally.
		s.discard()
		return nil, storageError(err)
	}
	_, err = conn.Exec(ctx, `INSERT INTO registry_sync_state
  (chain_id,platform,registry_address,start_block,decoder_version) VALUES ($1,$2,$3,$4,$5)
  ON CONFLICT (chain_id,platform,registry_address) DO NOTHING`, key.ChainID, key.Platform, key.Controller.Bytes(), int64(start), service.RegistryDecoderVersion)
	if err == nil {
		_, err = s.Cursor(ctx)
	}
	if err != nil {
		_ = s.Close()
		return nil, storageError(err)
	}
	return s, nil
}

type registrySession struct {
	conn    *pgxpool.Conn
	key     service.RegistryKey
	start   uint64
	lockKey string
}

func (s *registrySession) Cursor(ctx context.Context) (service.RegistryCursor, error) {
	var start int64
	var version int
	var last *int64
	var hash []byte
	err := s.conn.QueryRow(ctx, `SELECT start_block,decoder_version,last_scanned_block,last_scanned_hash
  FROM registry_sync_state WHERE chain_id=$1 AND platform=$2 AND registry_address=$3`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes()).Scan(&start, &version, &last, &hash)
	if err != nil {
		return service.RegistryCursor{}, storageError(err)
	}
	if uint64(start) != s.start || version != service.RegistryDecoderVersion {
		return service.RegistryCursor{}, errors.New("registry start block/decoder differs from saved configuration")
	}
	c := service.RegistryCursor{StartBlock: uint64(start)}
	if last != nil {
		c.Last = &service.RegistryCheckpoint{Block: uint64(*last), Hash: common.BytesToHash(hash)}
	}
	return c, nil
}

func (s *registrySession) PreviousCheckpoint(ctx context.Context, before uint64) (*service.RegistryCheckpoint, error) {
	if before > math.MaxInt64 {
		return nil, errors.New("invalid checkpoint block")
	}
	var block int64
	var hash []byte
	err := s.conn.QueryRow(ctx, `SELECT block_number,block_hash FROM registry_checkpoints
  WHERE chain_id=$1 AND platform=$2 AND registry_address=$3 AND block_number<$4
  ORDER BY block_number DESC LIMIT 1`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(before)).Scan(&block, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, storageError(err)
	}
	return &service.RegistryCheckpoint{Block: uint64(block), Hash: common.BytesToHash(hash)}, nil
}

func (s *registrySession) SaveRange(ctx context.Context, from uint64, end service.RegistryCheckpoint, events []service.RegistryEvent) error {
	if from > end.Block || end.Block > math.MaxInt64 || end.Hash == (common.Hash{}) {
		return errors.New("invalid registry range")
	}
	for _, e := range events {
		if err := e.Validate(from, end.Block); err != nil {
			return err
		}
	}
	cursor, err := s.Cursor(ctx)
	if err != nil {
		return err
	}
	expected := cursor.StartBlock
	if cursor.Last != nil {
		expected = cursor.Last.Block + 1
	}
	if from != expected {
		return errors.New("registry ranges must be saved contiguously")
	}
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return storageError(err)
	}
	defer rollback(tx)
	for _, e := range events {
		var vault, asset, receiver, protocol []byte
		if e.Kind == service.RegistryVaultRegistered {
			vault = e.Info.Vault.Bytes()
			asset = e.Info.Asset.Bytes()
			receiver = e.Info.RewardReceiver.Bytes()
			protocol = e.Info.ProtocolID[:]
		}
		_, err = tx.Exec(ctx, `INSERT INTO registry_events
   (chain_id,platform,registry_address,block_number,block_hash,transaction_hash,transaction_index,log_index,event_kind,gauge_address,vault_address,asset_address,reward_receiver,protocol_id)
   VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(e.Block), e.BlockHash.Bytes(), e.TxHash.Bytes(), int64(e.TxIndex), int64(e.LogIndex), e.Kind, e.Info.Gauge.Bytes(), vault, asset, receiver, protocol)
		if err != nil {
			return storageError(err)
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO registry_checkpoints (chain_id,platform,registry_address,block_number,block_hash)
  VALUES ($1,$2,$3,$4,$5)`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(end.Block), end.Hash.Bytes())
	if err != nil {
		return storageError(err)
	}
	_, err = tx.Exec(ctx, `UPDATE registry_sync_state SET last_scanned_block=$4,last_scanned_hash=$5,updated_at=now()
  WHERE chain_id=$1 AND platform=$2 AND registry_address=$3`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(end.Block), end.Hash.Bytes())
	if err != nil {
		return storageError(err)
	}
	_, err = tx.Exec(ctx, `DELETE FROM registry_checkpoints
  WHERE chain_id=$1 AND platform=$2 AND registry_address=$3 AND block_number < (
   SELECT block_number FROM registry_checkpoints
   WHERE chain_id=$1 AND platform=$2 AND registry_address=$3
   ORDER BY block_number DESC OFFSET $4 LIMIT 1)`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), service.RegistryCheckpointWindow-1)
	if err != nil {
		return storageError(err)
	}
	return storageError(tx.Commit(ctx))
}

func (s *registrySession) Rewind(ctx context.Context, point service.RegistryCheckpoint) error {
	if point.Block > math.MaxInt64 {
		return errors.New("invalid rewind block")
	}
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return storageError(err)
	}
	defer rollback(tx)
	var hash []byte
	err = tx.QueryRow(ctx, `SELECT block_hash FROM registry_checkpoints
  WHERE chain_id=$1 AND platform=$2 AND registry_address=$3 AND block_number=$4`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(point.Block)).Scan(&hash)
	if err != nil {
		return storageError(err)
	}
	if common.BytesToHash(hash) != point.Hash {
		return errors.New("rewind checkpoint hash differs")
	}
	for _, table := range []string{"registry_events", "registry_checkpoints"} {
		// Only fixed table names above, never user input.
		_, err = tx.Exec(ctx, `DELETE FROM `+table+` WHERE chain_id=$1 AND platform=$2 AND registry_address=$3 AND block_number>$4`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(point.Block))
		if err != nil {
			return storageError(err)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE registry_sync_state SET last_scanned_block=$4,last_scanned_hash=$5,updated_at=now()
  WHERE chain_id=$1 AND platform=$2 AND registry_address=$3`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(point.Block), point.Hash.Bytes())
	if err != nil {
		return storageError(err)
	}
	return storageError(tx.Commit(ctx))
}

func (s *registrySession) Registrations(ctx context.Context, through uint64) ([]service.StakeDAOVaultInfo, error) {
	if through > math.MaxInt64 {
		return nil, errors.New("invalid registry snapshot block")
	}
	rows, err := s.conn.Query(ctx, `SELECT DISTINCT ON (gauge_address)
  gauge_address,vault_address,asset_address,reward_receiver,protocol_id FROM registry_events
  WHERE chain_id=$1 AND platform=$2 AND registry_address=$3 AND event_kind='vault_registered' AND block_number<=$4
  ORDER BY gauge_address,block_number DESC,transaction_index DESC,log_index DESC`, s.key.ChainID, s.key.Platform, s.key.Controller.Bytes(), int64(through))
	if err != nil {
		return nil, storageError(err)
	}
	defer rows.Close()
	var all []service.StakeDAOVaultInfo
	for rows.Next() {
		var gauge, vault, asset, receiver, protocol []byte
		if err := rows.Scan(&gauge, &vault, &asset, &receiver, &protocol); err != nil {
			return nil, storageError(err)
		}
		info := service.StakeDAOVaultInfo{Gauge: common.BytesToAddress(gauge), Vault: common.BytesToAddress(vault), Asset: common.BytesToAddress(asset), RewardReceiver: common.BytesToAddress(receiver)}
		copy(info.ProtocolID[:], protocol)
		all = append(all, info)
	}
	return all, storageError(rows.Err())
}

func (s *registrySession) discard() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.conn.Conn().Close(ctx)
	s.conn.Release()
	s.conn = nil
}

func (s *registrySession) Close() error {
	if s.conn == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var unlocked bool
	lockKey := s.lockKey
	if lockKey == "" {
		lockKey = s.key.String()
	}
	err := s.conn.QueryRow(ctx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, lockKey).Scan(&unlocked)
	if err != nil || !unlocked {
		s.discard()
		return service.ErrRegistryUnavailable
	}
	s.conn.Release()
	s.conn = nil
	return nil
}

// Do not propagate DSNs/passwords or raw database errors into logs/HTTP bodies.
func storageError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return service.ErrRegistryUnavailable
}

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

// Swap history uses the same configured Postgres pool, with a separate stream
// lock and cursor. No second connection configuration or in-app DDL is needed.
func (r *RegistryPostgres) SwapBackend() string { return "postgres" }
func (r *RegistryPostgres) AcquireSwaps(ctx context.Context, key service.SwapKey) (service.SwapSession, error) {
	if e := key.Validate(); e != nil {
		return nil, e
	}
	conn, e := r.pool.Acquire(ctx)
	if e != nil {
		return nil, storageError(e)
	}
	base := &registrySession{conn: conn, lockKey: key.String()}
	s := &swapSession{registrySession: base, key: key}
	if _, e = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, key.String()); e != nil {
		base.discard()
		return nil, storageError(e)
	}
	_, e = conn.Exec(ctx, `INSERT INTO swap_streams(chain_id,platform,pool_address,event_topic,decoder_version) VALUES($1,$2,$3,$4,$5)
 ON CONFLICT(chain_id,platform,pool_address,event_topic) DO NOTHING`, key.ChainID, key.Platform, key.Pool.Bytes(), key.Event.Bytes(), service.SwapDecoderVersion)
	if e == nil {
		_, e = s.Cursor(ctx)
	}
	if e != nil {
		_ = s.Close()
		return nil, storageError(e)
	}
	return s, nil
}

type swapSession struct {
	*registrySession
	key service.SwapKey
}

func (s *swapSession) args() []interface{} {
	return []interface{}{s.key.ChainID, s.key.Platform, s.key.Pool.Bytes(), s.key.Event.Bytes()}
}
func (s *swapSession) Cursor(ctx context.Context) (*service.RegistryCheckpoint, error) {
	var version int
	var block *int64
	var hash []byte
	e := s.conn.QueryRow(ctx, `SELECT decoder_version,last_saved_block,last_saved_hash FROM swap_streams WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4`, s.args()...).Scan(&version, &block, &hash)
	if e != nil {
		return nil, storageError(e)
	}
	if version != service.SwapDecoderVersion {
		return nil, errors.New("swap decoder differs from stored configuration")
	}
	if block == nil {
		return nil, nil
	}
	return &service.RegistryCheckpoint{Block: uint64(*block), Hash: common.BytesToHash(hash)}, nil
}
func (s *swapSession) Coverage(ctx context.Context, from, to uint64) ([]service.SwapRange, error) {
	if from == 0 || from > to || to > math.MaxInt64 {
		return nil, errors.New("invalid swap coverage window")
	}
	args := append(s.args(), int64(from), int64(to))
	rows, e := s.conn.Query(ctx, `SELECT from_block,through_block,through_hash FROM swap_scan_ranges
 WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4 AND through_block>=$5 AND from_block<=$6 ORDER BY from_block`, args...)
	if e != nil {
		return nil, storageError(e)
	}
	defer rows.Close()
	result := []service.SwapRange{}
	for rows.Next() {
		var f, t int64
		var h []byte
		if e = rows.Scan(&f, &t, &h); e != nil {
			return nil, storageError(e)
		}
		result = append(result, service.SwapRange{From: uint64(f), Through: service.RegistryCheckpoint{Block: uint64(t), Hash: common.BytesToHash(h)}})
	}
	return result, storageError(rows.Err())
}
func (s *swapSession) SaveSwapRange(ctx context.Context, from uint64, end service.RegistryCheckpoint, logs []types.Log) error {
	if from == 0 || from > end.Block || end.Block >= math.MaxInt64 || end.Hash == (common.Hash{}) {
		return errors.New("invalid swap save range")
	}
	for _, l := range logs {
		if e := service.ValidateStoredSwap(s.key, l, from, end.Block); e != nil {
			return e
		}
	}
	tx, e := s.conn.Begin(ctx)
	if e != nil {
		return storageError(e)
	}
	defer rollback(tx)
	batch := &pgx.Batch{}
	for _, l := range logs {
		args := append(s.args(), int64(l.BlockNumber), l.BlockHash.Bytes(), l.TxHash.Bytes(), int64(l.TxIndex), int64(l.Index), l.Topics[1][12:], l.Data)
		batch.Queue(`INSERT INTO swap_events(chain_id,platform,pool_address,event_topic,block_number,block_hash,transaction_hash,transaction_index,log_index,buyer_address,event_data)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
 ON CONFLICT(chain_id,platform,pool_address,event_topic,block_number,log_index) DO UPDATE SET event_data=EXCLUDED.event_data
 WHERE swap_events.block_hash=EXCLUDED.block_hash AND swap_events.transaction_hash=EXCLUDED.transaction_hash
 AND swap_events.transaction_index=EXCLUDED.transaction_index AND swap_events.buyer_address=EXCLUDED.buyer_address AND swap_events.event_data=EXCLUDED.event_data`, args...)
	}
	br := tx.SendBatch(ctx, batch)
	for range logs {
		tag, err := br.Exec()
		if err != nil {
			_ = br.Close()
			return storageError(err)
		}
		if tag.RowsAffected() != 1 {
			_ = br.Close()
			return service.ErrRegistryReorg
		}
	}
	if e = br.Close(); e != nil {
		return storageError(e)
	}
	args := append(s.args(), int64(from)-1, int64(end.Block)+1)
	rows, e := tx.Query(ctx, `SELECT from_block,through_block,through_hash FROM swap_scan_ranges
 WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4 AND through_block>=$5 AND from_block<=$6 ORDER BY from_block`, args...)
	if e != nil {
		return storageError(e)
	}
	mergedFrom, mergedEnd, mergedHash := from, end.Block, end.Hash
	for rows.Next() {
		var f, t int64
		var h []byte
		if e = rows.Scan(&f, &t, &h); e != nil {
			rows.Close()
			return storageError(e)
		}
		mergedFrom = min(mergedFrom, uint64(f))
		if uint64(t) > mergedEnd {
			mergedEnd = uint64(t)
			mergedHash = common.BytesToHash(h)
		} else if uint64(t) == mergedEnd && !bytes.Equal(h, mergedHash.Bytes()) {
			rows.Close()
			return service.ErrRegistryReorg
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return storageError(e)
	}
	_, e = tx.Exec(ctx, `DELETE FROM swap_scan_ranges WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4 AND through_block>=$5 AND from_block<=$6`, args...)
	if e != nil {
		return storageError(e)
	}
	_, e = tx.Exec(ctx, `INSERT INTO swap_scan_ranges(chain_id,platform,pool_address,event_topic,from_block,through_block,through_hash) VALUES($1,$2,$3,$4,$5,$6,$7)`, append(s.args(), int64(mergedFrom), int64(mergedEnd), mergedHash.Bytes())...)
	if e != nil {
		return storageError(e)
	}
	if e = s.updateSwapCursor(ctx, tx); e != nil {
		return e
	}
	return storageError(tx.Commit(ctx))
}
func (s *swapSession) updateSwapCursor(ctx context.Context, tx pgx.Tx) error {
	_, e := tx.Exec(ctx, `UPDATE swap_streams s SET last_saved_block=(SELECT max(through_block) FROM swap_scan_ranges r
 WHERE r.chain_id=s.chain_id AND r.platform=s.platform AND r.pool_address=s.pool_address AND r.event_topic=s.event_topic),
 last_saved_hash=(SELECT through_hash FROM swap_scan_ranges r WHERE r.chain_id=s.chain_id AND r.platform=s.platform
 AND r.pool_address=s.pool_address AND r.event_topic=s.event_topic ORDER BY through_block DESC LIMIT 1),updated_at=now()
 WHERE s.chain_id=$1 AND s.platform=$2 AND s.pool_address=$3 AND s.event_topic=$4`, s.args()...)
	return storageError(e)
}
func (s *swapSession) DropSwapRange(ctx context.Context, r service.SwapRange) error {
	if r.From == 0 || r.From > r.Through.Block || r.Through.Block > math.MaxInt64 {
		return errors.New("invalid swap invalidation range")
	}
	tx, e := s.conn.Begin(ctx)
	if e != nil {
		return storageError(e)
	}
	defer rollback(tx)
	args := append(s.args(), int64(r.From), int64(r.Through.Block))
	var h []byte
	e = tx.QueryRow(ctx, `SELECT through_hash FROM swap_scan_ranges WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4 AND from_block=$5 AND through_block=$6`, args...).Scan(&h)
	if e != nil {
		return storageError(e)
	}
	if !bytes.Equal(h, r.Through.Hash.Bytes()) {
		return service.ErrRegistryReorg
	}
	_, e = tx.Exec(ctx, `DELETE FROM swap_events WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4 AND block_number BETWEEN $5 AND $6`, args...)
	if e != nil {
		return storageError(e)
	}
	_, e = tx.Exec(ctx, `DELETE FROM swap_scan_ranges WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4 AND from_block=$5 AND through_block=$6`, args...)
	if e != nil {
		return storageError(e)
	}
	if e = s.updateSwapCursor(ctx, tx); e != nil {
		return e
	}
	return storageError(tx.Commit(ctx))
}
func (s *swapSession) Swaps(ctx context.Context, from, to uint64) ([]types.Log, error) {
	if from == 0 || from > to || to > math.MaxInt64 {
		return nil, errors.New("invalid saved swap window")
	}
	rows, e := s.conn.Query(ctx, `SELECT block_number,block_hash,transaction_hash,transaction_index,log_index,buyer_address,event_data FROM swap_events
 WHERE chain_id=$1 AND platform=$2 AND pool_address=$3 AND event_topic=$4 AND block_number BETWEEN $5 AND $6
 ORDER BY block_number,log_index LIMIT 50001`, append(s.args(), int64(from), int64(to))...)
	if e != nil {
		return nil, storageError(e)
	}
	defer rows.Close()
	result := []types.Log{}
	for rows.Next() {
		var block, ti, li int64
		var bh, th, buyer, data []byte
		if e = rows.Scan(&block, &bh, &th, &ti, &li, &buyer, &data); e != nil {
			return nil, storageError(e)
		}
		result = append(result, types.Log{Address: s.key.Pool, BlockNumber: uint64(block), BlockHash: common.BytesToHash(bh), TxHash: common.BytesToHash(th), TxIndex: uint(ti), Index: uint(li), Topics: []common.Hash{s.key.Event, common.BytesToHash(buyer)}, Data: data})
		if len(result) > 50000 {
			return nil, errors.New("swap event count exceeds diagnostic limit")
		}
	}
	return result, storageError(rows.Err())
}
