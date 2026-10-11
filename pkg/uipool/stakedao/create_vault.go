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

// VaultControllerMetaData contains all meta data concerning the VaultController contract.
var VaultControllerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"AccountantAlreadySet\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GaugeAlreadyShutdown\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GaugeNotShutdown\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidAllocationTarget\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotPermissionSetter\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyRegistrar\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyStrategy\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAddress\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"gauge\",\"type\":\"address\"}],\"name\":\"GaugeShutdown\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"gauge\",\"type\":\"address\"}],\"name\":\"GaugeUnshutdown\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferStarted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes4\",\"name\":\"selector\",\"type\":\"bytes4\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"name\":\"PermissionSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"setter\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"name\":\"PermissionSetterSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"indexed\":true,\"internalType\":\"string\",\"name\":\"componentId\",\"type\":\"string\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"component\",\"type\":\"address\"}],\"name\":\"ProtocolComponentSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"ProtocolShutdown\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"registrar\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"name\":\"RegistrarPermissionSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"Unpaused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"gauge\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"vault\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"asset\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"rewardReceiver\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"VaultRegistered\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"acceptOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"accountant\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"allocator\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"selector\",\"type\":\"bytes4\"}],\"name\":\"allowed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge_\",\"type\":\"address\"}],\"name\":\"asset\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"factory\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"feeReceiver\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"gateway\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"gauge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"vault\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"asset\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rewardReceiver\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"bool\",\"name\":\"isShutdown\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"name\":\"isPaused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"registrar_\",\"type\":\"address\"}],\"name\":\"isRegistrar\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge_\",\"type\":\"address\"}],\"name\":\"isShutdown\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"}],\"name\":\"isValidAllocationTarget\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"locker\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"pause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pendingOwner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"permissionSetters\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge_\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"vault\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"asset\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"rewardReceiver\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"registerVault\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"registrar\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"}],\"name\":\"removeValidAllocationTarget\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge_\",\"type\":\"address\"}],\"name\":\"rewardReceiver\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"address\",\"name\":\"accountant\",\"type\":\"address\"}],\"name\":\"setAccountant\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"address\",\"name\":\"allocator\",\"type\":\"address\"}],\"name\":\"setAllocator\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"address\",\"name\":\"factory\",\"type\":\"address\"}],\"name\":\"setFactory\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"address\",\"name\":\"feeReceiver\",\"type\":\"address\"}],\"name\":\"setFeeReceiver\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"address\",\"name\":\"gateway\",\"type\":\"address\"}],\"name\":\"setGateway\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"address\",\"name\":\"locker\",\"type\":\"address\"}],\"name\":\"setLocker\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"caller\",\"type\":\"address\"},{\"internalType\":\"bytes4\",\"name\":\"selector\",\"type\":\"bytes4\"},{\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"name\":\"setPermission\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"setter\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"name\":\"setPermissionSetter\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"registrar_\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"name\":\"setRegistrar\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"},{\"internalType\":\"address\",\"name\":\"strategy\",\"type\":\"address\"}],\"name\":\"setStrategy\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"}],\"name\":\"setValidAllocationTarget\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge_\",\"type\":\"address\"}],\"name\":\"shutdown\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"strategy\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"protocolId\",\"type\":\"bytes4\"}],\"name\":\"unpause\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge_\",\"type\":\"address\"}],\"name\":\"unshutdown\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"gauge_\",\"type\":\"address\"}],\"name\":\"vault\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// VaultControllerABI is the input ABI used to generate the binding from.
// Deprecated: Use VaultControllerMetaData.ABI instead.
var VaultControllerABI = VaultControllerMetaData.ABI

// VaultController is an auto generated Go binding around an Ethereum contract.
type VaultController struct {
	VaultControllerCaller     // Read-only binding to the contract
	VaultControllerTransactor // Write-only binding to the contract
	VaultControllerFilterer   // Log filterer for contract events
}

// VaultControllerCaller is an auto generated read-only Go binding around an Ethereum contract.
type VaultControllerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VaultControllerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type VaultControllerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VaultControllerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type VaultControllerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VaultControllerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type VaultControllerSession struct {
	Contract     *VaultController  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VaultControllerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type VaultControllerCallerSession struct {
	Contract *VaultControllerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// VaultControllerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type VaultControllerTransactorSession struct {
	Contract     *VaultControllerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// VaultControllerRaw is an auto generated low-level Go binding around an Ethereum contract.
type VaultControllerRaw struct {
	Contract *VaultController // Generic contract binding to access the raw methods on
}

// VaultControllerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type VaultControllerCallerRaw struct {
	Contract *VaultControllerCaller // Generic read-only contract binding to access the raw methods on
}

// VaultControllerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type VaultControllerTransactorRaw struct {
	Contract *VaultControllerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewVaultController creates a new instance of VaultController, bound to a specific deployed contract.
func NewVaultController(address common.Address, backend bind.ContractBackend) (*VaultController, error) {
	contract, err := bindVaultController(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &VaultController{VaultControllerCaller: VaultControllerCaller{contract: contract}, VaultControllerTransactor: VaultControllerTransactor{contract: contract}, VaultControllerFilterer: VaultControllerFilterer{contract: contract}}, nil
}

// NewVaultControllerCaller creates a new read-only instance of VaultController, bound to a specific deployed contract.
func NewVaultControllerCaller(address common.Address, caller bind.ContractCaller) (*VaultControllerCaller, error) {
	contract, err := bindVaultController(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &VaultControllerCaller{contract: contract}, nil
}

// NewVaultControllerTransactor creates a new write-only instance of VaultController, bound to a specific deployed contract.
func NewVaultControllerTransactor(address common.Address, transactor bind.ContractTransactor) (*VaultControllerTransactor, error) {
	contract, err := bindVaultController(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &VaultControllerTransactor{contract: contract}, nil
}

// NewVaultControllerFilterer creates a new log filterer instance of VaultController, bound to a specific deployed contract.
func NewVaultControllerFilterer(address common.Address, filterer bind.ContractFilterer) (*VaultControllerFilterer, error) {
	contract, err := bindVaultController(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &VaultControllerFilterer{contract: contract}, nil
}

// bindVaultController binds a generic wrapper to an already deployed contract.
func bindVaultController(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := VaultControllerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VaultController *VaultControllerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _VaultController.Contract.VaultControllerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VaultController *VaultControllerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VaultController.Contract.VaultControllerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VaultController *VaultControllerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VaultController.Contract.VaultControllerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_VaultController *VaultControllerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _VaultController.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_VaultController *VaultControllerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VaultController.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_VaultController *VaultControllerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _VaultController.Contract.contract.Transact(opts, method, params...)
}

// Accountant is a free data retrieval call binding the contract method 0xcd398592.
//
// Solidity: function accountant(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCaller) Accountant(opts *bind.CallOpts, protocolId [4]byte) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "accountant", protocolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Accountant is a free data retrieval call binding the contract method 0xcd398592.
//
// Solidity: function accountant(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerSession) Accountant(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Accountant(&_VaultController.CallOpts, protocolId)
}

// Accountant is a free data retrieval call binding the contract method 0xcd398592.
//
// Solidity: function accountant(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCallerSession) Accountant(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Accountant(&_VaultController.CallOpts, protocolId)
}

// Allocator is a free data retrieval call binding the contract method 0xef64d763.
//
// Solidity: function allocator(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCaller) Allocator(opts *bind.CallOpts, protocolId [4]byte) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "allocator", protocolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Allocator is a free data retrieval call binding the contract method 0xef64d763.
//
// Solidity: function allocator(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerSession) Allocator(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Allocator(&_VaultController.CallOpts, protocolId)
}

// Allocator is a free data retrieval call binding the contract method 0xef64d763.
//
// Solidity: function allocator(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCallerSession) Allocator(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Allocator(&_VaultController.CallOpts, protocolId)
}

// Allowed is a free data retrieval call binding the contract method 0x05de62c0.
//
// Solidity: function allowed(address target, address caller, bytes4 selector) view returns(bool)
func (_VaultController *VaultControllerCaller) Allowed(opts *bind.CallOpts, target common.Address, caller common.Address, selector [4]byte) (bool, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "allowed", target, caller, selector)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Allowed is a free data retrieval call binding the contract method 0x05de62c0.
//
// Solidity: function allowed(address target, address caller, bytes4 selector) view returns(bool)
func (_VaultController *VaultControllerSession) Allowed(target common.Address, caller common.Address, selector [4]byte) (bool, error) {
	return _VaultController.Contract.Allowed(&_VaultController.CallOpts, target, caller, selector)
}

// Allowed is a free data retrieval call binding the contract method 0x05de62c0.
//
// Solidity: function allowed(address target, address caller, bytes4 selector) view returns(bool)
func (_VaultController *VaultControllerCallerSession) Allowed(target common.Address, caller common.Address, selector [4]byte) (bool, error) {
	return _VaultController.Contract.Allowed(&_VaultController.CallOpts, target, caller, selector)
}

// Asset is a free data retrieval call binding the contract method 0x9c4667a2.
//
// Solidity: function asset(address gauge_) view returns(address)
func (_VaultController *VaultControllerCaller) Asset(opts *bind.CallOpts, gauge_ common.Address) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "asset", gauge_)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Asset is a free data retrieval call binding the contract method 0x9c4667a2.
//
// Solidity: function asset(address gauge_) view returns(address)
func (_VaultController *VaultControllerSession) Asset(gauge_ common.Address) (common.Address, error) {
	return _VaultController.Contract.Asset(&_VaultController.CallOpts, gauge_)
}

// Asset is a free data retrieval call binding the contract method 0x9c4667a2.
//
// Solidity: function asset(address gauge_) view returns(address)
func (_VaultController *VaultControllerCallerSession) Asset(gauge_ common.Address) (common.Address, error) {
	return _VaultController.Contract.Asset(&_VaultController.CallOpts, gauge_)
}

// Factory is a free data retrieval call binding the contract method 0xb3890209.
//
// Solidity: function factory(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCaller) Factory(opts *bind.CallOpts, protocolId [4]byte) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "factory", protocolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Factory is a free data retrieval call binding the contract method 0xb3890209.
//
// Solidity: function factory(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerSession) Factory(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Factory(&_VaultController.CallOpts, protocolId)
}

// Factory is a free data retrieval call binding the contract method 0xb3890209.
//
// Solidity: function factory(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCallerSession) Factory(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Factory(&_VaultController.CallOpts, protocolId)
}

// FeeReceiver is a free data retrieval call binding the contract method 0x09f14272.
//
// Solidity: function feeReceiver(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCaller) FeeReceiver(opts *bind.CallOpts, protocolId [4]byte) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "feeReceiver", protocolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FeeReceiver is a free data retrieval call binding the contract method 0x09f14272.
//
// Solidity: function feeReceiver(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerSession) FeeReceiver(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.FeeReceiver(&_VaultController.CallOpts, protocolId)
}

// FeeReceiver is a free data retrieval call binding the contract method 0x09f14272.
//
// Solidity: function feeReceiver(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCallerSession) FeeReceiver(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.FeeReceiver(&_VaultController.CallOpts, protocolId)
}

// Gateway is a free data retrieval call binding the contract method 0x05f533cf.
//
// Solidity: function gateway(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCaller) Gateway(opts *bind.CallOpts, protocolId [4]byte) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "gateway", protocolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Gateway is a free data retrieval call binding the contract method 0x05f533cf.
//
// Solidity: function gateway(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerSession) Gateway(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Gateway(&_VaultController.CallOpts, protocolId)
}

// Gateway is a free data retrieval call binding the contract method 0x05f533cf.
//
// Solidity: function gateway(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCallerSession) Gateway(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Gateway(&_VaultController.CallOpts, protocolId)
}

// Gauge is a free data retrieval call binding the contract method 0x7664a7e8.
//
// Solidity: function gauge(address ) view returns(address vault, address asset, address rewardReceiver, bytes4 protocolId, bool isShutdown)
func (_VaultController *VaultControllerCaller) Gauge(opts *bind.CallOpts, arg0 common.Address) (struct {
	Vault          common.Address
	Asset          common.Address
	RewardReceiver common.Address
	ProtocolId     [4]byte
	IsShutdown     bool
}, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "gauge", arg0)

	outstruct := new(struct {
		Vault          common.Address
		Asset          common.Address
		RewardReceiver common.Address
		ProtocolId     [4]byte
		IsShutdown     bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Vault = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Asset = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.RewardReceiver = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	outstruct.ProtocolId = *abi.ConvertType(out[3], new([4]byte)).(*[4]byte)
	outstruct.IsShutdown = *abi.ConvertType(out[4], new(bool)).(*bool)

	return *outstruct, err

}

// Gauge is a free data retrieval call binding the contract method 0x7664a7e8.
//
// Solidity: function gauge(address ) view returns(address vault, address asset, address rewardReceiver, bytes4 protocolId, bool isShutdown)
func (_VaultController *VaultControllerSession) Gauge(arg0 common.Address) (struct {
	Vault          common.Address
	Asset          common.Address
	RewardReceiver common.Address
	ProtocolId     [4]byte
	IsShutdown     bool
}, error) {
	return _VaultController.Contract.Gauge(&_VaultController.CallOpts, arg0)
}

// Gauge is a free data retrieval call binding the contract method 0x7664a7e8.
//
// Solidity: function gauge(address ) view returns(address vault, address asset, address rewardReceiver, bytes4 protocolId, bool isShutdown)
func (_VaultController *VaultControllerCallerSession) Gauge(arg0 common.Address) (struct {
	Vault          common.Address
	Asset          common.Address
	RewardReceiver common.Address
	ProtocolId     [4]byte
	IsShutdown     bool
}, error) {
	return _VaultController.Contract.Gauge(&_VaultController.CallOpts, arg0)
}

// IsPaused is a free data retrieval call binding the contract method 0x09b65e66.
//
// Solidity: function isPaused(bytes4 ) view returns(bool)
func (_VaultController *VaultControllerCaller) IsPaused(opts *bind.CallOpts, arg0 [4]byte) (bool, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "isPaused", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPaused is a free data retrieval call binding the contract method 0x09b65e66.
//
// Solidity: function isPaused(bytes4 ) view returns(bool)
func (_VaultController *VaultControllerSession) IsPaused(arg0 [4]byte) (bool, error) {
	return _VaultController.Contract.IsPaused(&_VaultController.CallOpts, arg0)
}

// IsPaused is a free data retrieval call binding the contract method 0x09b65e66.
//
// Solidity: function isPaused(bytes4 ) view returns(bool)
func (_VaultController *VaultControllerCallerSession) IsPaused(arg0 [4]byte) (bool, error) {
	return _VaultController.Contract.IsPaused(&_VaultController.CallOpts, arg0)
}

// IsRegistrar is a free data retrieval call binding the contract method 0xd5db72eb.
//
// Solidity: function isRegistrar(address registrar_) view returns(bool)
func (_VaultController *VaultControllerCaller) IsRegistrar(opts *bind.CallOpts, registrar_ common.Address) (bool, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "isRegistrar", registrar_)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsRegistrar is a free data retrieval call binding the contract method 0xd5db72eb.
//
// Solidity: function isRegistrar(address registrar_) view returns(bool)
func (_VaultController *VaultControllerSession) IsRegistrar(registrar_ common.Address) (bool, error) {
	return _VaultController.Contract.IsRegistrar(&_VaultController.CallOpts, registrar_)
}

// IsRegistrar is a free data retrieval call binding the contract method 0xd5db72eb.
//
// Solidity: function isRegistrar(address registrar_) view returns(bool)
func (_VaultController *VaultControllerCallerSession) IsRegistrar(registrar_ common.Address) (bool, error) {
	return _VaultController.Contract.IsRegistrar(&_VaultController.CallOpts, registrar_)
}

// IsShutdown is a free data retrieval call binding the contract method 0x4bc5d735.
//
// Solidity: function isShutdown(address gauge_) view returns(bool)
func (_VaultController *VaultControllerCaller) IsShutdown(opts *bind.CallOpts, gauge_ common.Address) (bool, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "isShutdown", gauge_)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsShutdown is a free data retrieval call binding the contract method 0x4bc5d735.
//
// Solidity: function isShutdown(address gauge_) view returns(bool)
func (_VaultController *VaultControllerSession) IsShutdown(gauge_ common.Address) (bool, error) {
	return _VaultController.Contract.IsShutdown(&_VaultController.CallOpts, gauge_)
}

// IsShutdown is a free data retrieval call binding the contract method 0x4bc5d735.
//
// Solidity: function isShutdown(address gauge_) view returns(bool)
func (_VaultController *VaultControllerCallerSession) IsShutdown(gauge_ common.Address) (bool, error) {
	return _VaultController.Contract.IsShutdown(&_VaultController.CallOpts, gauge_)
}

// IsValidAllocationTarget is a free data retrieval call binding the contract method 0x4cf64f33.
//
// Solidity: function isValidAllocationTarget(address gauge, address target) view returns(bool)
func (_VaultController *VaultControllerCaller) IsValidAllocationTarget(opts *bind.CallOpts, gauge common.Address, target common.Address) (bool, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "isValidAllocationTarget", gauge, target)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidAllocationTarget is a free data retrieval call binding the contract method 0x4cf64f33.
//
// Solidity: function isValidAllocationTarget(address gauge, address target) view returns(bool)
func (_VaultController *VaultControllerSession) IsValidAllocationTarget(gauge common.Address, target common.Address) (bool, error) {
	return _VaultController.Contract.IsValidAllocationTarget(&_VaultController.CallOpts, gauge, target)
}

// IsValidAllocationTarget is a free data retrieval call binding the contract method 0x4cf64f33.
//
// Solidity: function isValidAllocationTarget(address gauge, address target) view returns(bool)
func (_VaultController *VaultControllerCallerSession) IsValidAllocationTarget(gauge common.Address, target common.Address) (bool, error) {
	return _VaultController.Contract.IsValidAllocationTarget(&_VaultController.CallOpts, gauge, target)
}

// Locker is a free data retrieval call binding the contract method 0x22f1cf69.
//
// Solidity: function locker(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCaller) Locker(opts *bind.CallOpts, protocolId [4]byte) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "locker", protocolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Locker is a free data retrieval call binding the contract method 0x22f1cf69.
//
// Solidity: function locker(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerSession) Locker(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Locker(&_VaultController.CallOpts, protocolId)
}

// Locker is a free data retrieval call binding the contract method 0x22f1cf69.
//
// Solidity: function locker(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCallerSession) Locker(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Locker(&_VaultController.CallOpts, protocolId)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_VaultController *VaultControllerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_VaultController *VaultControllerSession) Owner() (common.Address, error) {
	return _VaultController.Contract.Owner(&_VaultController.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_VaultController *VaultControllerCallerSession) Owner() (common.Address, error) {
	return _VaultController.Contract.Owner(&_VaultController.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_VaultController *VaultControllerCaller) PendingOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "pendingOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_VaultController *VaultControllerSession) PendingOwner() (common.Address, error) {
	return _VaultController.Contract.PendingOwner(&_VaultController.CallOpts)
}

// PendingOwner is a free data retrieval call binding the contract method 0xe30c3978.
//
// Solidity: function pendingOwner() view returns(address)
func (_VaultController *VaultControllerCallerSession) PendingOwner() (common.Address, error) {
	return _VaultController.Contract.PendingOwner(&_VaultController.CallOpts)
}

// PermissionSetters is a free data retrieval call binding the contract method 0x760ad7bb.
//
// Solidity: function permissionSetters(address ) view returns(bool)
func (_VaultController *VaultControllerCaller) PermissionSetters(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "permissionSetters", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PermissionSetters is a free data retrieval call binding the contract method 0x760ad7bb.
//
// Solidity: function permissionSetters(address ) view returns(bool)
func (_VaultController *VaultControllerSession) PermissionSetters(arg0 common.Address) (bool, error) {
	return _VaultController.Contract.PermissionSetters(&_VaultController.CallOpts, arg0)
}

// PermissionSetters is a free data retrieval call binding the contract method 0x760ad7bb.
//
// Solidity: function permissionSetters(address ) view returns(bool)
func (_VaultController *VaultControllerCallerSession) PermissionSetters(arg0 common.Address) (bool, error) {
	return _VaultController.Contract.PermissionSetters(&_VaultController.CallOpts, arg0)
}

// Registrar is a free data retrieval call binding the contract method 0xe397cb9e.
//
// Solidity: function registrar(address ) view returns(bool)
func (_VaultController *VaultControllerCaller) Registrar(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "registrar", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Registrar is a free data retrieval call binding the contract method 0xe397cb9e.
//
// Solidity: function registrar(address ) view returns(bool)
func (_VaultController *VaultControllerSession) Registrar(arg0 common.Address) (bool, error) {
	return _VaultController.Contract.Registrar(&_VaultController.CallOpts, arg0)
}

// Registrar is a free data retrieval call binding the contract method 0xe397cb9e.
//
// Solidity: function registrar(address ) view returns(bool)
func (_VaultController *VaultControllerCallerSession) Registrar(arg0 common.Address) (bool, error) {
	return _VaultController.Contract.Registrar(&_VaultController.CallOpts, arg0)
}

// RewardReceiver is a free data retrieval call binding the contract method 0x6720a120.
//
// Solidity: function rewardReceiver(address gauge_) view returns(address)
func (_VaultController *VaultControllerCaller) RewardReceiver(opts *bind.CallOpts, gauge_ common.Address) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "rewardReceiver", gauge_)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// RewardReceiver is a free data retrieval call binding the contract method 0x6720a120.
//
// Solidity: function rewardReceiver(address gauge_) view returns(address)
func (_VaultController *VaultControllerSession) RewardReceiver(gauge_ common.Address) (common.Address, error) {
	return _VaultController.Contract.RewardReceiver(&_VaultController.CallOpts, gauge_)
}

// RewardReceiver is a free data retrieval call binding the contract method 0x6720a120.
//
// Solidity: function rewardReceiver(address gauge_) view returns(address)
func (_VaultController *VaultControllerCallerSession) RewardReceiver(gauge_ common.Address) (common.Address, error) {
	return _VaultController.Contract.RewardReceiver(&_VaultController.CallOpts, gauge_)
}

// Strategy is a free data retrieval call binding the contract method 0x0595a654.
//
// Solidity: function strategy(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCaller) Strategy(opts *bind.CallOpts, protocolId [4]byte) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "strategy", protocolId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Strategy is a free data retrieval call binding the contract method 0x0595a654.
//
// Solidity: function strategy(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerSession) Strategy(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Strategy(&_VaultController.CallOpts, protocolId)
}

// Strategy is a free data retrieval call binding the contract method 0x0595a654.
//
// Solidity: function strategy(bytes4 protocolId) view returns(address)
func (_VaultController *VaultControllerCallerSession) Strategy(protocolId [4]byte) (common.Address, error) {
	return _VaultController.Contract.Strategy(&_VaultController.CallOpts, protocolId)
}

// Vault is a free data retrieval call binding the contract method 0xf815c03d.
//
// Solidity: function vault(address gauge_) view returns(address)
func (_VaultController *VaultControllerCaller) Vault(opts *bind.CallOpts, gauge_ common.Address) (common.Address, error) {
	var out []interface{}
	err := _VaultController.contract.Call(opts, &out, "vault", gauge_)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Vault is a free data retrieval call binding the contract method 0xf815c03d.
//
// Solidity: function vault(address gauge_) view returns(address)
func (_VaultController *VaultControllerSession) Vault(gauge_ common.Address) (common.Address, error) {
	return _VaultController.Contract.Vault(&_VaultController.CallOpts, gauge_)
}

// Vault is a free data retrieval call binding the contract method 0xf815c03d.
//
// Solidity: function vault(address gauge_) view returns(address)
func (_VaultController *VaultControllerCallerSession) Vault(gauge_ common.Address) (common.Address, error) {
	return _VaultController.Contract.Vault(&_VaultController.CallOpts, gauge_)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_VaultController *VaultControllerTransactor) AcceptOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "acceptOwnership")
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_VaultController *VaultControllerSession) AcceptOwnership() (*types.Transaction, error) {
	return _VaultController.Contract.AcceptOwnership(&_VaultController.TransactOpts)
}

// AcceptOwnership is a paid mutator transaction binding the contract method 0x79ba5097.
//
// Solidity: function acceptOwnership() returns()
func (_VaultController *VaultControllerTransactorSession) AcceptOwnership() (*types.Transaction, error) {
	return _VaultController.Contract.AcceptOwnership(&_VaultController.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x3aa83ec7.
//
// Solidity: function pause(bytes4 protocolId) returns()
func (_VaultController *VaultControllerTransactor) Pause(opts *bind.TransactOpts, protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "pause", protocolId)
}

// Pause is a paid mutator transaction binding the contract method 0x3aa83ec7.
//
// Solidity: function pause(bytes4 protocolId) returns()
func (_VaultController *VaultControllerSession) Pause(protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.Contract.Pause(&_VaultController.TransactOpts, protocolId)
}

// Pause is a paid mutator transaction binding the contract method 0x3aa83ec7.
//
// Solidity: function pause(bytes4 protocolId) returns()
func (_VaultController *VaultControllerTransactorSession) Pause(protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.Contract.Pause(&_VaultController.TransactOpts, protocolId)
}

// RegisterVault is a paid mutator transaction binding the contract method 0x7c663ae2.
//
// Solidity: function registerVault(address gauge_, address vault, address asset, address rewardReceiver, bytes4 protocolId) returns()
func (_VaultController *VaultControllerTransactor) RegisterVault(opts *bind.TransactOpts, gauge_ common.Address, vault common.Address, asset common.Address, rewardReceiver common.Address, protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "registerVault", gauge_, vault, asset, rewardReceiver, protocolId)
}

// RegisterVault is a paid mutator transaction binding the contract method 0x7c663ae2.
//
// Solidity: function registerVault(address gauge_, address vault, address asset, address rewardReceiver, bytes4 protocolId) returns()
func (_VaultController *VaultControllerSession) RegisterVault(gauge_ common.Address, vault common.Address, asset common.Address, rewardReceiver common.Address, protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.Contract.RegisterVault(&_VaultController.TransactOpts, gauge_, vault, asset, rewardReceiver, protocolId)
}

// RegisterVault is a paid mutator transaction binding the contract method 0x7c663ae2.
//
// Solidity: function registerVault(address gauge_, address vault, address asset, address rewardReceiver, bytes4 protocolId) returns()
func (_VaultController *VaultControllerTransactorSession) RegisterVault(gauge_ common.Address, vault common.Address, asset common.Address, rewardReceiver common.Address, protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.Contract.RegisterVault(&_VaultController.TransactOpts, gauge_, vault, asset, rewardReceiver, protocolId)
}

// RemoveValidAllocationTarget is a paid mutator transaction binding the contract method 0x23404e6c.
//
// Solidity: function removeValidAllocationTarget(address gauge, address target) returns()
func (_VaultController *VaultControllerTransactor) RemoveValidAllocationTarget(opts *bind.TransactOpts, gauge common.Address, target common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "removeValidAllocationTarget", gauge, target)
}

// RemoveValidAllocationTarget is a paid mutator transaction binding the contract method 0x23404e6c.
//
// Solidity: function removeValidAllocationTarget(address gauge, address target) returns()
func (_VaultController *VaultControllerSession) RemoveValidAllocationTarget(gauge common.Address, target common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.RemoveValidAllocationTarget(&_VaultController.TransactOpts, gauge, target)
}

// RemoveValidAllocationTarget is a paid mutator transaction binding the contract method 0x23404e6c.
//
// Solidity: function removeValidAllocationTarget(address gauge, address target) returns()
func (_VaultController *VaultControllerTransactorSession) RemoveValidAllocationTarget(gauge common.Address, target common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.RemoveValidAllocationTarget(&_VaultController.TransactOpts, gauge, target)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_VaultController *VaultControllerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_VaultController *VaultControllerSession) RenounceOwnership() (*types.Transaction, error) {
	return _VaultController.Contract.RenounceOwnership(&_VaultController.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_VaultController *VaultControllerTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _VaultController.Contract.RenounceOwnership(&_VaultController.TransactOpts)
}

// SetAccountant is a paid mutator transaction binding the contract method 0xd5fe1d13.
//
// Solidity: function setAccountant(bytes4 protocolId, address accountant) returns()
func (_VaultController *VaultControllerTransactor) SetAccountant(opts *bind.TransactOpts, protocolId [4]byte, accountant common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setAccountant", protocolId, accountant)
}

// SetAccountant is a paid mutator transaction binding the contract method 0xd5fe1d13.
//
// Solidity: function setAccountant(bytes4 protocolId, address accountant) returns()
func (_VaultController *VaultControllerSession) SetAccountant(protocolId [4]byte, accountant common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetAccountant(&_VaultController.TransactOpts, protocolId, accountant)
}

// SetAccountant is a paid mutator transaction binding the contract method 0xd5fe1d13.
//
// Solidity: function setAccountant(bytes4 protocolId, address accountant) returns()
func (_VaultController *VaultControllerTransactorSession) SetAccountant(protocolId [4]byte, accountant common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetAccountant(&_VaultController.TransactOpts, protocolId, accountant)
}

// SetAllocator is a paid mutator transaction binding the contract method 0xf46c6c67.
//
// Solidity: function setAllocator(bytes4 protocolId, address allocator) returns()
func (_VaultController *VaultControllerTransactor) SetAllocator(opts *bind.TransactOpts, protocolId [4]byte, allocator common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setAllocator", protocolId, allocator)
}

// SetAllocator is a paid mutator transaction binding the contract method 0xf46c6c67.
//
// Solidity: function setAllocator(bytes4 protocolId, address allocator) returns()
func (_VaultController *VaultControllerSession) SetAllocator(protocolId [4]byte, allocator common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetAllocator(&_VaultController.TransactOpts, protocolId, allocator)
}

// SetAllocator is a paid mutator transaction binding the contract method 0xf46c6c67.
//
// Solidity: function setAllocator(bytes4 protocolId, address allocator) returns()
func (_VaultController *VaultControllerTransactorSession) SetAllocator(protocolId [4]byte, allocator common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetAllocator(&_VaultController.TransactOpts, protocolId, allocator)
}

// SetFactory is a paid mutator transaction binding the contract method 0x0c642df1.
//
// Solidity: function setFactory(bytes4 protocolId, address factory) returns()
func (_VaultController *VaultControllerTransactor) SetFactory(opts *bind.TransactOpts, protocolId [4]byte, factory common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setFactory", protocolId, factory)
}

// SetFactory is a paid mutator transaction binding the contract method 0x0c642df1.
//
// Solidity: function setFactory(bytes4 protocolId, address factory) returns()
func (_VaultController *VaultControllerSession) SetFactory(protocolId [4]byte, factory common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetFactory(&_VaultController.TransactOpts, protocolId, factory)
}

// SetFactory is a paid mutator transaction binding the contract method 0x0c642df1.
//
// Solidity: function setFactory(bytes4 protocolId, address factory) returns()
func (_VaultController *VaultControllerTransactorSession) SetFactory(protocolId [4]byte, factory common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetFactory(&_VaultController.TransactOpts, protocolId, factory)
}

// SetFeeReceiver is a paid mutator transaction binding the contract method 0xb23a56f0.
//
// Solidity: function setFeeReceiver(bytes4 protocolId, address feeReceiver) returns()
func (_VaultController *VaultControllerTransactor) SetFeeReceiver(opts *bind.TransactOpts, protocolId [4]byte, feeReceiver common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setFeeReceiver", protocolId, feeReceiver)
}

// SetFeeReceiver is a paid mutator transaction binding the contract method 0xb23a56f0.
//
// Solidity: function setFeeReceiver(bytes4 protocolId, address feeReceiver) returns()
func (_VaultController *VaultControllerSession) SetFeeReceiver(protocolId [4]byte, feeReceiver common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetFeeReceiver(&_VaultController.TransactOpts, protocolId, feeReceiver)
}

// SetFeeReceiver is a paid mutator transaction binding the contract method 0xb23a56f0.
//
// Solidity: function setFeeReceiver(bytes4 protocolId, address feeReceiver) returns()
func (_VaultController *VaultControllerTransactorSession) SetFeeReceiver(protocolId [4]byte, feeReceiver common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetFeeReceiver(&_VaultController.TransactOpts, protocolId, feeReceiver)
}

// SetGateway is a paid mutator transaction binding the contract method 0x7b730d16.
//
// Solidity: function setGateway(bytes4 protocolId, address gateway) returns()
func (_VaultController *VaultControllerTransactor) SetGateway(opts *bind.TransactOpts, protocolId [4]byte, gateway common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setGateway", protocolId, gateway)
}

// SetGateway is a paid mutator transaction binding the contract method 0x7b730d16.
//
// Solidity: function setGateway(bytes4 protocolId, address gateway) returns()
func (_VaultController *VaultControllerSession) SetGateway(protocolId [4]byte, gateway common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetGateway(&_VaultController.TransactOpts, protocolId, gateway)
}

// SetGateway is a paid mutator transaction binding the contract method 0x7b730d16.
//
// Solidity: function setGateway(bytes4 protocolId, address gateway) returns()
func (_VaultController *VaultControllerTransactorSession) SetGateway(protocolId [4]byte, gateway common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetGateway(&_VaultController.TransactOpts, protocolId, gateway)
}

// SetLocker is a paid mutator transaction binding the contract method 0x791d5d31.
//
// Solidity: function setLocker(bytes4 protocolId, address locker) returns()
func (_VaultController *VaultControllerTransactor) SetLocker(opts *bind.TransactOpts, protocolId [4]byte, locker common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setLocker", protocolId, locker)
}

// SetLocker is a paid mutator transaction binding the contract method 0x791d5d31.
//
// Solidity: function setLocker(bytes4 protocolId, address locker) returns()
func (_VaultController *VaultControllerSession) SetLocker(protocolId [4]byte, locker common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetLocker(&_VaultController.TransactOpts, protocolId, locker)
}

// SetLocker is a paid mutator transaction binding the contract method 0x791d5d31.
//
// Solidity: function setLocker(bytes4 protocolId, address locker) returns()
func (_VaultController *VaultControllerTransactorSession) SetLocker(protocolId [4]byte, locker common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetLocker(&_VaultController.TransactOpts, protocolId, locker)
}

// SetPermission is a paid mutator transaction binding the contract method 0xe64624fa.
//
// Solidity: function setPermission(address target, address caller, bytes4 selector, bool allowed) returns()
func (_VaultController *VaultControllerTransactor) SetPermission(opts *bind.TransactOpts, target common.Address, caller common.Address, selector [4]byte, allowed bool) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setPermission", target, caller, selector, allowed)
}

// SetPermission is a paid mutator transaction binding the contract method 0xe64624fa.
//
// Solidity: function setPermission(address target, address caller, bytes4 selector, bool allowed) returns()
func (_VaultController *VaultControllerSession) SetPermission(target common.Address, caller common.Address, selector [4]byte, allowed bool) (*types.Transaction, error) {
	return _VaultController.Contract.SetPermission(&_VaultController.TransactOpts, target, caller, selector, allowed)
}

// SetPermission is a paid mutator transaction binding the contract method 0xe64624fa.
//
// Solidity: function setPermission(address target, address caller, bytes4 selector, bool allowed) returns()
func (_VaultController *VaultControllerTransactorSession) SetPermission(target common.Address, caller common.Address, selector [4]byte, allowed bool) (*types.Transaction, error) {
	return _VaultController.Contract.SetPermission(&_VaultController.TransactOpts, target, caller, selector, allowed)
}

// SetPermissionSetter is a paid mutator transaction binding the contract method 0xf1710d08.
//
// Solidity: function setPermissionSetter(address setter, bool allowed) returns()
func (_VaultController *VaultControllerTransactor) SetPermissionSetter(opts *bind.TransactOpts, setter common.Address, allowed bool) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setPermissionSetter", setter, allowed)
}

// SetPermissionSetter is a paid mutator transaction binding the contract method 0xf1710d08.
//
// Solidity: function setPermissionSetter(address setter, bool allowed) returns()
func (_VaultController *VaultControllerSession) SetPermissionSetter(setter common.Address, allowed bool) (*types.Transaction, error) {
	return _VaultController.Contract.SetPermissionSetter(&_VaultController.TransactOpts, setter, allowed)
}

// SetPermissionSetter is a paid mutator transaction binding the contract method 0xf1710d08.
//
// Solidity: function setPermissionSetter(address setter, bool allowed) returns()
func (_VaultController *VaultControllerTransactorSession) SetPermissionSetter(setter common.Address, allowed bool) (*types.Transaction, error) {
	return _VaultController.Contract.SetPermissionSetter(&_VaultController.TransactOpts, setter, allowed)
}

// SetRegistrar is a paid mutator transaction binding the contract method 0xc1bf1dcf.
//
// Solidity: function setRegistrar(address registrar_, bool allowed) returns()
func (_VaultController *VaultControllerTransactor) SetRegistrar(opts *bind.TransactOpts, registrar_ common.Address, allowed bool) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setRegistrar", registrar_, allowed)
}

// SetRegistrar is a paid mutator transaction binding the contract method 0xc1bf1dcf.
//
// Solidity: function setRegistrar(address registrar_, bool allowed) returns()
func (_VaultController *VaultControllerSession) SetRegistrar(registrar_ common.Address, allowed bool) (*types.Transaction, error) {
	return _VaultController.Contract.SetRegistrar(&_VaultController.TransactOpts, registrar_, allowed)
}

// SetRegistrar is a paid mutator transaction binding the contract method 0xc1bf1dcf.
//
// Solidity: function setRegistrar(address registrar_, bool allowed) returns()
func (_VaultController *VaultControllerTransactorSession) SetRegistrar(registrar_ common.Address, allowed bool) (*types.Transaction, error) {
	return _VaultController.Contract.SetRegistrar(&_VaultController.TransactOpts, registrar_, allowed)
}

// SetStrategy is a paid mutator transaction binding the contract method 0x8604dee6.
//
// Solidity: function setStrategy(bytes4 protocolId, address strategy) returns()
func (_VaultController *VaultControllerTransactor) SetStrategy(opts *bind.TransactOpts, protocolId [4]byte, strategy common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setStrategy", protocolId, strategy)
}

// SetStrategy is a paid mutator transaction binding the contract method 0x8604dee6.
//
// Solidity: function setStrategy(bytes4 protocolId, address strategy) returns()
func (_VaultController *VaultControllerSession) SetStrategy(protocolId [4]byte, strategy common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetStrategy(&_VaultController.TransactOpts, protocolId, strategy)
}

// SetStrategy is a paid mutator transaction binding the contract method 0x8604dee6.
//
// Solidity: function setStrategy(bytes4 protocolId, address strategy) returns()
func (_VaultController *VaultControllerTransactorSession) SetStrategy(protocolId [4]byte, strategy common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetStrategy(&_VaultController.TransactOpts, protocolId, strategy)
}

// SetValidAllocationTarget is a paid mutator transaction binding the contract method 0xa1292903.
//
// Solidity: function setValidAllocationTarget(address gauge, address target) returns()
func (_VaultController *VaultControllerTransactor) SetValidAllocationTarget(opts *bind.TransactOpts, gauge common.Address, target common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "setValidAllocationTarget", gauge, target)
}

// SetValidAllocationTarget is a paid mutator transaction binding the contract method 0xa1292903.
//
// Solidity: function setValidAllocationTarget(address gauge, address target) returns()
func (_VaultController *VaultControllerSession) SetValidAllocationTarget(gauge common.Address, target common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetValidAllocationTarget(&_VaultController.TransactOpts, gauge, target)
}

// SetValidAllocationTarget is a paid mutator transaction binding the contract method 0xa1292903.
//
// Solidity: function setValidAllocationTarget(address gauge, address target) returns()
func (_VaultController *VaultControllerTransactorSession) SetValidAllocationTarget(gauge common.Address, target common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.SetValidAllocationTarget(&_VaultController.TransactOpts, gauge, target)
}

// Shutdown is a paid mutator transaction binding the contract method 0xcce2f3fb.
//
// Solidity: function shutdown(address gauge_) returns()
func (_VaultController *VaultControllerTransactor) Shutdown(opts *bind.TransactOpts, gauge_ common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "shutdown", gauge_)
}

// Shutdown is a paid mutator transaction binding the contract method 0xcce2f3fb.
//
// Solidity: function shutdown(address gauge_) returns()
func (_VaultController *VaultControllerSession) Shutdown(gauge_ common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.Shutdown(&_VaultController.TransactOpts, gauge_)
}

// Shutdown is a paid mutator transaction binding the contract method 0xcce2f3fb.
//
// Solidity: function shutdown(address gauge_) returns()
func (_VaultController *VaultControllerTransactorSession) Shutdown(gauge_ common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.Shutdown(&_VaultController.TransactOpts, gauge_)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_VaultController *VaultControllerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_VaultController *VaultControllerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.TransferOwnership(&_VaultController.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_VaultController *VaultControllerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.TransferOwnership(&_VaultController.TransactOpts, newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0xbac1e94b.
//
// Solidity: function unpause(bytes4 protocolId) returns()
func (_VaultController *VaultControllerTransactor) Unpause(opts *bind.TransactOpts, protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "unpause", protocolId)
}

// Unpause is a paid mutator transaction binding the contract method 0xbac1e94b.
//
// Solidity: function unpause(bytes4 protocolId) returns()
func (_VaultController *VaultControllerSession) Unpause(protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.Contract.Unpause(&_VaultController.TransactOpts, protocolId)
}

// Unpause is a paid mutator transaction binding the contract method 0xbac1e94b.
//
// Solidity: function unpause(bytes4 protocolId) returns()
func (_VaultController *VaultControllerTransactorSession) Unpause(protocolId [4]byte) (*types.Transaction, error) {
	return _VaultController.Contract.Unpause(&_VaultController.TransactOpts, protocolId)
}

// Unshutdown is a paid mutator transaction binding the contract method 0x622e152d.
//
// Solidity: function unshutdown(address gauge_) returns()
func (_VaultController *VaultControllerTransactor) Unshutdown(opts *bind.TransactOpts, gauge_ common.Address) (*types.Transaction, error) {
	return _VaultController.contract.Transact(opts, "unshutdown", gauge_)
}

// Unshutdown is a paid mutator transaction binding the contract method 0x622e152d.
//
// Solidity: function unshutdown(address gauge_) returns()
func (_VaultController *VaultControllerSession) Unshutdown(gauge_ common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.Unshutdown(&_VaultController.TransactOpts, gauge_)
}

// Unshutdown is a paid mutator transaction binding the contract method 0x622e152d.
//
// Solidity: function unshutdown(address gauge_) returns()
func (_VaultController *VaultControllerTransactorSession) Unshutdown(gauge_ common.Address) (*types.Transaction, error) {
	return _VaultController.Contract.Unshutdown(&_VaultController.TransactOpts, gauge_)
}

// VaultControllerGaugeShutdownIterator is returned from FilterGaugeShutdown and is used to iterate over the raw logs and unpacked data for GaugeShutdown events raised by the VaultController contract.
type VaultControllerGaugeShutdownIterator struct {
	Event *VaultControllerGaugeShutdown // Event containing the contract specifics and raw log

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
func (it *VaultControllerGaugeShutdownIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerGaugeShutdown)
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
		it.Event = new(VaultControllerGaugeShutdown)
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
func (it *VaultControllerGaugeShutdownIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerGaugeShutdownIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerGaugeShutdown represents a GaugeShutdown event raised by the VaultController contract.
type VaultControllerGaugeShutdown struct {
	Gauge common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterGaugeShutdown is a free log retrieval operation binding the contract event 0x5303f5e3277cb33a424732af8355550ed3446778c3b06e5cb9b79474773c95f0.
//
// Solidity: event GaugeShutdown(address indexed gauge)
func (_VaultController *VaultControllerFilterer) FilterGaugeShutdown(opts *bind.FilterOpts, gauge []common.Address) (*VaultControllerGaugeShutdownIterator, error) {

	var gaugeRule []interface{}
	for _, gaugeItem := range gauge {
		gaugeRule = append(gaugeRule, gaugeItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "GaugeShutdown", gaugeRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerGaugeShutdownIterator{contract: _VaultController.contract, event: "GaugeShutdown", logs: logs, sub: sub}, nil
}

// WatchGaugeShutdown is a free log subscription operation binding the contract event 0x5303f5e3277cb33a424732af8355550ed3446778c3b06e5cb9b79474773c95f0.
//
// Solidity: event GaugeShutdown(address indexed gauge)
func (_VaultController *VaultControllerFilterer) WatchGaugeShutdown(opts *bind.WatchOpts, sink chan<- *VaultControllerGaugeShutdown, gauge []common.Address) (event.Subscription, error) {

	var gaugeRule []interface{}
	for _, gaugeItem := range gauge {
		gaugeRule = append(gaugeRule, gaugeItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "GaugeShutdown", gaugeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerGaugeShutdown)
				if err := _VaultController.contract.UnpackLog(event, "GaugeShutdown", log); err != nil {
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

// ParseGaugeShutdown is a log parse operation binding the contract event 0x5303f5e3277cb33a424732af8355550ed3446778c3b06e5cb9b79474773c95f0.
//
// Solidity: event GaugeShutdown(address indexed gauge)
func (_VaultController *VaultControllerFilterer) ParseGaugeShutdown(log types.Log) (*VaultControllerGaugeShutdown, error) {
	event := new(VaultControllerGaugeShutdown)
	if err := _VaultController.contract.UnpackLog(event, "GaugeShutdown", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerGaugeUnshutdownIterator is returned from FilterGaugeUnshutdown and is used to iterate over the raw logs and unpacked data for GaugeUnshutdown events raised by the VaultController contract.
type VaultControllerGaugeUnshutdownIterator struct {
	Event *VaultControllerGaugeUnshutdown // Event containing the contract specifics and raw log

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
func (it *VaultControllerGaugeUnshutdownIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerGaugeUnshutdown)
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
		it.Event = new(VaultControllerGaugeUnshutdown)
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
func (it *VaultControllerGaugeUnshutdownIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerGaugeUnshutdownIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerGaugeUnshutdown represents a GaugeUnshutdown event raised by the VaultController contract.
type VaultControllerGaugeUnshutdown struct {
	Gauge common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterGaugeUnshutdown is a free log retrieval operation binding the contract event 0xc8e74fb3d97c0a82e22800e7bf8248c005c7aed9376e0893f8a2506ce1992c89.
//
// Solidity: event GaugeUnshutdown(address indexed gauge)
func (_VaultController *VaultControllerFilterer) FilterGaugeUnshutdown(opts *bind.FilterOpts, gauge []common.Address) (*VaultControllerGaugeUnshutdownIterator, error) {

	var gaugeRule []interface{}
	for _, gaugeItem := range gauge {
		gaugeRule = append(gaugeRule, gaugeItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "GaugeUnshutdown", gaugeRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerGaugeUnshutdownIterator{contract: _VaultController.contract, event: "GaugeUnshutdown", logs: logs, sub: sub}, nil
}

// WatchGaugeUnshutdown is a free log subscription operation binding the contract event 0xc8e74fb3d97c0a82e22800e7bf8248c005c7aed9376e0893f8a2506ce1992c89.
//
// Solidity: event GaugeUnshutdown(address indexed gauge)
func (_VaultController *VaultControllerFilterer) WatchGaugeUnshutdown(opts *bind.WatchOpts, sink chan<- *VaultControllerGaugeUnshutdown, gauge []common.Address) (event.Subscription, error) {

	var gaugeRule []interface{}
	for _, gaugeItem := range gauge {
		gaugeRule = append(gaugeRule, gaugeItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "GaugeUnshutdown", gaugeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerGaugeUnshutdown)
				if err := _VaultController.contract.UnpackLog(event, "GaugeUnshutdown", log); err != nil {
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

// ParseGaugeUnshutdown is a log parse operation binding the contract event 0xc8e74fb3d97c0a82e22800e7bf8248c005c7aed9376e0893f8a2506ce1992c89.
//
// Solidity: event GaugeUnshutdown(address indexed gauge)
func (_VaultController *VaultControllerFilterer) ParseGaugeUnshutdown(log types.Log) (*VaultControllerGaugeUnshutdown, error) {
	event := new(VaultControllerGaugeUnshutdown)
	if err := _VaultController.contract.UnpackLog(event, "GaugeUnshutdown", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerOwnershipTransferStartedIterator is returned from FilterOwnershipTransferStarted and is used to iterate over the raw logs and unpacked data for OwnershipTransferStarted events raised by the VaultController contract.
type VaultControllerOwnershipTransferStartedIterator struct {
	Event *VaultControllerOwnershipTransferStarted // Event containing the contract specifics and raw log

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
func (it *VaultControllerOwnershipTransferStartedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerOwnershipTransferStarted)
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
		it.Event = new(VaultControllerOwnershipTransferStarted)
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
func (it *VaultControllerOwnershipTransferStartedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerOwnershipTransferStartedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerOwnershipTransferStarted represents a OwnershipTransferStarted event raised by the VaultController contract.
type VaultControllerOwnershipTransferStarted struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferStarted is a free log retrieval operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_VaultController *VaultControllerFilterer) FilterOwnershipTransferStarted(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*VaultControllerOwnershipTransferStartedIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerOwnershipTransferStartedIterator{contract: _VaultController.contract, event: "OwnershipTransferStarted", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferStarted is a free log subscription operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_VaultController *VaultControllerFilterer) WatchOwnershipTransferStarted(opts *bind.WatchOpts, sink chan<- *VaultControllerOwnershipTransferStarted, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "OwnershipTransferStarted", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerOwnershipTransferStarted)
				if err := _VaultController.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
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

// ParseOwnershipTransferStarted is a log parse operation binding the contract event 0x38d16b8cac22d99fc7c124b9cd0de2d3fa1faef420bfe791d8c362d765e22700.
//
// Solidity: event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner)
func (_VaultController *VaultControllerFilterer) ParseOwnershipTransferStarted(log types.Log) (*VaultControllerOwnershipTransferStarted, error) {
	event := new(VaultControllerOwnershipTransferStarted)
	if err := _VaultController.contract.UnpackLog(event, "OwnershipTransferStarted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the VaultController contract.
type VaultControllerOwnershipTransferredIterator struct {
	Event *VaultControllerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *VaultControllerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerOwnershipTransferred)
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
		it.Event = new(VaultControllerOwnershipTransferred)
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
func (it *VaultControllerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerOwnershipTransferred represents a OwnershipTransferred event raised by the VaultController contract.
type VaultControllerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_VaultController *VaultControllerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*VaultControllerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerOwnershipTransferredIterator{contract: _VaultController.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_VaultController *VaultControllerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *VaultControllerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerOwnershipTransferred)
				if err := _VaultController.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_VaultController *VaultControllerFilterer) ParseOwnershipTransferred(log types.Log) (*VaultControllerOwnershipTransferred, error) {
	event := new(VaultControllerOwnershipTransferred)
	if err := _VaultController.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the VaultController contract.
type VaultControllerPausedIterator struct {
	Event *VaultControllerPaused // Event containing the contract specifics and raw log

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
func (it *VaultControllerPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerPaused)
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
		it.Event = new(VaultControllerPaused)
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
func (it *VaultControllerPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerPaused represents a Paused event raised by the VaultController contract.
type VaultControllerPaused struct {
	ProtocolId [4]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0xa5ced19b26a7fec9bfb9e35d68605a1241d62b1ab9a3b862f36a219891fdf515.
//
// Solidity: event Paused(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) FilterPaused(opts *bind.FilterOpts, protocolId [][4]byte) (*VaultControllerPausedIterator, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "Paused", protocolIdRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerPausedIterator{contract: _VaultController.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0xa5ced19b26a7fec9bfb9e35d68605a1241d62b1ab9a3b862f36a219891fdf515.
//
// Solidity: event Paused(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *VaultControllerPaused, protocolId [][4]byte) (event.Subscription, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "Paused", protocolIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerPaused)
				if err := _VaultController.contract.UnpackLog(event, "Paused", log); err != nil {
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

// ParsePaused is a log parse operation binding the contract event 0xa5ced19b26a7fec9bfb9e35d68605a1241d62b1ab9a3b862f36a219891fdf515.
//
// Solidity: event Paused(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) ParsePaused(log types.Log) (*VaultControllerPaused, error) {
	event := new(VaultControllerPaused)
	if err := _VaultController.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerPermissionSetIterator is returned from FilterPermissionSet and is used to iterate over the raw logs and unpacked data for PermissionSet events raised by the VaultController contract.
type VaultControllerPermissionSetIterator struct {
	Event *VaultControllerPermissionSet // Event containing the contract specifics and raw log

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
func (it *VaultControllerPermissionSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerPermissionSet)
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
		it.Event = new(VaultControllerPermissionSet)
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
func (it *VaultControllerPermissionSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerPermissionSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerPermissionSet represents a PermissionSet event raised by the VaultController contract.
type VaultControllerPermissionSet struct {
	Target   common.Address
	Caller   common.Address
	Selector [4]byte
	Allowed  bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterPermissionSet is a free log retrieval operation binding the contract event 0x9d950e4919d7c91068179be3cd6a99029ee6fdfcf4fb488053aa79841b517e03.
//
// Solidity: event PermissionSet(address indexed target, address indexed caller, bytes4 indexed selector, bool allowed)
func (_VaultController *VaultControllerFilterer) FilterPermissionSet(opts *bind.FilterOpts, target []common.Address, caller []common.Address, selector [][4]byte) (*VaultControllerPermissionSetIterator, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}
	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}
	var selectorRule []interface{}
	for _, selectorItem := range selector {
		selectorRule = append(selectorRule, selectorItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "PermissionSet", targetRule, callerRule, selectorRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerPermissionSetIterator{contract: _VaultController.contract, event: "PermissionSet", logs: logs, sub: sub}, nil
}

// WatchPermissionSet is a free log subscription operation binding the contract event 0x9d950e4919d7c91068179be3cd6a99029ee6fdfcf4fb488053aa79841b517e03.
//
// Solidity: event PermissionSet(address indexed target, address indexed caller, bytes4 indexed selector, bool allowed)
func (_VaultController *VaultControllerFilterer) WatchPermissionSet(opts *bind.WatchOpts, sink chan<- *VaultControllerPermissionSet, target []common.Address, caller []common.Address, selector [][4]byte) (event.Subscription, error) {

	var targetRule []interface{}
	for _, targetItem := range target {
		targetRule = append(targetRule, targetItem)
	}
	var callerRule []interface{}
	for _, callerItem := range caller {
		callerRule = append(callerRule, callerItem)
	}
	var selectorRule []interface{}
	for _, selectorItem := range selector {
		selectorRule = append(selectorRule, selectorItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "PermissionSet", targetRule, callerRule, selectorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerPermissionSet)
				if err := _VaultController.contract.UnpackLog(event, "PermissionSet", log); err != nil {
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

// ParsePermissionSet is a log parse operation binding the contract event 0x9d950e4919d7c91068179be3cd6a99029ee6fdfcf4fb488053aa79841b517e03.
//
// Solidity: event PermissionSet(address indexed target, address indexed caller, bytes4 indexed selector, bool allowed)
func (_VaultController *VaultControllerFilterer) ParsePermissionSet(log types.Log) (*VaultControllerPermissionSet, error) {
	event := new(VaultControllerPermissionSet)
	if err := _VaultController.contract.UnpackLog(event, "PermissionSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerPermissionSetterSetIterator is returned from FilterPermissionSetterSet and is used to iterate over the raw logs and unpacked data for PermissionSetterSet events raised by the VaultController contract.
type VaultControllerPermissionSetterSetIterator struct {
	Event *VaultControllerPermissionSetterSet // Event containing the contract specifics and raw log

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
func (it *VaultControllerPermissionSetterSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerPermissionSetterSet)
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
		it.Event = new(VaultControllerPermissionSetterSet)
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
func (it *VaultControllerPermissionSetterSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerPermissionSetterSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerPermissionSetterSet represents a PermissionSetterSet event raised by the VaultController contract.
type VaultControllerPermissionSetterSet struct {
	Setter  common.Address
	Allowed bool
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterPermissionSetterSet is a free log retrieval operation binding the contract event 0x023ef6e50bf1aa5e0ac13acee4206b46ded01551f0247af798912f40d14baa79.
//
// Solidity: event PermissionSetterSet(address indexed setter, bool allowed)
func (_VaultController *VaultControllerFilterer) FilterPermissionSetterSet(opts *bind.FilterOpts, setter []common.Address) (*VaultControllerPermissionSetterSetIterator, error) {

	var setterRule []interface{}
	for _, setterItem := range setter {
		setterRule = append(setterRule, setterItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "PermissionSetterSet", setterRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerPermissionSetterSetIterator{contract: _VaultController.contract, event: "PermissionSetterSet", logs: logs, sub: sub}, nil
}

// WatchPermissionSetterSet is a free log subscription operation binding the contract event 0x023ef6e50bf1aa5e0ac13acee4206b46ded01551f0247af798912f40d14baa79.
//
// Solidity: event PermissionSetterSet(address indexed setter, bool allowed)
func (_VaultController *VaultControllerFilterer) WatchPermissionSetterSet(opts *bind.WatchOpts, sink chan<- *VaultControllerPermissionSetterSet, setter []common.Address) (event.Subscription, error) {

	var setterRule []interface{}
	for _, setterItem := range setter {
		setterRule = append(setterRule, setterItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "PermissionSetterSet", setterRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerPermissionSetterSet)
				if err := _VaultController.contract.UnpackLog(event, "PermissionSetterSet", log); err != nil {
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

// ParsePermissionSetterSet is a log parse operation binding the contract event 0x023ef6e50bf1aa5e0ac13acee4206b46ded01551f0247af798912f40d14baa79.
//
// Solidity: event PermissionSetterSet(address indexed setter, bool allowed)
func (_VaultController *VaultControllerFilterer) ParsePermissionSetterSet(log types.Log) (*VaultControllerPermissionSetterSet, error) {
	event := new(VaultControllerPermissionSetterSet)
	if err := _VaultController.contract.UnpackLog(event, "PermissionSetterSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerProtocolComponentSetIterator is returned from FilterProtocolComponentSet and is used to iterate over the raw logs and unpacked data for ProtocolComponentSet events raised by the VaultController contract.
type VaultControllerProtocolComponentSetIterator struct {
	Event *VaultControllerProtocolComponentSet // Event containing the contract specifics and raw log

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
func (it *VaultControllerProtocolComponentSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerProtocolComponentSet)
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
		it.Event = new(VaultControllerProtocolComponentSet)
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
func (it *VaultControllerProtocolComponentSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerProtocolComponentSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerProtocolComponentSet represents a ProtocolComponentSet event raised by the VaultController contract.
type VaultControllerProtocolComponentSet struct {
	ProtocolId  [4]byte
	ComponentId common.Hash
	Component   common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterProtocolComponentSet is a free log retrieval operation binding the contract event 0x23a80e454b9e756d9b2ebb718293c09fe024b7adf6ea29070e27dd44dded2464.
//
// Solidity: event ProtocolComponentSet(bytes4 indexed protocolId, string indexed componentId, address indexed component)
func (_VaultController *VaultControllerFilterer) FilterProtocolComponentSet(opts *bind.FilterOpts, protocolId [][4]byte, componentId []string, component []common.Address) (*VaultControllerProtocolComponentSetIterator, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}
	var componentIdRule []interface{}
	for _, componentIdItem := range componentId {
		componentIdRule = append(componentIdRule, componentIdItem)
	}
	var componentRule []interface{}
	for _, componentItem := range component {
		componentRule = append(componentRule, componentItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "ProtocolComponentSet", protocolIdRule, componentIdRule, componentRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerProtocolComponentSetIterator{contract: _VaultController.contract, event: "ProtocolComponentSet", logs: logs, sub: sub}, nil
}

// WatchProtocolComponentSet is a free log subscription operation binding the contract event 0x23a80e454b9e756d9b2ebb718293c09fe024b7adf6ea29070e27dd44dded2464.
//
// Solidity: event ProtocolComponentSet(bytes4 indexed protocolId, string indexed componentId, address indexed component)
func (_VaultController *VaultControllerFilterer) WatchProtocolComponentSet(opts *bind.WatchOpts, sink chan<- *VaultControllerProtocolComponentSet, protocolId [][4]byte, componentId []string, component []common.Address) (event.Subscription, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}
	var componentIdRule []interface{}
	for _, componentIdItem := range componentId {
		componentIdRule = append(componentIdRule, componentIdItem)
	}
	var componentRule []interface{}
	for _, componentItem := range component {
		componentRule = append(componentRule, componentItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "ProtocolComponentSet", protocolIdRule, componentIdRule, componentRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerProtocolComponentSet)
				if err := _VaultController.contract.UnpackLog(event, "ProtocolComponentSet", log); err != nil {
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

// ParseProtocolComponentSet is a log parse operation binding the contract event 0x23a80e454b9e756d9b2ebb718293c09fe024b7adf6ea29070e27dd44dded2464.
//
// Solidity: event ProtocolComponentSet(bytes4 indexed protocolId, string indexed componentId, address indexed component)
func (_VaultController *VaultControllerFilterer) ParseProtocolComponentSet(log types.Log) (*VaultControllerProtocolComponentSet, error) {
	event := new(VaultControllerProtocolComponentSet)
	if err := _VaultController.contract.UnpackLog(event, "ProtocolComponentSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerProtocolShutdownIterator is returned from FilterProtocolShutdown and is used to iterate over the raw logs and unpacked data for ProtocolShutdown events raised by the VaultController contract.
type VaultControllerProtocolShutdownIterator struct {
	Event *VaultControllerProtocolShutdown // Event containing the contract specifics and raw log

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
func (it *VaultControllerProtocolShutdownIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerProtocolShutdown)
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
		it.Event = new(VaultControllerProtocolShutdown)
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
func (it *VaultControllerProtocolShutdownIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerProtocolShutdownIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerProtocolShutdown represents a ProtocolShutdown event raised by the VaultController contract.
type VaultControllerProtocolShutdown struct {
	ProtocolId [4]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterProtocolShutdown is a free log retrieval operation binding the contract event 0xec8a49bce3751efb1292241109720b056d64fa7dd68887ba4e92f09b373d7f73.
//
// Solidity: event ProtocolShutdown(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) FilterProtocolShutdown(opts *bind.FilterOpts, protocolId [][4]byte) (*VaultControllerProtocolShutdownIterator, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "ProtocolShutdown", protocolIdRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerProtocolShutdownIterator{contract: _VaultController.contract, event: "ProtocolShutdown", logs: logs, sub: sub}, nil
}

// WatchProtocolShutdown is a free log subscription operation binding the contract event 0xec8a49bce3751efb1292241109720b056d64fa7dd68887ba4e92f09b373d7f73.
//
// Solidity: event ProtocolShutdown(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) WatchProtocolShutdown(opts *bind.WatchOpts, sink chan<- *VaultControllerProtocolShutdown, protocolId [][4]byte) (event.Subscription, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "ProtocolShutdown", protocolIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerProtocolShutdown)
				if err := _VaultController.contract.UnpackLog(event, "ProtocolShutdown", log); err != nil {
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

// ParseProtocolShutdown is a log parse operation binding the contract event 0xec8a49bce3751efb1292241109720b056d64fa7dd68887ba4e92f09b373d7f73.
//
// Solidity: event ProtocolShutdown(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) ParseProtocolShutdown(log types.Log) (*VaultControllerProtocolShutdown, error) {
	event := new(VaultControllerProtocolShutdown)
	if err := _VaultController.contract.UnpackLog(event, "ProtocolShutdown", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerRegistrarPermissionSetIterator is returned from FilterRegistrarPermissionSet and is used to iterate over the raw logs and unpacked data for RegistrarPermissionSet events raised by the VaultController contract.
type VaultControllerRegistrarPermissionSetIterator struct {
	Event *VaultControllerRegistrarPermissionSet // Event containing the contract specifics and raw log

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
func (it *VaultControllerRegistrarPermissionSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerRegistrarPermissionSet)
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
		it.Event = new(VaultControllerRegistrarPermissionSet)
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
func (it *VaultControllerRegistrarPermissionSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerRegistrarPermissionSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerRegistrarPermissionSet represents a RegistrarPermissionSet event raised by the VaultController contract.
type VaultControllerRegistrarPermissionSet struct {
	Registrar common.Address
	Allowed   bool
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRegistrarPermissionSet is a free log retrieval operation binding the contract event 0xd6a2ab6b0dbd98e539210db72e366dd068032c1abcfce8b8588bcee82052ded5.
//
// Solidity: event RegistrarPermissionSet(address indexed registrar, bool allowed)
func (_VaultController *VaultControllerFilterer) FilterRegistrarPermissionSet(opts *bind.FilterOpts, registrar []common.Address) (*VaultControllerRegistrarPermissionSetIterator, error) {

	var registrarRule []interface{}
	for _, registrarItem := range registrar {
		registrarRule = append(registrarRule, registrarItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "RegistrarPermissionSet", registrarRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerRegistrarPermissionSetIterator{contract: _VaultController.contract, event: "RegistrarPermissionSet", logs: logs, sub: sub}, nil
}

// WatchRegistrarPermissionSet is a free log subscription operation binding the contract event 0xd6a2ab6b0dbd98e539210db72e366dd068032c1abcfce8b8588bcee82052ded5.
//
// Solidity: event RegistrarPermissionSet(address indexed registrar, bool allowed)
func (_VaultController *VaultControllerFilterer) WatchRegistrarPermissionSet(opts *bind.WatchOpts, sink chan<- *VaultControllerRegistrarPermissionSet, registrar []common.Address) (event.Subscription, error) {

	var registrarRule []interface{}
	for _, registrarItem := range registrar {
		registrarRule = append(registrarRule, registrarItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "RegistrarPermissionSet", registrarRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerRegistrarPermissionSet)
				if err := _VaultController.contract.UnpackLog(event, "RegistrarPermissionSet", log); err != nil {
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

// ParseRegistrarPermissionSet is a log parse operation binding the contract event 0xd6a2ab6b0dbd98e539210db72e366dd068032c1abcfce8b8588bcee82052ded5.
//
// Solidity: event RegistrarPermissionSet(address indexed registrar, bool allowed)
func (_VaultController *VaultControllerFilterer) ParseRegistrarPermissionSet(log types.Log) (*VaultControllerRegistrarPermissionSet, error) {
	event := new(VaultControllerRegistrarPermissionSet)
	if err := _VaultController.contract.UnpackLog(event, "RegistrarPermissionSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerUnpausedIterator is returned from FilterUnpaused and is used to iterate over the raw logs and unpacked data for Unpaused events raised by the VaultController contract.
type VaultControllerUnpausedIterator struct {
	Event *VaultControllerUnpaused // Event containing the contract specifics and raw log

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
func (it *VaultControllerUnpausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerUnpaused)
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
		it.Event = new(VaultControllerUnpaused)
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
func (it *VaultControllerUnpausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerUnpausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerUnpaused represents a Unpaused event raised by the VaultController contract.
type VaultControllerUnpaused struct {
	ProtocolId [4]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterUnpaused is a free log retrieval operation binding the contract event 0xb95723779c5dae72a25db18011ec3a516b76f381329525a24e1b97605fb5fe98.
//
// Solidity: event Unpaused(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) FilterUnpaused(opts *bind.FilterOpts, protocolId [][4]byte) (*VaultControllerUnpausedIterator, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "Unpaused", protocolIdRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerUnpausedIterator{contract: _VaultController.contract, event: "Unpaused", logs: logs, sub: sub}, nil
}

// WatchUnpaused is a free log subscription operation binding the contract event 0xb95723779c5dae72a25db18011ec3a516b76f381329525a24e1b97605fb5fe98.
//
// Solidity: event Unpaused(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) WatchUnpaused(opts *bind.WatchOpts, sink chan<- *VaultControllerUnpaused, protocolId [][4]byte) (event.Subscription, error) {

	var protocolIdRule []interface{}
	for _, protocolIdItem := range protocolId {
		protocolIdRule = append(protocolIdRule, protocolIdItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "Unpaused", protocolIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerUnpaused)
				if err := _VaultController.contract.UnpackLog(event, "Unpaused", log); err != nil {
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

// ParseUnpaused is a log parse operation binding the contract event 0xb95723779c5dae72a25db18011ec3a516b76f381329525a24e1b97605fb5fe98.
//
// Solidity: event Unpaused(bytes4 indexed protocolId)
func (_VaultController *VaultControllerFilterer) ParseUnpaused(log types.Log) (*VaultControllerUnpaused, error) {
	event := new(VaultControllerUnpaused)
	if err := _VaultController.contract.UnpackLog(event, "Unpaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// VaultControllerVaultRegisteredIterator is returned from FilterVaultRegistered and is used to iterate over the raw logs and unpacked data for VaultRegistered events raised by the VaultController contract.
type VaultControllerVaultRegisteredIterator struct {
	Event *VaultControllerVaultRegistered // Event containing the contract specifics and raw log

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
func (it *VaultControllerVaultRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(VaultControllerVaultRegistered)
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
		it.Event = new(VaultControllerVaultRegistered)
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
func (it *VaultControllerVaultRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *VaultControllerVaultRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// VaultControllerVaultRegistered represents a VaultRegistered event raised by the VaultController contract.
type VaultControllerVaultRegistered struct {
	Gauge          common.Address
	Vault          common.Address
	Asset          common.Address
	RewardReceiver common.Address
	ProtocolId     [4]byte
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterVaultRegistered is a free log retrieval operation binding the contract event 0x01faf72cecd22e6d7b9a5d03dfdd583ecc5fd698872639d99a9731c240dbbcdd.
//
// Solidity: event VaultRegistered(address indexed gauge, address indexed vault, address indexed asset, address rewardReceiver, bytes4 protocolId)
func (_VaultController *VaultControllerFilterer) FilterVaultRegistered(opts *bind.FilterOpts, gauge []common.Address, vault []common.Address, asset []common.Address) (*VaultControllerVaultRegisteredIterator, error) {

	var gaugeRule []interface{}
	for _, gaugeItem := range gauge {
		gaugeRule = append(gaugeRule, gaugeItem)
	}
	var vaultRule []interface{}
	for _, vaultItem := range vault {
		vaultRule = append(vaultRule, vaultItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _VaultController.contract.FilterLogs(opts, "VaultRegistered", gaugeRule, vaultRule, assetRule)
	if err != nil {
		return nil, err
	}
	return &VaultControllerVaultRegisteredIterator{contract: _VaultController.contract, event: "VaultRegistered", logs: logs, sub: sub}, nil
}

// WatchVaultRegistered is a free log subscription operation binding the contract event 0x01faf72cecd22e6d7b9a5d03dfdd583ecc5fd698872639d99a9731c240dbbcdd.
//
// Solidity: event VaultRegistered(address indexed gauge, address indexed vault, address indexed asset, address rewardReceiver, bytes4 protocolId)
func (_VaultController *VaultControllerFilterer) WatchVaultRegistered(opts *bind.WatchOpts, sink chan<- *VaultControllerVaultRegistered, gauge []common.Address, vault []common.Address, asset []common.Address) (event.Subscription, error) {

	var gaugeRule []interface{}
	for _, gaugeItem := range gauge {
		gaugeRule = append(gaugeRule, gaugeItem)
	}
	var vaultRule []interface{}
	for _, vaultItem := range vault {
		vaultRule = append(vaultRule, vaultItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _VaultController.contract.WatchLogs(opts, "VaultRegistered", gaugeRule, vaultRule, assetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(VaultControllerVaultRegistered)
				if err := _VaultController.contract.UnpackLog(event, "VaultRegistered", log); err != nil {
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

// ParseVaultRegistered is a log parse operation binding the contract event 0x01faf72cecd22e6d7b9a5d03dfdd583ecc5fd698872639d99a9731c240dbbcdd.
//
// Solidity: event VaultRegistered(address indexed gauge, address indexed vault, address indexed asset, address rewardReceiver, bytes4 protocolId)
func (_VaultController *VaultControllerFilterer) ParseVaultRegistered(log types.Log) (*VaultControllerVaultRegistered, error) {
	event := new(VaultControllerVaultRegistered)
	if err := _VaultController.contract.UnpackLog(event, "VaultRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
