package stakedao

// Adapted from the user's curveService.go volume/fee approach. The original
// Curve service is deliberately untouched. See WEETH_FEE_MODEL_RU.md for limits.
import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"sort"
	"strings"
	"sync"

	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
)

var weETHFeeGauge = common.HexToAddress("0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2")
var weETHFeePool = common.HexToAddress("0xdb74dfdd3bb46be8ce6c33dc9d82777bcfc3ded5")
var weETHFeeCoins = [2]common.Address{common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), common.HexToAddress("0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee")}

const weETHFeeABI = `[
{"type":"function","name":"coins","inputs":[{"type":"uint256"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"balances","inputs":[{"type":"uint256"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"stored_rates","inputs":[],"outputs":[{"type":"uint256[]"}]},
{"type":"function","name":"dynamic_fee","inputs":[{"type":"int128"},{"type":"int128"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"admin_fee","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"event","name":"TokenExchange","anonymous":false,"inputs":[{"name":"buyer","type":"address","indexed":true},{"name":"sold_id","type":"int128","indexed":false},{"name":"tokens_sold","type":"uint256","indexed":false},{"name":"bought_id","type":"int128","indexed":false},{"name":"tokens_bought","type":"uint256","indexed":false}]}
]`

func ValidateWeETHFeeSelection(gauges []common.Address, days int) error {
	if days == 0 {
		return nil
	}
	if (days != 1 && days != 7 && days != 30) || len(gauges) != 1 || gauges[0] != weETHFeeGauge {
		return errors.New("weethFeeDays must be 1, 7 or 30 and requires only gauge 0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2")
	}
	return nil
}

type weETHFeeReference struct {
	rates               [2]decimal.Decimal
	capital, fee, admin decimal.Decimal
}

// Uses the observed service client / fixed block; no HTTP prices or global Curve maps.
func (s *StakeDAOService) weETHFeeReference(ctx context.Context, block *big.Int) (weETHFeeReference, error) {
	var r weETHFeeReference
	reads := []snapshotRead{}
	for i, coin := range weETHFeeCoins {
		reads = append(reads,
			readContract(weETHFeePool, weETHFeeABI, "coins", big.NewInt(int64(i))),
			readContract(weETHFeePool, weETHFeeABI, "balances", big.NewInt(int64(i))),
			readContract(coin, erc20DecimalsABI, "decimals"))
	}
	reads = append(reads, readContract(weETHFeePool, weETHFeeABI, "stored_rates"),
		readContract(weETHFeePool, weETHFeeABI, "dynamic_fee", big.NewInt(0), big.NewInt(1)),
		readContract(weETHFeePool, weETHFeeABI, "admin_fee"),
		readContract(common.HexToAddress("0x6A8cbed756804B16E05E741eDaBd5cB544AE21bf"), feePoolABI, "get_pool_asset_types", weETHFeePool))
	data := s.snapshotCalls(ctx, block, reads...)
	for _, d := range data {
		if d.err != nil {
			return r, errors.New("fee_reference_unavailable")
		}
	}
	if len(data[6].values) != 1 || len(data[9].values) != 1 {
		return r, errors.New("invalid_fee_reference")
	}
	rates, ok := data[6].values[0].([]*big.Int)
	kinds, kOK := data[9].values[0].([]uint8)
	if !ok || !kOK || len(rates) != 2 || len(kinds) != 2 || kinds[0] != 0 || kinds[1] != 1 {
		return r, errors.New("unsupported_rate_model")
	}
	for i, coin := range weETHFeeCoins {
		actual, e1 := data[3*i].address()
		balance, e2 := data[3*i+1].unsigned()
		decimals, e3 := data[3*i+2].unsigned()
		if e1 != nil || e2 != nil || e3 != nil || actual != coin || decimals.Cmp(big.NewInt(18)) != 0 || rates[i] == nil || rates[i].Sign() <= 0 {
			return r, errors.New("invalid_fee_reference")
		}
		r.rates[i] = decimal.NewFromBigInt(rates[i], -18)
		r.capital = r.capital.Add(decimal.NewFromBigInt(balance, -18).Mul(r.rates[i]))
	}
	f, e1 := data[7].unsigned()
	a, e2 := data[8].unsigned()
	if e1 != nil || e2 != nil || f.Sign() <= 0 || f.Cmp(big.NewInt(10000000000)) >= 0 || a.Cmp(big.NewInt(5000000000)) != 0 || !r.rates[0].Equal(decimal.NewFromInt(1)) || r.rates[1].GreaterThan(decimal.NewFromInt(10)) || !r.capital.IsPositive() {
		return r, errors.New("invalid_fee_reference")
	}
	r.fee = decimal.NewFromBigInt(f, -10)
	r.admin = decimal.NewFromBigInt(a, -10)
	return r, nil
}

// Ranges are complete or the entire estimate is unavailable. No missing chunks
// or malformed swaps silently become zero volume. sampleRPC owns the shared RPS.
func readWeETHSwaps(ctx context.Context, client stakeDAORPC, from, to uint64, event common.Hash) ([]types.Log, int, error) {
	if from > to {
		return nil, 0, errors.New("invalid_fee_window")
	}
	total := (to-from)/1500 + 1
	if total > 160 {
		return nil, 0, errors.New("fee_history_range_limit")
	}
	jobs := make(chan uint64, int(total))
	for n := from; n <= to; {
		jobs <- n
		if to-n < 1500 {
			break
		}
		n += 1500
	}
	close(jobs)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var failed error
	done := 0
	all := []types.Log{}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range jobs {
				if workCtx.Err() != nil {
					return
				}
				logs, err := client.FilterLogs(workCtx, ethereum.FilterQuery{FromBlock: new(big.Int).SetUint64(n), ToBlock: new(big.Int).SetUint64(min(n+1499, to)), Addresses: []common.Address{weETHFeePool}, Topics: [][]common.Hash{{event}}})
				mu.Lock()
				if err != nil {
					if failed == nil {
						failed = errors.New("swap_history_unavailable")
						cancel()
					}
				} else {
					for _, l := range logs {
						if l.BlockNumber < n || l.BlockNumber > min(n+1499, to) {
							failed = errors.New("invalid_swap_range")
							cancel()
							break
						}
					}
					all = append(all, logs...)
					done++
					if len(all) > 50000 {
						failed = errors.New("swap_event_limit")
						cancel()
					}
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if failed != nil {
		return nil, done, failed
	}
	if ctx.Err() != nil {
		return nil, done, ctx.Err()
	}
	if done != int(total) {
		return nil, done, errors.New("incomplete_swap_history")
	}
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.BlockNumber != b.BlockNumber {
			return a.BlockNumber < b.BlockNumber
		}
		if a.TxIndex != b.TxIndex {
			return a.TxIndex < b.TxIndex
		}
		return a.Index < b.Index
	})
	return all, done, nil
}

// Net tokens_bought are converted to gross-fee estimates via f/(1-f). Snapshot
// f and rates are approximations, NOT the actual historic per-swap parameters.
func estimateWeETHSwaps(logs []types.Log, parsed abi.ABI, r weETHFeeReference) (decimal.Decimal, int, error) {
	total := decimal.Zero
	count := 0
	seen := map[string]types.Log{}
	event := parsed.Events["TokenExchange"]
	for _, l := range logs {
		if l.Address != weETHFeePool || l.Removed || len(l.Topics) != 2 || l.Topics[0] != event.ID || len(l.Data) != 128 || l.BlockHash == (common.Hash{}) || l.TxHash == (common.Hash{}) {
			return decimal.Zero, 0, errors.New("invalid_swap_log")
		}
		key := l.TxHash.Hex() + ":" + new(big.Int).SetUint64(uint64(l.Index)).String()
		if old, ok := seen[key]; ok {
			if old.BlockNumber != l.BlockNumber || old.BlockHash != l.BlockHash || old.TxIndex != l.TxIndex || old.Topics[1] != l.Topics[1] || !bytes.Equal(old.Data, l.Data) {
				return decimal.Zero, 0, errors.New("conflicting_swap_log")
			}
			continue
		}
		seen[key] = l
		v, err := event.Inputs.NonIndexed().Unpack(l.Data)
		if err != nil || len(v) != 4 {
			return decimal.Zero, 0, errors.New("invalid_swap_log")
		}
		sold, sOK := v[0].(*big.Int)
		amount, aOK := v[1].(*big.Int)
		bought, bOK := v[2].(*big.Int)
		net, nOK := v[3].(*big.Int)
		if !sOK || !aOK || !bOK || !nOK || sold == nil || bought == nil || amount == nil || net == nil || !sold.IsInt64() || !bought.IsInt64() || sold.Sign() < 0 || sold.Int64() > 1 || bought.Sign() < 0 || bought.Int64() > 1 || sold.Cmp(bought) == 0 || amount.Sign() <= 0 || net.Sign() <= 0 {
			return decimal.Zero, 0, errors.New("invalid_swap_log")
		}
		fee := decimal.NewFromBigInt(net, -18).Mul(r.fee).DivRound(decimal.NewFromInt(1).Sub(r.fee), 36).Mul(decimal.NewFromInt(1).Sub(r.admin)).Mul(r.rates[int(bought.Int64())])
		total = total.Add(fee)
		count++
	}
	return total, count, nil
}

func (s *StakeDAOService) addWeETHFeeEstimate(ctx context.Context, end *types.Header, days int, pool *dto.StakeDAOPool) {
	if pool.Diagnostics == nil {
		pool.Diagnostics = &dto.StakeDAOPoolDiagnostics{}
	}
	d := &dto.WeETHFeeEstimate{Status: "unavailable", Model: "snapshot_parameter_swap_fee_estimate", Pool: weETHFeePool, WindowDays: days, EndBlock: end.Number.Uint64(), EndBlockHash: end.Hash(),
		Assumptions: []string{"snapshot_dynamic_fee_for_all_swaps", "snapshot_rates_and_pool_capital_in_eth", "historical_parameters_not_replayed", "liquidity_operation_fees_excluded", "not_trading_apy_not_included_in_vault_apr"}}
	pool.Diagnostics.WeETHFees = d
	fail := func(e error) { d.Reasons = []string{e.Error()} }
	if s.chainID != 1 || pool.GaugeAddress != weETHFeeGauge || pool.AssetAddress != weETHFeePool || pool.ProtocolID != "0xc715e373" {
		fail(errors.New("unsupported_fee_pool"))
		return
	}
	reference, err := s.weETHFeeReference(ctx, end.Number)
	if err != nil {
		fail(err)
		return
	}
	start, err := findFeeBoundary(ctx, s.client, end, days, map[uint64]*types.Header{end.Number.Uint64(): end})
	if err != nil {
		fail(err)
		return
	}
	n, h, seconds := start.Number.Uint64(), start.Hash(), end.Time-start.Time
	d.StartBlock, d.StartBlockHash, d.CoveredSeconds = &n, &h, &seconds
	if n >= end.Number.Uint64() || seconds == 0 {
		fail(errors.New("invalid_fee_window"))
		return
	}
	d.RangesTotal = int((end.Number.Uint64()-n-1)/1500 + 1)
	parsed, err := abi.JSON(strings.NewReader(weETHFeeABI))
	if err != nil {
		fail(errors.New("invalid_swap_abi"))
		return
	}
	logs, done, err := readWeETHSwaps(ctx, s.client, n+1, end.Number.Uint64(), parsed.Events["TokenExchange"].ID)
	d.RangesDone = done
	if err != nil {
		fail(err)
		return
	}
	fees, count, err := estimateWeETHSwaps(logs, parsed, reference)
	if err != nil {
		fail(err)
		return
	}
	verified, err := s.client.HeaderByNumber(ctx, start.Number)
	if err != nil || verified == nil || verified.Hash() != h {
		fail(errors.New("history_snapshot_changed_or_unavailable"))
		return
	}
	value, ok := finiteFloat(fees.DivRound(reference.capital, 36).Mul(decimal.NewFromInt(31536000)).DivRound(decimal.NewFromInt(int64(seconds)), 36).Mul(decimal.NewFromInt(100)))
	if !ok {
		fail(errors.New("invalid_fee_estimate"))
		return
	}
	capital, fraction, admin, total := reference.capital.String(), reference.fee.String(), reference.admin.String(), fees.String()
	d.ReferenceCapitalETH, d.DynamicFeeFraction, d.AdminFeeFraction, d.EstimatedLPFeesETH = &capital, &fraction, &admin, &total
	d.Swaps, d.EstimatedAPR, d.Status = count, &value, "estimate_available"
}
