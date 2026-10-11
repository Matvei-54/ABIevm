// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package uipool

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

// AddressProviderMetaData contains all meta data concerning the AddressProvider contract.
var AddressProviderMetaData = &bind.MetaData{
	ABI: "[{\"name\":\"NewAddressIdentifier\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"id\",\"indexed\":true},{\"type\":\"address\",\"name\":\"addr\",\"indexed\":false},{\"type\":\"string\",\"name\":\"description\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"AddressModified\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"id\",\"indexed\":true},{\"type\":\"address\",\"name\":\"new_address\",\"indexed\":false},{\"type\":\"uint256\",\"name\":\"version\",\"indexed\":false}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"CommitNewAdmin\",\"inputs\":[{\"type\":\"uint256\",\"name\":\"deadline\",\"indexed\":true},{\"type\":\"address\",\"name\":\"admin\",\"indexed\":true}],\"anonymous\":false,\"type\":\"event\"},{\"name\":\"NewAdmin\",\"inputs\":[{\"type\":\"address\",\"name\":\"admin\",\"indexed\":true}],\"anonymous\":false,\"type\":\"event\"},{\"outputs\":[],\"inputs\":[{\"type\":\"address\",\"name\":\"_admin\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"name\":\"get_registry\",\"outputs\":[{\"type\":\"address\",\"name\":\"\"}],\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"gas\":1061},{\"name\":\"max_id\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\"}],\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"gas\":1258},{\"name\":\"get_address\",\"outputs\":[{\"type\":\"address\",\"name\":\"\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"_id\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"gas\":1308},{\"name\":\"add_new_id\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\"}],\"inputs\":[{\"type\":\"address\",\"name\":\"_address\"},{\"type\":\"string\",\"name\":\"_description\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"gas\":291275},{\"name\":\"set_address\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"_id\"},{\"type\":\"address\",\"name\":\"_address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"gas\":182430},{\"name\":\"unset_address\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"_id\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"gas\":101348},{\"name\":\"commit_transfer_ownership\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\"}],\"inputs\":[{\"type\":\"address\",\"name\":\"_new_admin\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"gas\":74048},{\"name\":\"apply_transfer_ownership\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\"}],\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"gas\":60125},{\"name\":\"revert_transfer_ownership\",\"outputs\":[{\"type\":\"bool\",\"name\":\"\"}],\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\",\"gas\":21400},{\"name\":\"admin\",\"outputs\":[{\"type\":\"address\",\"name\":\"\"}],\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"gas\":1331},{\"name\":\"transfer_ownership_deadline\",\"outputs\":[{\"type\":\"uint256\",\"name\":\"\"}],\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"gas\":1361},{\"name\":\"future_admin\",\"outputs\":[{\"type\":\"address\",\"name\":\"\"}],\"inputs\":[],\"stateMutability\":\"view\",\"type\":\"function\",\"gas\":1391},{\"name\":\"get_id_info\",\"outputs\":[{\"type\":\"address\",\"name\":\"addr\"},{\"type\":\"bool\",\"name\":\"is_active\"},{\"type\":\"uint256\",\"name\":\"version\"},{\"type\":\"uint256\",\"name\":\"last_modified\"},{\"type\":\"string\",\"name\":\"description\"}],\"inputs\":[{\"type\":\"uint256\",\"name\":\"arg0\"}],\"stateMutability\":\"view\",\"type\":\"function\",\"gas\":12168}]",
}

// AddressProviderABI is the input ABI used to generate the binding from.
// Deprecated: Use AddressProviderMetaData.ABI instead.
var AddressProviderABI = AddressProviderMetaData.ABI

// AddressProvider is an auto generated Go binding around an Ethereum contract.
type AddressProvider struct {
	AddressProviderCaller     // Read-only binding to the contract
	AddressProviderTransactor // Write-only binding to the contract
	AddressProviderFilterer   // Log filterer for contract events
}

// AddressProviderCaller is an auto generated read-only Go binding around an Ethereum contract.
type AddressProviderCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressProviderTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AddressProviderTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressProviderFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AddressProviderFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AddressProviderSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AddressProviderSession struct {
	Contract     *AddressProvider  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AddressProviderCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AddressProviderCallerSession struct {
	Contract *AddressProviderCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// AddressProviderTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AddressProviderTransactorSession struct {
	Contract     *AddressProviderTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// AddressProviderRaw is an auto generated low-level Go binding around an Ethereum contract.
type AddressProviderRaw struct {
	Contract *AddressProvider // Generic contract binding to access the raw methods on
}

// AddressProviderCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AddressProviderCallerRaw struct {
	Contract *AddressProviderCaller // Generic read-only contract binding to access the raw methods on
}

// AddressProviderTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AddressProviderTransactorRaw struct {
	Contract *AddressProviderTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAddressProvider creates a new instance of AddressProvider, bound to a specific deployed contract.
func NewAddressProvider(address common.Address, backend bind.ContractBackend) (*AddressProvider, error) {
	contract, err := bindAddressProvider(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AddressProvider{AddressProviderCaller: AddressProviderCaller{contract: contract}, AddressProviderTransactor: AddressProviderTransactor{contract: contract}, AddressProviderFilterer: AddressProviderFilterer{contract: contract}}, nil
}

// NewAddressProviderCaller creates a new read-only instance of AddressProvider, bound to a specific deployed contract.
func NewAddressProviderCaller(address common.Address, caller bind.ContractCaller) (*AddressProviderCaller, error) {
	contract, err := bindAddressProvider(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AddressProviderCaller{contract: contract}, nil
}

// NewAddressProviderTransactor creates a new write-only instance of AddressProvider, bound to a specific deployed contract.
func NewAddressProviderTransactor(address common.Address, transactor bind.ContractTransactor) (*AddressProviderTransactor, error) {
	contract, err := bindAddressProvider(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AddressProviderTransactor{contract: contract}, nil
}

// NewAddressProviderFilterer creates a new log filterer instance of AddressProvider, bound to a specific deployed contract.
func NewAddressProviderFilterer(address common.Address, filterer bind.ContractFilterer) (*AddressProviderFilterer, error) {
	contract, err := bindAddressProvider(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AddressProviderFilterer{contract: contract}, nil
}

// bindAddressProvider binds a generic wrapper to an already deployed contract.
func bindAddressProvider(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AddressProviderMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AddressProvider *AddressProviderRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AddressProvider.Contract.AddressProviderCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AddressProvider *AddressProviderRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressProvider.Contract.AddressProviderTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AddressProvider *AddressProviderRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AddressProvider.Contract.AddressProviderTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AddressProvider *AddressProviderCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AddressProvider.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AddressProvider *AddressProviderTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressProvider.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AddressProvider *AddressProviderTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AddressProvider.Contract.contract.Transact(opts, method, params...)
}

// Admin is a free data retrieval call binding the contract method 0xf851a440.
//
// Solidity: function admin() view returns(address)
func (_AddressProvider *AddressProviderCaller) Admin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AddressProvider.contract.Call(opts, &out, "admin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Admin is a free data retrieval call binding the contract method 0xf851a440.
//
// Solidity: function admin() view returns(address)
func (_AddressProvider *AddressProviderSession) Admin() (common.Address, error) {
	return _AddressProvider.Contract.Admin(&_AddressProvider.CallOpts)
}

// Admin is a free data retrieval call binding the contract method 0xf851a440.
//
// Solidity: function admin() view returns(address)
func (_AddressProvider *AddressProviderCallerSession) Admin() (common.Address, error) {
	return _AddressProvider.Contract.Admin(&_AddressProvider.CallOpts)
}

// FutureAdmin is a free data retrieval call binding the contract method 0x17f7182a.
//
// Solidity: function future_admin() view returns(address)
func (_AddressProvider *AddressProviderCaller) FutureAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AddressProvider.contract.Call(opts, &out, "future_admin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FutureAdmin is a free data retrieval call binding the contract method 0x17f7182a.
//
// Solidity: function future_admin() view returns(address)
func (_AddressProvider *AddressProviderSession) FutureAdmin() (common.Address, error) {
	return _AddressProvider.Contract.FutureAdmin(&_AddressProvider.CallOpts)
}

// FutureAdmin is a free data retrieval call binding the contract method 0x17f7182a.
//
// Solidity: function future_admin() view returns(address)
func (_AddressProvider *AddressProviderCallerSession) FutureAdmin() (common.Address, error) {
	return _AddressProvider.Contract.FutureAdmin(&_AddressProvider.CallOpts)
}

// GetAddress is a free data retrieval call binding the contract method 0x493f4f74.
//
// Solidity: function get_address(uint256 _id) view returns(address)
func (_AddressProvider *AddressProviderCaller) GetAddress(opts *bind.CallOpts, _id *big.Int) (common.Address, error) {
	var out []interface{}
	err := _AddressProvider.contract.Call(opts, &out, "get_address", _id)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetAddress is a free data retrieval call binding the contract method 0x493f4f74.
//
// Solidity: function get_address(uint256 _id) view returns(address)
func (_AddressProvider *AddressProviderSession) GetAddress(_id *big.Int) (common.Address, error) {
	return _AddressProvider.Contract.GetAddress(&_AddressProvider.CallOpts, _id)
}

// GetAddress is a free data retrieval call binding the contract method 0x493f4f74.
//
// Solidity: function get_address(uint256 _id) view returns(address)
func (_AddressProvider *AddressProviderCallerSession) GetAddress(_id *big.Int) (common.Address, error) {
	return _AddressProvider.Contract.GetAddress(&_AddressProvider.CallOpts, _id)
}

// GetIdInfo is a free data retrieval call binding the contract method 0x92668ecb.
//
// Solidity: function get_id_info(uint256 arg0) view returns(address addr, bool is_active, uint256 version, uint256 last_modified, string description)
func (_AddressProvider *AddressProviderCaller) GetIdInfo(opts *bind.CallOpts, arg0 *big.Int) (struct {
	Addr         common.Address
	IsActive     bool
	Version      *big.Int
	LastModified *big.Int
	Description  string
}, error) {
	var out []interface{}
	err := _AddressProvider.contract.Call(opts, &out, "get_id_info", arg0)

	outstruct := new(struct {
		Addr         common.Address
		IsActive     bool
		Version      *big.Int
		LastModified *big.Int
		Description  string
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Addr = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.IsActive = *abi.ConvertType(out[1], new(bool)).(*bool)
	outstruct.Version = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.LastModified = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Description = *abi.ConvertType(out[4], new(string)).(*string)

	return *outstruct, err

}

// GetIdInfo is a free data retrieval call binding the contract method 0x92668ecb.
//
// Solidity: function get_id_info(uint256 arg0) view returns(address addr, bool is_active, uint256 version, uint256 last_modified, string description)
func (_AddressProvider *AddressProviderSession) GetIdInfo(arg0 *big.Int) (struct {
	Addr         common.Address
	IsActive     bool
	Version      *big.Int
	LastModified *big.Int
	Description  string
}, error) {
	return _AddressProvider.Contract.GetIdInfo(&_AddressProvider.CallOpts, arg0)
}

// GetIdInfo is a free data retrieval call binding the contract method 0x92668ecb.
//
// Solidity: function get_id_info(uint256 arg0) view returns(address addr, bool is_active, uint256 version, uint256 last_modified, string description)
func (_AddressProvider *AddressProviderCallerSession) GetIdInfo(arg0 *big.Int) (struct {
	Addr         common.Address
	IsActive     bool
	Version      *big.Int
	LastModified *big.Int
	Description  string
}, error) {
	return _AddressProvider.Contract.GetIdInfo(&_AddressProvider.CallOpts, arg0)
}

// GetRegistry is a free data retrieval call binding the contract method 0xa262904b.
//
// Solidity: function get_registry() view returns(address)
func (_AddressProvider *AddressProviderCaller) GetRegistry(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _AddressProvider.contract.Call(opts, &out, "get_registry")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetRegistry is a free data retrieval call binding the contract method 0xa262904b.
//
// Solidity: function get_registry() view returns(address)
func (_AddressProvider *AddressProviderSession) GetRegistry() (common.Address, error) {
	return _AddressProvider.Contract.GetRegistry(&_AddressProvider.CallOpts)
}

// GetRegistry is a free data retrieval call binding the contract method 0xa262904b.
//
// Solidity: function get_registry() view returns(address)
func (_AddressProvider *AddressProviderCallerSession) GetRegistry() (common.Address, error) {
	return _AddressProvider.Contract.GetRegistry(&_AddressProvider.CallOpts)
}

// MaxId is a free data retrieval call binding the contract method 0x0c6d784f.
//
// Solidity: function max_id() view returns(uint256)
func (_AddressProvider *AddressProviderCaller) MaxId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AddressProvider.contract.Call(opts, &out, "max_id")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxId is a free data retrieval call binding the contract method 0x0c6d784f.
//
// Solidity: function max_id() view returns(uint256)
func (_AddressProvider *AddressProviderSession) MaxId() (*big.Int, error) {
	return _AddressProvider.Contract.MaxId(&_AddressProvider.CallOpts)
}

// MaxId is a free data retrieval call binding the contract method 0x0c6d784f.
//
// Solidity: function max_id() view returns(uint256)
func (_AddressProvider *AddressProviderCallerSession) MaxId() (*big.Int, error) {
	return _AddressProvider.Contract.MaxId(&_AddressProvider.CallOpts)
}

// TransferOwnershipDeadline is a free data retrieval call binding the contract method 0xe0a0b586.
//
// Solidity: function transfer_ownership_deadline() view returns(uint256)
func (_AddressProvider *AddressProviderCaller) TransferOwnershipDeadline(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _AddressProvider.contract.Call(opts, &out, "transfer_ownership_deadline")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TransferOwnershipDeadline is a free data retrieval call binding the contract method 0xe0a0b586.
//
// Solidity: function transfer_ownership_deadline() view returns(uint256)
func (_AddressProvider *AddressProviderSession) TransferOwnershipDeadline() (*big.Int, error) {
	return _AddressProvider.Contract.TransferOwnershipDeadline(&_AddressProvider.CallOpts)
}

// TransferOwnershipDeadline is a free data retrieval call binding the contract method 0xe0a0b586.
//
// Solidity: function transfer_ownership_deadline() view returns(uint256)
func (_AddressProvider *AddressProviderCallerSession) TransferOwnershipDeadline() (*big.Int, error) {
	return _AddressProvider.Contract.TransferOwnershipDeadline(&_AddressProvider.CallOpts)
}

// AddNewId is a paid mutator transaction binding the contract method 0x168f9579.
//
// Solidity: function add_new_id(address _address, string _description) returns(uint256)
func (_AddressProvider *AddressProviderTransactor) AddNewId(opts *bind.TransactOpts, _address common.Address, _description string) (*types.Transaction, error) {
	return _AddressProvider.contract.Transact(opts, "add_new_id", _address, _description)
}

// AddNewId is a paid mutator transaction binding the contract method 0x168f9579.
//
// Solidity: function add_new_id(address _address, string _description) returns(uint256)
func (_AddressProvider *AddressProviderSession) AddNewId(_address common.Address, _description string) (*types.Transaction, error) {
	return _AddressProvider.Contract.AddNewId(&_AddressProvider.TransactOpts, _address, _description)
}

// AddNewId is a paid mutator transaction binding the contract method 0x168f9579.
//
// Solidity: function add_new_id(address _address, string _description) returns(uint256)
func (_AddressProvider *AddressProviderTransactorSession) AddNewId(_address common.Address, _description string) (*types.Transaction, error) {
	return _AddressProvider.Contract.AddNewId(&_AddressProvider.TransactOpts, _address, _description)
}

// ApplyTransferOwnership is a paid mutator transaction binding the contract method 0x6a1c05ae.
//
// Solidity: function apply_transfer_ownership() returns(bool)
func (_AddressProvider *AddressProviderTransactor) ApplyTransferOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressProvider.contract.Transact(opts, "apply_transfer_ownership")
}

// ApplyTransferOwnership is a paid mutator transaction binding the contract method 0x6a1c05ae.
//
// Solidity: function apply_transfer_ownership() returns(bool)
func (_AddressProvider *AddressProviderSession) ApplyTransferOwnership() (*types.Transaction, error) {
	return _AddressProvider.Contract.ApplyTransferOwnership(&_AddressProvider.TransactOpts)
}

// ApplyTransferOwnership is a paid mutator transaction binding the contract method 0x6a1c05ae.
//
// Solidity: function apply_transfer_ownership() returns(bool)
func (_AddressProvider *AddressProviderTransactorSession) ApplyTransferOwnership() (*types.Transaction, error) {
	return _AddressProvider.Contract.ApplyTransferOwnership(&_AddressProvider.TransactOpts)
}

// CommitTransferOwnership is a paid mutator transaction binding the contract method 0x6b441a40.
//
// Solidity: function commit_transfer_ownership(address _new_admin) returns(bool)
func (_AddressProvider *AddressProviderTransactor) CommitTransferOwnership(opts *bind.TransactOpts, _new_admin common.Address) (*types.Transaction, error) {
	return _AddressProvider.contract.Transact(opts, "commit_transfer_ownership", _new_admin)
}

// CommitTransferOwnership is a paid mutator transaction binding the contract method 0x6b441a40.
//
// Solidity: function commit_transfer_ownership(address _new_admin) returns(bool)
func (_AddressProvider *AddressProviderSession) CommitTransferOwnership(_new_admin common.Address) (*types.Transaction, error) {
	return _AddressProvider.Contract.CommitTransferOwnership(&_AddressProvider.TransactOpts, _new_admin)
}

// CommitTransferOwnership is a paid mutator transaction binding the contract method 0x6b441a40.
//
// Solidity: function commit_transfer_ownership(address _new_admin) returns(bool)
func (_AddressProvider *AddressProviderTransactorSession) CommitTransferOwnership(_new_admin common.Address) (*types.Transaction, error) {
	return _AddressProvider.Contract.CommitTransferOwnership(&_AddressProvider.TransactOpts, _new_admin)
}

// RevertTransferOwnership is a paid mutator transaction binding the contract method 0x86fbf193.
//
// Solidity: function revert_transfer_ownership() returns(bool)
func (_AddressProvider *AddressProviderTransactor) RevertTransferOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AddressProvider.contract.Transact(opts, "revert_transfer_ownership")
}

// RevertTransferOwnership is a paid mutator transaction binding the contract method 0x86fbf193.
//
// Solidity: function revert_transfer_ownership() returns(bool)
func (_AddressProvider *AddressProviderSession) RevertTransferOwnership() (*types.Transaction, error) {
	return _AddressProvider.Contract.RevertTransferOwnership(&_AddressProvider.TransactOpts)
}

// RevertTransferOwnership is a paid mutator transaction binding the contract method 0x86fbf193.
//
// Solidity: function revert_transfer_ownership() returns(bool)
func (_AddressProvider *AddressProviderTransactorSession) RevertTransferOwnership() (*types.Transaction, error) {
	return _AddressProvider.Contract.RevertTransferOwnership(&_AddressProvider.TransactOpts)
}

// SetAddress is a paid mutator transaction binding the contract method 0x6a84cad0.
//
// Solidity: function set_address(uint256 _id, address _address) returns(bool)
func (_AddressProvider *AddressProviderTransactor) SetAddress(opts *bind.TransactOpts, _id *big.Int, _address common.Address) (*types.Transaction, error) {
	return _AddressProvider.contract.Transact(opts, "set_address", _id, _address)
}

// SetAddress is a paid mutator transaction binding the contract method 0x6a84cad0.
//
// Solidity: function set_address(uint256 _id, address _address) returns(bool)
func (_AddressProvider *AddressProviderSession) SetAddress(_id *big.Int, _address common.Address) (*types.Transaction, error) {
	return _AddressProvider.Contract.SetAddress(&_AddressProvider.TransactOpts, _id, _address)
}

// SetAddress is a paid mutator transaction binding the contract method 0x6a84cad0.
//
// Solidity: function set_address(uint256 _id, address _address) returns(bool)
func (_AddressProvider *AddressProviderTransactorSession) SetAddress(_id *big.Int, _address common.Address) (*types.Transaction, error) {
	return _AddressProvider.Contract.SetAddress(&_AddressProvider.TransactOpts, _id, _address)
}

// UnsetAddress is a paid mutator transaction binding the contract method 0x5eec0daa.
//
// Solidity: function unset_address(uint256 _id) returns(bool)
func (_AddressProvider *AddressProviderTransactor) UnsetAddress(opts *bind.TransactOpts, _id *big.Int) (*types.Transaction, error) {
	return _AddressProvider.contract.Transact(opts, "unset_address", _id)
}

// UnsetAddress is a paid mutator transaction binding the contract method 0x5eec0daa.
//
// Solidity: function unset_address(uint256 _id) returns(bool)
func (_AddressProvider *AddressProviderSession) UnsetAddress(_id *big.Int) (*types.Transaction, error) {
	return _AddressProvider.Contract.UnsetAddress(&_AddressProvider.TransactOpts, _id)
}

// UnsetAddress is a paid mutator transaction binding the contract method 0x5eec0daa.
//
// Solidity: function unset_address(uint256 _id) returns(bool)
func (_AddressProvider *AddressProviderTransactorSession) UnsetAddress(_id *big.Int) (*types.Transaction, error) {
	return _AddressProvider.Contract.UnsetAddress(&_AddressProvider.TransactOpts, _id)
}

// AddressProviderAddressModifiedIterator is returned from FilterAddressModified and is used to iterate over the raw logs and unpacked data for AddressModified events raised by the AddressProvider contract.
type AddressProviderAddressModifiedIterator struct {
	Event *AddressProviderAddressModified // Event containing the contract specifics and raw log

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
func (it *AddressProviderAddressModifiedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AddressProviderAddressModified)
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
		it.Event = new(AddressProviderAddressModified)
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
func (it *AddressProviderAddressModifiedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AddressProviderAddressModifiedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AddressProviderAddressModified represents a AddressModified event raised by the AddressProvider contract.
type AddressProviderAddressModified struct {
	Id         *big.Int
	NewAddress common.Address
	Version    *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterAddressModified is a free log retrieval operation binding the contract event 0xe7a6334c4f573efdf292d404d59adacec345f4f7c76495a034008edda0acef47.
//
// Solidity: event AddressModified(uint256 indexed id, address new_address, uint256 version)
func (_AddressProvider *AddressProviderFilterer) FilterAddressModified(opts *bind.FilterOpts, id []*big.Int) (*AddressProviderAddressModifiedIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _AddressProvider.contract.FilterLogs(opts, "AddressModified", idRule)
	if err != nil {
		return nil, err
	}
	return &AddressProviderAddressModifiedIterator{contract: _AddressProvider.contract, event: "AddressModified", logs: logs, sub: sub}, nil
}

// WatchAddressModified is a free log subscription operation binding the contract event 0xe7a6334c4f573efdf292d404d59adacec345f4f7c76495a034008edda0acef47.
//
// Solidity: event AddressModified(uint256 indexed id, address new_address, uint256 version)
func (_AddressProvider *AddressProviderFilterer) WatchAddressModified(opts *bind.WatchOpts, sink chan<- *AddressProviderAddressModified, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _AddressProvider.contract.WatchLogs(opts, "AddressModified", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AddressProviderAddressModified)
				if err := _AddressProvider.contract.UnpackLog(event, "AddressModified", log); err != nil {
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

// ParseAddressModified is a log parse operation binding the contract event 0xe7a6334c4f573efdf292d404d59adacec345f4f7c76495a034008edda0acef47.
//
// Solidity: event AddressModified(uint256 indexed id, address new_address, uint256 version)
func (_AddressProvider *AddressProviderFilterer) ParseAddressModified(log types.Log) (*AddressProviderAddressModified, error) {
	event := new(AddressProviderAddressModified)
	if err := _AddressProvider.contract.UnpackLog(event, "AddressModified", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AddressProviderCommitNewAdminIterator is returned from FilterCommitNewAdmin and is used to iterate over the raw logs and unpacked data for CommitNewAdmin events raised by the AddressProvider contract.
type AddressProviderCommitNewAdminIterator struct {
	Event *AddressProviderCommitNewAdmin // Event containing the contract specifics and raw log

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
func (it *AddressProviderCommitNewAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AddressProviderCommitNewAdmin)
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
		it.Event = new(AddressProviderCommitNewAdmin)
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
func (it *AddressProviderCommitNewAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AddressProviderCommitNewAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AddressProviderCommitNewAdmin represents a CommitNewAdmin event raised by the AddressProvider contract.
type AddressProviderCommitNewAdmin struct {
	Deadline *big.Int
	Admin    common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterCommitNewAdmin is a free log retrieval operation binding the contract event 0x181aa3aa17d4cbf99265dd4443eba009433d3cde79d60164fde1d1a192beb935.
//
// Solidity: event CommitNewAdmin(uint256 indexed deadline, address indexed admin)
func (_AddressProvider *AddressProviderFilterer) FilterCommitNewAdmin(opts *bind.FilterOpts, deadline []*big.Int, admin []common.Address) (*AddressProviderCommitNewAdminIterator, error) {

	var deadlineRule []interface{}
	for _, deadlineItem := range deadline {
		deadlineRule = append(deadlineRule, deadlineItem)
	}
	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _AddressProvider.contract.FilterLogs(opts, "CommitNewAdmin", deadlineRule, adminRule)
	if err != nil {
		return nil, err
	}
	return &AddressProviderCommitNewAdminIterator{contract: _AddressProvider.contract, event: "CommitNewAdmin", logs: logs, sub: sub}, nil
}

// WatchCommitNewAdmin is a free log subscription operation binding the contract event 0x181aa3aa17d4cbf99265dd4443eba009433d3cde79d60164fde1d1a192beb935.
//
// Solidity: event CommitNewAdmin(uint256 indexed deadline, address indexed admin)
func (_AddressProvider *AddressProviderFilterer) WatchCommitNewAdmin(opts *bind.WatchOpts, sink chan<- *AddressProviderCommitNewAdmin, deadline []*big.Int, admin []common.Address) (event.Subscription, error) {

	var deadlineRule []interface{}
	for _, deadlineItem := range deadline {
		deadlineRule = append(deadlineRule, deadlineItem)
	}
	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _AddressProvider.contract.WatchLogs(opts, "CommitNewAdmin", deadlineRule, adminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AddressProviderCommitNewAdmin)
				if err := _AddressProvider.contract.UnpackLog(event, "CommitNewAdmin", log); err != nil {
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
func (_AddressProvider *AddressProviderFilterer) ParseCommitNewAdmin(log types.Log) (*AddressProviderCommitNewAdmin, error) {
	event := new(AddressProviderCommitNewAdmin)
	if err := _AddressProvider.contract.UnpackLog(event, "CommitNewAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AddressProviderNewAddressIdentifierIterator is returned from FilterNewAddressIdentifier and is used to iterate over the raw logs and unpacked data for NewAddressIdentifier events raised by the AddressProvider contract.
type AddressProviderNewAddressIdentifierIterator struct {
	Event *AddressProviderNewAddressIdentifier // Event containing the contract specifics and raw log

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
func (it *AddressProviderNewAddressIdentifierIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AddressProviderNewAddressIdentifier)
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
		it.Event = new(AddressProviderNewAddressIdentifier)
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
func (it *AddressProviderNewAddressIdentifierIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AddressProviderNewAddressIdentifierIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AddressProviderNewAddressIdentifier represents a NewAddressIdentifier event raised by the AddressProvider contract.
type AddressProviderNewAddressIdentifier struct {
	Id          *big.Int
	Addr        common.Address
	Description string
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterNewAddressIdentifier is a free log retrieval operation binding the contract event 0x5b0f9b31dc08c19adcc0181c1b97ad54a84487faf0a4fdcb88c8681724298af9.
//
// Solidity: event NewAddressIdentifier(uint256 indexed id, address addr, string description)
func (_AddressProvider *AddressProviderFilterer) FilterNewAddressIdentifier(opts *bind.FilterOpts, id []*big.Int) (*AddressProviderNewAddressIdentifierIterator, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _AddressProvider.contract.FilterLogs(opts, "NewAddressIdentifier", idRule)
	if err != nil {
		return nil, err
	}
	return &AddressProviderNewAddressIdentifierIterator{contract: _AddressProvider.contract, event: "NewAddressIdentifier", logs: logs, sub: sub}, nil
}

// WatchNewAddressIdentifier is a free log subscription operation binding the contract event 0x5b0f9b31dc08c19adcc0181c1b97ad54a84487faf0a4fdcb88c8681724298af9.
//
// Solidity: event NewAddressIdentifier(uint256 indexed id, address addr, string description)
func (_AddressProvider *AddressProviderFilterer) WatchNewAddressIdentifier(opts *bind.WatchOpts, sink chan<- *AddressProviderNewAddressIdentifier, id []*big.Int) (event.Subscription, error) {

	var idRule []interface{}
	for _, idItem := range id {
		idRule = append(idRule, idItem)
	}

	logs, sub, err := _AddressProvider.contract.WatchLogs(opts, "NewAddressIdentifier", idRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AddressProviderNewAddressIdentifier)
				if err := _AddressProvider.contract.UnpackLog(event, "NewAddressIdentifier", log); err != nil {
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

// ParseNewAddressIdentifier is a log parse operation binding the contract event 0x5b0f9b31dc08c19adcc0181c1b97ad54a84487faf0a4fdcb88c8681724298af9.
//
// Solidity: event NewAddressIdentifier(uint256 indexed id, address addr, string description)
func (_AddressProvider *AddressProviderFilterer) ParseNewAddressIdentifier(log types.Log) (*AddressProviderNewAddressIdentifier, error) {
	event := new(AddressProviderNewAddressIdentifier)
	if err := _AddressProvider.contract.UnpackLog(event, "NewAddressIdentifier", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AddressProviderNewAdminIterator is returned from FilterNewAdmin and is used to iterate over the raw logs and unpacked data for NewAdmin events raised by the AddressProvider contract.
type AddressProviderNewAdminIterator struct {
	Event *AddressProviderNewAdmin // Event containing the contract specifics and raw log

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
func (it *AddressProviderNewAdminIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AddressProviderNewAdmin)
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
		it.Event = new(AddressProviderNewAdmin)
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
func (it *AddressProviderNewAdminIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AddressProviderNewAdminIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AddressProviderNewAdmin represents a NewAdmin event raised by the AddressProvider contract.
type AddressProviderNewAdmin struct {
	Admin common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterNewAdmin is a free log retrieval operation binding the contract event 0x71614071b88dee5e0b2ae578a9dd7b2ebbe9ae832ba419dc0242cd065a290b6c.
//
// Solidity: event NewAdmin(address indexed admin)
func (_AddressProvider *AddressProviderFilterer) FilterNewAdmin(opts *bind.FilterOpts, admin []common.Address) (*AddressProviderNewAdminIterator, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _AddressProvider.contract.FilterLogs(opts, "NewAdmin", adminRule)
	if err != nil {
		return nil, err
	}
	return &AddressProviderNewAdminIterator{contract: _AddressProvider.contract, event: "NewAdmin", logs: logs, sub: sub}, nil
}

// WatchNewAdmin is a free log subscription operation binding the contract event 0x71614071b88dee5e0b2ae578a9dd7b2ebbe9ae832ba419dc0242cd065a290b6c.
//
// Solidity: event NewAdmin(address indexed admin)
func (_AddressProvider *AddressProviderFilterer) WatchNewAdmin(opts *bind.WatchOpts, sink chan<- *AddressProviderNewAdmin, admin []common.Address) (event.Subscription, error) {

	var adminRule []interface{}
	for _, adminItem := range admin {
		adminRule = append(adminRule, adminItem)
	}

	logs, sub, err := _AddressProvider.contract.WatchLogs(opts, "NewAdmin", adminRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AddressProviderNewAdmin)
				if err := _AddressProvider.contract.UnpackLog(event, "NewAdmin", log); err != nil {
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
func (_AddressProvider *AddressProviderFilterer) ParseNewAdmin(log types.Log) (*AddressProviderNewAdmin, error) {
	event := new(AddressProviderNewAdmin)
	if err := _AddressProvider.contract.UnpackLog(event, "NewAdmin", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
