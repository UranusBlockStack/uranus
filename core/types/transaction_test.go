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
	"fmt"
	"math/big"
	"testing"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/stretchr/testify/assert"
)

var (
	to     = utils.HexToAddress("0x970e8128ab834e8eac17ab8e3812f010678cf791")
	testTx = NewTransaction(
		Binary,
		3,
		big.NewInt(10),
		2000,
		big.NewInt(1),
		utils.FromHex("55"),
		&to,
	)
)

func TestTxEncodeAndDecode(t *testing.T) {
	txb, err := rlp.EncodeToBytes(testTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	assert.Equal(t, utils.FromHex("df8003018207d0d594970e8128ab834e8eac17ab8e3812f010678cf7910a5580"), txb)

	tmpTx, err := decodeTx(txb)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	assert.Equal(t, testTx.Nonce(), tmpTx.Nonce())
	assert.Equal(t, testTx.To().Hex(), tmpTx.To().Hex())
	assert.Equal(t, testTx.Value().Int64(), tmpTx.Value().Int64())
	assert.Equal(t, testTx.Gas(), tmpTx.Gas())
	assert.Equal(t, testTx.GasPrice().Int64(), tmpTx.GasPrice().Int64())
	assert.Equal(t, testTx.Payload(), tmpTx.Payload())

	fmt.Println(testTx.Hash().Hex())
}

func decodeTx(data []byte) (*Transaction, error) {
	var tx Transaction
	return &tx, rlp.Decode(bytes.NewReader(data), &tx)
}
