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
	"testing"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/state"
)

func TestRewindChain(t *testing.T) {
	stateCache := state.NewDatabase(db.NewMemDatabase())
	ledger := New(&Config{}, db.NewMemDatabase(), func(hash utils.Hash) bool {
		_, err := stateCache.OpenTrie(hash)
		return err == nil
	})
	genesisBlock, _ := DefaultGenesis().ToBlock(ledger.chain)
	DefaultGenesis().Commit(ledger.chain)

	ledger.CheckLastBlock(genesisBlock)

	block := ledger.GetBlockByHeight(0)

	utils.AssertEquals(t, block.Hash(), genesisBlock.Hash())
}
