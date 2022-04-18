// Copyright 2017 The Celo Authors
// This file is part of the celo library.
//
// The celo library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The celo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the celo library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	cgExchangeRateNum = big.NewInt(1)
	cgExchangeRateDen = big.NewInt(1)

	// selector is first 4 bytes of keccak256 of "getExchangeRate(address,address)"
	// Source:
	// pip3 install pyethereum
	// python3 -c 'from ethereum.utils import sha3; print(sha3("getExchangeRate(address,address)")[0:4].hex())'
	getExchangeRateFuncABI = hexutil.MustDecode("0xbaaa61be")

	// selector is first 4 bytes of keccak256 of "balanceOf(address)"
	// Source:
	// pip3 install pyethereum
	// python3 -c 'from ethereum.utils import sha3; print(sha3("balanceOf(address)")[0:4].hex())'
	getBalanceFuncABI = hexutil.MustDecode("0x70a08231")

	// selector is first 4 bytes of keccak256 of "getWhitelist()"
	// Source:
	// pip3 install pyethereum
	// python3 -c 'from ethereum.utils import sha3; print(sha3("getWhitelist()")[0:4].hex())'
	getWhiteListFuncABI = hexutil.MustDecode("0xd01f63f5")

	getWhiteListFuncReturnABI, _ = abi.JSON(strings.NewReader(`[{ "name" : "addressSliceSingle", "constant" : false, "outputs": [ { "type" : "address[]" } ] }]`))

	errExchangeRateCacheMiss = errors.New("exchange rate cache miss")
)

type exchangeRate struct {
	Numerator   *big.Int
	Denominator *big.Int
}

type PriceComparator struct {
	gasCurrencyAddresses *[]common.Address                // The set of currencies that will have their exchange rate monitored
	exchangeRates        map[common.Address]*exchangeRate // indexedCurrency:CeloGold exchange rate
	blockchain           *BlockChain                      // Used to construct the EVM object needed to make the call the medianator contract
	chainConfig          *params.ChainConfig              // The config object of the eth object
}

func (pc *PriceComparator) getExchangeRate(currency *common.Address) (*big.Int, *big.Int, error) {
	if currency == nil {
		return cgExchangeRateNum, cgExchangeRateDen, nil
	} else {
		if exchangeRate, ok := pc.exchangeRates[*currency]; !ok {
			return nil, nil, errExchangeRateCacheMiss
		} else {
			return exchangeRate.Numerator, exchangeRate.Denominator, nil
		}
	}
}

func (pc *PriceComparator) Cmp(val1 *big.Int, currency1 *common.Address, val2 *big.Int, currency2 *common.Address) int {
	if currency1 == currency2 {
		return val1.Cmp(val2)
	}

	exchangeRate1Num, exchangeRate1Den, err1 := pc.getExchangeRate(currency1)
	exchangeRate2Num, exchangeRate2Den, err2 := pc.getExchangeRate(currency2)

	if err1 != nil || err2 != nil {
		currency1Output := "nil"
		if currency1 != nil {
			currency1Output = currency1.Hex()
		}
		currency2Output := "nil"
		if currency2 != nil {
			currency2Output = currency2.Hex()
		}
		log.Warn("Error in retrieving cached exchange rate.  Will do comparison of two values without exchange rate conversion.", "currency1", currency1Output, "err1", err1, "currency2", currency2Output, "err2", err2)
		return val1.Cmp(val2)
	}

	// Below code block is basically evaluating this comparison:
	// val1 * exchangeRate1Num/exchangeRate1Den < val2 * exchangeRate2Num/exchangeRate2Den
	// It will transform that comparison to this, to remove having to deal with fractional values.
	// val1 * exchangeRate1Num * exchangeRate2Den < val2 * exchangeRate2Num * exchangeRate1Den
	leftSide := new(big.Int).Mul(val1, new(big.Int).Mul(exchangeRate1Num, exchangeRate2Den))
	rightSide := new(big.Int).Mul(val2, new(big.Int).Mul(exchangeRate2Num, exchangeRate1Den))
	return leftSide.Cmp(rightSide)
}

// This function will retrieve the exchange rates from the Medianator contract and cache them.
// Medianator must have a function with the following signature:
// "function getExchangeRate(address, address)"
func (pc *PriceComparator) retrieveExchangeRates() {
	log.Trace("retrieveExchangeRates",
		"gasCurrencyAddresses", fmt.Sprintf("%v", *pc.gasCurrencyAddresses))

	header := pc.blockchain.CurrentBlock().Header()
	state, err := pc.blockchain.StateAt(header.Root)
	if err != nil {
		log.Error("PriceComparator.retrieveExchangeRates - Error in retrieving the state from the blockchain")
		return
	}

	// The EVM Context requires a msg, but the actual field values don't really matter.  Putting in
	// zero values.
	msg := types.NewMessage(common.HexToAddress("0x0"), nil, 0, common.Big0, 0, common.Big0, nil, []byte{}, false)
	context := NewEVMContext(msg, header, pc.blockchain, nil)
	evm := vm.NewEVM(context, state, pc.chainConfig, *pc.blockchain.GetVMConfig())

	for _, gasCurrencyAddress := range *pc.gasCurrencyAddresses {
		transactionData := common.GetEncodedAbi(
			getExchangeRateFuncABI, [][]byte{common.AddressToAbi(params.CeloGoldAddress), common.AddressToAbi(gasCurrencyAddress)})
		anyCaller := vm.AccountRef(common.HexToAddress("0x0")) // any caller will work

		// Some reasonable gas limit to avoid a potentially bad Oracle from running expensive computations.
		gas := uint64(20 * 1000)
		log.Trace("PriceComparator.retrieveExchangeRates - Calling getExchangeRate", "caller", anyCaller, "customTokenContractAddress",
			params.MedianatorAddress, "gas", gas, "transactionData", hexutil.Encode(transactionData))

		ret, leftoverGas, err := evm.StaticCall(anyCaller, params.MedianatorAddress, transactionData, gas)

		if err != nil {
			log.Error("PriceComparator.retrieveExchangeRates - Error in retrieving exchange rate",
				"base", params.CeloGoldAddress, "counter", gasCurrencyAddress, "err", err)
			continue
		}

		if len(ret) != 2*32 {
			log.Error("PriceComparator.retrieveExchangeRates - Unexpected return value in retrieving exchange rate",
				"base", params.CeloGoldAddress, "counter", gasCurrencyAddress, "ret", hexutil.Encode(ret))
			continue
		}

		log.Trace("getExchangeRate", "ret", ret, "leftoverGas", leftoverGas, "err", err)
		baseAmount := new(big.Int).SetBytes(ret[0:32])
		counterAmount := new(big.Int).SetBytes(ret[32:64])
		log.Trace("getExchangeRate", "baseAmount", baseAmount, "counterAmount", counterAmount)

		if _, ok := pc.exchangeRates[gasCurrencyAddress]; !ok {
			pc.exchangeRates[gasCurrencyAddress] = &exchangeRate{}
		}

		pc.exchangeRates[gasCurrencyAddress].Numerator = baseAmount
		pc.exchangeRates[gasCurrencyAddress].Denominator = counterAmount
	}
}

func (pc *PriceComparator) mainLoop() {
	pc.retrieveExchangeRates()
	ticker := time.NewTicker(10 * time.Second)

	for range ticker.C {
		pc.retrieveExchangeRates()
	}
}

func NewPriceComparator(gasCurrencyAddresses *[]common.Address, chainConfig *params.ChainConfig, blockchain *BlockChain) *PriceComparator {
	exchangeRates := make(map[common.Address]*exchangeRate)

	pc := &PriceComparator{
		gasCurrencyAddresses: gasCurrencyAddresses,
		exchangeRates:        exchangeRates,
		blockchain:           blockchain,
		chainConfig:          chainConfig,
	}

	if pc.gasCurrencyAddresses != nil && len(*pc.gasCurrencyAddresses) > 0 {
		go pc.mainLoop()
	}

	return pc
}

// This function will retrieve the balance of an ERC20 token.  Specifically, the contract must have the
// following function.
// "function balanceOf(address _owner) public view returns (uint256)"
func GetBalanceOf(accountOwner common.Address, contractAddress *common.Address, evm *vm.EVM, gas uint64) (
	balance *big.Int, gasUsed uint64, err error) {

	transactionData := common.GetEncodedAbi(getBalanceFuncABI, [][]byte{common.AddressToAbi(accountOwner)})
	anyCaller := vm.AccountRef(common.HexToAddress("0x0")) // any caller will work
	log.Trace("getBalanceOf", "caller", anyCaller, "customTokenContractAddress",
		*contractAddress, "gas", gas, "transactionData", hexutil.Encode(transactionData))
	ret, leftoverGas, err := evm.StaticCall(anyCaller, *contractAddress, transactionData, gas)
	gasUsed = gas - leftoverGas
	if err != nil {
		log.Debug("getBalanceOf error occurred", "Error", err)
		return nil, gasUsed, err
	}
	result := big.NewInt(0)
	result.SetBytes(ret)
	log.Trace("getBalanceOf balance", "account", accountOwner.Hash(), "Balance", result.String(),
		"gas used", gasUsed)
	return result, gasUsed, nil
}

type GasCurrencyWhitelist struct {
	whitelistedAddresses   map[common.Address]bool
	whitelistedAddressesMu sync.RWMutex
	blockchain             *BlockChain         // Used to construct the EVM object needed to make the call the medianator contract
	chainConfig            *params.ChainConfig // The config object of the eth object
}

func (gcWl *GasCurrencyWhitelist) retrieveWhitelist() []common.Address {
	log.Trace("GasCurrencyWhitelist.retrieveWhitelist")

	returnList := []common.Address{}

	if gcWl.blockchain == nil {
		log.Warn("GasCurrencyWhitelist.retrieveWhitelist - gcWl.blockchain is nil, returning empty whitelist")
		return returnList
	}

	header := gcWl.blockchain.CurrentBlock().Header()
	state, err := gcWl.blockchain.StateAt(header.Root)
	if err != nil {
		log.Error("GasCurrencyWhitelist.retrieveWhitelist - Error in retrieving the state from the blockchain")

		// If we can't retrieve the whitelist, be conservative and assume no currencies are whitelisted
		return returnList
	}

	// The EVM Context requires a msg, but the actual field values don't really matter.  Putting in
	// zero values.
	msg := types.NewMessage(common.HexToAddress("0x0"), nil, 0, common.Big0, 0, common.Big0, nil, []byte{}, false)
	context := NewEVMContext(msg, header, gcWl.blockchain, nil)
	evm := vm.NewEVM(context, state, gcWl.chainConfig, *gcWl.blockchain.GetVMConfig())

	anyCaller := vm.AccountRef(common.HexToAddress("0x0")) // any caller will work
	transactionData := common.GetEncodedAbi(getWhiteListFuncABI, [][]byte{})
	gas := uint64(20 * 1000)
	log.Trace("GasCurrencyWhiteList.retrieveWhiteList() - Calling retrieveWhiteList", "caller", anyCaller, "GasCurrencyWhiteList",
		params.GasCurrencyWhitelistAddress, "gas", gas, "transactionData", hexutil.Encode(transactionData))

	ret, leftoverGas, err := evm.StaticCall(anyCaller, params.GasCurrencyWhitelistAddress, transactionData, gas)

	if err != nil {
		log.Error("Error in retrieving the gas currency whitelist", "err", err)
		return returnList
	}

	log.Trace("retrieveWhitelist", "ret", ret, "leftoverGas", leftoverGas)

	if err := getWhiteListFuncReturnABI.Unpack(&returnList, "addressSliceSingle", ret); err != nil {
		log.Trace("Error in unpacking gas currency whitelist", "err", err)
		return returnList
	}

	outputWhiteList := make([]string, len(returnList))
	for _, address := range returnList {
		outputWhiteList = append(outputWhiteList, address.Hex())
	}
	log.Trace("retrieveWhitelist", "whitelist", outputWhiteList)
	return returnList
}

func (gcWl *GasCurrencyWhitelist) RefreshWhitelist() {
	addresses := gcWl.retrieveWhitelist()

	gcWl.whitelistedAddressesMu.Lock()

	for k := range gcWl.whitelistedAddresses {
		delete(gcWl.whitelistedAddresses, k)
	}

	for _, address := range addresses {
		gcWl.whitelistedAddresses[address] = true
	}

	gcWl.whitelistedAddressesMu.Unlock()
}

func (gcWl *GasCurrencyWhitelist) IsWhitelisted(gasCurrencyAddress common.Address) bool {
	gcWl.whitelistedAddressesMu.RLock()

	_, ok := gcWl.whitelistedAddresses[gasCurrencyAddress]

	gcWl.whitelistedAddressesMu.RUnlock()

	return ok
}

func NewGasCurrencyWhitelist(chainConfig *params.ChainConfig, blockchain *BlockChain) *GasCurrencyWhitelist {
	gcWl := &GasCurrencyWhitelist{
		whitelistedAddresses: make(map[common.Address]bool),
		blockchain:           blockchain,
		chainConfig:          chainConfig,
	}

	return gcWl
}
