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

package ledger

import (
	"math/big"
	"testing"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/stretchr/testify/assert"
)

var testCache = newCache(&Config{})

func createTestBlock(extraData string) *types.Block {
	return types.NewBlockWithBlockHeader(&types.BlockHeader{
		ExtraData:        []byte(extraData),
		TransactionsRoot: utils.Hash{},
		ReceiptsRoot:     utils.Hash{},
	})
}

func TestBlockCache(t *testing.T) {
	block := createTestBlock("test block")
	if testCache.blockAdd(block.Hash(), block) {
		tmpBlock := testCache.getBlock(block.Hash())
		assert.Equal(t, block, tmpBlock)
	}

}
func TestTxsCache(t *testing.T) {
	block := createTestBlock("test block")
	if testCache.txsAdd(block.Hash(), block.Transactions()) {
		tmptxs := testCache.getTxs(block.Hash())
		assert.Equal(t, block.Transactions(), tmptxs)
	}
}
func TestTdCache(t *testing.T) {
	td := big.NewInt(1000)
	if testCache.tdAdd(utils.BigToHash(td), td) {
		tmptd := testCache.getTd(utils.BigToHash(td))
		assert.Equal(t, td, tmptd)
	}
}

func TestFutureBlocks(t *testing.T) {
	fblock1 := createTestBlock("test future block 1 ")
	fblock2 := createTestBlock("test future block 2")
	var fbs types.Blocks
	fbs = append(append(fbs, fblock1), fblock2)
	if testCache.futureBlockAdd(fblock1.Hash(), fblock1) {
		tmpBlock1 := testCache.getFutureBlock(fblock1.Hash())
		assert.Equal(t, fblock1, tmpBlock1)
	}
	// test getAllFutureBlock
	testCache.futureBlockAdd(fblock2.Hash(), fblock2)
	tmpBlocks := testCache.getAllFutureBlock()
	assert.Equal(t, fbs, tmpBlocks)

	// test clean all
	testCache.cleanAll()
	assert.Equal(t, 0, testCache.blockCache.Len())
	assert.Equal(t, 0, testCache.txsCache.Len())
	assert.Equal(t, 0, testCache.tdCache.Len())
	assert.Equal(t, 0, testCache.futureBlocks.Len())
}
