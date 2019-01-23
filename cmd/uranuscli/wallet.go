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
	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"github.com/UranusBlockStack/uranus/wallet"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var createAccountCmd = &cobra.Command{
	Use:   "createAccount <passphrase>",
	Short: "Create a new account.",
	Long:  `Create a new account.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result := utils.Address{}
		cmdutils.ClientCall("Wallet.NewAccount", args[0], &result)
		cmdutils.PrintJSON(result)
	},
}

var deleteAccountCmd = &cobra.Command{
	Use:   "deleteAccount <address> <passphrase>",
	Short: "Delete a account by address and passphrase.",
	Long:  `Delete a account by address and passphrase.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		cmdutils.ClientCall("Wallet.Delete", rpcapi.DeleteArgs{
			Address:    utils.HexToAddress(cmdutils.IsHexAddr(args[0])),
			Passphrase: args[1]}, &result)
		cmdutils.PrintJSON(result)
	},
}

var updateAccountCmd = &cobra.Command{
	Use:   "updateAccount <address> <passphrase> <newpassphrase>",
	Short: "Update a account passphrase.",
	Long:  `Update a account passphrase.`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		cmdutils.ClientCall("Wallet.Update", rpcapi.UpdateArgs{
			Address:       utils.HexToAddress(cmdutils.IsHexAddr(args[0])),
			Passphrase:    args[1],
			NewPassphrase: args[2]}, &result)
		cmdutils.PrintJSON(result)
	},
}

// listAccount represents the version command
var listAccountsCmd = &cobra.Command{
	Use:   "listAccounts",
	Short: "List all exist accounts.",
	Long:  `List all exist accounts.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := wallet.Accounts{}
		cmdutils.ClientCall("Wallet.Accounts", nil, &result)
		if cmdutils.OneLine {
			for i, a := range result {
				jww.FEEDBACK.Print(i, ":", a.Address.String())
			}
		} else {
			cmdutils.PrintJSONList(result)
		}
	},
}

var importRawKeyCmd = &cobra.Command{
	Use:   "importRawKey <privKeyHex> <passphrase>",
	Short: "Import raw PrivateKey into walet.",
	Long:  `Import raw PrivateKey into walet.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		result := utils.Address{}
		cmdutils.ClientCall("Wallet.ImportRawKey", rpcapi.ImportRawKeyArgs{
			PrivKeyHex: args[0],
			Passphrase: args[1]}, &result)
		cmdutils.PrintJSON(result)
	},
}

var exportRawKeyCmd = &cobra.Command{
	Use:   "exportRawKey <address> <passphrase>",
	Short: "export raw PrivateKey as hex string.",
	Long:  `export raw PrivateKey as hex string.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var result string
		cmdutils.ClientCall("Wallet.ExportRawKey", rpcapi.ExportRawKeyArgs{
			Address:    utils.HexToAddress(cmdutils.IsHexAddr(args[0])),
			Passphrase: args[1]}, &result)
		cmdutils.PrintJSON(result)
	},
}
