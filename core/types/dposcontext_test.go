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

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/stretchr/testify/assert"
)

func TestDposContextSnapshot(t *testing.T) {
	dbMem := db.NewMemDatabase()
	db := mtp.NewDatabase(dbMem)
	dposContext, err := NewDposContext(db)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := dposContext.Snapshot()
	assert.Equal(t, dposContext.Root(), snapshot.Root())
	assert.NotEqual(t, dposContext, snapshot)

	// change dposContext
	if err := dposContext.BecomeCandidate(utils.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")); err != nil {
		t.Fatal(err)
	}
	assert.NotEqual(t, dposContext.Root(), snapshot.Root())

	// revert snapshot
	dposContext.RevertToSnapShot(snapshot)
	assert.Equal(t, dposContext.Root(), snapshot.Root())
	assert.NotEqual(t, dposContext, snapshot)
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

	assert.Equal(t, len(candidates), len(candidateMap))
	for _, candidate := range candidates {
		assert.Equal(t, candidateMap[candidate], true)
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
		if err := dposContext.Delegate(candidate, []*utils.Address{&candidate}); err != nil {
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
		voters := new([]*utils.Address)

		if err := rlp.DecodeBytes(voteIter.Value, voters); err != nil {
			t.Fatal(err)
		}

		voteMap[*(*voters)[0]] = true
	}
	for i, candidate := range candidates {
		delegateIter := mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
		if i == kickIdx {
			assert.Equal(t, delegateIter.Next(), false)
			assert.Equal(t, candidateMap[candidate], false)
			assert.Equal(t, voteMap[candidate], false)
			continue
		}
		assert.Equal(t, delegateIter.Next(), true)
		assert.Equal(t, candidateMap[candidate], true)
		assert.Equal(t, voteMap[candidate], true)
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
	testaddr := utils.HexToAddress("0xab")
	if err := dposContext.Delegate(delegator, []*utils.Address{&testaddr}); err != nil && err.Error() != "invalid candidate 0x00000000000000000000000000000000000000AB to delegate" {
		t.Fatal(err)
	}

	// delegator delegate to old candidate
	if err := dposContext.Delegate(delegator, []*utils.Address{&candidate}); err != nil {
		t.Fatal(err)
	}
	delegateIter := mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
	if delegateIter.Next() {
		assert.Equal(t, append(delegatePrefix, append(candidate.Bytes(), delegator.Bytes()...)...), delegateIter.Key)
		assert.Equal(t, delegator, utils.BytesToAddress(delegateIter.Value))
	}
	voteIter := mtp.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	if voteIter.Next() {
		assert.Equal(t, append(votePrefix, delegator.Bytes()...), voteIter.Key)
		assert.Equal(t, candidate, utils.BytesToAddress(voteIter.Value))
	}

	// delegator delegate to new candidate
	if err := dposContext.Delegate(delegator, []*utils.Address{&newCandidate}); err != nil {
		t.Fatal(err)
	}
	delegateIter = mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
	assert.Equal(t, delegateIter.Next(), false)
	delegateIter = mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(newCandidate.Bytes()))
	if delegateIter.Next() {
		assert.Equal(t, append(delegatePrefix, append(newCandidate.Bytes(), delegator.Bytes()...)...), delegateIter.Key)
		assert.Equal(t, delegator, utils.BytesToAddress(delegateIter.Value))
	}
	voteIter = mtp.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	if voteIter.Next() {
		assert.Equal(t, append(votePrefix, delegator.Bytes()...), voteIter.Key)
		assert.Equal(t, newCandidate, utils.BytesToAddress(voteIter.Value))
	}

	// delegator undelegate to not exist candidate
	if err := dposContext.UnDelegate(utils.HexToAddress("0x00")); err != nil && err.Error() != "invalid candidate to undelegate" {
		t.Fatal(err)
	}

	// delegator undelegate to new candidate
	if err := dposContext.UnDelegate(delegator); err != nil {
		t.Fatal(err)
	}
	delegateIter = mtp.NewIterator(dposContext.delegateTrie.PrefixIterator(newCandidate.Bytes()))
	assert.Equal(t, delegateIter.Next(), false)
	voteIter = mtp.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	assert.Equal(t, voteIter.Next(), false)
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

	assert.Equal(t, len(validators), len(result))
	validatorMap := map[utils.Address]bool{}
	for _, validator := range validators {
		validatorMap[validator] = true
	}
	for _, validator := range result {
		assert.Equal(t, validatorMap[validator], true)
	}
}
