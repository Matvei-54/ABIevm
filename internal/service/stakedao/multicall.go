package stakedao

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Ethereum Mainnet Multicall3. Calls are simulated by eth_call; no transaction is sent.
var stakeDAOMulticallAddress = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// Subset of the ABI already checked into contracts/curve/Multicall3Curve.json.
const stakeDAOMulticallABI = `[{"type":"function","name":"aggregate3","stateMutability":"payable","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"allowFailure","type":"bool"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

const stakeDAOMulticallBatchSize = 24

type multicallInput struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

type multicallOutput struct {
	Success    bool
	ReturnData []byte
}

type snapshotRead struct {
	target  common.Address
	abiJSON string
	method  string
	args    []interface{}
}

func readContract(target common.Address, abiJSON, method string, args ...interface{}) snapshotRead {
	return snapshotRead{target: target, abiJSON: abiJSON, method: method, args: args}
}

type snapshotResult struct {
	values []interface{}
	err    error
}

func (r snapshotResult) address() (common.Address, error) {
	if r.err != nil {
		return common.Address{}, r.err
	}
	if len(r.values) == 1 {
		if address, ok := r.values[0].(common.Address); ok {
			return address, nil
		}
	}
	return common.Address{}, errors.New("invalid address output")
}

func (r snapshotResult) unsigned() (*big.Int, error) {
	if r.err != nil {
		return nil, r.err
	}
	if len(r.values) == 1 {
		switch value := r.values[0].(type) {
		case *big.Int:
			if value != nil && value.Sign() >= 0 {
				return value, nil
			}
		case uint8:
			return new(big.Int).SetUint64(uint64(value)), nil
		case uint32:
			return new(big.Int).SetUint64(uint64(value)), nil
		}
	}
	return nil, errors.New("invalid unsigned output")
}

// Only explicitly selected, caller-independent getters belong here: subcalls see
// Multicall3 as msg.sender. Dependent reads remain in separate stages. Results
// retain their input order and failures; a failed batch never triggers direct retries.
func (s *StakeDAOService) snapshotCalls(ctx context.Context, block *big.Int, reads ...snapshotRead) []snapshotResult {
	results := make([]snapshotResult, len(reads))
	failFrom := func(start int, err error) []snapshotResult {
		for i := start; i < len(results); i++ {
			results[i].err = err
		}
		return results
	}
	if len(reads) == 0 {
		return results
	}
	if block == nil || block.Sign() < 0 {
		return failFrom(0, errors.New("Multicall requires a fixed nonnegative snapshot block"))
	}
	aggregate, err := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
	if err != nil {
		return failFrom(0, err)
	}
	for start := 0; start < len(reads); start += stakeDAOMulticallBatchSize {
		if err := ctx.Err(); err != nil {
			return failFrom(start, err)
		}
		end := min(start+stakeDAOMulticallBatchSize, len(reads))
		calls := make([]multicallInput, end-start)
		parsers := make([]abi.ABI, end-start)
		for j, read := range reads[start:end] {
			parsed, err := abi.JSON(strings.NewReader(read.abiJSON))
			if err != nil {
				return failFrom(start, err)
			}
			data, err := parsed.Pack(read.method, read.args...)
			if err != nil {
				return failFrom(start, err)
			}
			calls[j] = multicallInput{Target: read.target, AllowFailure: true, CallData: data}
			parsers[j] = parsed
		}
		data, err := aggregate.Pack("aggregate3", calls)
		if err != nil {
			return failFrom(start, err)
		}
		raw, err := s.client.CallContract(ctx, ethereum.CallMsg{To: &stakeDAOMulticallAddress, Data: data}, block)
		if err != nil {
			return failFrom(start, fmt.Errorf("%w: Multicall at block %s: %w", ErrRPCUnavailable, block, err))
		}
		out, err := aggregate.Unpack("aggregate3", raw)
		if err != nil || len(out) != 1 {
			return failFrom(start, fmt.Errorf("%w: invalid Multicall response", ErrRPCUnavailable))
		}
		decoded := *abi.ConvertType(out[0], new([]multicallOutput)).(*[]multicallOutput)
		if len(decoded) != len(calls) {
			return failFrom(start, fmt.Errorf("%w: Multicall result count mismatch", ErrRPCUnavailable))
		}
		for j, result := range decoded {
			if !result.Success {
				results[start+j].err = fmt.Errorf("%s reverted in Multicall", reads[start+j].method)
				continue
			}
			results[start+j].values, results[start+j].err = parsers[j].Unpack(reads[start+j].method, result.ReturnData)
		}
	}
	return results
}
