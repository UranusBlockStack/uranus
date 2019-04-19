// Copyright 2018 The uranus Authors
// This file is part of the uranus library.
//
// The uranus library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The uranus library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the uranus library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"math"
	"math/big"

	database "github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/spf13/cobra"
)

var (
	contractHexAddr string
	input           string
	methodName      string
)

func init() {
	callCmd.Flags().StringVarP(&input, "input", "i", "", "call function input")
	callCmd.Flags().StringVarP(&methodName, "method", "m", "", "call function method name")
	callCmd.Flags().StringVarP(&contractHexAddr, "contractAddr", "c", "", "the contract address")
	callCmd.Flags().StringVarP(&account, "account", "a", "", "invoking the address of calling the smart contract(Default is random and has 1 seele)")
	rootCmd.AddCommand(callCmd)
}

var callCmd = &cobra.Command{
	Use:   "call",
	Short: "call a contract",
	Long:  `All contract could callable. This is Seele contract simulator's`,
	Run: func(cmd *cobra.Command, args []string) {
		callContract(contractHexAddr)
	},
}

func callContract(contractHexAddr string) {
	db, statedb, bcStore, dispose, err := preprocessContract()
	if err != nil {
		fmt.Println("failed to prepare the simulator environment,", err.Error())
		return
	}
	defer dispose()

	// Get the invoking address
	from := getFromAddress(statedb)
	if from.IsEmpty() {
		return
	}

	// Contract address
	contractAddr := getContractAddress(db)
	if contractAddr.IsEmpty() {
		return
	}

	// Input message to call contract
	input := getContractInputMsg(db, contractAddr.Bytes())
	if len(input) == 0 {
		return
	}

	// Call method and input parameters
	msg, err := utils.HexToBytes(input)
	if err != nil {
		fmt.Println("Invalid input message,", err.Error())
		return
	}

	// Create a call message transaction
	callContractTx, err := types.NewMessageTransaction(from, contractAddr, big.NewInt(0), big.NewInt(1), math.MaxUint64, DefaultNonce, msg)
	if err != nil {
		fmt.Println("failed to create message tx,", err.Error())
		return
	}

	receipt, err := processContract(statedb, bcStore, callContractTx)
	if err != nil {
		fmt.Println("failed to call contract,", err.Error())
		return
	}

	// Print the result
	fmt.Println()
	fmt.Println("contract called successfully")

	if len(receipt.Result) > 0 {
		fmt.Println("Result (raw):", receipt.Result)
		fmt.Println("Result (hex):", utils.BytesToHex(receipt.Result))
		fmt.Println("Result (big):", new(big.Int).SetBytes(receipt.Result))
	}

	for i, log := range receipt.Logs {
		fmt.Printf("Log[%v]:\n", i)
		fmt.Println("\taddress:", log.Address.Hex())
		if len(log.Topics) == 1 {
			fmt.Println("\ttopics:", log.Topics[0].Hex())
		} else {
			fmt.Println("\ttopics:", log.Topics)
		}
		dataLen := len(log.Data) / 32
		for i := 0; i < dataLen; i++ {
			fmt.Printf("\tdata[%v]: %v\n", i, log.Data[i*32:i*32+32])
		}
	}
}

func getContractAddress(db database.Database) utils.Address {
	if len(contractHexAddr) == 0 {
		addr := getGlobalContractAddress(db)
		if addr.IsEmpty() {
			fmt.Println("Contract address not specified.")
		}

		return addr
	}

	addr, err := utils.HexToAddress(contractHexAddr)
	if err != nil {
		fmt.Println("Invalid contract address,", err.Error())
		return utils.EmptyAddress
	}

	return addr
}

func getContractInputMsg(db database.Database, contractAddr []byte) string {
	if len(input) > 0 {
		return input
	}

	if len(methodName) == 0 {
		fmt.Println("Input or method not specified.")
		return ""
	}

	output := getContractCompilationOutput(db, contractAddr)
	if output == nil {
		fmt.Println("Cannot find the contract info in DB.")
		return ""
	}

	method := output.getMethodByName(methodName)
	if method == nil {
		return ""
	}

	return method.createInput()
}
