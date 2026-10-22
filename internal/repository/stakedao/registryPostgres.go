// Package stakedao implements PostgreSQL persistence; blockchain reads stay in the service.
package stakedao

import (
	"context"
	"errors"
	"math"
	"time"

	service "ABIevm/internal/service/stakedao"
	"github.com/ethereum/go-ethereum/common"
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
	if err != nil {
		pool.Close()
		return nil, errors.New("Postgres connection/schema unavailable; check DATABASE_URL and apply migrations/000001_registry.up.sql")
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
	conn  *pgxpool.Conn
	key   service.RegistryKey
	start uint64
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
	err := s.conn.QueryRow(ctx, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, s.key.String()).Scan(&unlocked)
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
