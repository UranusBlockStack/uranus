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
	"math/big"

	"github.com/UranusBlockStack/uranus/common/bloom"
	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/params"
	"golang.org/x/crypto/sha3"
)

func rlpHash(x interface{}) (h utils.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type writeCounter utils.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

// DerivableList array
type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

// DeriveRootHash root hash
func DeriveRootHash(list DerivableList) utils.Hash {
	keybuf := new(bytes.Buffer)
	trie := new(mtp.Trie)
	for i := 0; i < list.Len(); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		trie.Update(keybuf.Bytes(), list.GetRlp(i))
	}
	return trie.Hash()
}

// Transactions is is a wrapper around a Transaction array to implement DerivableList.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s Transactions) GetRlp(i int) []byte {
	enc, err := rlp.EncodeToBytes(s[i])
	if err != nil {
		panic(err)
	}

	return enc
}

// Receipts is a wrapper around a Receipt array to implement DerivableList.
type Receipts []*Receipt

// Len returns the number of receipts in this list.
func (r Receipts) Len() int { return len(r) }

// GetRlp returns the RLP encoding of one receipt from the list.
func (r Receipts) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(r[i])
	if err != nil {
		panic(err)
	}
	return bytes
}

// Logs is a wrapper around a Log array to implement DerivableList.
type Logs []*Log

// Len returns the number of logs in this list.
func (l Logs) Len() int { return len(l) }

// GetRlp returns the RLP encoding of one log from the list.
func (l Logs) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(l[i])
	if err != nil {
		panic(err)
	}
	return bytes
}

func CreateBloom(receipts Receipts) bloom.Bloom {
	bin := new(big.Int)
	for _, receipt := range receipts {
		bin.Or(bin, LogsBloom(receipt.Logs))
	}

	return bloom.BytesToBloom(bin.Bytes())
}

func LogsBloom(logs []*Log) *big.Int {
	bin := new(big.Int)
	for _, log := range logs {
		bin.Or(bin, bloom.Bloom9(log.Address.Bytes()))
		for _, b := range log.Topics {
			bin.Or(bin, bloom.Bloom9(b[:]))
		}
	}

	return bin
}

// CalcGasLimit computes the gas limit of the next block after parent.
func CalcGasLimit(parent *Block) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor
	// decay = parentGasLimit / 1024 -1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if limit < params.GenesisGasLimit {
		limit = parent.GasLimit() + decay
		if limit > params.GenesisGasLimit {
			limit = params.GenesisGasLimit
		}
	}
	return limit
}
