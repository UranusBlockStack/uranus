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

package txpool

import (
	"math/big"

	"github.com/UranusBlockStack/uranus/common/utils"
)

// Sort primarily by price, returning the cheaper one
type priceNonce struct {
	hash  utils.Hash
	price *big.Int
	nonce uint64
}

type priceHeap []*priceNonce

func (h priceHeap) Len() int      { return len(h) }
func (h priceHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h priceHeap) Less(i, j int) bool {
	switch h[i].price.Cmp(h[j].price) {
	case -1:
		return true
	case 1:
		return false
	}
	return h[i].nonce > h[j].nonce
}

func (h *priceHeap) Push(x interface{}) {
	*h = append(*h, x.(*priceNonce))
}

func (h *priceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type timeActionHash struct {
	hash      utils.Hash
	timestamp *big.Int
}

type timeHeap []*timeActionHash

func (t timeHeap) Len() int { return len(t) }
func (t timeHeap) Less(i, j int) bool {
	switch t[i].timestamp.Cmp(t[j].timestamp) {
	case 1:
		return false
	default:
		return true
	}
}
func (t timeHeap) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t *timeHeap) Push(x interface{}) {
	*t = append(*t, x.(*timeActionHash))
}
func (t *timeHeap) Pop() interface{} {
	old := *t
	n := len(old)
	x := old[n-1]
	*t = old[0 : n-1]
	return x
}
