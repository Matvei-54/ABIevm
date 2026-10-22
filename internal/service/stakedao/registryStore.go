package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrRegistryUnavailable = errors.New("StakeDAO registry storage unavailable")
	ErrRegistryReorg       = errors.New("StakeDAO registry checkpoint is not canonical")
)

const RegistryDecoderVersion = 1

// Retain bounded rollback checkpoints, not one forever for each polling cycle.
const RegistryCheckpointWindow = 16

// RegistryKey isolates platforms, networks and replacement Controllers.
type RegistryKey struct {
	ChainID    int64
	Platform   string
	Controller common.Address
}

func (k RegistryKey) Validate() error {
	if k.ChainID <= 0 || k.Platform == "" || len(k.Platform) > 64 || k.Controller == (common.Address{}) {
		return errors.New("invalid registry key")
	}
	return nil
}

func (k RegistryKey) String() string {
	return fmt.Sprintf("%d:%s:%s", k.ChainID, k.Platform, k.Controller.Hex())
}

type RegistryCheckpoint struct {
	Block uint64
	Hash  common.Hash
}

type RegistryCursor struct {
	StartBlock uint64
	Last       *RegistryCheckpoint
}

type RegistryEvent struct {
	Kind      string
	Info      StakeDAOVaultInfo
	Block     uint64
	BlockHash common.Hash
	TxHash    common.Hash
	TxIndex   uint64
	LogIndex  uint64
}

const (
	RegistryVaultRegistered = "vault_registered"
	RegistryGaugeShutdown   = "gauge_shutdown"
	RegistryGaugeUnshutdown = "gauge_unshutdown"
)

// Validate rejects malformed or out-of-range events before a database transaction.
func (e RegistryEvent) Validate(from, to uint64) error {
	if e.Block < from || e.Block > to || to > math.MaxInt64 || e.TxIndex > math.MaxInt64 || e.LogIndex > math.MaxInt64 ||
		e.BlockHash == (common.Hash{}) || e.TxHash == (common.Hash{}) || e.Info.Gauge == (common.Address{}) {
		return errors.New("invalid registry event coordinates")
	}
	switch e.Kind {
	case RegistryVaultRegistered:
		if e.Info.Vault == (common.Address{}) || e.Info.Asset == (common.Address{}) {
			return errors.New("invalid vault registration")
		}
	case RegistryGaugeShutdown, RegistryGaugeUnshutdown:
		if e.Info.Vault != (common.Address{}) || e.Info.Asset != (common.Address{}) || e.Info.RewardReceiver != (common.Address{}) || e.Info.ProtocolID != ([4]byte{}) {
			return errors.New("invalid gauge status event")
		}
	default:
		return errors.New("unknown registry event kind")
	}
	return nil
}

// A session exclusively owns one stream until Close, including across processes.
// SaveRange must commit events, checkpoint and cursor atomically. Failed ranges
// must leave the cursor unchanged; Rewind removes both orphaned events and progress.
type RegistryStore interface {
	Acquire(context.Context, RegistryKey, uint64) (RegistrySession, error)
}

type RegistrySession interface {
	Cursor(context.Context) (RegistryCursor, error)
	PreviousCheckpoint(context.Context, uint64) (*RegistryCheckpoint, error)
	SaveRange(context.Context, uint64, RegistryCheckpoint, []RegistryEvent) error
	Rewind(context.Context, RegistryCheckpoint) error
	Registrations(context.Context, uint64) ([]StakeDAOVaultInfo, error)
	Close() error
}
