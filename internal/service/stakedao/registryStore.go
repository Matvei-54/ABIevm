package stakedao

import (
	dto "ABIevm/internal/service/stakedao/dto"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

// Swap streams are isolated from Controller registrations, including their
// coverage/cursors. A range cache supports expanding a window backwards too.
const SwapDecoderVersion = 1

type SwapKey struct {
	ChainID  int64
	Platform string
	Pool     common.Address
	Event    common.Hash
}

func (k SwapKey) Validate() error {
	if err := (RegistryKey{ChainID: k.ChainID, Platform: k.Platform, Controller: k.Pool}).Validate(); err != nil {
		return err
	}
	if k.Event == (common.Hash{}) {
		return errors.New("invalid swap event key")
	}
	return nil
}
func (k SwapKey) String() string {
	return "swaps:" + (RegistryKey{ChainID: k.ChainID, Platform: k.Platform, Controller: k.Pool}).String() + ":" + k.Event.Hex()
}

type SwapRange struct {
	From    uint64
	Through RegistryCheckpoint
}

type SwapStore interface {
	AcquireSwaps(context.Context, SwapKey) (SwapSession, error)
}
type SwapSession interface {
	Cursor(context.Context) (*RegistryCheckpoint, error)
	Coverage(context.Context, uint64, uint64) ([]SwapRange, error)
	SaveSwapRange(context.Context, uint64, RegistryCheckpoint, []types.Log) error
	DropSwapRange(context.Context, SwapRange) error
	Swaps(context.Context, uint64, uint64) ([]types.Log, error)
	Close() error
}

type SwapSyncStats = dto.SwapStorageStats

func ValidateStoredSwap(key SwapKey, entry types.Log, from, to uint64) error {
	if err := key.Validate(); err != nil {
		return err
	}
	if from == 0 || from > to || to > math.MaxInt64 || entry.BlockNumber < from || entry.BlockNumber > to || entry.Address != key.Pool || entry.Removed || len(entry.Topics) != 2 || entry.Topics[0] != key.Event || len(entry.Data) != 128 || entry.BlockHash == (common.Hash{}) || entry.TxHash == (common.Hash{}) || uint64(entry.Index) > math.MaxInt64 || uint64(entry.TxIndex) > math.MaxInt64 || new(big.Int).SetBytes(entry.Topics[1][:12]).Sign() != 0 {
		return errors.New("invalid stored swap coordinates")
	}
	return nil
}
