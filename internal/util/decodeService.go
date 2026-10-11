package util

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func normalize(v interface{}) interface{} {
	switch val := v.(type) {
	case []interface{}:
		out := make([]interface{}, 0, len(val))
		for _, e := range val {
			out = append(out, normalize(e))
		}
		return out
	case abi.Type:
		// Tuple is represented as []interface{} by UnpackIntoInterface, handle below
		return val
	case common.Address:
		return val.Hex()
	case *common.Address:
		return val.Hex()
	case []byte:
		// bytes or dynamic bytes — return hex
		return "0x" + hex.EncodeToString(val)
	case *big.Int:
		return val.String()
	case big.Int:
		return val.String()
	case string:
		return val
	case bool:
		return val
	default:
		// try reflect fallback
		return val
	}
}

// Decode uses the provided abi.ABI to unpack outputBytes from functionName and returns normalized []interface{}
func Decode(parsedABI abi.ABI, functionName string, outputBytes []byte) ([]interface{}, error) {
	var decoded []interface{}
	// UnpackIntoInterface will fill decoded with function output values
	if err := parsedABI.UnpackIntoInterface(&decoded, functionName, outputBytes); err != nil {
		return nil, err
	}

	// normalize each top-level return value
	res := make([]interface{}, 0, len(decoded))
	for _, v := range decoded {
		res = append(res, normalize(v))
	}
	return res, nil
}

// Helper: build calldata for getReservesData(address)
// selector = keccak("getReservesData(address)")[0:4] = ec489c21
func BuildGetReservesDataCalldata(provider string) []byte {
	// ensure 0x-less lower-case
	addr := strings.TrimPrefix(provider, "0x")
	// selector + 32 bytes padded address
	hexData := "ec489c21" + strings.Repeat("0", 64-len(addr)) + addr
	b, _ := hex.DecodeString(hexData)
	return b
}
