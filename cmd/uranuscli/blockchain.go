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
	"os"
	"strconv"

	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

var getBlockByHeightCmd = &cobra.Command{
	Use:   "getBlockByHeight <height> [fullTx]",
	Short: "Returns the requested block by height.",
	Long:  `Returns the requested block by height.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		req := rpcapi.GetBlockByHeightArgs{}
		switch len(args) {
		case 1:
			req.BlockHeight = cmdutils.GetBlockheight(args[0])
		case 2:
			req.BlockHeight = cmdutils.GetBlockheight(args[0])
			full, err := strconv.ParseBool(args[1])
			if err != nil {
				jww.ERROR.Printf("Invalid fulltx value: %v err: %v", args[1], err)
				os.Exit(1)
			}
			req.FullTx = full
		}

		result := map[string]interface{}{}
		cmdutils.ClientCall("BlockChain.GetBlockByHeight", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var getBlockByHashCmd = &cobra.Command{
	Use:   "getBlockByHash <hash> [fullTx]",
	Short: "Returns the requested block by hash.",
	Long:  `Returns the requested block by hash.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		req := rpcapi.GetBlockByHashArgs{}

		switch len(args) {
		case 1:
			req.BlockHash = utils.HexToHash(cmdutils.IsHexHash(args[0]))
		case 2:
			req.BlockHash = utils.HexToHash(cmdutils.IsHexHash(args[0]))
			full, err := strconv.ParseBool(args[1])
			if err != nil {
				jww.ERROR.Printf("ParseBool args: %v err: %v", args[1], err)
				os.Exit(1)
			}
			req.FullTx = full
		}
		result := map[string]interface{}{}
		cmdutils.ClientCall("BlockChain.GetBlockByHash", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var getTransactionByHashCmd = &cobra.Command{
	Use:   "getTransactionByHash <hash>",
	Short: "Returns the transaction for the given transaction hash.",
	Long:  `Returns the transaction for the given transaction hash.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result := &rpcapi.RPCTransaction{}
		cmdutils.ClientCall("BlockChain.GetTransactionByHash", utils.HexToHash(cmdutils.IsHexHash(args[0])), &result)
		cmdutils.PrintJSON(result)
	},
}

var getTransactionReceiptCmd = &cobra.Command{
	Use:   "getTransactionReceipt <hash>",
	Short: "Returns the transaction receipt for the given transaction hash.",
	Long:  `Returns the transaction receipt for the given transaction hash.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		result := map[string]interface{}{}
		cmdutils.ClientCall("BlockChain.GetTransactionReceipt", utils.HexToHash(cmdutils.IsHexHash(args[0])), &result)
		cmdutils.PrintJSON(result)
	},
}
