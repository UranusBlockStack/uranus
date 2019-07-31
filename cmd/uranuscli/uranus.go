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
	"encoding/json"
	"math/big"
	"strconv"

	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var suggestGasPriceCmd = &cobra.Command{
	Use:   "suggestGasPrice ",
	Short: "Return suggest gas price.",
	Long:  `Return suggest gas price.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)
		cmdutils.ClientCall("Uranus.SuggestGasPrice", nil, &result)
		cmdutils.PrintJSON(result)
	},
}

var getBalanceCmd = &cobra.Command{
	Use:   "getBalance <address> [height]",
	Short: "returns the amount of wei for the given address in the state of the given block number.",
	Long:  `returns the amount of wei for the given address in the state of the given block number.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)

		req := &rpcapi.GetBalanceArgs{}
		switch len(args) {
		case 1:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
		case 2:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
			req.BlockHeight = cmdutils.GetBlockheight(args[1])
		}

		cmdutils.ClientCall("Uranus.GetBalance", req, &result)
		if cmdutils.OneLine {
			x := big.NewInt(0).Div(result.ToInt(), big.NewInt(1e14))
			y := float64(x.Int64())
			y /= 1e4
			jww.FEEDBACK.Print(result.String(), " URAC:", strconv.FormatFloat(y, 'f', -1, 64))
		} else {
			cmdutils.PrintJSON(result)
		}
	},
}

var getLockedBalanceCmd = &cobra.Command{
	Use:   "getLockedBalance <address> [height]",
	Short: "returns the locked amount of wei for the given address in the state of the given block number.",
	Long:  `returns the locked amount of wei for the given address in the state of the given block number.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)

		req := &rpcapi.GetBalanceArgs{}
		switch len(args) {
		case 1:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
		case 2:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
			req.BlockHeight = cmdutils.GetBlockheight(args[1])
		}

		cmdutils.ClientCall("Uranus.GetLockedBalance", req, &result)
		if cmdutils.OneLine {
			x := big.NewInt(0).Div(result.ToInt(), big.NewInt(1e14))
			y := float64(x.Int64())
			y /= 1e4
			jww.FEEDBACK.Print(result.String(), " URAC:", strconv.FormatFloat(y, 'f', -1, 64))
		} else {
			cmdutils.PrintJSON(result)
		}
	},
}

var getUnLockedBalanceCmd = &cobra.Command{
	Use:   "getUnLockedBalance <address> [height]",
	Short: "returns the unlocked amount of wei for the given address in the state of the given block number.",
	Long:  `returns the unlocked amount of wei for the given address in the state of the given block number.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)

		req := &rpcapi.GetBalanceArgs{}
		switch len(args) {
		case 1:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
		case 2:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
			req.BlockHeight = cmdutils.GetBlockheight(args[1])
		}

		cmdutils.ClientCall("Uranus.GetUnLockedBalance", req, &result)
		if cmdutils.OneLine {
			x := big.NewInt(0).Div(result.ToInt(), big.NewInt(1e14))
			y := float64(x.Int64())
			y /= 1e4
			jww.FEEDBACK.Print(result.String(), " URAC:", strconv.FormatFloat(y, 'f', -1, 64))
		} else {
			cmdutils.PrintJSON(result)
		}
	},
}

var getDelegateTimeCmd = &cobra.Command{
	Use:   "getDelegateTime <address> [height]",
	Short: "returns the time of locked amount of wei for the given address in the state of the given block number.",
	Long:  `returns the time of locked amount of wei for the given address in the state of the given block number.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)

		req := &rpcapi.GetBalanceArgs{}
		switch len(args) {
		case 1:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
		case 2:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
			req.BlockHeight = cmdutils.GetBlockheight(args[1])
		}

		cmdutils.ClientCall("Uranus.GetDelegateTimestamp", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var getUnDelegateTimeCmd = &cobra.Command{
	Use:   "getUnDelegateTime <address> [height]",
	Short: "returns the time of unlocked amount of wei for the given address in the state of the given block number.",
	Long:  `returns the time of unlocked amount of wei for the given address in the state of the given block number.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)

		req := &rpcapi.GetBalanceArgs{}
		switch len(args) {
		case 1:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
		case 2:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
			req.BlockHeight = cmdutils.GetBlockheight(args[1])
		}

		cmdutils.ClientCall("Uranus.GetUnDelegateTimestamp", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var getNonceCmd = &cobra.Command{
	Use:   "getNonce <address> [height]",
	Short: "returns nonce for the given address.",
	Long:  `returns nonce for the given address.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Uint64)

		req := &rpcapi.GetNonceArgs{}
		switch len(args) {
		case 1:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
		case 2:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
			req.BlockHeight = cmdutils.GetBlockheight(args[1])
		}

		cmdutils.ClientCall("Uranus.GetNonce", req, &result)
		if cmdutils.OneLine {
			jww.FEEDBACK.Print(result.String(), " Nonce:", strconv.FormatUint(uint64(*result), 10))
		} else {
			cmdutils.PrintJSON(result)
		}
	},
}
var getCodeCmd = &cobra.Command{
	Use:   "getCode <address> [height]",
	Short: "returns contract code for the given contract address.",
	Long:  `returns contract code for the given contract address.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Bytes)
		req := &rpcapi.GetCodeArgs{}
		switch len(args) {
		case 1:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
		case 2:
			req.Address = utils.HexToAddress(cmdutils.IsHexAddr(args[0]))
			req.BlockHeight = cmdutils.GetBlockheight(args[1])
		}

		cmdutils.ClientCall("Uranus.GetCode", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var sendRawTransactionCmd = &cobra.Command{
	Use:   "sendRawTransaction <TxHex>",
	Short: "Add the signed transaction to the transaction pool.",
	Long:  `Add the signed transaction to the transaction pool.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result := &utils.Hash{}
		cmdutils.ClientCall("Uranus.SendRawTransaction", utils.HexToBytes(args[0]), &result)
		cmdutils.PrintJSON(result)
	},
}

var signAndSendTransactionCmd = &cobra.Command{
	Use:   "signAndSendTransaction <SendTxArgs json>",
	Short: "Connecting to a remote node.",
	Long:  `Connecting to a remote node.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result := &utils.Hash{}
		req := &rpcapi.SendTxArgs{}
		if err := json.Unmarshal([]byte(args[0]), req); err != nil {
			jww.ERROR.Println(err)
		}
		cmdutils.ClientCall("Uranus.SignAndSendTransaction", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var callCmd = &cobra.Command{
	Use:   "call <CallArgs json>",
	Short: "executes the given transaction on the state for the given block number..",
	Long:  `executes the given transaction on the state for the given block number..`,
	Args:  cobra.ExactArgs(1),

	Run: func(cmd *cobra.Command, args []string) {
		result := map[string]interface{}{}
		req := &rpcapi.CallArgs{}
		if err := json.Unmarshal([]byte(args[0]), req); err != nil {
			jww.ERROR.Println(err)
		}
		cmdutils.ClientCall("Uranus.Call", req, &result)
		cmdutils.PrintJSON(result)
	},
}
