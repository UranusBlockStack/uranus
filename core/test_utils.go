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

package core

import (
	"fmt"
	"io/ioutil"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/params"
)

var (
	legitimate = 1
	fork       = 2
)

func newLegitimate(engine consensus.Engine, n int) (db.Database, *BlockChain, error) {

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, nil, err
	}

	ldb, err := db.NewLDB(dir, 0, 0)
	if err != nil {
		return nil, nil, err
	}

	genesis, statedb, err := new(ledger.Genesis).Commit(ledger.NewChain(ldb))
	if err != nil {
		return ldb, nil, err
	}

	bc, err := NewBlockChain(nil, params.TestChainConfig, statedb, ldb, engine, &vm.Config{})
	if err != nil {
		return ldb, bc, err
	}

	blocks := makeBlocks(genesis, n, engine, ldb, legitimate)

	if _, err := bc.InsertChain(blocks); err != nil {
		return ldb, bc, err
	}
	return ldb, bc, nil
}

func makeBlocks(parent *types.Block, n int, engine consensus.Engine, db db.Database, seed int) []*types.Block {
	blocks, _ := generateChainBlocks(params.TestChainConfig, parent, engine, db, n, func(i int, header *types.BlockHeader) {
		header.Miner = utils.Address{0: byte(seed), 19: byte(i)}
	})
	return blocks
}

func makeHeader(parent *types.Block, state *state.StateDB, engine consensus.Engine) *types.BlockHeader {
	var time *big.Int
	if parent.Time() == nil {
		time = big.NewInt(10)
	} else {
		time = new(big.Int).Add(parent.Time(), big.NewInt(10)) // block time is fixed at 10 seconds
	}

	return &types.BlockHeader{
		PreviousHash: parent.Hash(),
		Miner:        parent.Miner(),
		Difficulty: engine.CalcDifficulty(nil, params.TestChainConfig, time.Uint64(), &types.BlockHeader{
			Height:     parent.Height(),
			TimeStamp:  new(big.Int).Sub(time, big.NewInt(10)),
			Difficulty: parent.Difficulty(),
		}),
		GasLimit:  types.CalcGasLimit(parent),
		Height:    new(big.Int).Add(parent.Height(), big.NewInt(1)),
		TimeStamp: time,
	}
}

func generateChainBlocks(config *params.ChainConfig, parent *types.Block, engine consensus.Engine, db db.Database, n int, gen func(int, *types.BlockHeader)) ([]*types.Block, []types.Receipts) {
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) (*types.Block, types.Receipts) {

		blockchain, err := NewBlockChain(nil, config, statedb.Database(), db, engine, &vm.Config{})
		if err != nil {
			panic(err)
		}
		defer blockchain.Stop()

		header := makeHeader(parent, statedb, engine)

		if gen != nil {
			gen(i, header)
			statedb.AddBalance(header.Miner, params.BlockReward)
			header.StateRoot = statedb.IntermediateRoot(false)
		}

		block := types.NewBlock(header, nil, nil, nil)

		// Write state changes to db
		root, err := statedb.Commit(true)
		if err != nil {
			panic(fmt.Sprintf("state write error: %v", err))
		}
		if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
			panic(fmt.Sprintf("trie write error: %v", err))
		}
		return block, nil
	}

	for i := 0; i < n; i++ {
		statedb, err := state.New(parent.StateRoot(), state.NewDatabase(db))
		if err != nil {
			panic(err)
		}
		block, receipt := genblock(i, parent, statedb)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
	}
	return blocks, receipts
}
