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
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"github.com/spf13/cobra"
)

var createAccountCmd = &cobra.Command{
	Use:   "createAccount <passphrase>",
	Short: "Create a new account.",
	Long:  `Create a new account.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result := utils.Address{}
		ClientCall("Wallet.NewAccount", args[0], &result)
		printJSON(result)
	},
}

var deleteAccountCmd = &cobra.Command{
	Use:   "deleteAccount <address> <passphrase>",
	Short: "Delete a account by address and passphrase.",
	Long:  `Delete a account by address and passphrase.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		ClientCall("Wallet.Delete", rpcapi.DeleteArgs{
			Address:    utils.HexToAddress(isHexAddr(args[0])),
			Passphrase: args[1]}, &result)
		printJSON(result)
	},
}

var updateAccountCmd = &cobra.Command{
	Use:   "updateAccount <address> <passphrase> <newpassphrase>",
	Short: "Update a account passphrase.",
	Long:  `Update a account passphrase.`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		ClientCall("Wallet.Update", rpcapi.UpdateArgs{
			Address:       utils.HexToAddress(isHexAddr(args[0])),
			Passphrase:    args[1],
			NewPassphrase: args[2]}, &result)
		printJSON(result)
	},
}

// listAccount represents the version command
var listAccountsCmd = &cobra.Command{
	Use:   "listAccounts",
	Short: "List all exist accounts.",
	Long:  `List all exist accounts.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := []utils.Address{}
		ClientCall("Wallet.Accounts", nil, &result)
		printJSONList(result)
	},
}

var importRawKeyCmd = &cobra.Command{
	Use:   "importRawKey <privKeyHex> <passphrase>",
	Short: "Import raw PrivateKey into walet.",
	Long:  `Import raw PrivateKey into walet.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		result := utils.Address{}
		ClientCall("Wallet.ImportRawKey", rpcapi.ImportRawKeyArgs{
			PrivKeyHex: args[0],
			Passphrase: args[1]}, &result)
		printJSON(result)
	},
}
