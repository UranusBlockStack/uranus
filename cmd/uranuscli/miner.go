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
			num := getUint64(args[0])
			*threads = int32(num)
		}
		ClientCall("Miner.Start", threads, &result)
		printJSON(result)
	},
}

var stopMinerCmd = &cobra.Command{
	Use:   "stopMiner",
	Short: "Stop miner.",
	Long:  `Stop miner.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		ClientCall("Miner.Stop", nil, &result)
		printJSON(result)
	},
}

var setCoinbaseCmd = &cobra.Command{
	Use:   "setCoinbase <address>",
	Short: "Set the coinbase of the miner.",
	Long:  `Set the coinbase of the miner.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		ClientCall("Miner.SetCoinbase", utils.HexToAddress(isHexAddr(args[0])), &result)
		printJSON(result)
	},
}
