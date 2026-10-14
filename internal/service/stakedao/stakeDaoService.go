package stakedao

import (
	_ "embed"
	"strings"
)

//go:embed abi/vault-v2.json
var vaultV2ABI string

//go:embed abi/accountant-v2.json
var accountantV2ABI string

//go:embed abi/strategy-v2.json
var strategyV2ABI string

//go:embed abi/sidecar-v2.json
var sidecarV2ABI string

//go:embed abi/curve-gauge.json
var curveGaugeABI string

//go:embed abi/curve-controller.json
var curveControllerABI string

const allocatorV2ABI = `[{"name":"getAllocationTargets","type":"function","stateMutability":"view","inputs":[{"type":"address"}],"outputs":[{"type":"address[]"}]}]`
const convexPoolABI = `[
{"name":"rewardToken","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]},
{"name":"totalSupply","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]},
{"name":"rewardRate","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]},
{"name":"periodFinish","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]}]`

const PROTOCOL_CONTROLLER_ETH_ARB = "0x2d8BcE1FaE00a959354aCD9eBf9174337A64d4fb"

// Controller ABI matches contracts/stakedao/CreateVault.json.
const protocolControllerABI = `[
	{"inputs":[{"internalType":"address","name":"gauge","type":"address"}],"name":"vault","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"address","name":"gauge","type":"address"}],"name":"asset","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"name":"protocolId","type":"bytes4"}],"name":"accountant","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}
]`

const rewardVaultERC4626ABI = `[
	{"inputs":[],"name":"totalAssets","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"uint256","name":"shares","type":"uint256"}],"name":"convertToAssets","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]`

const erc20TotalSupplyABI = `[
	{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
]`

const erc20DecimalsABI = `[
	{"inputs":[],"name":"decimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
]`

const erc20NameABI = `[
	{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}
]`

const erc20SymbolABI = `[
	{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"}
]`

// Getter signatures are documented in the official staking-v2 audit reports.
const rewardVaultAccountantABI = `[{"inputs":[],"name":"ACCOUNTANT","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`
const accountantRewardTokenABI = `[{"inputs":[],"name":"REWARD_TOKEN","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}]`

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
