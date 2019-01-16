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

var getValidatorsCmd = &cobra.Command{
	Use:   "getValidators <height> ",
	Short: "Returns the list of dpos validators by height.",
	Long:  `Returns the list of dpos validators by height.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := cmdutils.GetBlockheight(args[0])
		result := []utils.Address{}
		cmdutils.ClientCall("Dpos.GetValidators", req, &result)
		cmdutils.PrintJSONList(result)
	},
}

var getVotersCmd = &cobra.Command{
	Use:   "getVoters <height> ",
	Short: "Returns the list of dpos voters by height.",
	Long:  `Returns the list of dpos voters by height.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := cmdutils.GetBlockheight(args[0])
		result := make(map[utils.Address]utils.Big)
		cmdutils.ClientCall("Dpos.GetVoters", req, &result)
		cmdutils.PrintJSON(result)
	},
}

var getCandidatesCmd = &cobra.Command{
	Use:   "getCandidates <height> ",
	Short: "Returns the list of dpos candidates by height.",
	Long:  `Returns the list of dpos candidates by height.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := cmdutils.GetBlockheight(args[0])
		result := []utils.Address{}
		cmdutils.ClientCall("Dpos.GetCandidates", req, &result)
		cmdutils.PrintJSONList(result)
	},
}

var getDelegatorsCmd = &cobra.Command{
	Use:   "getDelegators <height> <candidate>",
	Short: "Returns the list of dpos delegators by height.",
	Long:  `Returns the list of dpos delegators by height.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		req := rpcapi.CandidateArgs{
			Number:    cmdutils.GetBlockheight(args[0]),
			Candidate: utils.HexToAddress(cmdutils.IsHexAddr(args[1])),
		}
		result := []utils.Address{}
		cmdutils.ClientCall("Dpos.GetDelegators", req, &result)
		cmdutils.PrintJSONList(result)
	},
}

var getConfirmedBlockNumberCmd = &cobra.Command{
	Use:   "getConfirmedBlockNumber ",
	Short: "Returns the confirmed block height.",
	Long:  `Returns the confirmed block height.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)
		cmdutils.ClientCall("Dpos.GetConfirmedBlockNumber", nil, &result)
		cmdutils.PrintJSON(result)
	},
}

var getBFTConfirmedBlockNumberCmd = &cobra.Command{
	Use:   "GetBFTConfirmedBlockNumber ",
	Short: "Returns the bft confirmed block height.",
	Long:  `Returns the bft confirmed block height.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)
		cmdutils.ClientCall("Dpos.GetBFTConfirmedBlockNumber", nil, &result)
		cmdutils.PrintJSON(result)
	},
}
