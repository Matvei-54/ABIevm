// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package aave

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

// DataTypesEModeCategory is an auto generated low-level Go binding around an user-defined struct.
type DataTypesEModeCategory struct {
	Ltv                  uint16
	LiquidationThreshold uint16
	LiquidationBonus     uint16
	CollateralBitmap     *big.Int
	Label                string
	BorrowableBitmap     *big.Int
}

// IUiPoolDataProviderV3AggregatedReserveData is an auto generated low-level Go binding around an user-defined struct.
type IUiPoolDataProviderV3AggregatedReserveData struct {
	UnderlyingAsset                common.Address
	Name                           string
	Symbol                         string
	Decimals                       *big.Int
	BaseLTVasCollateral            *big.Int
	ReserveLiquidationThreshold    *big.Int
	ReserveLiquidationBonus        *big.Int
	ReserveFactor                  *big.Int
	UsageAsCollateralEnabled       bool
	BorrowingEnabled               bool
	IsActive                       bool
	IsFrozen                       bool
	LiquidityIndex                 *big.Int
	VariableBorrowIndex            *big.Int
	LiquidityRate                  *big.Int
	VariableBorrowRate             *big.Int
	LastUpdateTimestamp            *big.Int
	ATokenAddress                  common.Address
	VariableDebtTokenAddress       common.Address
	InterestRateStrategyAddress    common.Address
	AvailableLiquidity             *big.Int
	TotalScaledVariableDebt        *big.Int
	PriceInMarketReferenceCurrency *big.Int
	PriceOracle                    common.Address
	VariableRateSlope1             *big.Int
	VariableRateSlope2             *big.Int
	BaseVariableBorrowRate         *big.Int
	OptimalUsageRatio              *big.Int
	IsPaused                       bool
	IsSiloedBorrowing              bool
	AccruedToTreasury              *big.Int
	Unbacked                       *big.Int
	IsolationModeTotalDebt         *big.Int
	FlashLoanEnabled               bool
	DebtCeiling                    *big.Int
	DebtCeilingDecimals            *big.Int
	BorrowCap                      *big.Int
	SupplyCap                      *big.Int
	BorrowableInIsolation          bool
	VirtualAccActive               bool
	VirtualUnderlyingBalance       *big.Int
}

// IUiPoolDataProviderV3BaseCurrencyInfo is an auto generated low-level Go binding around an user-defined struct.
type IUiPoolDataProviderV3BaseCurrencyInfo struct {
	MarketReferenceCurrencyUnit       *big.Int
	MarketReferenceCurrencyPriceInUsd *big.Int
	NetworkBaseTokenPriceInUsd        *big.Int
	NetworkBaseTokenPriceDecimals     uint8
}

// IUiPoolDataProviderV3Emode is an auto generated low-level Go binding around an user-defined struct.
type IUiPoolDataProviderV3Emode struct {
	Id    uint8
	EMode DataTypesEModeCategory
}

// IUiPoolDataProviderV3UserReserveData is an auto generated low-level Go binding around an user-defined struct.
type IUiPoolDataProviderV3UserReserveData struct {
	UnderlyingAsset                common.Address
	ScaledATokenBalance            *big.Int
	UsageAsCollateralEnabledOnUser bool
	ScaledVariableDebt             *big.Int
}

// UipoolMetaData contains all meta data concerning the Uipool contract.
var UipoolMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIEACAggregatorProxy\",\"name\":\"_networkBaseTokenPriceInUsdProxyAggregator\",\"type\":\"address\"},{\"internalType\":\"contractIEACAggregatorProxy\",\"name\":\"_marketReferenceCurrencyPriceInUsdProxyAggregator\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"ETH_CURRENCY_UNIT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MKR_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_bytes32\",\"type\":\"bytes32\"}],\"name\":\"bytes32ToString\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIPoolAddressesProvider\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"getEModes\",\"outputs\":[{\"components\":[{\"internalType\":\"uint8\",\"name\":\"id\",\"type\":\"uint8\"},{\"components\":[{\"internalType\":\"uint16\",\"name\":\"ltv\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"liquidationThreshold\",\"type\":\"uint16\"},{\"internalType\":\"uint16\",\"name\":\"liquidationBonus\",\"type\":\"uint16\"},{\"internalType\":\"uint128\",\"name\":\"collateralBitmap\",\"type\":\"uint128\"},{\"internalType\":\"string\",\"name\":\"label\",\"type\":\"string\"},{\"internalType\":\"uint128\",\"name\":\"borrowableBitmap\",\"type\":\"uint128\"}],\"internalType\":\"structDataTypes.EModeCategory\",\"name\":\"eMode\",\"type\":\"tuple\"}],\"internalType\":\"structIUiPoolDataProviderV3.Emode[]\",\"name\":\"\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIPoolAddressesProvider\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"getReservesData\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"underlyingAsset\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"decimals\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseLTVasCollateral\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserveLiquidationThreshold\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserveLiquidationBonus\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"reserveFactor\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"usageAsCollateralEnabled\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"borrowingEnabled\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isActive\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isFrozen\",\"type\":\"bool\"},{\"internalType\":\"uint128\",\"name\":\"liquidityIndex\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"variableBorrowIndex\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"liquidityRate\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"variableBorrowRate\",\"type\":\"uint128\"},{\"internalType\":\"uint40\",\"name\":\"lastUpdateTimestamp\",\"type\":\"uint40\"},{\"internalType\":\"address\",\"name\":\"aTokenAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"variableDebtTokenAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"interestRateStrategyAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"availableLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalScaledVariableDebt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"priceInMarketReferenceCurrency\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"priceOracle\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"variableRateSlope1\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"variableRateSlope2\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseVariableBorrowRate\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"optimalUsageRatio\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isPaused\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isSiloedBorrowing\",\"type\":\"bool\"},{\"internalType\":\"uint128\",\"name\":\"accruedToTreasury\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"unbacked\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"isolationModeTotalDebt\",\"type\":\"uint128\"},{\"internalType\":\"bool\",\"name\":\"flashLoanEnabled\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"debtCeiling\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"debtCeilingDecimals\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"borrowCap\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"supplyCap\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"borrowableInIsolation\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"virtualAccActive\",\"type\":\"bool\"},{\"internalType\":\"uint128\",\"name\":\"virtualUnderlyingBalance\",\"type\":\"uint128\"}],\"internalType\":\"structIUiPoolDataProviderV3.AggregatedReserveData[]\",\"name\":\"\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"marketReferenceCurrencyUnit\",\"type\":\"uint256\"},{\"internalType\":\"int256\",\"name\":\"marketReferenceCurrencyPriceInUsd\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"networkBaseTokenPriceInUsd\",\"type\":\"int256\"},{\"internalType\":\"uint8\",\"name\":\"networkBaseTokenPriceDecimals\",\"type\":\"uint8\"}],\"internalType\":\"structIUiPoolDataProviderV3.BaseCurrencyInfo\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIPoolAddressesProvider\",\"name\":\"provider\",\"type\":\"address\"}],\"name\":\"getReservesList\",\"outputs\":[{\"internalType\":\"address[]\",\"name\":\"\",\"type\":\"address[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIPoolAddressesProvider\",\"name\":\"provider\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"user\",\"type\":\"address\"}],\"name\":\"getUserReservesData\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"underlyingAsset\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"scaledATokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"usageAsCollateralEnabledOnUser\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"scaledVariableDebt\",\"type\":\"uint256\"}],\"internalType\":\"structIUiPoolDataProviderV3.UserReserveData[]\",\"name\":\"\",\"type\":\"tuple[]\"},{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"marketReferenceCurrencyPriceInUsdProxyAggregator\",\"outputs\":[{\"internalType\":\"contractIEACAggregatorProxy\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"networkBaseTokenPriceInUsdProxyAggregator\",\"outputs\":[{\"internalType\":\"contractIEACAggregatorProxy\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
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

// ETHCURRENCYUNIT is a free data retrieval call binding the contract method 0x0496f53a.
//
// Solidity: function ETH_CURRENCY_UNIT() view returns(uint256)
func (_Uipool *UipoolCaller) ETHCURRENCYUNIT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "ETH_CURRENCY_UNIT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ETHCURRENCYUNIT is a free data retrieval call binding the contract method 0x0496f53a.
//
// Solidity: function ETH_CURRENCY_UNIT() view returns(uint256)
func (_Uipool *UipoolSession) ETHCURRENCYUNIT() (*big.Int, error) {
	return _Uipool.Contract.ETHCURRENCYUNIT(&_Uipool.CallOpts)
}

// ETHCURRENCYUNIT is a free data retrieval call binding the contract method 0x0496f53a.
//
// Solidity: function ETH_CURRENCY_UNIT() view returns(uint256)
func (_Uipool *UipoolCallerSession) ETHCURRENCYUNIT() (*big.Int, error) {
	return _Uipool.Contract.ETHCURRENCYUNIT(&_Uipool.CallOpts)
}

// MKRADDRESS is a free data retrieval call binding the contract method 0x825ffd92.
//
// Solidity: function MKR_ADDRESS() view returns(address)
func (_Uipool *UipoolCaller) MKRADDRESS(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "MKR_ADDRESS")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MKRADDRESS is a free data retrieval call binding the contract method 0x825ffd92.
//
// Solidity: function MKR_ADDRESS() view returns(address)
func (_Uipool *UipoolSession) MKRADDRESS() (common.Address, error) {
	return _Uipool.Contract.MKRADDRESS(&_Uipool.CallOpts)
}

// MKRADDRESS is a free data retrieval call binding the contract method 0x825ffd92.
//
// Solidity: function MKR_ADDRESS() view returns(address)
func (_Uipool *UipoolCallerSession) MKRADDRESS() (common.Address, error) {
	return _Uipool.Contract.MKRADDRESS(&_Uipool.CallOpts)
}

// Bytes32ToString is a free data retrieval call binding the contract method 0x9201de55.
//
// Solidity: function bytes32ToString(bytes32 _bytes32) pure returns(string)
func (_Uipool *UipoolCaller) Bytes32ToString(opts *bind.CallOpts, _bytes32 [32]byte) (string, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "bytes32ToString", _bytes32)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Bytes32ToString is a free data retrieval call binding the contract method 0x9201de55.
//
// Solidity: function bytes32ToString(bytes32 _bytes32) pure returns(string)
func (_Uipool *UipoolSession) Bytes32ToString(_bytes32 [32]byte) (string, error) {
	return _Uipool.Contract.Bytes32ToString(&_Uipool.CallOpts, _bytes32)
}

// Bytes32ToString is a free data retrieval call binding the contract method 0x9201de55.
//
// Solidity: function bytes32ToString(bytes32 _bytes32) pure returns(string)
func (_Uipool *UipoolCallerSession) Bytes32ToString(_bytes32 [32]byte) (string, error) {
	return _Uipool.Contract.Bytes32ToString(&_Uipool.CallOpts, _bytes32)
}

// GetEModes is a free data retrieval call binding the contract method 0x6f90b9d1.
//
// Solidity: function getEModes(address provider) view returns((uint8,(uint16,uint16,uint16,uint128,string,uint128))[])
func (_Uipool *UipoolCaller) GetEModes(opts *bind.CallOpts, provider common.Address) ([]IUiPoolDataProviderV3Emode, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "getEModes", provider)

	if err != nil {
		return *new([]IUiPoolDataProviderV3Emode), err
	}

	out0 := *abi.ConvertType(out[0], new([]IUiPoolDataProviderV3Emode)).(*[]IUiPoolDataProviderV3Emode)

	return out0, err

}

// GetEModes is a free data retrieval call binding the contract method 0x6f90b9d1.
//
// Solidity: function getEModes(address provider) view returns((uint8,(uint16,uint16,uint16,uint128,string,uint128))[])
func (_Uipool *UipoolSession) GetEModes(provider common.Address) ([]IUiPoolDataProviderV3Emode, error) {
	return _Uipool.Contract.GetEModes(&_Uipool.CallOpts, provider)
}

// GetEModes is a free data retrieval call binding the contract method 0x6f90b9d1.
//
// Solidity: function getEModes(address provider) view returns((uint8,(uint16,uint16,uint16,uint128,string,uint128))[])
func (_Uipool *UipoolCallerSession) GetEModes(provider common.Address) ([]IUiPoolDataProviderV3Emode, error) {
	return _Uipool.Contract.GetEModes(&_Uipool.CallOpts, provider)
}

// GetReservesData is a free data retrieval call binding the contract method 0xec489c21.
//
// Solidity: function getReservesData(address provider) view returns((address,string,string,uint256,uint256,uint256,uint256,uint256,bool,bool,bool,bool,uint128,uint128,uint128,uint128,uint40,address,address,address,uint256,uint256,uint256,address,uint256,uint256,uint256,uint256,bool,bool,uint128,uint128,uint128,bool,uint256,uint256,uint256,uint256,bool,bool,uint128)[], (uint256,int256,int256,uint8))
func (_Uipool *UipoolCaller) GetReservesData(opts *bind.CallOpts, provider common.Address) ([]IUiPoolDataProviderV3AggregatedReserveData, IUiPoolDataProviderV3BaseCurrencyInfo, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "getReservesData", provider)

	if err != nil {
		return *new([]IUiPoolDataProviderV3AggregatedReserveData), *new(IUiPoolDataProviderV3BaseCurrencyInfo), err
	}

	out0 := *abi.ConvertType(out[0], new([]IUiPoolDataProviderV3AggregatedReserveData)).(*[]IUiPoolDataProviderV3AggregatedReserveData)
	out1 := *abi.ConvertType(out[1], new(IUiPoolDataProviderV3BaseCurrencyInfo)).(*IUiPoolDataProviderV3BaseCurrencyInfo)

	return out0, out1, err

}

// GetReservesData is a free data retrieval call binding the contract method 0xec489c21.
//
// Solidity: function getReservesData(address provider) view returns((address,string,string,uint256,uint256,uint256,uint256,uint256,bool,bool,bool,bool,uint128,uint128,uint128,uint128,uint40,address,address,address,uint256,uint256,uint256,address,uint256,uint256,uint256,uint256,bool,bool,uint128,uint128,uint128,bool,uint256,uint256,uint256,uint256,bool,bool,uint128)[], (uint256,int256,int256,uint8))
func (_Uipool *UipoolSession) GetReservesData(provider common.Address) ([]IUiPoolDataProviderV3AggregatedReserveData, IUiPoolDataProviderV3BaseCurrencyInfo, error) {
	return _Uipool.Contract.GetReservesData(&_Uipool.CallOpts, provider)
}

// GetReservesData is a free data retrieval call binding the contract method 0xec489c21.
//
// Solidity: function getReservesData(address provider) view returns((address,string,string,uint256,uint256,uint256,uint256,uint256,bool,bool,bool,bool,uint128,uint128,uint128,uint128,uint40,address,address,address,uint256,uint256,uint256,address,uint256,uint256,uint256,uint256,bool,bool,uint128,uint128,uint128,bool,uint256,uint256,uint256,uint256,bool,bool,uint128)[], (uint256,int256,int256,uint8))
func (_Uipool *UipoolCallerSession) GetReservesData(provider common.Address) ([]IUiPoolDataProviderV3AggregatedReserveData, IUiPoolDataProviderV3BaseCurrencyInfo, error) {
	return _Uipool.Contract.GetReservesData(&_Uipool.CallOpts, provider)
}

// GetReservesList is a free data retrieval call binding the contract method 0x586c1442.
//
// Solidity: function getReservesList(address provider) view returns(address[])
func (_Uipool *UipoolCaller) GetReservesList(opts *bind.CallOpts, provider common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "getReservesList", provider)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetReservesList is a free data retrieval call binding the contract method 0x586c1442.
//
// Solidity: function getReservesList(address provider) view returns(address[])
func (_Uipool *UipoolSession) GetReservesList(provider common.Address) ([]common.Address, error) {
	return _Uipool.Contract.GetReservesList(&_Uipool.CallOpts, provider)
}

// GetReservesList is a free data retrieval call binding the contract method 0x586c1442.
//
// Solidity: function getReservesList(address provider) view returns(address[])
func (_Uipool *UipoolCallerSession) GetReservesList(provider common.Address) ([]common.Address, error) {
	return _Uipool.Contract.GetReservesList(&_Uipool.CallOpts, provider)
}

// GetUserReservesData is a free data retrieval call binding the contract method 0x51974cc0.
//
// Solidity: function getUserReservesData(address provider, address user) view returns((address,uint256,bool,uint256)[], uint8)
func (_Uipool *UipoolCaller) GetUserReservesData(opts *bind.CallOpts, provider common.Address, user common.Address) ([]IUiPoolDataProviderV3UserReserveData, uint8, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "getUserReservesData", provider, user)

	if err != nil {
		return *new([]IUiPoolDataProviderV3UserReserveData), *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new([]IUiPoolDataProviderV3UserReserveData)).(*[]IUiPoolDataProviderV3UserReserveData)
	out1 := *abi.ConvertType(out[1], new(uint8)).(*uint8)

	return out0, out1, err

}

// GetUserReservesData is a free data retrieval call binding the contract method 0x51974cc0.
//
// Solidity: function getUserReservesData(address provider, address user) view returns((address,uint256,bool,uint256)[], uint8)
func (_Uipool *UipoolSession) GetUserReservesData(provider common.Address, user common.Address) ([]IUiPoolDataProviderV3UserReserveData, uint8, error) {
	return _Uipool.Contract.GetUserReservesData(&_Uipool.CallOpts, provider, user)
}

// GetUserReservesData is a free data retrieval call binding the contract method 0x51974cc0.
//
// Solidity: function getUserReservesData(address provider, address user) view returns((address,uint256,bool,uint256)[], uint8)
func (_Uipool *UipoolCallerSession) GetUserReservesData(provider common.Address, user common.Address) ([]IUiPoolDataProviderV3UserReserveData, uint8, error) {
	return _Uipool.Contract.GetUserReservesData(&_Uipool.CallOpts, provider, user)
}

// MarketReferenceCurrencyPriceInUsdProxyAggregator is a free data retrieval call binding the contract method 0xd22cf68a.
//
// Solidity: function marketReferenceCurrencyPriceInUsdProxyAggregator() view returns(address)
func (_Uipool *UipoolCaller) MarketReferenceCurrencyPriceInUsdProxyAggregator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "marketReferenceCurrencyPriceInUsdProxyAggregator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// MarketReferenceCurrencyPriceInUsdProxyAggregator is a free data retrieval call binding the contract method 0xd22cf68a.
//
// Solidity: function marketReferenceCurrencyPriceInUsdProxyAggregator() view returns(address)
func (_Uipool *UipoolSession) MarketReferenceCurrencyPriceInUsdProxyAggregator() (common.Address, error) {
	return _Uipool.Contract.MarketReferenceCurrencyPriceInUsdProxyAggregator(&_Uipool.CallOpts)
}

// MarketReferenceCurrencyPriceInUsdProxyAggregator is a free data retrieval call binding the contract method 0xd22cf68a.
//
// Solidity: function marketReferenceCurrencyPriceInUsdProxyAggregator() view returns(address)
func (_Uipool *UipoolCallerSession) MarketReferenceCurrencyPriceInUsdProxyAggregator() (common.Address, error) {
	return _Uipool.Contract.MarketReferenceCurrencyPriceInUsdProxyAggregator(&_Uipool.CallOpts)
}

// NetworkBaseTokenPriceInUsdProxyAggregator is a free data retrieval call binding the contract method 0x3c1740ed.
//
// Solidity: function networkBaseTokenPriceInUsdProxyAggregator() view returns(address)
func (_Uipool *UipoolCaller) NetworkBaseTokenPriceInUsdProxyAggregator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Uipool.contract.Call(opts, &out, "networkBaseTokenPriceInUsdProxyAggregator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// NetworkBaseTokenPriceInUsdProxyAggregator is a free data retrieval call binding the contract method 0x3c1740ed.
//
// Solidity: function networkBaseTokenPriceInUsdProxyAggregator() view returns(address)
func (_Uipool *UipoolSession) NetworkBaseTokenPriceInUsdProxyAggregator() (common.Address, error) {
	return _Uipool.Contract.NetworkBaseTokenPriceInUsdProxyAggregator(&_Uipool.CallOpts)
}

// NetworkBaseTokenPriceInUsdProxyAggregator is a free data retrieval call binding the contract method 0x3c1740ed.
//
// Solidity: function networkBaseTokenPriceInUsdProxyAggregator() view returns(address)
func (_Uipool *UipoolCallerSession) NetworkBaseTokenPriceInUsdProxyAggregator() (common.Address, error) {
	return _Uipool.Contract.NetworkBaseTokenPriceInUsdProxyAggregator(&_Uipool.CallOpts)
}
