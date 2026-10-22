package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const registryStatusEventsABI = `[
{"type":"event","name":"GaugeShutdown","anonymous":false,"inputs":[{"name":"gauge","type":"address","indexed":true}]},
{"type":"event","name":"GaugeUnshutdown","anonymous":false,"inputs":[{"name":"gauge","type":"address","indexed":true}]}]`

type registryChainReader interface {
	vaultLogReader
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

// RegistryService owns discovery/cursor reconciliation, not SQL or pricing.
// Only finalized state is persisted. The small unfinalized tail is read afresh
// for each fixed calculation block and never advances the durable cursor.
type RegistryService struct{ store RegistryStore }

func NewRegistryService(store RegistryStore) *RegistryService { return &RegistryService{store: store} }

func (s *RegistryService) SyncAndList(ctx context.Context, client registryChainReader, key RegistryKey, start, snapshot uint64) ([]StakeDAOVaultInfo, error) {
	if err := key.Validate(); err != nil {
		return nil, err
	}
	if s.store == nil || start == 0 || start > snapshot {
		return nil, errors.New("invalid registry synchronization configuration")
	}
	progress := progressFrom(ctx)
	progress.setStage("registry_lock")
	session, err := s.store.Acquire(ctx, key, start)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	progress.setStage("registry_checkpoints")
	finalized, err := client.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return nil, fmt.Errorf("%w: finalized header: %w", ErrRPCUnavailable, err)
	}
	if finalized == nil || finalized.Number == nil || !finalized.Number.IsUint64() {
		return nil, errors.New("invalid finalized header")
	}
	finalizedHead := finalized.Number.Uint64()
	through := finalizedHead
	if through > snapshot {
		through = snapshot
		finalized, err = readRegistryHeader(ctx, client, through)
		if err != nil {
			return nil, err
		}
	}
	cursor, err := s.checkCursor(ctx, client, session, finalizedHead)
	if err != nil {
		return nil, err
	}
	progress.registryProgress(cursor.Last, through)
	from := start
	if cursor.Last != nil {
		from = cursor.Last.Block + 1
	}
	if from <= through {
		progress.setStage("registry_finalized")
		parsed, err := abi.JSON(strings.NewReader(strings.TrimSuffix(vaultRegisteredABI, "]") + "," + strings.TrimPrefix(registryStatusEventsABI, "[")))
		if err != nil {
			return nil, err
		}
		topics := map[common.Hash]int{}
		for _, event := range parsed.Events {
			count := 1
			for _, input := range event.Inputs {
				if input.Indexed {
					count++
				}
			}
			topics[event.ID] = count
		}
		err = readRegistryRanges(ctx, client, key.Controller, topics, from, through, func(interval discoveryRange, logs []types.Log) error {
			events, err := decodeRegistryEvents(parsed, logs, interval)
			if err != nil {
				return err
			}
			header, err := readRegistryHeader(ctx, client, interval.to)
			if err != nil {
				return err
			}
			if interval.to == through && header.Hash() != finalized.Hash() {
				return ErrRegistryReorg
			}
			// Registry events are sparse: check their hashes without fetching a
			// header for every empty chain block.
			hashes := map[uint64]common.Hash{interval.to: header.Hash()}
			for _, event := range events {
				hash, found := hashes[event.Block]
				if !found {
					eventHeader, err := readRegistryHeader(ctx, client, event.Block)
					if err != nil {
						return err
					}
					hash = eventHeader.Hash()
					hashes[event.Block] = hash
				}
				if hash != event.BlockHash {
					return ErrRegistryReorg
				}
			}
			point := RegistryCheckpoint{Block: interval.to, Hash: header.Hash()}
			if err := session.SaveRange(ctx, interval.from, point, events); err != nil {
				return err
			}
			progress.registryProgress(&point, through)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if through >= start {
		verify, err := readRegistryHeader(ctx, client, through)
		if err != nil {
			return nil, err
		}
		if verify.Hash() != finalized.Hash() {
			return nil, ErrRegistryReorg
		}
	}
	saved, err := session.Registrations(ctx, through)
	if err != nil {
		return nil, err
	}
	// Merge by gauge, keeping the newest registration from the unfinalized tail.
	latest := make(map[common.Address]StakeDAOVaultInfo, len(saved))
	for _, info := range saved {
		latest[info.Gauge] = info
	}
	tailFrom := through + 1
	if tailFrom < start {
		tailFrom = start
	}
	if tailFrom <= snapshot {
		progress.setStage("registry_tail")
		tail, err := DiscoverStakeDAOVaults(ctx, client, key.Controller, tailFrom, snapshot)
		if err != nil {
			return nil, err
		}
		for _, info := range tail {
			latest[info.Gauge] = info
		}
	}
	all := make([]StakeDAOVaultInfo, 0, len(latest))
	for _, info := range latest {
		all = append(all, info)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Gauge.Hex() < all[j].Gauge.Hex() })
	return all, nil
}

// Never silently reset history on a different start block, lagging node, or a
// deep reorganization. The bounded checkpoint search avoids an unbounded RPC scan.
func (s *RegistryService) checkCursor(ctx context.Context, client registryChainReader, session RegistrySession, through uint64) (RegistryCursor, error) {
	cursor, err := session.Cursor(ctx)
	if err != nil {
		return cursor, err
	}
	if cursor.Last == nil {
		return cursor, nil
	}
	if cursor.Last.Block > through {
		return cursor, fmt.Errorf("%w: finalized RPC head is behind the saved registry cursor", ErrRPCUnavailable)
	}
	point := cursor.Last
	for attempt := 0; attempt < RegistryCheckpointWindow && point != nil; attempt++ {
		header, err := readRegistryHeader(ctx, client, point.Block)
		if err != nil {
			return cursor, err
		}
		if header.Hash() == point.Hash {
			if point.Block != cursor.Last.Block {
				if err := session.Rewind(ctx, *point); err != nil {
					return cursor, err
				}
				cursor.Last = point
			}
			return cursor, nil
		}
		point, err = session.PreviousCheckpoint(ctx, point.Block)
		if err != nil {
			return cursor, err
		}
	}
	return cursor, fmt.Errorf("%w: no matching checkpoint in the rollback window; explicit registry recovery is required", ErrRegistryReorg)
}

func readRegistryHeader(ctx context.Context, client registryChainReader, block uint64) (*types.Header, error) {
	header, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(block))
	if err != nil {
		return nil, fmt.Errorf("%w: registry block %d: %w", ErrRPCUnavailable, block, err)
	}
	if header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Uint64() != block {
		return nil, errors.New("invalid registry checkpoint header")
	}
	return header, nil
}

func decodeRegistryEvents(parsed abi.ABI, logs []types.Log, interval discoveryRange) ([]RegistryEvent, error) {
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		if logs[i].TxIndex != logs[j].TxIndex {
			return logs[i].TxIndex < logs[j].TxIndex
		}
		return logs[i].Index < logs[j].Index
	})
	events := make([]RegistryEvent, 0, len(logs))
	for _, entry := range logs {
		if entry.Removed {
			continue
		}
		if len(entry.Topics) == 0 {
			return nil, errors.New("registry log has no event topic")
		}
		event, err := parsed.EventByID(entry.Topics[0])
		if err != nil {
			return nil, err
		}
		e := RegistryEvent{Block: entry.BlockNumber, BlockHash: entry.BlockHash, TxHash: entry.TxHash, TxIndex: uint64(entry.TxIndex), LogIndex: uint64(entry.Index)}
		switch event.Name {
		case "VaultRegistered":
			e.Kind = RegistryVaultRegistered
			e.Info, err = decodeVaultRegistration(parsed, entry)
		case "GaugeShutdown", "GaugeUnshutdown":
			if len(entry.Topics) != 2 || len(entry.Data) != 0 || new(big.Int).SetBytes(entry.Topics[1][:12]).Sign() != 0 {
				return nil, errors.New("invalid gauge status log")
			}
			e.Info.Gauge = common.BytesToAddress(entry.Topics[1].Bytes())
			e.Kind = RegistryGaugeShutdown
			if event.Name == "GaugeUnshutdown" {
				e.Kind = RegistryGaugeUnshutdown
			}
		default:
			return nil, errors.New("unexpected registry event")
		}
		if err != nil {
			return nil, err
		}
		if err := e.Validate(interval.from, interval.to); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
