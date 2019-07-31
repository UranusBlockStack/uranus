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
	"fmt"
	"math/big"
	"strconv"
	"time"

	cmdutils "github.com/UranusBlockStack/uranus/cmd/utils"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
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
		result := []*rpcapi.VoterInfo{}
		cmdutils.ClientCall("Dpos.GetVoters", req, &result)
		if cmdutils.OneLine {
			for i := 0; i < len(result); i++ {
				x := big.NewInt(0).Div(result[i].LockedBalance.ToInt(), big.NewInt(1e14))
				y := float64(x.Int64())
				y /= 1e4
				t := time.Unix(result[i].TimeStamp.ToInt().Int64()/int64(time.Second), result[i].TimeStamp.ToInt().Int64()%int64(time.Second))
				s := fmt.Sprint("voter:", result[i].VoterAddr.String(),
					" locked:", strconv.FormatFloat(y, 'f', -1, 64),
					" time:", t)
				jww.FEEDBACK.Print(s)
			}
		} else {
			cmdutils.PrintJSONList(result)
		}
	},
}

var getCandidatesCmd = &cobra.Command{
	Use:   "getCandidates <height> ",
	Short: "Returns the list of dpos candidates by height.",
	Long:  `Returns the list of dpos candidates by height.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := cmdutils.GetBlockheight(args[0])
		result := []*rpcapi.CandidateInfo{}
		cmdutils.ClientCall("Dpos.GetCandidates", req, &result)
		if cmdutils.OneLine {
			for i := 0; i < len(result); i++ {
				x := big.NewInt(0).Div(result[i].Total.ToInt(), big.NewInt(1e14))
				y := float64(x.Int64())
				y /= 1e4
				s := fmt.Sprint("Candidate:", result[i].CandidateAddr.String(),
					" Total:", strconv.FormatFloat(y, 'f', -1, 64),
					" weight:", result[i].Weight)
				jww.FEEDBACK.Print(s)
			}
		} else {
			cmdutils.PrintJSONList(result)
		}
	},
}

var getDelegatorsCmd = &cobra.Command{
	Use:   "getDelegators <height> <candidate>",
	Short: "Returns the list of dpos delegators by height.",
	Long:  `Returns the list of dpos delegators by height.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		req := rpcapi.CandidateArgs{
			BlockHeight: cmdutils.GetBlockheight(args[0]),
			Candidate:   utils.HexToAddress(cmdutils.IsHexAddr(args[1])),
		}
		result := []*rpcapi.VoterInfo{}
		cmdutils.ClientCall("Dpos.GetDelegators", req, &result)
		cmdutils.PrintJSONList(result)
	},
}

var getVoterInfoCmd = &cobra.Command{
	Use:   "getVoter <height> <delegator>",
	Short: "Returns voter info by delegator and height.",
	Long:  `Returns voter info by delegator and height.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		req := rpcapi.VoterInfoArgs{
			BlockHeight: cmdutils.GetBlockheight(args[0]),
			Delegator:   utils.HexToAddress(cmdutils.IsHexAddr(args[1])),
		}
		result := rpcapi.VoterInfo{}
		cmdutils.ClientCall("Dpos.GetVoter", req, &result)
		cmdutils.PrintJSON(result)
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
		if cmdutils.OneLine {
			jww.FEEDBACK.Print(result.String(), " Confirm Height:", strconv.FormatInt(result.ToInt().Int64(), 10))
		} else {
			cmdutils.PrintJSON(result)
		}
	},
}

var getBFTConfirmedBlockNumberCmd = &cobra.Command{
	Use:   "getBFTConfirmedBlockNumber ",
	Short: "Returns the bft confirmed block height.",
	Long:  `Returns the bft confirmed block height.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := new(utils.Big)
		cmdutils.ClientCall("Dpos.GetBFTConfirmedBlockNumber", nil, &result)
		if cmdutils.OneLine {
			jww.FEEDBACK.Print(result.String(), " BFT Height:", strconv.FormatInt(result.ToInt().Int64(), 10))
		} else {
			cmdutils.PrintJSON(result)
		}
	},
}
