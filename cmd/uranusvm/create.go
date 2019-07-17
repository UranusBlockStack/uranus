/// Copyright 2018 The uranus Authors
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
	"io/ioutil"
	"math/big"
	"path/filepath"

	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var (
	code         string
	solFile      string
	solcCompiler string
	account      string

	defaultDir = filepath.Join(cmdutils.DefaultDataDir(), "simulator")
)

func init() {
	createCmd.Flags().StringVarP(&code, "code", "c", "", "the binary code of the smart contract to create, or the name of a readable file that contains the binary contract code in the local directory(Required)")
	createCmd.Flags().StringVarP(&solFile, "file", "f", "", "solidity file path")
	createCmd.Flags().StringVarP(&solcCompiler, "solc", "s", "./build/solc", "solc compiler path")
	createCmd.Flags().StringVarP(&account, "account", "a", "", "the account address(Default is random and has 1 seele)")
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a contract",
	Long:  "Create a contract with specified bytecodes or compiled bytecodes from specified solidity file.",
	Run: func(cmd *cobra.Command, args []string) {
		createContract()
	},
}

func createContract() {
	if len(solFile) == 0 && len(code) == 0 {
		jww.ERROR.Println("Code or solidity file not specified.")
		return
	}

	// compile solidity file if specified.
	var compileOutput *solCompileOutput
	if len(solFile) > 0 {
		output, dispose := compile(solFile, solcCompiler)
		if output == nil {
			return
		}

		compileOutput = output
		code = output.HexByteCodes
		defer dispose()
	}

	// Try to read the file, if successful, use the file code
	if bytecode, err := ioutil.ReadFile(code); err == nil {
		code = string(bytecode)
	}

	bytecode := utils.HexToBytes(utils.RemovePrefix(code))

	db, statedb, exec, dispose, err := preprocessContract()
	if err != nil {
		jww.FEEDBACK.Println("Failed to prepare the simulator environment,", err.Error())
		return
	}
	defer dispose()

	// Get an account to create the contract
	from := getFromAddress(statedb)
	if from.IsEmpty() {
		return
	}

	// Create a contract
	//createContractTx, err := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(1), math.MaxUint64, DefaultNonce, bytecode)
	accountNonce := statedb.GetNonce(from)
	createContractTx := types.NewTransaction(types.Binary, accountNonce, big.NewInt(0), uint64(3000000), big.NewInt(1), bytecode)

	_, receipt, err := processContract(from, statedb, exec, createContractTx)
	if err != nil {
		jww.FEEDBACK.Println("Failed to create contract,", err.Error())
		return
	}

	// Print the contract Address
	jww.FEEDBACK.Println()
	jww.FEEDBACK.Println("contract created successfully")
	jww.FEEDBACK.Println("Contract address:", receipt.ContractAddress.Hex())

	// Save contract info
	setGlobalContractAddress(db, receipt.ContractAddress)

	if compileOutput != nil {
		setContractCompilationOutput(db, receipt.ContractAddress.Bytes(), compileOutput)
	}
}
