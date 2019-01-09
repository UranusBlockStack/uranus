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
	"math/big"
	"sync/atomic"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

// Action represents the Redeem transaction deferred Action.
type Action struct {
	TxHash       utils.Hash    `json:"txHash"`
	Sender       utils.Address `json:"sender"`
	GenTimeStamp *big.Int      `json:"generateTime"`
	DelayDur     *big.Int      `json:"delayDuration"`

	// cache
	hash atomic.Value
}

// NewAction new action.
func NewAction(txHash utils.Hash, sender utils.Address, gen, delay *big.Int) *Action {
	return &Action{
		TxHash:       txHash,
		Sender:       sender,
		GenTimeStamp: gen,
		DelayDur:     delay,
	}
}

// Hash returns the action hash of the action.
func (a *Action) Hash() utils.Hash {
	if hash := a.hash.Load(); hash != nil {
		return hash.(utils.Hash)
	}
	hash := rlpHash(a)
	a.hash.Store(hash)
	return hash
}

// Actions .
type Actions []*Action

// Len returns the number of actions in this list.
func (a Actions) Len() int { return len(a) }

// GetRlp returns the RLP encoding of one action from the list.
func (a Actions) GetRlp(i int) []byte {
	bytes, err := rlp.EncodeToBytes(a[i])
	if err != nil {
		panic(err)
	}
	return bytes
}
