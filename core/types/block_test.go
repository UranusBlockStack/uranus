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
	"time"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/stretchr/testify/assert"
)

var (
	testHeader = &BlockHeader{
		PreviousHash: utils.HexToHash("0x0d1729168cb1cd29427da495451152be0b1c557f4cbcc16589394d8d176cbd01"),
		Miner:        utils.HexToAddress("0x970e8128ab834e8eac17ab8e3812f010678cf791"),
		StateRoot:    utils.HexToHash("0x5ac94daa3bf60af6faf0c80a8b7378a672268fb20e0908e8c2cac7383916519b"),
		Difficulty:   big.NewInt(1000000),
		Height:       big.NewInt(100),
		GasLimit:     10000,
		GasUsed:      1000,
		TimeStamp:    big.NewInt(time.Now().Unix()),
		ExtraData:    []byte("test block"),
		Nonce:        EncodeNonce(uint64(0xa13a5a8c8f2bb1c4)),
	}
	testBlock = NewBlock(testHeader, []*Transaction{testTx}, []*Receipt{testReceipt})
)

func TestBlockEncodeAndDecode(t *testing.T) {

	blockb, err := rlp.EncodeToBytes(testBlock)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	tmpBlock, err := decodeBlock(blockb)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	assert.Equal(t, testBlock.PreviousHash(), tmpBlock.PreviousHash())
	assert.Equal(t, testBlock.Miner().Hex(), tmpBlock.Miner().Hex())
	assert.Equal(t, testBlock.StateRoot(), tmpBlock.StateRoot())
	assert.Equal(t, testBlock.TransactionsRoot(), tmpBlock.TransactionsRoot())
	assert.Equal(t, testBlock.ReceiptsRoot(), tmpBlock.ReceiptsRoot())
	assert.Equal(t, testBlock.Difficulty().Int64(), tmpBlock.Difficulty().Int64())
	assert.Equal(t, testBlock.Height().Int64(), tmpBlock.Height().Int64())
	assert.Equal(t, testBlock.GasLimit(), tmpBlock.GasLimit())
	assert.Equal(t, testBlock.GasUsed(), tmpBlock.GasUsed())
	assert.Equal(t, testBlock.Time().Int64(), tmpBlock.Time().Int64())
	assert.Equal(t, testBlock.ExtraData(), tmpBlock.ExtraData())
	assert.Equal(t, testBlock.Nonce(), tmpBlock.Nonce())
	assert.Equal(t, testBlock.Transactions()[0].Hash(), tmpBlock.Transactions()[0].Hash())

}

func decodeBlock(data []byte) (*Block, error) {
	var b Block
	return &b, rlp.Decode(bytes.NewReader(data), &b)
}
