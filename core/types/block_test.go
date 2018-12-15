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
	testBlock = NewBlock(testHeader, []*Transaction{testTx}, []*Action{testAction}, []*Receipt{testReceipt})
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

	utils.AssertEquals(t, tmpBlock.PreviousHash(), testBlock.PreviousHash())
	utils.AssertEquals(t, tmpBlock.Miner().Hex(), testBlock.Miner().Hex())
	utils.AssertEquals(t, tmpBlock.StateRoot(), testBlock.StateRoot())
	utils.AssertEquals(t, tmpBlock.TransactionsRoot(), testBlock.TransactionsRoot())
	utils.AssertEquals(t, tmpBlock.ReceiptsRoot(), testBlock.ReceiptsRoot())
	utils.AssertEquals(t, tmpBlock.Difficulty().Int64(), testBlock.Difficulty().Int64())
	utils.AssertEquals(t, tmpBlock.Height().Int64(), testBlock.Height().Int64())
	utils.AssertEquals(t, tmpBlock.GasLimit(), testBlock.GasLimit())
	utils.AssertEquals(t, tmpBlock.GasUsed(), testBlock.GasUsed())
	utils.AssertEquals(t, tmpBlock.Time().Int64(), testBlock.Time().Int64())
	utils.AssertEquals(t, tmpBlock.ExtraData(), testBlock.ExtraData())
	utils.AssertEquals(t, tmpBlock.Nonce(), testBlock.Nonce())
	utils.AssertEquals(t, tmpBlock.Transactions()[0].Hash(), testBlock.Transactions()[0].Hash())
	utils.AssertEquals(t, tmpBlock.Actions()[0].Hash(), testBlock.Actions()[0].Hash())

}

func decodeBlock(data []byte) (*Block, error) {
	var b Block
	return &b, rlp.Decode(bytes.NewReader(data), &b)
}
