// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package registry

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// UipoolMetaData contains all meta data concerning the Uipool contract.
var UipoolMetaData = &bind.MetaData{
	ABI: "[{\"name\":\"CommitNewAdmin\",\"inputs\":[{\"name\":\"deadline\",\"type\":\"uint256\",\"indexed\":true},{\"name\":\"admin\",\"type\":\"address\",\"indexed\":true}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"NewAdmin\",\"inputs\":[{\"name\":\"admin\",\"type\":\"address\",\"indexed\":true}],\"anonymous\":false,\"type\":\"event\"},{\"stateMutability\":\"nonpayable\",\"type\":\"constructor\",\"inputs\":[{\"name\":\"_address_provider\",\"type\":\"address\"}],\"outputs\":[]},{\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"add_registry_handler\",\"inputs\":[{\"name\":\"_registry_handler\",\"type\":\"address\"}],\"outputs\":[]},{\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"name\":\"update_registry_handler\",\"inputs\":[{\"name\":\"_index\",\"type\":\"uint256\"},{\"name\":\"_registry_handler\",\"type\":\"address\"}],\"outputs\":[]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_registry_handlers_from_pool\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address[10]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_base_registry\",\"inputs\":[{\"name\":\"registry_handler\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"find_pool_for_coins\",\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"find_pool_for_coins\",\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"i\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"find_pools_for_coins\",\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_admin_balances\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_admin_balances\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_balances\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_balances\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_base_pool\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_base_pool\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_coin_indices\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\"},{\"name\":\"\",\"type\":\"int128\"},{\"name\":\"\",\"type\":\"bool\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_coin_indices\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\"},{\"name\":\"\",\"type\":\"int128\"},{\"name\":\"\",\"type\":\"bool\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_decimals\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_decimals\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_fees\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[10]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_fees\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[10]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_gauge\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_gauge\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"gauge_idx\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_gauge\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"gauge_idx\",\"type\":\"uint256\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_gauge_type\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_gauge_type\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"gauge_idx\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_gauge_type\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"gauge_idx\",\"type\":\"uint256\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_lp_token\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_lp_token\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_n_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_n_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_n_underlying_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_n_underlying_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_asset_type\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_asset_type\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_from_lp_token\",\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_from_lp_token\",\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_params\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[20]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_params\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[20]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_name\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_pool_name\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"string\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_underlying_balances\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_underlying_balances\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_underlying_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_underlying_coins\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_underlying_decimals\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_underlying_decimals\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[8]\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_virtual_price_from_lp_token\",\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_virtual_price_from_lp_token\",\"inputs\":[{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"is_meta\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"is_meta\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"is_registered\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"is_registered\",\"inputs\":[{\"name\":\"_pool\",\"type\":\"address\"},{\"name\":\"_handler_id\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"pool_count\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"pool_list\",\"inputs\":[{\"name\":\"_index\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"address_provider\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"get_registry\",\"inputs\":[{\"name\":\"arg0\",\"type\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\"}]},{\"stateMutability\":\"view\",\"type\":\"function\",\"name\":\"registry_length\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}]}]",
}

// UipoolABI is the input ABI used to generate the binding from.
// Deprecated: Use UipoolMetaData.ABI instead.
var UipoolABI = UipoolMetaData.ABI

// Uipool is an auto generated Go binding around an Ethereum contract.
type Uipool struct {
	UipoolCaller     // Read-only binding to the contract
	UipoolTransactor // Write-only binding to the contract
	UipoolFilterer   // Log filterer for contract events
}

// UipoolCaller is an auto generated read-only Go binding around an Ethereum contract.
type UipoolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UipoolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type UipoolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UipoolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type UipoolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// UipoolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type UipoolSession struct {
	Contract     *Uipool           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// UipoolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type UipoolCallerSession struct {
	Contract *UipoolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// UipoolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type UipoolTransactorSession struct {
	Contract     *UipoolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// UipoolRaw is an auto generated low-level Go binding around an Ethereum contract.
type UipoolRaw struct {
	Contract *Uipool // Generic contract binding to access the raw methods on
}

// UipoolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type UipoolCallerRaw struct {
	Contract *UipoolCaller // Generic read-only contract binding to access the raw methods on
}

// UipoolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type UipoolTransactorRaw struct {
	Contract *UipoolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewUipool creates a new instance of Uipool, bound to a specific deployed contract.
func NewUipool(address common.Address, backend bind.ContractBackend) (*Uipool, error) {
	contract, err := bindUipool(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Uipool{UipoolCaller: UipoolCaller{contract: contract}, UipoolTransactor: UipoolTransactor{contract: contract}, UipoolFilterer: UipoolFilterer{contract: contract}}, nil
}

// NewUipoolCaller creates a new read-only instance of Uipool, bound to a specific deployed contract.
func NewUipoolCaller(address common.Address, caller bind.ContractCaller) (*UipoolCaller, error) {
	contract, err := bindUipool(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &UipoolCaller{contract: contract}, nil
}

// NewUipoolTransactor creates a new write-only instance of Uipool, bound to a specific deployed contract.
func NewUipoolTransactor(address common.Address, transactor bind.ContractTransactor) (*UipoolTransactor, error) {
	contract, err := bindUipool(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &UipoolTransactor{contract: contract}, nil
}

// NewUipoolFilterer creates a new log filterer instance of Uipool, bound to a specific deployed contract.
func NewUipoolFilterer(address common.Address, filterer bind.ContractFilterer) (*UipoolFilterer, error) {
	contract, err := bindUipool(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &UipoolFilterer{contract: contract}, nil
}

// bindUipool binds a generic wrapper to an already deployed contract.
func bindUipool(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := UipoolMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Uipool *UipoolRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Uipool.Contract.UipoolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Uipool *UipoolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Uipool.Contract.UipoolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Uipool *UipoolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Uipool.Contract.UipoolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Uipool *UipoolCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Uipool.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Uipool *UipoolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Uipool.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Uipool *UipoolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Uipool.Contract.contract.Transact(opts, method, params...)
}

// AddressProvider is a free data retrieval call binding the contract method 0xce50c2e7.
//
// Solidity: function address_provider() view returns(address)
func (_Uipool *UipoolCaller) AddressProvider(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "address_provider")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AddressProvider is a free data retrieval call binding the contract method 0xce50c2e7.
//
// Solidity: function address_provider() view returns(address)
func (_Uipool *UipoolSession) AddressProvider() (common.Address, error) {
	return _Uipool.Contract.AddressProvider(&_Uipool.CallOpts)
}

// AddressProvider is a free data retrieval call binding the contract method 0xce50c2e7.
//
// Solidity: function address_provider() view returns(address)
func (_Uipool *UipoolCallerSession) AddressProvider() (common.Address, error) {
	return _Uipool.Contract.AddressProvider(&_Uipool.CallOpts)
}

// FindPoolForCoins is a free data retrieval call binding the contract method 0xa87df06c.
//
// Solidity: function find_pool_for_coins(address _from, address _to) view returns(address)
func (_Uipool *UipoolCaller) FindPoolForCoins(opts *bind.CallOpts, _from common.Address, _to common.Address) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "find_pool_for_coins", _from, _to)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FindPoolForCoins is a free data retrieval call binding the contract method 0xa87df06c.
//
// Solidity: function find_pool_for_coins(address _from, address _to) view returns(address)
func (_Uipool *UipoolSession) FindPoolForCoins(_from common.Address, _to common.Address) (common.Address, error) {
	return _Uipool.Contract.FindPoolForCoins(&_Uipool.CallOpts, _from, _to)
}

// FindPoolForCoins is a free data retrieval call binding the contract method 0xa87df06c.
//
// Solidity: function find_pool_for_coins(address _from, address _to) view returns(address)
func (_Uipool *UipoolCallerSession) FindPoolForCoins(_from common.Address, _to common.Address) (common.Address, error) {
	return _Uipool.Contract.FindPoolForCoins(&_Uipool.CallOpts, _from, _to)
}

// FindPoolForCoins0 is a free data retrieval call binding the contract method 0x6982eb0b.
//
// Solidity: function find_pool_for_coins(address _from, address _to, uint256 i) view returns(address)
func (_Uipool *UipoolCaller) FindPoolForCoins0(opts *bind.CallOpts, _from common.Address, _to common.Address, i *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "find_pool_for_coins0", _from, _to, i)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FindPoolForCoins0 is a free data retrieval call binding the contract method 0x6982eb0b.
//
// Solidity: function find_pool_for_coins(address _from, address _to, uint256 i) view returns(address)
func (_Uipool *UipoolSession) FindPoolForCoins0(_from common.Address, _to common.Address, i *big.Int) (common.Address, error) {
	return _Uipool.Contract.FindPoolForCoins0(&_Uipool.CallOpts, _from, _to, i)
}

// FindPoolForCoins0 is a free data retrieval call binding the contract method 0x6982eb0b.
//
// Solidity: function find_pool_for_coins(address _from, address _to, uint256 i) view returns(address)
func (_Uipool *UipoolCallerSession) FindPoolForCoins0(_from common.Address, _to common.Address, i *big.Int) (common.Address, error) {
	return _Uipool.Contract.FindPoolForCoins0(&_Uipool.CallOpts, _from, _to, i)
}

// FindPoolsForCoins is a free data retrieval call binding the contract method 0xa064072b.
//
// Solidity: function find_pools_for_coins(address _from, address _to) view returns(address[])
func (_Uipool *UipoolCaller) FindPoolsForCoins(opts *bind.CallOpts, _from common.Address, _to common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "find_pools_for_coins", _from, _to)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// FindPoolsForCoins is a free data retrieval call binding the contract method 0xa064072b.
//
// Solidity: function find_pools_for_coins(address _from, address _to) view returns(address[])
func (_Uipool *UipoolSession) FindPoolsForCoins(_from common.Address, _to common.Address) ([]common.Address, error) {
	return _Uipool.Contract.FindPoolsForCoins(&_Uipool.CallOpts, _from, _to)
}

// FindPoolsForCoins is a free data retrieval call binding the contract method 0xa064072b.
//
// Solidity: function find_pools_for_coins(address _from, address _to) view returns(address[])
func (_Uipool *UipoolCallerSession) FindPoolsForCoins(_from common.Address, _to common.Address) ([]common.Address, error) {
	return _Uipool.Contract.FindPoolsForCoins(&_Uipool.CallOpts, _from, _to)
}

// GetAdminBalances is a free data retrieval call binding the contract method 0xc11e45b8.
//
// Solidity: function get_admin_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetAdminBalances(opts *bind.CallOpts, _pool common.Address) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_admin_balances", _pool)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetAdminBalances is a free data retrieval call binding the contract method 0xc11e45b8.
//
// Solidity: function get_admin_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolSession) GetAdminBalances(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetAdminBalances(&_Uipool.CallOpts, _pool)
}

// GetAdminBalances is a free data retrieval call binding the contract method 0xc11e45b8.
//
// Solidity: function get_admin_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetAdminBalances(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetAdminBalances(&_Uipool.CallOpts, _pool)
}

// GetAdminBalances0 is a free data retrieval call binding the contract method 0xc0bf4cbf.
//
// Solidity: function get_admin_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetAdminBalances0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_admin_balances0", _pool, _handler_id)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetAdminBalances0 is a free data retrieval call binding the contract method 0xc0bf4cbf.
//
// Solidity: function get_admin_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolSession) GetAdminBalances0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetAdminBalances0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetAdminBalances0 is a free data retrieval call binding the contract method 0xc0bf4cbf.
//
// Solidity: function get_admin_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetAdminBalances0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetAdminBalances0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetBalances is a free data retrieval call binding the contract method 0x92e3cc2d.
//
// Solidity: function get_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetBalances(opts *bind.CallOpts, _pool common.Address) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_balances", _pool)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetBalances is a free data retrieval call binding the contract method 0x92e3cc2d.
//
// Solidity: function get_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolSession) GetBalances(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetBalances(&_Uipool.CallOpts, _pool)
}

// GetBalances is a free data retrieval call binding the contract method 0x92e3cc2d.
//
// Solidity: function get_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetBalances(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetBalances(&_Uipool.CallOpts, _pool)
}

// GetBalances0 is a free data retrieval call binding the contract method 0xaa85169c.
//
// Solidity: function get_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetBalances0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_balances0", _pool, _handler_id)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetBalances0 is a free data retrieval call binding the contract method 0xaa85169c.
//
// Solidity: function get_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolSession) GetBalances0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetBalances0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetBalances0 is a free data retrieval call binding the contract method 0xaa85169c.
//
// Solidity: function get_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetBalances0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetBalances0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetBasePool is a free data retrieval call binding the contract method 0x6f20d6dd.
//
// Solidity: function get_base_pool(address _pool) view returns(address)
func (_Uipool *UipoolCaller) GetBasePool(opts *bind.CallOpts, _pool common.Address) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_base_pool", _pool)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetBasePool is a free data retrieval call binding the contract method 0x6f20d6dd.
//
// Solidity: function get_base_pool(address _pool) view returns(address)
func (_Uipool *UipoolSession) GetBasePool(_pool common.Address) (common.Address, error) {
	return _Uipool.Contract.GetBasePool(&_Uipool.CallOpts, _pool)
}

// GetBasePool is a free data retrieval call binding the contract method 0x6f20d6dd.
//
// Solidity: function get_base_pool(address _pool) view returns(address)
func (_Uipool *UipoolCallerSession) GetBasePool(_pool common.Address) (common.Address, error) {
	return _Uipool.Contract.GetBasePool(&_Uipool.CallOpts, _pool)
}

// GetBasePool0 is a free data retrieval call binding the contract method 0xa54e3ade.
//
// Solidity: function get_base_pool(address _pool, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCaller) GetBasePool0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_base_pool0", _pool, _handler_id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetBasePool0 is a free data retrieval call binding the contract method 0xa54e3ade.
//
// Solidity: function get_base_pool(address _pool, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolSession) GetBasePool0(_pool common.Address, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetBasePool0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetBasePool0 is a free data retrieval call binding the contract method 0xa54e3ade.
//
// Solidity: function get_base_pool(address _pool, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCallerSession) GetBasePool0(_pool common.Address, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetBasePool0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetBaseRegistry is a free data retrieval call binding the contract method 0x84e1710d.
//
// Solidity: function get_base_registry(address registry_handler) view returns(address)
func (_Uipool *UipoolCaller) GetBaseRegistry(opts *bind.CallOpts, registry_handler common.Address) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_base_registry", registry_handler)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetBaseRegistry is a free data retrieval call binding the contract method 0x84e1710d.
//
// Solidity: function get_base_registry(address registry_handler) view returns(address)
func (_Uipool *UipoolSession) GetBaseRegistry(registry_handler common.Address) (common.Address, error) {
	return _Uipool.Contract.GetBaseRegistry(&_Uipool.CallOpts, registry_handler)
}

// GetBaseRegistry is a free data retrieval call binding the contract method 0x84e1710d.
//
// Solidity: function get_base_registry(address registry_handler) view returns(address)
func (_Uipool *UipoolCallerSession) GetBaseRegistry(registry_handler common.Address) (common.Address, error) {
	return _Uipool.Contract.GetBaseRegistry(&_Uipool.CallOpts, registry_handler)
}

// GetCoinIndices is a free data retrieval call binding the contract method 0xeb85226d.
//
// Solidity: function get_coin_indices(address _pool, address _from, address _to) view returns(int128, int128, bool)
func (_Uipool *UipoolCaller) GetCoinIndices(opts *bind.CallOpts, _pool common.Address, _from common.Address, _to common.Address) (*big.Int, *big.Int, bool, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_coin_indices", _pool, _from, _to)

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(bool)).(*bool)

	return out0, out1, out2, err

}

// GetCoinIndices is a free data retrieval call binding the contract method 0xeb85226d.
//
// Solidity: function get_coin_indices(address _pool, address _from, address _to) view returns(int128, int128, bool)
func (_Uipool *UipoolSession) GetCoinIndices(_pool common.Address, _from common.Address, _to common.Address) (*big.Int, *big.Int, bool, error) {
	return _Uipool.Contract.GetCoinIndices(&_Uipool.CallOpts, _pool, _from, _to)
}

// GetCoinIndices is a free data retrieval call binding the contract method 0xeb85226d.
//
// Solidity: function get_coin_indices(address _pool, address _from, address _to) view returns(int128, int128, bool)
func (_Uipool *UipoolCallerSession) GetCoinIndices(_pool common.Address, _from common.Address, _to common.Address) (*big.Int, *big.Int, bool, error) {
	return _Uipool.Contract.GetCoinIndices(&_Uipool.CallOpts, _pool, _from, _to)
}

// GetCoinIndices0 is a free data retrieval call binding the contract method 0x63fb3ddb.
//
// Solidity: function get_coin_indices(address _pool, address _from, address _to, uint256 _handler_id) view returns(int128, int128, bool)
func (_Uipool *UipoolCaller) GetCoinIndices0(opts *bind.CallOpts, _pool common.Address, _from common.Address, _to common.Address, _handler_id *big.Int) (*big.Int, *big.Int, bool, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_coin_indices0", _pool, _from, _to, _handler_id)

	if err != nil {
		return *new(*big.Int), *new(*big.Int), *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	out2 := *abi.ConvertType(out[2], new(bool)).(*bool)

	return out0, out1, out2, err

}

// GetCoinIndices0 is a free data retrieval call binding the contract method 0x63fb3ddb.
//
// Solidity: function get_coin_indices(address _pool, address _from, address _to, uint256 _handler_id) view returns(int128, int128, bool)
func (_Uipool *UipoolSession) GetCoinIndices0(_pool common.Address, _from common.Address, _to common.Address, _handler_id *big.Int) (*big.Int, *big.Int, bool, error) {
	return _Uipool.Contract.GetCoinIndices0(&_Uipool.CallOpts, _pool, _from, _to, _handler_id)
}

// GetCoinIndices0 is a free data retrieval call binding the contract method 0x63fb3ddb.
//
// Solidity: function get_coin_indices(address _pool, address _from, address _to, uint256 _handler_id) view returns(int128, int128, bool)
func (_Uipool *UipoolCallerSession) GetCoinIndices0(_pool common.Address, _from common.Address, _to common.Address, _handler_id *big.Int) (*big.Int, *big.Int, bool, error) {
	return _Uipool.Contract.GetCoinIndices0(&_Uipool.CallOpts, _pool, _from, _to, _handler_id)
}

// GetCoins is a free data retrieval call binding the contract method 0x9ac90d3d.
//
// Solidity: function get_coins(address _pool) view returns(address[8])
func (_Uipool *UipoolCaller) GetCoins(opts *bind.CallOpts, _pool common.Address) ([8]common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_coins", _pool)

	if err != nil {
		return *new([8]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([8]common.Address)).(*[8]common.Address)

	return out0, err

}

// GetCoins is a free data retrieval call binding the contract method 0x9ac90d3d.
//
// Solidity: function get_coins(address _pool) view returns(address[8])
func (_Uipool *UipoolSession) GetCoins(_pool common.Address) ([8]common.Address, error) {
	return _Uipool.Contract.GetCoins(&_Uipool.CallOpts, _pool)
}

// GetCoins is a free data retrieval call binding the contract method 0x9ac90d3d.
//
// Solidity: function get_coins(address _pool) view returns(address[8])
func (_Uipool *UipoolCallerSession) GetCoins(_pool common.Address) ([8]common.Address, error) {
	return _Uipool.Contract.GetCoins(&_Uipool.CallOpts, _pool)
}

// GetCoins0 is a free data retrieval call binding the contract method 0x6ebe51fc.
//
// Solidity: function get_coins(address _pool, uint256 _handler_id) view returns(address[8])
func (_Uipool *UipoolCaller) GetCoins0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([8]common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_coins0", _pool, _handler_id)

	if err != nil {
		return *new([8]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([8]common.Address)).(*[8]common.Address)

	return out0, err

}

// GetCoins0 is a free data retrieval call binding the contract method 0x6ebe51fc.
//
// Solidity: function get_coins(address _pool, uint256 _handler_id) view returns(address[8])
func (_Uipool *UipoolSession) GetCoins0(_pool common.Address, _handler_id *big.Int) ([8]common.Address, error) {
	return _Uipool.Contract.GetCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetCoins0 is a free data retrieval call binding the contract method 0x6ebe51fc.
//
// Solidity: function get_coins(address _pool, uint256 _handler_id) view returns(address[8])
func (_Uipool *UipoolCallerSession) GetCoins0(_pool common.Address, _handler_id *big.Int) ([8]common.Address, error) {
	return _Uipool.Contract.GetCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetDecimals is a free data retrieval call binding the contract method 0x52b51555.
//
// Solidity: function get_decimals(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetDecimals(opts *bind.CallOpts, _pool common.Address) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_decimals", _pool)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetDecimals is a free data retrieval call binding the contract method 0x52b51555.
//
// Solidity: function get_decimals(address _pool) view returns(uint256[8])
func (_Uipool *UipoolSession) GetDecimals(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetDecimals(&_Uipool.CallOpts, _pool)
}

// GetDecimals is a free data retrieval call binding the contract method 0x52b51555.
//
// Solidity: function get_decimals(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetDecimals(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetDecimals(&_Uipool.CallOpts, _pool)
}

// GetDecimals0 is a free data retrieval call binding the contract method 0x403f502f.
//
// Solidity: function get_decimals(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetDecimals0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_decimals0", _pool, _handler_id)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetDecimals0 is a free data retrieval call binding the contract method 0x403f502f.
//
// Solidity: function get_decimals(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolSession) GetDecimals0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetDecimals0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetDecimals0 is a free data retrieval call binding the contract method 0x403f502f.
//
// Solidity: function get_decimals(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetDecimals0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetDecimals0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetFees is a free data retrieval call binding the contract method 0x7cdb72b0.
//
// Solidity: function get_fees(address _pool) view returns(uint256[10])
func (_Uipool *UipoolCaller) GetFees(opts *bind.CallOpts, _pool common.Address) ([10]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_fees", _pool)

	if err != nil {
		return *new([10]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([10]*big.Int)).(*[10]*big.Int)

	return out0, err

}

// GetFees is a free data retrieval call binding the contract method 0x7cdb72b0.
//
// Solidity: function get_fees(address _pool) view returns(uint256[10])
func (_Uipool *UipoolSession) GetFees(_pool common.Address) ([10]*big.Int, error) {
	return _Uipool.Contract.GetFees(&_Uipool.CallOpts, _pool)
}

// GetFees is a free data retrieval call binding the contract method 0x7cdb72b0.
//
// Solidity: function get_fees(address _pool) view returns(uint256[10])
func (_Uipool *UipoolCallerSession) GetFees(_pool common.Address) ([10]*big.Int, error) {
	return _Uipool.Contract.GetFees(&_Uipool.CallOpts, _pool)
}

// GetFees0 is a free data retrieval call binding the contract method 0x0ed5a427.
//
// Solidity: function get_fees(address _pool, uint256 _handler_id) view returns(uint256[10])
func (_Uipool *UipoolCaller) GetFees0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([10]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_fees0", _pool, _handler_id)

	if err != nil {
		return *new([10]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([10]*big.Int)).(*[10]*big.Int)

	return out0, err

}

// GetFees0 is a free data retrieval call binding the contract method 0x0ed5a427.
//
// Solidity: function get_fees(address _pool, uint256 _handler_id) view returns(uint256[10])
func (_Uipool *UipoolSession) GetFees0(_pool common.Address, _handler_id *big.Int) ([10]*big.Int, error) {
	return _Uipool.Contract.GetFees0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetFees0 is a free data retrieval call binding the contract method 0x0ed5a427.
//
// Solidity: function get_fees(address _pool, uint256 _handler_id) view returns(uint256[10])
func (_Uipool *UipoolCallerSession) GetFees0(_pool common.Address, _handler_id *big.Int) ([10]*big.Int, error) {
	return _Uipool.Contract.GetFees0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetGauge is a free data retrieval call binding the contract method 0xdaf297b9.
//
// Solidity: function get_gauge(address _pool) view returns(address)
func (_Uipool *UipoolCaller) GetGauge(opts *bind.CallOpts, _pool common.Address) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_gauge", _pool)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetGauge is a free data retrieval call binding the contract method 0xdaf297b9.
//
// Solidity: function get_gauge(address _pool) view returns(address)
func (_Uipool *UipoolSession) GetGauge(_pool common.Address) (common.Address, error) {
	return _Uipool.Contract.GetGauge(&_Uipool.CallOpts, _pool)
}

// GetGauge is a free data retrieval call binding the contract method 0xdaf297b9.
//
// Solidity: function get_gauge(address _pool) view returns(address)
func (_Uipool *UipoolCallerSession) GetGauge(_pool common.Address) (common.Address, error) {
	return _Uipool.Contract.GetGauge(&_Uipool.CallOpts, _pool)
}

// GetGauge0 is a free data retrieval call binding the contract method 0xe4081220.
//
// Solidity: function get_gauge(address _pool, uint256 gauge_idx) view returns(address)
func (_Uipool *UipoolCaller) GetGauge0(opts *bind.CallOpts, _pool common.Address, gauge_idx *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_gauge0", _pool, gauge_idx)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetGauge0 is a free data retrieval call binding the contract method 0xe4081220.
//
// Solidity: function get_gauge(address _pool, uint256 gauge_idx) view returns(address)
func (_Uipool *UipoolSession) GetGauge0(_pool common.Address, gauge_idx *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetGauge0(&_Uipool.CallOpts, _pool, gauge_idx)
}

// GetGauge0 is a free data retrieval call binding the contract method 0xe4081220.
//
// Solidity: function get_gauge(address _pool, uint256 gauge_idx) view returns(address)
func (_Uipool *UipoolCallerSession) GetGauge0(_pool common.Address, gauge_idx *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetGauge0(&_Uipool.CallOpts, _pool, gauge_idx)
}

// GetGauge1 is a free data retrieval call binding the contract method 0x773fb7e3.
//
// Solidity: function get_gauge(address _pool, uint256 gauge_idx, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCaller) GetGauge1(opts *bind.CallOpts, _pool common.Address, gauge_idx *big.Int, _handler_id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_gauge1", _pool, gauge_idx, _handler_id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetGauge1 is a free data retrieval call binding the contract method 0x773fb7e3.
//
// Solidity: function get_gauge(address _pool, uint256 gauge_idx, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolSession) GetGauge1(_pool common.Address, gauge_idx *big.Int, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetGauge1(&_Uipool.CallOpts, _pool, gauge_idx, _handler_id)
}

// GetGauge1 is a free data retrieval call binding the contract method 0x773fb7e3.
//
// Solidity: function get_gauge(address _pool, uint256 gauge_idx, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCallerSession) GetGauge1(_pool common.Address, gauge_idx *big.Int, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetGauge1(&_Uipool.CallOpts, _pool, gauge_idx, _handler_id)
}

// GetGaugeType is a free data retrieval call binding the contract method 0x25fa5d13.
//
// Solidity: function get_gauge_type(address _pool) view returns(int128)
func (_Uipool *UipoolCaller) GetGaugeType(opts *bind.CallOpts, _pool common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_gauge_type", _pool)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetGaugeType is a free data retrieval call binding the contract method 0x25fa5d13.
//
// Solidity: function get_gauge_type(address _pool) view returns(int128)
func (_Uipool *UipoolSession) GetGaugeType(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetGaugeType(&_Uipool.CallOpts, _pool)
}

// GetGaugeType is a free data retrieval call binding the contract method 0x25fa5d13.
//
// Solidity: function get_gauge_type(address _pool) view returns(int128)
func (_Uipool *UipoolCallerSession) GetGaugeType(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetGaugeType(&_Uipool.CallOpts, _pool)
}

// GetGaugeType0 is a free data retrieval call binding the contract method 0x7c51db55.
//
// Solidity: function get_gauge_type(address _pool, uint256 gauge_idx) view returns(int128)
func (_Uipool *UipoolCaller) GetGaugeType0(opts *bind.CallOpts, _pool common.Address, gauge_idx *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_gauge_type0", _pool, gauge_idx)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetGaugeType0 is a free data retrieval call binding the contract method 0x7c51db55.
//
// Solidity: function get_gauge_type(address _pool, uint256 gauge_idx) view returns(int128)
func (_Uipool *UipoolSession) GetGaugeType0(_pool common.Address, gauge_idx *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetGaugeType0(&_Uipool.CallOpts, _pool, gauge_idx)
}

// GetGaugeType0 is a free data retrieval call binding the contract method 0x7c51db55.
//
// Solidity: function get_gauge_type(address _pool, uint256 gauge_idx) view returns(int128)
func (_Uipool *UipoolCallerSession) GetGaugeType0(_pool common.Address, gauge_idx *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetGaugeType0(&_Uipool.CallOpts, _pool, gauge_idx)
}

// GetGaugeType1 is a free data retrieval call binding the contract method 0xf8b661e2.
//
// Solidity: function get_gauge_type(address _pool, uint256 gauge_idx, uint256 _handler_id) view returns(int128)
func (_Uipool *UipoolCaller) GetGaugeType1(opts *bind.CallOpts, _pool common.Address, gauge_idx *big.Int, _handler_id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_gauge_type1", _pool, gauge_idx, _handler_id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetGaugeType1 is a free data retrieval call binding the contract method 0xf8b661e2.
//
// Solidity: function get_gauge_type(address _pool, uint256 gauge_idx, uint256 _handler_id) view returns(int128)
func (_Uipool *UipoolSession) GetGaugeType1(_pool common.Address, gauge_idx *big.Int, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetGaugeType1(&_Uipool.CallOpts, _pool, gauge_idx, _handler_id)
}

// GetGaugeType1 is a free data retrieval call binding the contract method 0xf8b661e2.
//
// Solidity: function get_gauge_type(address _pool, uint256 gauge_idx, uint256 _handler_id) view returns(int128)
func (_Uipool *UipoolCallerSession) GetGaugeType1(_pool common.Address, gauge_idx *big.Int, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetGaugeType1(&_Uipool.CallOpts, _pool, gauge_idx, _handler_id)
}

// GetLpToken is a free data retrieval call binding the contract method 0x37951049.
//
// Solidity: function get_lp_token(address _pool) view returns(address)
func (_Uipool *UipoolCaller) GetLpToken(opts *bind.CallOpts, _pool common.Address) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_lp_token", _pool)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetLpToken is a free data retrieval call binding the contract method 0x37951049.
//
// Solidity: function get_lp_token(address _pool) view returns(address)
func (_Uipool *UipoolSession) GetLpToken(_pool common.Address) (common.Address, error) {
	return _Uipool.Contract.GetLpToken(&_Uipool.CallOpts, _pool)
}

// GetLpToken is a free data retrieval call binding the contract method 0x37951049.
//
// Solidity: function get_lp_token(address _pool) view returns(address)
func (_Uipool *UipoolCallerSession) GetLpToken(_pool common.Address) (common.Address, error) {
	return _Uipool.Contract.GetLpToken(&_Uipool.CallOpts, _pool)
}

// GetLpToken0 is a free data retrieval call binding the contract method 0x0881715f.
//
// Solidity: function get_lp_token(address _pool, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCaller) GetLpToken0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_lp_token0", _pool, _handler_id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetLpToken0 is a free data retrieval call binding the contract method 0x0881715f.
//
// Solidity: function get_lp_token(address _pool, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolSession) GetLpToken0(_pool common.Address, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetLpToken0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetLpToken0 is a free data retrieval call binding the contract method 0x0881715f.
//
// Solidity: function get_lp_token(address _pool, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCallerSession) GetLpToken0(_pool common.Address, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetLpToken0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetNCoins is a free data retrieval call binding the contract method 0x940494f1.
//
// Solidity: function get_n_coins(address _pool) view returns(uint256)
func (_Uipool *UipoolCaller) GetNCoins(opts *bind.CallOpts, _pool common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_n_coins", _pool)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNCoins is a free data retrieval call binding the contract method 0x940494f1.
//
// Solidity: function get_n_coins(address _pool) view returns(uint256)
func (_Uipool *UipoolSession) GetNCoins(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetNCoins(&_Uipool.CallOpts, _pool)
}

// GetNCoins is a free data retrieval call binding the contract method 0x940494f1.
//
// Solidity: function get_n_coins(address _pool) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetNCoins(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetNCoins(&_Uipool.CallOpts, _pool)
}

// GetNCoins0 is a free data retrieval call binding the contract method 0x11d81279.
//
// Solidity: function get_n_coins(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCaller) GetNCoins0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_n_coins0", _pool, _handler_id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNCoins0 is a free data retrieval call binding the contract method 0x11d81279.
//
// Solidity: function get_n_coins(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolSession) GetNCoins0(_pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetNCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetNCoins0 is a free data retrieval call binding the contract method 0x11d81279.
//
// Solidity: function get_n_coins(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetNCoins0(_pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetNCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetNUnderlyingCoins is a free data retrieval call binding the contract method 0x0a700c08.
//
// Solidity: function get_n_underlying_coins(address _pool) view returns(uint256)
func (_Uipool *UipoolCaller) GetNUnderlyingCoins(opts *bind.CallOpts, _pool common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_n_underlying_coins", _pool)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNUnderlyingCoins is a free data retrieval call binding the contract method 0x0a700c08.
//
// Solidity: function get_n_underlying_coins(address _pool) view returns(uint256)
func (_Uipool *UipoolSession) GetNUnderlyingCoins(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetNUnderlyingCoins(&_Uipool.CallOpts, _pool)
}

// GetNUnderlyingCoins is a free data retrieval call binding the contract method 0x0a700c08.
//
// Solidity: function get_n_underlying_coins(address _pool) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetNUnderlyingCoins(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetNUnderlyingCoins(&_Uipool.CallOpts, _pool)
}

// GetNUnderlyingCoins0 is a free data retrieval call binding the contract method 0xdecdf68f.
//
// Solidity: function get_n_underlying_coins(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCaller) GetNUnderlyingCoins0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_n_underlying_coins0", _pool, _handler_id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNUnderlyingCoins0 is a free data retrieval call binding the contract method 0xdecdf68f.
//
// Solidity: function get_n_underlying_coins(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolSession) GetNUnderlyingCoins0(_pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetNUnderlyingCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetNUnderlyingCoins0 is a free data retrieval call binding the contract method 0xdecdf68f.
//
// Solidity: function get_n_underlying_coins(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetNUnderlyingCoins0(_pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetNUnderlyingCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetPoolAssetType is a free data retrieval call binding the contract method 0x66d3966c.
//
// Solidity: function get_pool_asset_type(address _pool) view returns(uint256)
func (_Uipool *UipoolCaller) GetPoolAssetType(opts *bind.CallOpts, _pool common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_asset_type", _pool)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPoolAssetType is a free data retrieval call binding the contract method 0x66d3966c.
//
// Solidity: function get_pool_asset_type(address _pool) view returns(uint256)
func (_Uipool *UipoolSession) GetPoolAssetType(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetPoolAssetType(&_Uipool.CallOpts, _pool)
}

// GetPoolAssetType is a free data retrieval call binding the contract method 0x66d3966c.
//
// Solidity: function get_pool_asset_type(address _pool) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetPoolAssetType(_pool common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetPoolAssetType(&_Uipool.CallOpts, _pool)
}

// GetPoolAssetType0 is a free data retrieval call binding the contract method 0x90d1dd2d.
//
// Solidity: function get_pool_asset_type(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCaller) GetPoolAssetType0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_asset_type0", _pool, _handler_id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPoolAssetType0 is a free data retrieval call binding the contract method 0x90d1dd2d.
//
// Solidity: function get_pool_asset_type(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolSession) GetPoolAssetType0(_pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetPoolAssetType0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetPoolAssetType0 is a free data retrieval call binding the contract method 0x90d1dd2d.
//
// Solidity: function get_pool_asset_type(address _pool, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetPoolAssetType0(_pool common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetPoolAssetType0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetPoolFromLpToken is a free data retrieval call binding the contract method 0xbdf475c3.
//
// Solidity: function get_pool_from_lp_token(address _token) view returns(address)
func (_Uipool *UipoolCaller) GetPoolFromLpToken(opts *bind.CallOpts, _token common.Address) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_from_lp_token", _token)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPoolFromLpToken is a free data retrieval call binding the contract method 0xbdf475c3.
//
// Solidity: function get_pool_from_lp_token(address _token) view returns(address)
func (_Uipool *UipoolSession) GetPoolFromLpToken(_token common.Address) (common.Address, error) {
	return _Uipool.Contract.GetPoolFromLpToken(&_Uipool.CallOpts, _token)
}

// GetPoolFromLpToken is a free data retrieval call binding the contract method 0xbdf475c3.
//
// Solidity: function get_pool_from_lp_token(address _token) view returns(address)
func (_Uipool *UipoolCallerSession) GetPoolFromLpToken(_token common.Address) (common.Address, error) {
	return _Uipool.Contract.GetPoolFromLpToken(&_Uipool.CallOpts, _token)
}

// GetPoolFromLpToken0 is a free data retrieval call binding the contract method 0x36678b80.
//
// Solidity: function get_pool_from_lp_token(address _token, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCaller) GetPoolFromLpToken0(opts *bind.CallOpts, _token common.Address, _handler_id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_from_lp_token0", _token, _handler_id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetPoolFromLpToken0 is a free data retrieval call binding the contract method 0x36678b80.
//
// Solidity: function get_pool_from_lp_token(address _token, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolSession) GetPoolFromLpToken0(_token common.Address, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetPoolFromLpToken0(&_Uipool.CallOpts, _token, _handler_id)
}

// GetPoolFromLpToken0 is a free data retrieval call binding the contract method 0x36678b80.
//
// Solidity: function get_pool_from_lp_token(address _token, uint256 _handler_id) view returns(address)
func (_Uipool *UipoolCallerSession) GetPoolFromLpToken0(_token common.Address, _handler_id *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetPoolFromLpToken0(&_Uipool.CallOpts, _token, _handler_id)
}

// GetPoolName is a free data retrieval call binding the contract method 0x5c911741.
//
// Solidity: function get_pool_name(address _pool) view returns(string)
func (_Uipool *UipoolCaller) GetPoolName(opts *bind.CallOpts, _pool common.Address) (string, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_name", _pool)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GetPoolName is a free data retrieval call binding the contract method 0x5c911741.
//
// Solidity: function get_pool_name(address _pool) view returns(string)
func (_Uipool *UipoolSession) GetPoolName(_pool common.Address) (string, error) {
	return _Uipool.Contract.GetPoolName(&_Uipool.CallOpts, _pool)
}

// GetPoolName is a free data retrieval call binding the contract method 0x5c911741.
//
// Solidity: function get_pool_name(address _pool) view returns(string)
func (_Uipool *UipoolCallerSession) GetPoolName(_pool common.Address) (string, error) {
	return _Uipool.Contract.GetPoolName(&_Uipool.CallOpts, _pool)
}

// GetPoolName0 is a free data retrieval call binding the contract method 0x92234f45.
//
// Solidity: function get_pool_name(address _pool, uint256 _handler_id) view returns(string)
func (_Uipool *UipoolCaller) GetPoolName0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (string, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_name0", _pool, _handler_id)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// GetPoolName0 is a free data retrieval call binding the contract method 0x92234f45.
//
// Solidity: function get_pool_name(address _pool, uint256 _handler_id) view returns(string)
func (_Uipool *UipoolSession) GetPoolName0(_pool common.Address, _handler_id *big.Int) (string, error) {
	return _Uipool.Contract.GetPoolName0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetPoolName0 is a free data retrieval call binding the contract method 0x92234f45.
//
// Solidity: function get_pool_name(address _pool, uint256 _handler_id) view returns(string)
func (_Uipool *UipoolCallerSession) GetPoolName0(_pool common.Address, _handler_id *big.Int) (string, error) {
	return _Uipool.Contract.GetPoolName0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetPoolParams is a free data retrieval call binding the contract method 0x688532aa.
//
// Solidity: function get_pool_params(address _pool) view returns(uint256[20])
func (_Uipool *UipoolCaller) GetPoolParams(opts *bind.CallOpts, _pool common.Address) ([20]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_params", _pool)

	if err != nil {
		return *new([20]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([20]*big.Int)).(*[20]*big.Int)

	return out0, err

}

// GetPoolParams is a free data retrieval call binding the contract method 0x688532aa.
//
// Solidity: function get_pool_params(address _pool) view returns(uint256[20])
func (_Uipool *UipoolSession) GetPoolParams(_pool common.Address) ([20]*big.Int, error) {
	return _Uipool.Contract.GetPoolParams(&_Uipool.CallOpts, _pool)
}

// GetPoolParams is a free data retrieval call binding the contract method 0x688532aa.
//
// Solidity: function get_pool_params(address _pool) view returns(uint256[20])
func (_Uipool *UipoolCallerSession) GetPoolParams(_pool common.Address) ([20]*big.Int, error) {
	return _Uipool.Contract.GetPoolParams(&_Uipool.CallOpts, _pool)
}

// GetPoolParams0 is a free data retrieval call binding the contract method 0x7a65d2dd.
//
// Solidity: function get_pool_params(address _pool, uint256 _handler_id) view returns(uint256[20])
func (_Uipool *UipoolCaller) GetPoolParams0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([20]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_pool_params0", _pool, _handler_id)

	if err != nil {
		return *new([20]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([20]*big.Int)).(*[20]*big.Int)

	return out0, err

}

// GetPoolParams0 is a free data retrieval call binding the contract method 0x7a65d2dd.
//
// Solidity: function get_pool_params(address _pool, uint256 _handler_id) view returns(uint256[20])
func (_Uipool *UipoolSession) GetPoolParams0(_pool common.Address, _handler_id *big.Int) ([20]*big.Int, error) {
	return _Uipool.Contract.GetPoolParams0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetPoolParams0 is a free data retrieval call binding the contract method 0x7a65d2dd.
//
// Solidity: function get_pool_params(address _pool, uint256 _handler_id) view returns(uint256[20])
func (_Uipool *UipoolCallerSession) GetPoolParams0(_pool common.Address, _handler_id *big.Int) ([20]*big.Int, error) {
	return _Uipool.Contract.GetPoolParams0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetRegistry is a free data retrieval call binding the contract method 0x913d9b4d.
//
// Solidity: function get_registry(uint256 arg0) view returns(address)
func (_Uipool *UipoolCaller) GetRegistry(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_registry", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRegistry is a free data retrieval call binding the contract method 0x913d9b4d.
//
// Solidity: function get_registry(uint256 arg0) view returns(address)
func (_Uipool *UipoolSession) GetRegistry(arg0 *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetRegistry(&_Uipool.CallOpts, arg0)
}

// GetRegistry is a free data retrieval call binding the contract method 0x913d9b4d.
//
// Solidity: function get_registry(uint256 arg0) view returns(address)
func (_Uipool *UipoolCallerSession) GetRegistry(arg0 *big.Int) (common.Address, error) {
	return _Uipool.Contract.GetRegistry(&_Uipool.CallOpts, arg0)
}

// GetRegistryHandlersFromPool is a free data retrieval call binding the contract method 0x308d1b6d.
//
// Solidity: function get_registry_handlers_from_pool(address _pool) view returns(address[10])
func (_Uipool *UipoolCaller) GetRegistryHandlersFromPool(opts *bind.CallOpts, _pool common.Address) ([10]common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_registry_handlers_from_pool", _pool)

	if err != nil {
		return *new([10]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([10]common.Address)).(*[10]common.Address)

	return out0, err

}

// GetRegistryHandlersFromPool is a free data retrieval call binding the contract method 0x308d1b6d.
//
// Solidity: function get_registry_handlers_from_pool(address _pool) view returns(address[10])
func (_Uipool *UipoolSession) GetRegistryHandlersFromPool(_pool common.Address) ([10]common.Address, error) {
	return _Uipool.Contract.GetRegistryHandlersFromPool(&_Uipool.CallOpts, _pool)
}

// GetRegistryHandlersFromPool is a free data retrieval call binding the contract method 0x308d1b6d.
//
// Solidity: function get_registry_handlers_from_pool(address _pool) view returns(address[10])
func (_Uipool *UipoolCallerSession) GetRegistryHandlersFromPool(_pool common.Address) ([10]common.Address, error) {
	return _Uipool.Contract.GetRegistryHandlersFromPool(&_Uipool.CallOpts, _pool)
}

// GetUnderlyingBalances is a free data retrieval call binding the contract method 0x59f4f351.
//
// Solidity: function get_underlying_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetUnderlyingBalances(opts *bind.CallOpts, _pool common.Address) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_underlying_balances", _pool)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetUnderlyingBalances is a free data retrieval call binding the contract method 0x59f4f351.
//
// Solidity: function get_underlying_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolSession) GetUnderlyingBalances(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingBalances(&_Uipool.CallOpts, _pool)
}

// GetUnderlyingBalances is a free data retrieval call binding the contract method 0x59f4f351.
//
// Solidity: function get_underlying_balances(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetUnderlyingBalances(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingBalances(&_Uipool.CallOpts, _pool)
}

// GetUnderlyingBalances0 is a free data retrieval call binding the contract method 0x876112e6.
//
// Solidity: function get_underlying_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetUnderlyingBalances0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_underlying_balances0", _pool, _handler_id)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetUnderlyingBalances0 is a free data retrieval call binding the contract method 0x876112e6.
//
// Solidity: function get_underlying_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolSession) GetUnderlyingBalances0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingBalances0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetUnderlyingBalances0 is a free data retrieval call binding the contract method 0x876112e6.
//
// Solidity: function get_underlying_balances(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetUnderlyingBalances0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingBalances0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetUnderlyingCoins is a free data retrieval call binding the contract method 0xa77576ef.
//
// Solidity: function get_underlying_coins(address _pool) view returns(address[8])
func (_Uipool *UipoolCaller) GetUnderlyingCoins(opts *bind.CallOpts, _pool common.Address) ([8]common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_underlying_coins", _pool)

	if err != nil {
		return *new([8]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([8]common.Address)).(*[8]common.Address)

	return out0, err

}

// GetUnderlyingCoins is a free data retrieval call binding the contract method 0xa77576ef.
//
// Solidity: function get_underlying_coins(address _pool) view returns(address[8])
func (_Uipool *UipoolSession) GetUnderlyingCoins(_pool common.Address) ([8]common.Address, error) {
	return _Uipool.Contract.GetUnderlyingCoins(&_Uipool.CallOpts, _pool)
}

// GetUnderlyingCoins is a free data retrieval call binding the contract method 0xa77576ef.
//
// Solidity: function get_underlying_coins(address _pool) view returns(address[8])
func (_Uipool *UipoolCallerSession) GetUnderlyingCoins(_pool common.Address) ([8]common.Address, error) {
	return _Uipool.Contract.GetUnderlyingCoins(&_Uipool.CallOpts, _pool)
}

// GetUnderlyingCoins0 is a free data retrieval call binding the contract method 0x2fc90bcf.
//
// Solidity: function get_underlying_coins(address _pool, uint256 _handler_id) view returns(address[8])
func (_Uipool *UipoolCaller) GetUnderlyingCoins0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([8]common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_underlying_coins0", _pool, _handler_id)

	if err != nil {
		return *new([8]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([8]common.Address)).(*[8]common.Address)

	return out0, err

}

// GetUnderlyingCoins0 is a free data retrieval call binding the contract method 0x2fc90bcf.
//
// Solidity: function get_underlying_coins(address _pool, uint256 _handler_id) view returns(address[8])
func (_Uipool *UipoolSession) GetUnderlyingCoins0(_pool common.Address, _handler_id *big.Int) ([8]common.Address, error) {
	return _Uipool.Contract.GetUnderlyingCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetUnderlyingCoins0 is a free data retrieval call binding the contract method 0x2fc90bcf.
//
// Solidity: function get_underlying_coins(address _pool, uint256 _handler_id) view returns(address[8])
func (_Uipool *UipoolCallerSession) GetUnderlyingCoins0(_pool common.Address, _handler_id *big.Int) ([8]common.Address, error) {
	return _Uipool.Contract.GetUnderlyingCoins0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetUnderlyingDecimals is a free data retrieval call binding the contract method 0x4cb088f1.
//
// Solidity: function get_underlying_decimals(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetUnderlyingDecimals(opts *bind.CallOpts, _pool common.Address) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_underlying_decimals", _pool)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetUnderlyingDecimals is a free data retrieval call binding the contract method 0x4cb088f1.
//
// Solidity: function get_underlying_decimals(address _pool) view returns(uint256[8])
func (_Uipool *UipoolSession) GetUnderlyingDecimals(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingDecimals(&_Uipool.CallOpts, _pool)
}

// GetUnderlyingDecimals is a free data retrieval call binding the contract method 0x4cb088f1.
//
// Solidity: function get_underlying_decimals(address _pool) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetUnderlyingDecimals(_pool common.Address) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingDecimals(&_Uipool.CallOpts, _pool)
}

// GetUnderlyingDecimals0 is a free data retrieval call binding the contract method 0x622d64f4.
//
// Solidity: function get_underlying_decimals(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCaller) GetUnderlyingDecimals0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_underlying_decimals0", _pool, _handler_id)

	if err != nil {
		return *new([8]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([8]*big.Int)).(*[8]*big.Int)

	return out0, err

}

// GetUnderlyingDecimals0 is a free data retrieval call binding the contract method 0x622d64f4.
//
// Solidity: function get_underlying_decimals(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolSession) GetUnderlyingDecimals0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingDecimals0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetUnderlyingDecimals0 is a free data retrieval call binding the contract method 0x622d64f4.
//
// Solidity: function get_underlying_decimals(address _pool, uint256 _handler_id) view returns(uint256[8])
func (_Uipool *UipoolCallerSession) GetUnderlyingDecimals0(_pool common.Address, _handler_id *big.Int) ([8]*big.Int, error) {
	return _Uipool.Contract.GetUnderlyingDecimals0(&_Uipool.CallOpts, _pool, _handler_id)
}

// GetVirtualPriceFromLpToken is a free data retrieval call binding the contract method 0xc5b7074a.
//
// Solidity: function get_virtual_price_from_lp_token(address _token) view returns(uint256)
func (_Uipool *UipoolCaller) GetVirtualPriceFromLpToken(opts *bind.CallOpts, _token common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_virtual_price_from_lp_token", _token)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetVirtualPriceFromLpToken is a free data retrieval call binding the contract method 0xc5b7074a.
//
// Solidity: function get_virtual_price_from_lp_token(address _token) view returns(uint256)
func (_Uipool *UipoolSession) GetVirtualPriceFromLpToken(_token common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetVirtualPriceFromLpToken(&_Uipool.CallOpts, _token)
}

// GetVirtualPriceFromLpToken is a free data retrieval call binding the contract method 0xc5b7074a.
//
// Solidity: function get_virtual_price_from_lp_token(address _token) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetVirtualPriceFromLpToken(_token common.Address) (*big.Int, error) {
	return _Uipool.Contract.GetVirtualPriceFromLpToken(&_Uipool.CallOpts, _token)
}

// GetVirtualPriceFromLpToken0 is a free data retrieval call binding the contract method 0x4460f66f.
//
// Solidity: function get_virtual_price_from_lp_token(address _token, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCaller) GetVirtualPriceFromLpToken0(opts *bind.CallOpts, _token common.Address, _handler_id *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "get_virtual_price_from_lp_token0", _token, _handler_id)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetVirtualPriceFromLpToken0 is a free data retrieval call binding the contract method 0x4460f66f.
//
// Solidity: function get_virtual_price_from_lp_token(address _token, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolSession) GetVirtualPriceFromLpToken0(_token common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetVirtualPriceFromLpToken0(&_Uipool.CallOpts, _token, _handler_id)
}

// GetVirtualPriceFromLpToken0 is a free data retrieval call binding the contract method 0x4460f66f.
//
// Solidity: function get_virtual_price_from_lp_token(address _token, uint256 _handler_id) view returns(uint256)
func (_Uipool *UipoolCallerSession) GetVirtualPriceFromLpToken0(_token common.Address, _handler_id *big.Int) (*big.Int, error) {
	return _Uipool.Contract.GetVirtualPriceFromLpToken0(&_Uipool.CallOpts, _token, _handler_id)
}

// IsMeta is a free data retrieval call binding the contract method 0xe4d332a9.
//
// Solidity: function is_meta(address _pool) view returns(bool)
func (_Uipool *UipoolCaller) IsMeta(opts *bind.CallOpts, _pool common.Address) (bool, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "is_meta", _pool)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMeta is a free data retrieval call binding the contract method 0xe4d332a9.
//
// Solidity: function is_meta(address _pool) view returns(bool)
func (_Uipool *UipoolSession) IsMeta(_pool common.Address) (bool, error) {
	return _Uipool.Contract.IsMeta(&_Uipool.CallOpts, _pool)
}

// IsMeta is a free data retrieval call binding the contract method 0xe4d332a9.
//
// Solidity: function is_meta(address _pool) view returns(bool)
func (_Uipool *UipoolCallerSession) IsMeta(_pool common.Address) (bool, error) {
	return _Uipool.Contract.IsMeta(&_Uipool.CallOpts, _pool)
}

// IsMeta0 is a free data retrieval call binding the contract method 0xa64485a1.
//
// Solidity: function is_meta(address _pool, uint256 _handler_id) view returns(bool)
func (_Uipool *UipoolCaller) IsMeta0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (bool, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "is_meta0", _pool, _handler_id)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMeta0 is a free data retrieval call binding the contract method 0xa64485a1.
//
// Solidity: function is_meta(address _pool, uint256 _handler_id) view returns(bool)
func (_Uipool *UipoolSession) IsMeta0(_pool common.Address, _handler_id *big.Int) (bool, error) {
	return _Uipool.Contract.IsMeta0(&_Uipool.CallOpts, _pool, _handler_id)
}

// IsMeta0 is a free data retrieval call binding the contract method 0xa64485a1.
//
// Solidity: function is_meta(address _pool, uint256 _handler_id) view returns(bool)
func (_Uipool *UipoolCallerSession) IsMeta0(_pool common.Address, _handler_id *big.Int) (bool, error) {
	return _Uipool.Contract.IsMeta0(&_Uipool.CallOpts, _pool, _handler_id)
}

// IsRegistered is a free data retrieval call binding the contract method 0x619ea806.
//
// Solidity: function is_registered(address _pool) view returns(bool)
func (_Uipool *UipoolCaller) IsRegistered(opts *bind.CallOpts, _pool common.Address) (bool, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "is_registered", _pool)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRegistered is a free data retrieval call binding the contract method 0x619ea806.
//
// Solidity: function is_registered(address _pool) view returns(bool)
func (_Uipool *UipoolSession) IsRegistered(_pool common.Address) (bool, error) {
	return _Uipool.Contract.IsRegistered(&_Uipool.CallOpts, _pool)
}

// IsRegistered is a free data retrieval call binding the contract method 0x619ea806.
//
// Solidity: function is_registered(address _pool) view returns(bool)
func (_Uipool *UipoolCallerSession) IsRegistered(_pool common.Address) (bool, error) {
	return _Uipool.Contract.IsRegistered(&_Uipool.CallOpts, _pool)
}

// IsRegistered0 is a free data retrieval call binding the contract method 0xc9c4f3ec.
//
// Solidity: function is_registered(address _pool, uint256 _handler_id) view returns(bool)
func (_Uipool *UipoolCaller) IsRegistered0(opts *bind.CallOpts, _pool common.Address, _handler_id *big.Int) (bool, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "is_registered0", _pool, _handler_id)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRegistered0 is a free data retrieval call binding the contract method 0xc9c4f3ec.
//
// Solidity: function is_registered(address _pool, uint256 _handler_id) view returns(bool)
func (_Uipool *UipoolSession) IsRegistered0(_pool common.Address, _handler_id *big.Int) (bool, error) {
	return _Uipool.Contract.IsRegistered0(&_Uipool.CallOpts, _pool, _handler_id)
}

// IsRegistered0 is a free data retrieval call binding the contract method 0xc9c4f3ec.
//
// Solidity: function is_registered(address _pool, uint256 _handler_id) view returns(bool)
func (_Uipool *UipoolCallerSession) IsRegistered0(_pool common.Address, _handler_id *big.Int) (bool, error) {
	return _Uipool.Contract.IsRegistered0(&_Uipool.CallOpts, _pool, _handler_id)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Uipool *UipoolCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Uipool *UipoolSession) Owner() (common.Address, error) {
	return _Uipool.Contract.Owner(&_Uipool.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Uipool *UipoolCallerSession) Owner() (common.Address, error) {
	return _Uipool.Contract.Owner(&_Uipool.CallOpts)
}

// PoolCount is a free data retrieval call binding the contract method 0x956aae3a.
//
// Solidity: function pool_count() view returns(uint256)
func (_Uipool *UipoolCaller) PoolCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "pool_count")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PoolCount is a free data retrieval call binding the contract method 0x956aae3a.
//
// Solidity: function pool_count() view returns(uint256)
func (_Uipool *UipoolSession) PoolCount() (*big.Int, error) {
	return _Uipool.Contract.PoolCount(&_Uipool.CallOpts)
}

// PoolCount is a free data retrieval call binding the contract method 0x956aae3a.
//
// Solidity: function pool_count() view returns(uint256)
func (_Uipool *UipoolCallerSession) PoolCount() (*big.Int, error) {
	return _Uipool.Contract.PoolCount(&_Uipool.CallOpts)
}

// PoolList is a free data retrieval call binding the contract method 0x3a1d5d8e.
//
// Solidity: function pool_list(uint256 _index) view returns(address)
func (_Uipool *UipoolCaller) PoolList(opts *bind.CallOpts, _index *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "pool_list", _index)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoolList is a free data retrieval call binding the contract method 0x3a1d5d8e.
//
// Solidity: function pool_list(uint256 _index) view returns(address)
func (_Uipool *UipoolSession) PoolList(_index *big.Int) (common.Address, error) {
	return _Uipool.Contract.PoolList(&_Uipool.CallOpts, _index)
}

// PoolList is a free data retrieval call binding the contract method 0x3a1d5d8e.
//
// Solidity: function pool_list(uint256 _index) view returns(address)
func (_Uipool *UipoolCallerSession) PoolList(_index *big.Int) (common.Address, error) {
	return _Uipool.Contract.PoolList(&_Uipool.CallOpts, _index)
}

// RegistryLength is a free data retrieval call binding the contract method 0x083297d2.
//
// Solidity: function registry_length() view returns(uint256)
func (_Uipool *UipoolCaller) RegistryLength(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "registry_length")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RegistryLength is a free data retrieval call binding the contract method 0x083297d2.
//
// Solidity: function registry_length() view returns(uint256)
func (_Uipool *UipoolSession) RegistryLength() (*big.Int, error) {
	return _Uipool.Contract.RegistryLength(&_Uipool.CallOpts)
}

// RegistryLength is a free data retrieval call binding the contract method 0x083297d2.
//
// Solidity: function registry_length() view returns(uint256)
func (_Uipool *UipoolCallerSession) RegistryLength() (*big.Int, error) {
	return _Uipool.Contract.RegistryLength(&_Uipool.CallOpts)
}

// AddRegistryHandler is a paid mutator transaction binding the contract method 0x22f595c8.
//
// Solidity: function add_registry_handler(address _registry_handler) returns()
func (_Uipool *UipoolTransactor) AddRegistryHandler(opts *bind.TransactOpts, _registry_handler common.Address) (*types.Transaction, error) {
	return _Uipool.contract.Transact(opts, "add_registry_handler", _registry_handler)
}

// AddRegistryHandler is a paid mutator transaction binding the contract method 0x22f595c8.
//
// Solidity: function add_registry_handler(address _registry_handler) returns()
func (_Uipool *UipoolSession) AddRegistryHandler(_registry_handler common.Address) (*types.Transaction, error) {
	return _Uipool.Contract.AddRegistryHandler(&_Uipool.TransactOpts, _registry_handler)
}

// AddRegistryHandler is a paid mutator transaction binding the contract method 0x22f595c8.
//
// Solidity: function add_registry_handler(address _registry_handler) returns()
func (_Uipool *UipoolTransactorSession) AddRegistryHandler(_registry_handler common.Address) (*types.Transaction, error) {
	return _Uipool.Contract.AddRegistryHandler(&_Uipool.TransactOpts, _registry_handler)
}

// UpdateRegistryHandler is a paid mutator transaction binding the contract method 0x0dbdc9ff.
//
// Solidity: function update_registry_handler(uint256 _index, address _registry_handler) returns()
func (_Uipool *UipoolTransactor) UpdateRegistryHandler(opts *bind.TransactOpts, _index *big.Int, _registry_handler common.Address) (*types.Transaction, error) {
	return _Uipool.contract.Transact(opts, "update_registry_handler", _index, _registry_handler)
}

// UpdateRegistryHandler is a paid mutator transaction binding the contract method 0x0dbdc9ff.
//
// Solidity: function update_registry_handler(uint256 _index, address _registry_handler) returns()
func (_Uipool *UipoolSession) UpdateRegistryHandler(_index *big.Int, _registry_handler common.Address) (*types.Transaction, error) {
	return _Uipool.Contract.UpdateRegistryHandler(&_Uipool.TransactOpts, _index, _registry_handler)
}

// UpdateRegistryHandler is a paid mutator transaction binding the contract method 0x0dbdc9ff.
//
// Solidity: function update_registry_handler(uint256 _index, address _registry_handler) returns()
func (_Uipool *UipoolTransactorSession) UpdateRegistryHandler(_index *big.Int, _registry_handler common.Address) (*types.Transaction, error) {
	return _Uipool.Contract.UpdateRegistryHandler(&_Uipool.TransactOpts, _index, _registry_handler)
}

// UipoolCommitNewAdminIterator is returned from FilterCommitNewAdmin and is used to iterate over the raw logs and unpacked data for CommitNewAdmin events raised by the Uipool contract.
type UipoolCommitNewAdminIterator struct {
	Event *UipoolCommitNewAdmin // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *UipoolCommitNewAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UipoolCommitNewAdmin)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(UipoolCommitNewAdmin)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *UipoolCommitNewAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UipoolCommitNewAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UipoolCommitNewAdmin represents a CommitNewAdmin event raised by the Uipool contract.
type UipoolCommitNewAdmin struct {
	Deadline *big.Int
	Admin    common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterCommitNewAdmin is a free log retrieval operation binding the contract event 0x181aa3aa17d4cbf99265dd4443eba009433d3cde79d60164fde1d1a192beb935.
//
// Solidity: event CommitNewAdmin(uint256 indexed deadline, address indexed admin)
func (_Uipool *UipoolFilterer) FilterCommitNewAdmin(opts *bind.FilterOpts, deadline []*big.Int, admin []common.Address) (*UipoolCommitNewAdminIterator, error) {

	var deadlineRule []interface{}
	for _, deadlineItem := range deadline {
		deadlineRule = append(deadlineRule, deadlineItem)
	}
	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _Uipool.contract.FilterLogs(opts, "CommitNewAdmin", deadlineRule, adminRule)
	if err != nil {
		return nil, err
	}
	return &UipoolCommitNewAdminIterator{contract: _Uipool.contract, event: "CommitNewAdmin", logs: logs, sub: sub}, nil
}

// WatchCommitNewAdmin is a free log subscription operation binding the contract event 0x181aa3aa17d4cbf99265dd4443eba009433d3cde79d60164fde1d1a192beb935.
//
// Solidity: event CommitNewAdmin(uint256 indexed deadline, address indexed admin)
func (_Uipool *UipoolFilterer) WatchCommitNewAdmin(opts *bind.WatchOpts, sink chan<- *UipoolCommitNewAdmin, deadline []*big.Int, admin []common.Address) (event.Subscription, error) {

	var deadlineRule []interface{}
	for _, deadlineItem := range deadline {
		deadlineRule = append(deadlineRule, deadlineItem)
	}
	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _Uipool.contract.WatchLogs(opts, "CommitNewAdmin", deadlineRule, adminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UipoolCommitNewAdmin)
				if err := _Uipool.contract.UnpackLog(event, "CommitNewAdmin", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCommitNewAdmin is a log parse operation binding the contract event 0x181aa3aa17d4cbf99265dd4443eba009433d3cde79d60164fde1d1a192beb935.
//
// Solidity: event CommitNewAdmin(uint256 indexed deadline, address indexed admin)
func (_Uipool *UipoolFilterer) ParseCommitNewAdmin(log types.Log) (*UipoolCommitNewAdmin, error) {
	event := new(UipoolCommitNewAdmin)
	if err := _Uipool.contract.UnpackLog(event, "CommitNewAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// UipoolNewAdminIterator is returned from FilterNewAdmin and is used to iterate over the raw logs and unpacked data for NewAdmin events raised by the Uipool contract.
type UipoolNewAdminIterator struct {
	Event *UipoolNewAdmin // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *UipoolNewAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(UipoolNewAdmin)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(UipoolNewAdmin)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *UipoolNewAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *UipoolNewAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// UipoolNewAdmin represents a NewAdmin event raised by the Uipool contract.
type UipoolNewAdmin struct {
	Admin common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterNewAdmin is a free log retrieval operation binding the contract event 0x71614071b88dee5e0b2ae578a9dd7b2ebbe9ae832ba419dc0242cd065a290b6c.
//
// Solidity: event NewAdmin(address indexed admin)
func (_Uipool *UipoolFilterer) FilterNewAdmin(opts *bind.FilterOpts, admin []common.Address) (*UipoolNewAdminIterator, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _Uipool.contract.FilterLogs(opts, "NewAdmin", adminRule)
	if err != nil {
		return nil, err
	}
	return &UipoolNewAdminIterator{contract: _Uipool.contract, event: "NewAdmin", logs: logs, sub: sub}, nil
}

// WatchNewAdmin is a free log subscription operation binding the contract event 0x71614071b88dee5e0b2ae578a9dd7b2ebbe9ae832ba419dc0242cd065a290b6c.
//
// Solidity: event NewAdmin(address indexed admin)
func (_Uipool *UipoolFilterer) WatchNewAdmin(opts *bind.WatchOpts, sink chan<- *UipoolNewAdmin, admin []common.Address) (event.Subscription, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _Uipool.contract.WatchLogs(opts, "NewAdmin", adminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(UipoolNewAdmin)
				if err := _Uipool.contract.UnpackLog(event, "NewAdmin", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseNewAdmin is a log parse operation binding the contract event 0x71614071b88dee5e0b2ae578a9dd7b2ebbe9ae832ba419dc0242cd065a290b6c.
//
// Solidity: event NewAdmin(address indexed admin)
func (_Uipool *UipoolFilterer) ParseNewAdmin(log types.Log) (*UipoolNewAdmin, error) {
	event := new(UipoolNewAdmin)
	if err := _Uipool.contract.UnpackLog(event, "NewAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
