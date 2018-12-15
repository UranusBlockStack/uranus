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
	"testing"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

func TestDeferredList(t *testing.T) {
	dl := newDeferredList()
	nilHash := utils.Hash{}
	nilAddr := utils.Address{}
	actions := []*types.Action{
		types.NewAction(nilHash, nilAddr, big.NewInt(0), nil),
		types.NewAction(nilHash, nilAddr, big.NewInt(1), nil),
		types.NewAction(nilHash, nilAddr, big.NewInt(3), nil),
		types.NewAction(nilHash, nilAddr, big.NewInt(4), nil),
		types.NewAction(nilHash, nilAddr, big.NewInt(2), nil),
	}

	for _, a := range actions {
		dl.Put(a)
	}

	threshold := big.NewInt(3)
	tmpActions := dl.Cap(threshold)
	for _, a := range tmpActions {
		if a.GenTimeStamp.Cmp(threshold) >= 0 {
			t.Fatalf("generate timestamp:%v must lower: %v \n", a.GenTimeStamp.String(), threshold.String())
		}
	}

}
