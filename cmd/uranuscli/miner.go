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
	"github.com/spf13/cobra"
)

var startMinerCmd = &cobra.Command{
	Use:   "startMiner [threads]",
	Short: "Start miner with threads.",
	Long:  `Start miner with threads.`,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			result  bool
			threads *int32
		)
		if len(args) != 0 {
			num := cmdutils.GetUint64(args[0])
			*threads = int32(num)
		}
		cmdutils.ClientCall("Miner.Start", threads, &result)
		cmdutils.PrintJSON(result)
	},
}

var stopMinerCmd = &cobra.Command{
	Use:   "stopMiner",
	Short: "Stop miner.",
	Long:  `Stop miner.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		cmdutils.ClientCall("Miner.Stop", nil, &result)
		cmdutils.PrintJSON(result)
	},
}

var setCoinbaseCmd = &cobra.Command{
	Use:   "setCoinbase <address>",
	Short: "Set the coinbase of the miner.",
	Long:  `Set the coinbase of the miner.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		cmdutils.ClientCall("Miner.SetCoinbase", utils.HexToAddress(cmdutils.IsHexAddr(args[0])), &result)
		cmdutils.PrintJSON(result)
	},
}
