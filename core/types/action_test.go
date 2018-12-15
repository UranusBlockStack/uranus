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
	"testing"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

var (
	txHash     = utils.HexToHash("0x317b45ef844c4108432a06a4466aca2e11720b6dc1df3e7035a065d02829eca6")
	sender     = utils.HexToAddress("0x970e8128ab834e8eac17ab8e3812f010678cf791")
	gen        = big.NewInt(1)
	delay      = big.NewInt(2)
	testAction = NewAction(txHash, sender, gen, delay)
)

func TestActionEncodeAndDecode(t *testing.T) {

	actionBytes, err := rlp.EncodeToBytes(testAction)
	if err != nil {
		t.Fatal(err)
	}

	var act Action
	rlp.Decode(bytes.NewReader(actionBytes), &act)

	utils.AssertEquals(t, act.Hash(), testAction.Hash())
	utils.AssertEquals(t, act.TxHash, testAction.TxHash)
	utils.AssertEquals(t, act.Sender, testAction.Sender)
	utils.AssertEquals(t, act.GenTimeStamp, testAction.GenTimeStamp)
	utils.AssertEquals(t, act.DelayDur, testAction.DelayDur)
}
