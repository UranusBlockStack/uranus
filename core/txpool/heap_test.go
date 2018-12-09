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
	"sort"
	"testing"

	"github.com/UranusBlockStack/uranus/common/utils"
)

func TestNonceHeap(t *testing.T) {
	var nh nonceHeap
	array := []uint64{2, 1, 4, 3}
	for _, v := range array {
		nh.Push(v)
	}
	for i := 0; i < 4; i++ {
		utils.AssertEquals(t, array[3-i], nh.Pop().(uint64))
	}

	//test sort
	sortarray := []uint64{4, 3, 2, 1}

	for _, v := range array {
		nh.Push(v)
	}
	sort.Sort(nh)
	for i := 0; i < 4; i++ {
		utils.AssertEquals(t, sortarray[i], nh.Pop().(uint64))
	}
}

func TestPriceHeap(t *testing.T) {
	var ph priceHeap
	pns := []*priceNonce{
		&priceNonce{hash: utils.BytesToHash([]byte("2")), nonce: 2, price: big.NewInt(200)},
		&priceNonce{hash: utils.BytesToHash([]byte("1")), nonce: 1, price: big.NewInt(200)},
		&priceNonce{hash: utils.BytesToHash([]byte("4")), nonce: 4, price: big.NewInt(400)},
		&priceNonce{hash: utils.BytesToHash([]byte("3")), nonce: 3, price: big.NewInt(100)},
	}
	for _, v := range pns {
		ph.Push(v)
	}
	for i := 0; i < 4; i++ {
		utils.AssertEquals(t, pns[3-i], ph.Pop().(*priceNonce))
	}

	//test sort,first sort by price,if the price is equal,sort by nonce,high nonce is worse.
	sortPns := []*priceNonce{
		&priceNonce{hash: utils.BytesToHash([]byte("4")), nonce: 4, price: big.NewInt(400)},
		&priceNonce{hash: utils.BytesToHash([]byte("1")), nonce: 1, price: big.NewInt(200)},
		&priceNonce{hash: utils.BytesToHash([]byte("2")), nonce: 2, price: big.NewInt(200)},
		&priceNonce{hash: utils.BytesToHash([]byte("3")), nonce: 3, price: big.NewInt(100)},
	}

	for _, v := range sortPns {
		ph.Push(v)
	}
	sort.Sort(ph)
	for i := 0; i < 4; i++ {
		pn := ph.Pop().(*priceNonce)
		utils.AssertEquals(t, sortPns[i].nonce, pn.nonce)
		utils.AssertEquals(t, sortPns[i].hash, pn.hash)
		utils.AssertEquals(t, sortPns[i].price, pn.price)
	}
}

func TestTimeHeap(t *testing.T) {
	var th timeHeap

	array := []*timeActionHash{
		&timeActionHash{timestamp: big.NewInt(0)},
		&timeActionHash{timestamp: big.NewInt(2)},
		&timeActionHash{timestamp: big.NewInt(4)},
		&timeActionHash{timestamp: big.NewInt(5)},
		&timeActionHash{timestamp: big.NewInt(3)},
		&timeActionHash{timestamp: big.NewInt(1)},
	}
	for _, v := range array {
		th.Push(v)
	}

	sort.Sort(th)

	for i := 0; i < len(array); i++ {
		utils.AssertEquals(t, uint64(len(array)-1-i), th.Pop().(*timeActionHash).timestamp.Uint64())
	}
}
