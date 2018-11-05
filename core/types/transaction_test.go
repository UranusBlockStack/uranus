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
	to     = utils.HexToAddress("0x970e8128ab834e8eac17ab8e3812f010678cf791")
	testTx = NewTransaction(
		Binary,
		3,
		&to,
		big.NewInt(10),
		2000,
		big.NewInt(1),
		utils.FromHex("55"),
	)
)

func TestTxEncodeAndDecode(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	utils.AssertEquals(t, txb, utils.FromHex("de8003018207d094970e8128ab834e8eac17ab8e3812f010678cf7910a5580"))

	tmpTx, err := decodeTx(txb)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	utils.AssertEquals(t, tmpTx.Nonce(), testTx.Nonce())
	utils.AssertEquals(t, tmpTx.To().Hex(), testTx.To().Hex())
	utils.AssertEquals(t, tmpTx.Value().Int64(), testTx.Value().Int64())
	utils.AssertEquals(t, tmpTx.Gas(), testTx.Gas())
	utils.AssertEquals(t, tmpTx.GasPrice().Int64(), testTx.GasPrice().Int64())
	utils.AssertEquals(t, tmpTx.Payload(), testTx.Payload())

}

func decodeTx(data []byte) (*Transaction, error) {
	var tx Transaction
	return &tx, rlp.Decode(bytes.NewReader(data), &tx)
}
