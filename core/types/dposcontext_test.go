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

package types

import (
	"testing"

	"github.com/UranusBlockStack/uranus/common/rlp"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

func TestDposContextSnapshot(t *testing.T) {
	dbMem := db.NewMemDatabase()
	db := mtp.NewDatabase(dbMem)
	dposContext, err := NewDposContext(db)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := dposContext.Snapshot()
	utils.AssertEquals(t, dposContext.Root(), snapshot.Root())
	utils.AssertNotEquals(t, dposContext, snapshot)

	// change dposContext
	if err := dposContext.BecomeCandidate(utils.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")); err != nil {
		t.Fatal(err)
	}
	utils.AssertNotEquals(t, dposContext.Root(), snapshot.Root())

	// revert snapshot
	dposContext.RevertToSnapShot(snapshot)
	utils.AssertEquals(t, dposContext.Root(), snapshot.Root())
	utils.AssertNotEquals(t, dposContext, snapshot)
}

func TestDposContextBecomeCandidate(t *testing.T) {
	candidates := []utils.Address{
		utils.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		utils.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		utils.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}
	dbMem := db.NewMemDatabase()
	db := mtp.NewDatabase(dbMem)
	dposContext, err := NewDposContext(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if err := dposContext.BecomeCandidate(candidate); err != nil {
			t.Fatal(err)
		}
	}

	candidateMap := map[utils.Address]bool{}
	candidateIter := mtp.NewIterator(dposContext.candidateTrie.NodeIterator(nil))

	for candidateIter.Next() {
		info := &CandidateInfo{}
		if err := rlp.DecodeBytes(candidateIter.Value, info); err != nil {
			t.Fatal(err)
		}
		candidateMap[info.Addr] = true
	}

	utils.AssertEquals(t, len(candidates), len(candidateMap))

	for _, candidate := range candidates {
		utils.AssertEquals(t, candidateMap[candidate], true)
	}

}

func TestDposContextKickoutCandidate(t *testing.T) {
	candidates := []utils.Address{
		utils.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		utils.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		utils.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}
	dbMem := db.NewMemDatabase()
	db := mtp.NewDatabase(dbMem)
	dposContext, err := NewDposContext(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if err := dposContext.BecomeCandidate(candidate); err != nil {
			t.Fatal(err)
		}
		if err := dposContext.Delegate(candidate, candidate); err != nil {
			t.Fatal(err)
		}
	}

	kickIdx := 1
	if err := dposContext.KickoutCandidate(candidates[kickIdx]); err != nil {
		t.Fatal(err)
	}
	candidateMap := map[utils.Address]bool{}
	candidateIter := mtp.NewIterator(dposContext.candidateTrie.NodeIterator(nil))
	for candidateIter.Next() {
		info := &CandidateInfo{}
		if err := rlp.DecodeBytes(candidateIter.Value, info); err != nil {
			t.Fatal(err)
		}
		candidateMap[info.Addr] = true
	}
	voteIter := mtp.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	voteMap := map[utils.Address]bool{}
	for voteIter.Next() {
		voteMap[utils.BytesToAddress(voteIter.Value)] = true
	}

	for i, candidate := range candidates {
		delegateIter := mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
		if i == kickIdx {
			utils.AssertEquals(t, delegateIter.Next(), false)
			utils.AssertEquals(t, candidateMap[candidate], false)
			utils.AssertEquals(t, voteMap[candidate], false)
			continue
		}
		utils.AssertEquals(t, delegateIter.Next(), true)
		utils.AssertEquals(t, candidateMap[candidate], true)
		utils.AssertEquals(t, voteMap[candidate], true)
	}
}

func TestDposContextDelegateAndUnDelegate(t *testing.T) {
	candidate := utils.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e")
	newCandidate := utils.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2")
	delegator := utils.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670")
	dbMem := db.NewMemDatabase()
	db := mtp.NewDatabase(dbMem)
	dposContext, err := NewDposContext(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := dposContext.BecomeCandidate(candidate); err != nil {
		t.Fatal(err)
	}
	if err := dposContext.BecomeCandidate(newCandidate); err != nil {
		t.Fatal(err)
	}

	// delegator delegate to not exist candidate
	candidateIter := mtp.NewIterator(dposContext.candidateTrie.NodeIterator(nil))
	candidateMap := map[string]bool{}
	for candidateIter.Next() {
		candidateMap[string(candidateIter.Value)] = true
	}
	if err := dposContext.Delegate(delegator, utils.HexToAddress("0xab")); err != nil && err.Error() != "invalid candidate to delegate" {
		t.Fatal(err)
	}

	// delegator delegate to old candidate
	if err := dposContext.Delegate(delegator, candidate); err != nil {
		t.Fatal(err)
	}
	delegateIter := mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
	if delegateIter.Next() {
		utils.AssertEquals(t, append(delegatePrefix, append(candidate.Bytes(), delegator.Bytes()...)...), delegateIter.Key)
		utils.AssertEquals(t, delegator, utils.BytesToAddress(delegateIter.Value))
	}
	voteIter := mtp.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	if voteIter.Next() {
		utils.AssertEquals(t, append(votePrefix, delegator.Bytes()...), voteIter.Key)
		utils.AssertEquals(t, candidate, utils.BytesToAddress(voteIter.Value))
	}

	// delegator delegate to new candidate
	if err := dposContext.Delegate(delegator, newCandidate); err != nil {
		t.Fatal(err)
	}
	delegateIter = mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
	utils.AssertEquals(t, delegateIter.Next(), false)
	delegateIter = mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(newCandidate.Bytes()))
	if delegateIter.Next() {
		utils.AssertEquals(t, append(delegatePrefix, append(newCandidate.Bytes(), delegator.Bytes()...)...), delegateIter.Key)
		utils.AssertEquals(t, delegator, utils.BytesToAddress(delegateIter.Value))
	}
	voteIter = mtp.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	if voteIter.Next() {
		utils.AssertEquals(t, append(votePrefix, delegator.Bytes()...), voteIter.Key)
		utils.AssertEquals(t, newCandidate, utils.BytesToAddress(voteIter.Value))
	}

	// delegator undelegate to not exist candidate
	if err := dposContext.UnDelegate(utils.HexToAddress("0x00"), candidate); err != nil && err.Error() != "mismatch candidate to undelegate" {
		t.Fatal(err)
	}

	// delegator undelegate to old candidate
	if err := dposContext.UnDelegate(delegator, candidate); err != nil && err.Error() != "mismatch candidate to undelegate" {
		t.Fatal(err)
	}

	// delegator undelegate to new candidate
	if err := dposContext.UnDelegate(delegator, newCandidate); err != nil {
		t.Fatal(err)
	}
	delegateIter = mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(newCandidate.Bytes()))
	utils.AssertEquals(t, delegateIter.Next(), false)
	voteIter = mtp.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	utils.AssertEquals(t, voteIter.Next(), false)
}

func TestDposContextValidators(t *testing.T) {
	validators := []utils.Address{
		utils.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		utils.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		utils.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}

	dbMem := db.NewMemDatabase()
	db := mtp.NewDatabase(dbMem)
	dposContext, err := NewDposContext(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := dposContext.SetValidators(validators); err != nil {
		t.Fatal(err)
	}

	result, err := dposContext.GetValidators()
	if err != nil {
		t.Fatal(err)
	}

	utils.AssertEquals(t, len(validators), len(result))
	validatorMap := map[utils.Address]bool{}
	for _, validator := range validators {
		validatorMap[validator] = true
	}
	for _, validator := range result {
		utils.AssertEquals(t, validatorMap[validator], true)
	}
}
