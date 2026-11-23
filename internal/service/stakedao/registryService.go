package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math"
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

// SyncSwapEvents extends the existing storage service, without mixing its
// stream/cursor with Controller registrations. Save only finalized intervals;
// expanding 1 -> 7 -> 30 days backfills holes instead of resetting a cursor.
func (s *RegistryService) SyncSwapEvents(ctx context.Context, client registryChainReader, key SwapKey, from, to uint64) (logs []types.Log, stats SwapSyncStats, err error) {
	if e := key.Validate(); e != nil {
		return nil, stats, e
	}
	if from == 0 || from > to || to > math.MaxInt64 || (to-from)/1500+1 > 160 {
		return nil, stats, errors.New("invalid_swap_history_window")
	}
	store, ok := s.store.(SwapStore)
	if !ok {
		stats.Backend = "not_supported"
		logs, stats.RPCRangesRead, err = readSwapLogs(ctx, client, key.Pool, from, to, key.Event)
		stats.FetchedEvents = len(logs)
		return
	}
	stats.Backend = "configured_store"
	if named, ok := store.(interface{ SwapBackend() string }); ok {
		stats.Backend = named.SwapBackend()
	}
	session, e := store.AcquireSwaps(ctx, key)
	if e != nil {
		return nil, stats, swapStorageError(e)
	}
	defer session.Close()
	updateCursor := func() error {
		point, e := session.Cursor(ctx)
		if e != nil {
			return swapStorageError(e)
		}
		if point != nil {
			n := point.Block
			stats.LastSavedBlock = &n
		} else {
			stats.LastSavedBlock = nil
		}
		return nil
	}
	if e = updateCursor(); e != nil {
		return nil, stats, e
	}
	finalized, e := client.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if e != nil || finalized == nil || finalized.Number == nil || !finalized.Number.IsUint64() {
		return nil, stats, errors.New("swap_finalized_header_unavailable")
	}
	finalizedHead := finalized.Number.Uint64()
	through := min(to, finalizedHead)
	stats.FinalizedThrough = &through
	var covered []SwapRange
	{
		covered, e = session.Coverage(ctx, from, to)
		if e != nil {
			return nil, stats, swapStorageError(e)
		}
		valid := []SwapRange{}
		for _, r := range covered {
			if r.From == 0 || r.From > r.Through.Block || r.Through.Hash == (common.Hash{}) {
				return nil, stats, errors.New("invalid_swap_coverage")
			}
			if r.Through.Block > finalizedHead {
				return nil, stats, errors.New("logs_rpc_finalized_head_behind_storage")
			}
			header, e := readRegistryHeader(ctx, client, r.Through.Block)
			if e != nil {
				return nil, stats, errors.New("swap_checkpoint_header_unavailable")
			}
			if header.Hash() != r.Through.Hash {
				if e = session.DropSwapRange(ctx, r); e != nil {
					return nil, stats, swapStorageError(e)
				}
				if e = updateCursor(); e != nil {
					return nil, stats, e
				}
			} else {
				valid = append(valid, r)
			}
		}
		covered = valid
	}
	var cached []types.Log
	if from <= through {
		cached, e = session.Swaps(ctx, from, through)
		if e != nil {
			return nil, stats, swapStorageError(e)
		}
	}
	stats.CachedEvents = len(cached)
	for _, hole := range missingSwapRanges(from, through, covered) {
		// A bounded batch keeps successful progress across interruptions while adding
		// only two header reads per 16 log ranges (not per swap or per log range).
		for next := hole.From; next <= hole.Through.Block; {
			end := min(next+24000-1, hole.Through.Block)
			before, e := readRegistryHeader(ctx, client, end)
			if e != nil {
				return nil, stats, errors.New("swap_checkpoint_header_unavailable")
			}
			rows, done, e := readSwapLogs(ctx, client, key.Pool, next, end, key.Event)
			stats.RPCRangesRead += done
			if e != nil {
				return nil, stats, e
			}
			rows, e = validateSwapEventRange(key, rows, next, end)
			if e != nil {
				return nil, stats, e
			}
			after, e := readRegistryHeader(ctx, client, end)
			if e != nil || after.Hash() != before.Hash() {
				return nil, stats, errors.New("swap_history_snapshot_changed")
			}
			if e = session.SaveSwapRange(ctx, next, RegistryCheckpoint{Block: end, Hash: after.Hash()}, rows); e != nil {
				return nil, stats, swapStorageError(e)
			}
			stats.FetchedEvents += len(rows)
			if e = updateCursor(); e != nil {
				return nil, stats, e
			}
			if end == hole.Through.Block {
				break
			}
			next = end + 1
		}
	}
	var saved []types.Log
	if from <= through {
		saved, e = session.Swaps(ctx, from, through)
		if e != nil {
			return nil, stats, swapStorageError(e)
		}
	}
	tailFrom := from
	if through >= from {
		tailFrom = through + 1
	}
	if tailFrom <= to {
		tail, done, e := readSwapLogs(ctx, client, key.Pool, tailFrom, to, key.Event)
		stats.RPCRangesRead += done
		if e != nil {
			return nil, stats, e
		}
		tail, e = validateSwapEventRange(key, tail, tailFrom, to)
		if e != nil {
			return nil, stats, e
		}
		stats.FetchedEvents += len(tail)
		saved = append(saved, tail...)
	}
	if len(saved) > 50000 {
		return nil, stats, errors.New("swap_event_limit")
	}
	logs, e = validateSwapEventRange(key, saved, from, to)
	if e != nil {
		return nil, stats, e
	}
	if e = updateCursor(); e != nil {
		return nil, stats, e
	}
	return logs, stats, nil
}

func swapStorageError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return errors.New("swap_storage_unavailable")
}
func missingSwapRanges(from, to uint64, covered []SwapRange) []SwapRange {
	if from > to {
		return nil
	}
	sort.Slice(covered, func(i, j int) bool { return covered[i].From < covered[j].From })
	out := []SwapRange{}
	next := from
	for _, r := range covered {
		if r.Through.Block < next {
			continue
		}
		if r.From > to {
			break
		}
		if r.From > next {
			out = append(out, SwapRange{From: next, Through: RegistryCheckpoint{Block: min(to, r.From-1)}})
		}
		if r.Through.Block >= to {
			return out
		}
		next = max(next, r.Through.Block+1)
	}
	if next <= to {
		out = append(out, SwapRange{From: next, Through: RegistryCheckpoint{Block: to}})
	}
	return out
}
func validateSwapEventRange(key SwapKey, logs []types.Log, from, to uint64) ([]types.Log, error) {
	parsed, e := abi.JSON(strings.NewReader(weETHFeeABI))
	if e != nil {
		return nil, e
	}
	event := parsed.Events["TokenExchange"]
	if key.Event != event.ID {
		return nil, errors.New("unsupported_swap_decoder")
	}
	positions := map[[2]uint64]common.Hash{}
	for _, entry := range logs {
		if e := ValidateStoredSwap(key, entry, from, to); e != nil {
			return nil, e
		}
		position := [2]uint64{entry.BlockNumber, uint64(entry.Index)}
		if old, ok := positions[position]; ok && old != entry.TxHash {
			return nil, errors.New("conflicting_swap_position")
		}
		positions[position] = entry.TxHash
	}
	decoded, e := decodeCurveSwaps(logs, key.Pool, event)
	if e != nil {
		return nil, e
	}
	result := make([]types.Log, 0, len(decoded))
	for _, v := range decoded {
		result = append(result, v.Log)
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.BlockNumber != b.BlockNumber {
			return a.BlockNumber < b.BlockNumber
		}
		return a.Index < b.Index
	})
	return result, nil
}
