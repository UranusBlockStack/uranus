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
	"bytes"
	"errors"
	"fmt"

	"github.com/UranusBlockStack/uranus/common/crypto/sha3"
	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

type DposContext struct {
	epochTrie     *mtp.Trie
	delegateTrie  *mtp.Trie
	voteTrie      *mtp.Trie
	candidateTrie *mtp.Trie
	mintCntTrie   *mtp.Trie

	db *mtp.Database
}

type CandidateInfo struct {
	Addr        utils.Address
	Weight      uint64 // 100
	DegradeTime uint64
}

var (
	epochPrefix     = []byte("epoch-")
	delegatePrefix  = []byte("delegate-")
	votePrefix      = []byte("vote-")
	candidatePrefix = []byte("candidate-")
	mintCntPrefix   = []byte("mintCnt-")
)

func NewEpochTrie(root utils.Hash, db *mtp.Database) (*mtp.Trie, error) {
	return mtp.NewWithPrefix(root, epochPrefix, db)
}

func NewDelegateTrie(root utils.Hash, db *mtp.Database) (*mtp.Trie, error) {
	return mtp.NewWithPrefix(root, delegatePrefix, db)
}

func NewVoteTrie(root utils.Hash, db *mtp.Database) (*mtp.Trie, error) {
	return mtp.NewWithPrefix(root, votePrefix, db)
}

func NewCandidateTrie(root utils.Hash, db *mtp.Database) (*mtp.Trie, error) {
	return mtp.NewWithPrefix(root, candidatePrefix, db)
}

func NewMintCntTrie(root utils.Hash, db *mtp.Database) (*mtp.Trie, error) {
	return mtp.NewWithPrefix(root, mintCntPrefix, db)
}

func NewDposContext(db *mtp.Database) (*DposContext, error) {
	epochTrie, err := NewEpochTrie(utils.Hash{}, db)
	if err != nil {
		return nil, err
	}
	delegateTrie, err := NewDelegateTrie(utils.Hash{}, db)
	if err != nil {
		return nil, err
	}
	voteTrie, err := NewVoteTrie(utils.Hash{}, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(utils.Hash{}, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(utils.Hash{}, db)
	if err != nil {
		return nil, err
	}
	return &DposContext{
		epochTrie:     epochTrie,
		delegateTrie:  delegateTrie,
		voteTrie:      voteTrie,
		candidateTrie: candidateTrie,
		mintCntTrie:   mintCntTrie,
		db:            db,
	}, nil
}

func NewDposContextFromProto(db *mtp.Database, ctxProto *DposContextProto) (*DposContext, error) {
	epochTrie, err := NewEpochTrie(ctxProto.EpochHash, db)
	if err != nil {
		return nil, err
	}
	delegateTrie, err := NewDelegateTrie(ctxProto.DelegateHash, db)
	if err != nil {
		return nil, err
	}
	voteTrie, err := NewVoteTrie(ctxProto.VoteHash, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(ctxProto.CandidateHash, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(ctxProto.MintCntHash, db)
	if err != nil {
		return nil, err
	}
	return &DposContext{
		epochTrie:     epochTrie,
		delegateTrie:  delegateTrie,
		voteTrie:      voteTrie,
		candidateTrie: candidateTrie,
		mintCntTrie:   mintCntTrie,
		db:            db,
	}, nil
}

func (d *DposContext) Copy() *DposContext {
	epochTrie := *d.epochTrie
	delegateTrie := *d.delegateTrie
	voteTrie := *d.voteTrie
	candidateTrie := *d.candidateTrie
	mintCntTrie := *d.mintCntTrie
	return &DposContext{
		epochTrie:     &epochTrie,
		delegateTrie:  &delegateTrie,
		voteTrie:      &voteTrie,
		candidateTrie: &candidateTrie,
		mintCntTrie:   &mintCntTrie,
	}
}

func (d *DposContext) Root() (h utils.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, d.epochTrie.Hash())
	rlp.Encode(hw, d.delegateTrie.Hash())
	rlp.Encode(hw, d.candidateTrie.Hash())
	rlp.Encode(hw, d.voteTrie.Hash())
	rlp.Encode(hw, d.mintCntTrie.Hash())
	hw.Sum(h[:0])
	return h
}

func (d *DposContext) Snapshot() *DposContext {
	return d.Copy()
}

func (d *DposContext) RevertToSnapShot(snapshot *DposContext) {
	d.epochTrie = snapshot.epochTrie
	d.delegateTrie = snapshot.delegateTrie
	d.candidateTrie = snapshot.candidateTrie
	d.voteTrie = snapshot.voteTrie
	d.mintCntTrie = snapshot.mintCntTrie
}

func (d *DposContext) FromProto(dcp *DposContextProto) error {
	var err error
	d.epochTrie, err = NewEpochTrie(dcp.EpochHash, d.db)
	if err != nil {
		return err
	}
	d.delegateTrie, err = NewDelegateTrie(dcp.DelegateHash, d.db)
	if err != nil {
		return err
	}
	d.candidateTrie, err = NewCandidateTrie(dcp.CandidateHash, d.db)
	if err != nil {
		return err
	}
	d.voteTrie, err = NewVoteTrie(dcp.VoteHash, d.db)
	if err != nil {
		return err
	}
	d.mintCntTrie, err = NewMintCntTrie(dcp.MintCntHash, d.db)
	return err
}

type DposContextProto struct {
	EpochHash     utils.Hash `json:"epochRoot"        gencodec:"required"`
	DelegateHash  utils.Hash `json:"delegateRoot"     gencodec:"required"`
	CandidateHash utils.Hash `json:"candidateRoot"    gencodec:"required"`
	VoteHash      utils.Hash `json:"voteRoot"         gencodec:"required"`
	MintCntHash   utils.Hash `json:"mintCntRoot"      gencodec:"required"`
}

func (d *DposContext) ToProto() *DposContextProto {
	return &DposContextProto{
		EpochHash:     d.epochTrie.Hash(),
		DelegateHash:  d.delegateTrie.Hash(),
		CandidateHash: d.candidateTrie.Hash(),
		VoteHash:      d.voteTrie.Hash(),
		MintCntHash:   d.mintCntTrie.Hash(),
	}
}

func (p *DposContextProto) Root() (h utils.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, p.EpochHash)
	rlp.Encode(hw, p.DelegateHash)
	rlp.Encode(hw, p.CandidateHash)
	rlp.Encode(hw, p.VoteHash)
	rlp.Encode(hw, p.MintCntHash)
	hw.Sum(h[:0])
	return h
}

func (d *DposContext) KickoutCandidate(candidateAddr utils.Address) error {
	candidate := candidateAddr.Bytes()
	err := d.candidateTrie.TryDelete(candidate)
	if err != nil {
		if _, ok := err.(*mtp.MissingNodeError); !ok {
			return err
		}
	}
	iter := mtp.NewIterator(d.delegateTrie.PrefixIterator(candidate))
	for iter.Next() {
		delegator := iter.Value
		key := append(candidate, delegator...)
		err = d.delegateTrie.TryDelete(key)
		if err != nil {
			if _, ok := err.(*mtp.MissingNodeError); !ok {
				return err
			}
		}
		v, err := d.voteTrie.TryGet(delegator)
		if err != nil {
			if _, ok := err.(*mtp.MissingNodeError); !ok {
				return err
			}
		}

		oldCandidateAddrs := []*utils.Address{}
		if err := rlp.DecodeBytes(v, &oldCandidateAddrs); err != nil {
			return err
		}
		for index, oldCandidateAddr := range oldCandidateAddrs {
			if bytes.Equal(oldCandidateAddr.Bytes(), candidate) {
				oldCandidateAddrs = append(oldCandidateAddrs[:index], oldCandidateAddrs[index+1:]...)
				if len(oldCandidateAddrs) == 0 {
					err = d.voteTrie.TryDelete(delegator)
				} else {
					if candidate, err := rlp.EncodeToBytes(oldCandidateAddrs); err != nil {
						return err
					} else {
						err = d.voteTrie.TryUpdate(delegator, candidate)
					}
				}
				if err != nil {
					if _, ok := err.(*mtp.MissingNodeError); !ok {
						return err
					}
				}

			}
		}

	}
	return nil
}

func (d *DposContext) BecomeCandidate(candidateAddr utils.Address) error {
	candidate := candidateAddr.Bytes()
	candidateInfo := &CandidateInfo{
		Addr:   candidateAddr,
		Weight: 100,
	}
	val, err := rlp.EncodeToBytes(candidateInfo)
	if err != nil {
		return err
	}
	return d.candidateTrie.TryUpdate(candidate, val)
}

func (d *DposContext) Delegate(delegatorAddr utils.Address, candidateAddrs []*utils.Address) error {
	delegator := delegatorAddr.Bytes()
	for _, candidateAddr := range candidateAddrs {
		candidate := candidateAddr.Bytes()
		// the candidate must be candidate
		candidateInTrie, err := d.candidateTrie.TryGet(candidate)
		if err != nil {
			return err
		}
		if candidateInTrie == nil {
			return errors.New("invalid candidate to delegate")
		}
	}

	// delete old candidate if exists
	oldCandidate, err := d.voteTrie.TryGet(delegator)
	if err != nil {
		if _, ok := err.(*mtp.MissingNodeError); !ok {
			return err
		}
	}
	if oldCandidate != nil {
		oldCandidateAddrs := []*utils.Address{}
		if err := rlp.DecodeBytes(oldCandidate, &oldCandidateAddrs); err != nil {
			return err
		}
		for _, oldCandidateAddr := range oldCandidateAddrs {
			d.delegateTrie.Delete(append(oldCandidateAddr.Bytes(), delegator...))
		}
	}

	for _, candidateAddr := range candidateAddrs {
		if err = d.delegateTrie.TryUpdate(append(candidateAddr.Bytes(), delegator...), delegator); err != nil {
			return err
		}
	}

	candidate, err := rlp.EncodeToBytes(candidateAddrs)
	if err != nil {
		return err
	}
	return d.voteTrie.TryUpdate(delegator, candidate)
}

func (d *DposContext) UnDelegate(delegatorAddr utils.Address, candidateAddrs []*utils.Address) error {
	delegator := delegatorAddr.Bytes()
	for _, candidateAddr := range candidateAddrs {
		candidate := candidateAddr.Bytes()
		// the candidate must be candidate
		candidateInTrie, err := d.candidateTrie.TryGet(candidate)
		if err != nil {
			return err
		}
		if candidateInTrie == nil {
			return errors.New("invalid candidate to undelegate")
		}
	}

	candidate, err := rlp.EncodeToBytes(candidateAddrs)
	if err != nil {
		return err
	}

	oldCandidate, err := d.voteTrie.TryGet(delegator)
	if err != nil {
		return err
	}

	if len(candidateAddrs) != 0 && !bytes.Equal(candidate, oldCandidate) {
		return errors.New("mismatch candidate to undelegate")
	}

	oldCandidateAddrs := []*utils.Address{}
	if err := rlp.DecodeBytes(oldCandidate, &oldCandidateAddrs); err != nil {
		return err
	}
	for _, oldCandidateAddr := range oldCandidateAddrs {
		if err = d.delegateTrie.TryDelete(append(oldCandidateAddr.Bytes(), delegator...)); err != nil {
			return err
		}
	}

	return d.voteTrie.TryDelete(delegator)
}

func (d *DposContext) CommitTo(dbw *mtp.Database) (*DposContextProto, error) {
	epochRoot, err := d.epochTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	delegateRoot, err := d.delegateTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}

	voteRoot, err := d.voteTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	candidateRoot, err := d.candidateTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	mintCntRoot, err := d.mintCntTrie.CommitTo(dbw)
	if err != nil {
		return nil, err
	}
	if err := dbw.Commit(epochRoot, false); err != nil {
		return nil, err
	}
	if err := dbw.Commit(delegateRoot, false); err != nil {
		return nil, err
	}
	if err := dbw.Commit(voteRoot, false); err != nil {
		return nil, err
	}
	if err := dbw.Commit(candidateRoot, false); err != nil {
		return nil, err
	}
	if err := dbw.Commit(mintCntRoot, false); err != nil {
		return nil, err
	}
	// fmt.Println("===Debug=====")
	// fmt.Println("===CommitTo epochRoot 		===>", epochRoot.Hex())
	// fmt.Println("===CommitTo delegateRoot	===>", delegateRoot.Hex())
	// fmt.Println("===CommitTo voteRoot		===>", voteRoot.Hex())
	// fmt.Println("===CommitTo candidateRoot	===>", candidateRoot.Hex())
	// fmt.Println("===CommitTo mintCntRoot		===>", mintCntRoot.Hex())

	return &DposContextProto{
		EpochHash:     epochRoot,
		DelegateHash:  delegateRoot,
		VoteHash:      voteRoot,
		CandidateHash: candidateRoot,
		MintCntHash:   mintCntRoot,
	}, nil
}

func (d *DposContext) CandidateTrie() *mtp.Trie          { return d.candidateTrie }
func (d *DposContext) DelegateTrie() *mtp.Trie           { return d.delegateTrie }
func (d *DposContext) VoteTrie() *mtp.Trie               { return d.voteTrie }
func (d *DposContext) EpochTrie() *mtp.Trie              { return d.epochTrie }
func (d *DposContext) MintCntTrie() *mtp.Trie            { return d.mintCntTrie }
func (d *DposContext) DB() *mtp.Database                 { return d.db }
func (dc *DposContext) SetEpoch(epoch *mtp.Trie)         { dc.epochTrie = epoch }
func (dc *DposContext) SetDelegate(delegate *mtp.Trie)   { dc.delegateTrie = delegate }
func (dc *DposContext) SetVote(vote *mtp.Trie)           { dc.voteTrie = vote }
func (dc *DposContext) SetCandidate(candidate *mtp.Trie) { dc.candidateTrie = candidate }
func (dc *DposContext) SetMintCnt(mintCnt *mtp.Trie)     { dc.mintCntTrie = mintCnt }

func (dc *DposContext) GetCandidates() ([]utils.Address, error) {
	candidates := []utils.Address{}
	candidateIterator := mtp.NewIterator(dc.candidateTrie.NodeIterator(nil))
	for candidateIterator.Next() {
		candidateInfo := &CandidateInfo{}
		err := rlp.DecodeBytes(candidateIterator.Value, candidateInfo)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidateInfo.Addr)
	}
	return candidates, nil
}

func (dc *DposContext) GetDelegators(candidate utils.Address) ([]utils.Address, error) {
	delegators := []utils.Address{}
	iter := mtp.NewIterator(dc.delegateTrie.PrefixIterator(candidate.Bytes()))
	for iter.Next() {
		delegator := iter.Value
		delegators = append(delegators, utils.BytesToAddress(delegator))
	}
	return delegators, nil
}

func (dc *DposContext) GetValidators() ([]utils.Address, error) {
	var validators []utils.Address
	key := []byte("validator")
	validatorsRLP := dc.epochTrie.Get(key)
	if err := rlp.DecodeBytes(validatorsRLP, &validators); err != nil {
		return nil, fmt.Errorf("failed to decode validators: %s", err)
	}
	return validators, nil
}

func (dc *DposContext) SetValidators(validators []utils.Address) error {
	key := []byte("validator")
	validatorsRLP, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return fmt.Errorf("failed to encode validators to rlp bytes: %s", err)
	}
	dc.epochTrie.Update(key, validatorsRLP)
	return nil
}

func (dc *DposContext) IsDpos() bool {
	var validators []utils.Address
	key := []byte("validator")
	validatorsRLP := dc.epochTrie.Get(key)
	if err := rlp.DecodeBytes(validatorsRLP, &validators); err != nil {
		return false
	}
	if len(validators) == 1 {
		return false
	}
	return true
}
