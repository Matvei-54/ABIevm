package internal

type NetworkConfig struct {
	rpcUrl                      string
	contractPoolAddressProvider string
	contractUIProvider          string
}

var Networks = map[string]NetworkConfig{
	"ethereum": {
		rpcUrl:                      "https://eth-mainnet.g.alchemy.com/v2/----",
		contractPoolAddressProvider: "0x2f39d218133AFaB8F2B819B1066c7E434Ad94E9e",
		contractUIProvider:          "0x3F78BBD206e4D3c504Eb854232EdA7e47E9Fd8FC"},
	"arbitrum": {
		rpcUrl:                      "https://arb-mainnet.g.alchemy.com/v2/----",
		contractPoolAddressProvider: "0xa71dF25a3b0fa2040d0df516d5f318668bc22975",
		contractUIProvider:          "0xdBbFaFC45983B4659E368a3025bDb4fB7C1a595E",
	},
}
