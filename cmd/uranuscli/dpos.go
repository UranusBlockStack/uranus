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

var getValidatorsCmd = &cobra.Command{
	Use:   "getValidators <height> ",
	Short: "Returns the list of dpos validators by height.",
	Long:  `Returns the list of dpos validators by height.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := getBlockheight(args[0])
		result := []utils.Address{}
		ClientCall("Dpos.GetValidators", req, &result)
		printJSONList(result)
	},
}

var getVotersCmd = &cobra.Command{
	Use:   "getVoters <height> ",
	Short: "Returns the list of dpos voters by height.",
	Long:  `Returns the list of dpos voters by height.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := getBlockheight(args[0])
		result := make(map[utils.Address]utils.Big)
		ClientCall("Dpos.GetVoters", req, &result)
		printJSON(result)
	},
}

var getCandidatesCmd = &cobra.Command{
	Use:   "getCandidates <height> ",
	Short: "Returns the list of dpos candidates by height.",
	Long:  `Returns the list of dpos candidates by height.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := getBlockheight(args[0])
		result := []utils.Address{}
		ClientCall("Dpos.GetCandidates", req, &result)
		printJSONList(result)
	},
}
