package stakedao

import (
	"context"
	"errors"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type multicallProbeRPC struct {
	stakeDAORPC
	calls int
	call  func(ethereum.CallMsg, *big.Int) ([]byte, error)
}

func (p *multicallProbeRPC) CallContract(_ context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	p.calls++
	return p.call(msg, block)
}

func repositoryMulticallABI(t *testing.T) abi.ABI {
	t.Helper()
	data, err := os.ReadFile("../../../contracts/curve/Multicall3Curve.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := abi.JSON(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func decodeMulticall(t *testing.T, parsed abi.ABI, msg ethereum.CallMsg, block *big.Int) []multicallInput {
	t.Helper()
	if msg.To == nil || *msg.To != stakeDAOMulticallAddress || block == nil || block.Cmp(big.NewInt(4500)) != 0 || msg.From != (common.Address{}) {
		t.Fatalf("wrong Multicall target/snapshot: %+v block %v", msg, block)
	}
	method, err := parsed.MethodById(msg.Data[:4])
	if err != nil || method.Name != "aggregate3" {
		t.Fatalf("wrong Multicall selector: %v", err)
	}
	args, err := method.Inputs.Unpack(msg.Data[4:])
	if err != nil {
		t.Fatal(err)
	}
	calls := *abi.ConvertType(args[0], new([]multicallInput)).(*[]multicallInput)
	for _, call := range calls {
		if !call.AllowFailure {
			t.Fatal("optional subcall would revert the entire batch")
		}
	}
	return calls
}

func TestMulticallChunksAndPreservesOrderAndSnapshot(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	getter, _ := abi.JSON(strings.NewReader(rewardVaultERC4626ABI))
	var sizes []int
	client := &multicallProbeRPC{call: func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		calls := decodeMulticall(t, parsed, msg, block)
		sizes = append(sizes, len(calls))
		results := make([]multicallOutput, len(calls))
		for i, call := range calls {
			method, err := getter.MethodById(call.CallData[:4])
			if err != nil || method.Name != "totalAssets" {
				t.Fatal("wrong subcall selector")
			}
			data, err := method.Outputs.Pack(call.Target.Big())
			if err != nil {
				t.Fatal(err)
			}
			results[i] = multicallOutput{Success: true, ReturnData: data}
		}
		return parsed.Methods["aggregate3"].Outputs.Pack(results)
	}}
	service := &StakeDAOService{client: client}
	var reads []snapshotRead
	for i := 1; i <= 49; i++ {
		reads = append(reads, readContract(common.BigToAddress(big.NewInt(int64(i))), rewardVaultERC4626ABI, "totalAssets"))
	}
	results := service.snapshotCalls(context.Background(), big.NewInt(4500), reads...)
	if !reflect.DeepEqual(sizes, []int{24, 24, 1}) {
		t.Fatalf("unbounded batches: %v", sizes)
	}
	for i, result := range results {
		value, err := result.unsigned()
		if err != nil || value.Cmp(big.NewInt(int64(i+1))) != 0 {
			t.Fatalf("result %d reordered/lost: %v %v", i, value, err)
		}
	}
}

func TestMulticallIsolatesRevertsAndMalformedGetterData(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	getter, _ := abi.JSON(strings.NewReader(erc20DecimalsABI))
	valid, _ := getter.Methods["decimals"].Outputs.Pack(uint8(6))
	client := &multicallProbeRPC{call: func(msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		decodeMulticall(t, parsed, msg, block)
		return parsed.Methods["aggregate3"].Outputs.Pack([]multicallOutput{
			{Success: false, ReturnData: valid},
			{Success: true, ReturnData: nil},
			{Success: true, ReturnData: valid},
		})
	}}
	read := readContract(common.HexToAddress("0xaa"), erc20DecimalsABI, "decimals")
	results := (&StakeDAOService{client: client}).snapshotCalls(context.Background(), big.NewInt(4500), read, read, read)
	for _, result := range results[:2] {
		if value, err := result.unsigned(); err == nil || value != nil {
			t.Fatalf("failed getter turned into a number: %v %v", value, err)
		}
	}
	if value, err := results[2].unsigned(); err != nil || value.Uint64() != 6 || client.calls != 1 {
		t.Fatalf("healthy getter lost or retried: %v %v calls=%d", value, err, client.calls)
	}
}

func TestMulticallRejectsInvalidBatchesWithoutFallback(t *testing.T) {
	parsed := repositoryMulticallABI(t)
	read := readContract(common.HexToAddress("0xaa"), erc20DecimalsABI, "decimals")
	for _, mode := range []string{"transport", "malformed", "too-few", "too-many"} {
		t.Run(mode, func(t *testing.T) {
			client := &multicallProbeRPC{call: func(ethereum.CallMsg, *big.Int) ([]byte, error) {
				switch mode {
				case "transport":
					return nil, errors.New("provider unavailable")
				case "malformed":
					return []byte{1}, nil
				case "too-few":
					return parsed.Methods["aggregate3"].Outputs.Pack([]multicallOutput{})
				default:
					return parsed.Methods["aggregate3"].Outputs.Pack(make([]multicallOutput, 3))
				}
			}}
			results := (&StakeDAOService{client: client}).snapshotCalls(context.Background(), big.NewInt(4500), read, read)
			if client.calls != 1 {
				t.Fatalf("unexpected retry/fallback: %d calls", client.calls)
			}
			for _, result := range results {
				if !errors.Is(result.err, ErrRPCUnavailable) || len(result.values) != 0 {
					t.Fatalf("invalid batch accepted: %+v", result)
				}
			}
		})
	}
	client := &multicallProbeRPC{}
	service := &StakeDAOService{client: client}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result := service.snapshotCalls(ctx, big.NewInt(4500), read); !errors.Is(result[0].err, context.Canceled) || client.calls != 0 {
		t.Fatalf("cancelled batch was sent: %+v", result)
	}
	if result := service.snapshotCalls(context.Background(), nil, read); result[0].err == nil || client.calls != 0 {
		t.Fatal("unfixed block accepted")
	}
}

func TestMulticallConsumesOneSampleAttemptForMultipleGetters(t *testing.T) {
	_, f := newPoolFixture(t)
	client := newSampleRPC(f)
	client.remaining = 1
	read := readContract(common.HexToAddress("0xbb"), erc20DecimalsABI, "decimals")
	results := (&StakeDAOService{client: client}).snapshotCalls(context.Background(), big.NewInt(4500), read, read, read)
	for _, result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
	}
	if client.remaining != 0 || client.exhausted || len(f.callBlocks) != 1 || !reflect.DeepEqual(f.multicallSizes, []int{3}) {
		t.Fatal("sample budget counted subcalls as separate RPC attempts")
	}
}

func TestMulticallReducesFullPoolReadsWithoutChangingRates(t *testing.T) {
	s, f := newPoolFixture(t)
	f.fullPrimary, f.sidecar = true, true
	result, err := s.GetPools(context.Background(), "ethereum")
	if err != nil || len(result.Errors) != 0 || len(result.Pools) != 1 {
		t.Fatalf("pool loading failed: %+v %v", result, err)
	}
	pool := result.Pools[0]
	if pool.AssetAmount == nil || *pool.AssetAmount != "1.234567" || pool.TVL == nil || *pool.TVL != 2.469134 || pool.Rewards[0].RatePerSecond == nil || *pool.Rewards[0].RatePerSecond != "1.1688" || pool.APRStatus != "available" {
		t.Fatalf("pool values changed: %+v", pool)
	}
	getters := f.directCalls
	for _, size := range f.multicallSizes {
		getters += size
	}
	requests := len(f.multicallSizes) + f.directCalls
	if getters < 30 || requests*2 >= getters {
		t.Fatalf("expected >2x fewer eth_call requests: %d requests for %d getters", requests, getters)
	}
	t.Logf("%d contract getters in %d eth_call requests (%d Multicall batches)", getters, requests, len(f.multicallSizes))
}
