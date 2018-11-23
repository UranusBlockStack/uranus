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
	"github.com/spf13/cobra"
)

var getContentCmd = &cobra.Command{
	Use:   "getContent ",
	Short: "Returns the transactions contained within the transaction pool.",
	Long:  `Returns the transactions contained within the transaction pool.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := map[string]map[string]map[string]*rpcapi.RPCTransaction{}
		cmdutils.ClientCall("TxPool.Content", nil, &result)
		cmdutils.PrintJSON(result)
	},
}
var getStatusCmd = &cobra.Command{
	Use:   "getStatus ",
	Short: "returns the number of pending and queued transaction in the pool.",
	Long:  `returns the number of pending and queued transaction in the pool.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := map[string]utils.Uint{}
		cmdutils.ClientCall("TxPool.Status", nil, &result)
		cmdutils.PrintJSON(result)
	},
}
