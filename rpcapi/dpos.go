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

package rpcapi

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus/dpos"
	"github.com/UranusBlockStack/uranus/core/types"
)

// DposAPI exposes methods for the RPC interface
type DposAPI struct {
	b Backend
}

// NewDposAPI creates a new API definition for dpos methods of the node itself.
func NewDposAPI(b Backend) *DposAPI {
	return &DposAPI{b}
}

// GetValidators retrieves the list of the validators at specified block
func (api *DposAPI) GetValidators(number *BlockHeight, reply *[]utils.Address) error {
	var block *types.Block
	if number == nil || *number == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *number)
	}

	if block == nil {
		return fmt.Errorf("not found block %v", *number)
	}
	header := block.BlockHeader()

	statedb, err := api.b.BlockChain().State()
	if err != nil {
		return err
	}
	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), header.DposContext)
	if err != nil {
		return err
	}

	epochContext := &dpos.EpochContext{DposContext: dposContext, Statedb: statedb}
	validators, err := epochContext.DposContext.GetValidators()
	if err != nil {
		return err
	}

	*reply = validators
	return nil
}

type VoterInfoArgs struct {
	BlockHeight *BlockHeight
	Delegator   utils.Address
}

type VoterInfo struct {
	VoterAddr      utils.Address   `json:"voter"`
	LockedBalance  *utils.Big      `json:"lockedBalance"`
	TimeStamp      *utils.Big      `json:"timestamp"`
	CandidateAddrs []utils.Address `json:"candidates"`
}

type CandidateInfo struct {
	CandidateAddr utils.Address `json:"candidate"`
	Weight        uint64        `json:"weight"`
	Total         *big.Int      `json:"total"`
	Validate      *big.Int      `json:"-"`
}

type CandidateInfos []*CandidateInfo

func (p CandidateInfos) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p CandidateInfos) Len() int      { return len(p) }
func (p CandidateInfos) Less(i, j int) bool {
	if p[i].Validate.Cmp(p[j].Validate) < 0 {
		return false
	} else if p[i].Validate.Cmp(p[j].Validate) > 0 {
		return true
	} else {
		return p[i].CandidateAddr.String() < p[j].CandidateAddr.String()
	}
}

// GetVoter retrieves voter info at specified block
func (api *DposAPI) GetVoter(args *VoterInfoArgs, reply *VoterInfo) error {
	var block *types.Block
	if args.BlockHeight == nil || *args.BlockHeight == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *args.BlockHeight)
	}
	if block == nil {
		return fmt.Errorf("not found block %v", *args.BlockHeight)
	}

	header := block.BlockHeader()

	statedb, err := api.b.BlockChain().StateAt(block.StateRoot())
	if err != nil {
		return err
	}

	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), header.DposContext)
	if err != nil {
		return err
	}

	addrs, err := dposContext.GetCandidateAddrs(args.Delegator)
	if err != nil {
		return err
	}

	*reply = VoterInfo{
		VoterAddr:      args.Delegator,
		LockedBalance:  (*utils.Big)(statedb.GetLockedBalance(args.Delegator)),
		TimeStamp:      (*utils.Big)(statedb.GetDelegateTimestamp(args.Delegator)),
		CandidateAddrs: addrs,
	}

	return nil
}

// GetVoters retrieves the list of the voters at specified block
func (api *DposAPI) GetVoters(number *BlockHeight, reply *[]*VoterInfo) error {
	var block *types.Block
	if number == nil || *number == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *number)
	}
	if block == nil {
		return fmt.Errorf("not found block %v", *number)
	}
	header := block.BlockHeader()

	statedb, err := api.b.BlockChain().State()
	if err != nil {
		return err
	}

	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), header.DposContext)
	if err != nil {
		return err
	}
	epochContext := &dpos.EpochContext{DposContext: dposContext, Statedb: statedb}

	result := []*VoterInfo{}
	iter := mtp.NewIterator(epochContext.DposContext.VoteTrie().NodeIterator(nil))
	for iter.Next() {
		voter := utils.BytesToAddress(iter.Key)
		candidateAddrs := []utils.Address{}
		candidateAddrsBytes := iter.Value
		if err := rlp.DecodeBytes(candidateAddrsBytes, &candidateAddrs); err != nil {
			return err
		}
		voterInfo := &VoterInfo{
			VoterAddr:      voter,
			LockedBalance:  (*utils.Big)(statedb.GetLockedBalance(voter)),
			TimeStamp:      (*utils.Big)(statedb.GetDelegateTimestamp(voter)),
			CandidateAddrs: candidateAddrs,
		}
		result = append(result, voterInfo)
	}

	*reply = result
	return nil
}

// GetCandidates retrieves the list of the candidate at specified block
func (api *DposAPI) GetCandidates(number *BlockHeight, reply *[]*CandidateInfo) error {
	var block *types.Block
	if number == nil || *number == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *number)
	}
	if block == nil {
		return fmt.Errorf("not found block %v", *number)
	}
	header := block.BlockHeader()

	statedb, err := api.b.BlockChain().State()
	if err != nil {
		return err
	}
	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), header.DposContext)

	if err != nil {
		return err
	}
	epochContext := &dpos.EpochContext{DposContext: dposContext, Statedb: statedb}
	candidates, err := epochContext.DposContext.GetCandidates()
	if err != nil {
		return err
	}

	votes, _, _ := epochContext.CountVotes()
	result := CandidateInfos{}

	for _, validator := range candidates {
		candidateInfo := &CandidateInfo{
			CandidateAddr: validator.Addr,
			Weight:        validator.Weight,
			Validate:      votes[validator.Addr],
		}
		candidateInfo.Total = new(big.Int).Div(candidateInfo.Validate, big.NewInt(int64(validator.Weight)))
		result = append(result, candidateInfo)
	}
	sort.Sort(result)
	*reply = result
	return nil
}

type CandidateArgs struct {
	BlockHeight *BlockHeight
	Candidate   utils.Address
}

// GetDelegators retrieves the list of the delegators of specified candidate at specified block
func (api *DposAPI) GetDelegators(args *CandidateArgs, reply *[]*VoterInfo) error {
	var block *types.Block
	if args.BlockHeight == nil || *args.BlockHeight == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *args.BlockHeight)
	}
	if block == nil {
		return fmt.Errorf("not found block %v", *args.BlockHeight)
	}
	header := block.BlockHeader()

	statedb, err := api.b.BlockChain().State()
	if err != nil {
		return err
	}

	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), header.DposContext)
	if err != nil {
		return err
	}
	epochContext := &dpos.EpochContext{DposContext: dposContext, Statedb: statedb}
	delegators, err := epochContext.DposContext.GetDelegators(args.Candidate)
	if err != nil {
		return err
	}

	result := []*VoterInfo{}

	for _, delegator := range delegators {
		addrs, _ := dposContext.GetCandidateAddrs(delegator)
		voterInfo := &VoterInfo{
			VoterAddr:      delegator,
			LockedBalance:  (*utils.Big)(statedb.GetLockedBalance(delegator)),
			TimeStamp:      (*utils.Big)(statedb.GetDelegateTimestamp(delegator)),
			CandidateAddrs: addrs,
		}
		result = append(result, voterInfo)
	}

	*reply = result

	return nil
}

// GetConfirmedBlockNumber retrieves the latest irreversible block
func (api *DposAPI) GetConfirmedBlockNumber(ignore string, reply *utils.Big) error {
	n, err := api.b.GetConfirmedBlockNumber()
	if err != nil {
		return err
	}
	*reply = *(*utils.Big)(n)
	return nil
}

// GetBFTConfirmedBlockNumber retrieves  the bft latest irreversible block
func (api *DposAPI) GetBFTConfirmedBlockNumber(ignore string, reply *utils.Big) error {
	n, err := api.b.GetBFTConfirmedBlockNumber()
	if err != nil {
		return err
	}
	*reply = *(*utils.Big)(n)
	return nil
}
