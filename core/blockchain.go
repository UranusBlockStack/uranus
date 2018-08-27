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
	"math/big"
	"math/rand"
	"sort"
	"sync/atomic"
	"time"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	exec "github.com/UranusBlockStack/uranus/core/executor"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/txpool"
	"github.com/UranusBlockStack/uranus/core/types"
	blockValidator "github.com/UranusBlockStack/uranus/core/validator"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/feed"
	"github.com/UranusBlockStack/uranus/params"
)

// Processor is an interface for processing blocks using a given initial state.
type Processor interface {
	Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error)
}

// BlockChain manages chain imports, reverts, chain reorganisations.
type BlockChain struct {
	config   *params.ChainConfig
	vmConfig *vm.Config

	*ledger.Ledger

	genesisBlock *types.Block
	currentBlock atomic.Value

	stateCache state.Database // State database to reuse between imports (contains state cache)
	validator  *blockValidator.Validator

	chainHeadFeed       feed.Feed
	chainBlockscription feed.Subscription

	executor *exec.Executor
	engine   consensus.Engine
	quit     chan struct{} // blockchain quit channel
}

// NewBlockChain returns a fully initialised block chain using information available in the database.
func NewBlockChain(cfg *ledger.Config, chainCfg *params.ChainConfig, db db.Database, engine consensus.Engine, vmCfg *vm.Config) (*BlockChain, error) {
	stateCache := state.NewDatabase(db)
	ledger := ledger.New(cfg, db, func(hash utils.Hash) bool {
		_, err := stateCache.OpenTrie(hash)
		return err == nil
	})
	bc := &BlockChain{
		config:     chainCfg,
		vmConfig:   vmCfg,
		stateCache: stateCache,
		Ledger:     ledger,
		validator:  blockValidator.New(ledger, engine),
		executor:   exec.NewExecutor(chainCfg, ledger, engine),
		engine:     engine,
		quit:       make(chan struct{}),
	}

	// check chain before blockchian service start.
	if err := bc.preCheck(); err != nil {
		return nil, err
	}

	go bc.loop()
	return bc, nil
}

func (bc *BlockChain) preCheck() error {
	bc.genesisBlock = bc.GetBlockByHeight(0)
	if bc.genesisBlock == nil {
		return ledger.ErrNoGenesis
	}
	bc.loadLastState()
	return nil
}

func (bc *BlockChain) loadLastState() {
	currentBlock := bc.CheckLastBlock(bc.genesisBlock)
Head:
	if _, err := state.New(currentBlock.StateRoot(), bc.stateCache); err != nil {
		log.Warnf("Head state missing, repairing chain height: %v,hash: %v", currentBlock.Height(), currentBlock.Hash())
		currentBlock = bc.GetBlock(currentBlock.PreviousHash())
		goto Head
	}

	bc.currentBlock.Store(currentBlock)
	blockTd := bc.GetTd(currentBlock.Hash())
	log.Infof("Loaded most recent local full block number: %v,hash: %v,td: %v", currentBlock.Height(), currentBlock.Hash(), blockTd)
	return
}

func (bc *BlockChain) loop() {
	futureTimer := time.NewTicker(5 * time.Second)
	defer futureTimer.Stop()
	for {
		select {
		case <-futureTimer.C:
			bc.processBlocks()
		case <-bc.quit:
			bc.chainBlockscription.Unsubscribe()
			log.Info("blockchain service stop.")
			return
		}
	}
}

func (bc *BlockChain) processBlocks() {
	blocks := bc.GetFutureBlock()
	// sort by number
	sort.Sort(blocks)

	for _, b := range blocks {
		// insert future block one by one.
		event, _, err := bc.insertChain(b)
		if err != nil {
			log.Errorf("inster chain err :%v", err)
			continue
		}
		bc.chainHeadFeed.Send(event)

	}

}

func (bc *BlockChain) postEvent(events interface{}, logs []*types.Log) {
	// todo
}

func (bc *BlockChain) InsertChain(blocks types.Blocks) (int, error) {
	n := 0
	for _, blk := range blocks {
		if bc.HasBlock(blk.Hash()) {
			continue
		}
		if _, _, err := bc.insertChain(blk); err == nil {
			n++
		} else {
			return n, err
		}
	}
	return n, nil
}

func (bc *BlockChain) insertChain(block *types.Block) (interface{}, []*types.Log, error) {
	err := bc.validator.ValidateHeader(block.BlockHeader(), bc.config, true)
	if err == nil {
		err = bc.validator.ValidateTxs(block)
	}

	switch err {
	case blockValidator.ErrKnownBlock:
		if bc.CurrentBlock().Height().Uint64() >= block.Height().Uint64() {
			log.Warnf("Block and state both already known, block heigt: %v", block.Height())
		}
		return nil, nil, blockValidator.ErrKnownBlock
	case blockValidator.ErrFutureBlock:
		return nil, nil, bc.PutFutureBlock(block)
	case blockValidator.ErrUnknownAncestor:
		if bc.HasFutureBlock(block.PreviousHash()) {
			return nil, nil, bc.PutFutureBlock(block)
		}
	case blockValidator.ErrPrunedAncestor:
		currentBlock := bc.CurrentBlock()
		localTd := bc.GetTd(currentBlock.Hash())
		externTd := new(big.Int).Add(bc.GetTd(block.PreviousHash()), block.Difficulty())
		if localTd.Cmp(externTd) > 0 {
			if err = bc.WriteBlockWithoutState(block, externTd); err != nil {
				return nil, nil, err
			}
		}
		parent := bc.GetBlock(block.PreviousHash())
		for !bc.HasState(parent.StateRoot()) {
			parent = bc.GetBlock(parent.PreviousHash())
		}
		return bc.insertChain(parent)
	}

	if err != nil {
		return nil, nil, err
	}

	receipts, logs, state, err := bc.execBlock(block)
	if err != nil {
		return nil, nil, err
	}

	forking, err := bc.WriteBlockWithState(block, receipts, state)
	if err != nil {
		return nil, nil, err
	}

	if forking {
		log.Debugf("Inserted forked block number: %v,hash: %v,diff: %v,txsLen: %v,gas: %v.", block.Height(), block.Hash(), block.Difficulty(), len(block.Transactions()), block.GasUsed())
		return feed.ForkBlockEvent{Block: block}, logs, nil
	}
	log.Debugf("Inserted new block number: %v,hash: %v,txsLen: %v,gas: %v, diff: %v", block.Height(), block.Hash(), len(block.Transactions()), block.GasUsed(), block.Difficulty())
	return feed.BlockAndLogsEvent{Block: block, Logs: logs}, logs, nil
}

func (bc *BlockChain) execBlock(block *types.Block) (types.Receipts, []*types.Log, *state.StateDB, error) {
	parent := bc.GetBlock(block.PreviousHash())

	state, err := state.New(parent.StateRoot(), bc.stateCache)
	if err != nil {
		return nil, nil, nil, err
	}
	// Process block using the parent state as reference point.
	receipts, logs, usedGas, err := bc.executor.ExecBlock(block, state, *bc.vmConfig)
	if err != nil {
		return nil, nil, nil, err
	}
	// Validate the state using the default validator
	err = bc.validator.ValidateState(block, parent, state, receipts, true, usedGas)
	if err != nil {
		return nil, nil, nil, err
	}
	return receipts, logs, state, nil
}

// ExecTransaction execute transaction and return receipts
func (bc *BlockChain) ExecTransaction(author *utils.Address,
	gp *utils.GasPool, statedb *state.StateDB, header *types.BlockHeader,
	tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
	return bc.executor.ExecTransaction(author, gp, statedb, header, tx, usedGas, cfg)
}

//WriteBlockWithState write the block to the chain and get the status.
func (bc *BlockChain) WriteBlockWithState(block *types.Block, receipts types.Receipts, state *state.StateDB) (bool, error) {
	// get the total difficulty of the block
	ptd := bc.GetTd(block.PreviousHash())
	if ptd == nil {
		return false, blockValidator.ErrUnknownAncestor
	}
	currentBlock := bc.CurrentBlock()
	localTd := bc.GetTd(currentBlock.Hash())
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// write total difficulty
	bc.WriteTd(block.Hash(), externTd)

	root, err := state.Commit(true)
	if err != nil {
		return false, err
	}
	triedb := bc.stateCache.TrieDB()

	if err := triedb.Commit(root, false); err != nil {
		return false, err
	}

	reorg := externTd.Cmp(localTd) > 0
	currentBlock = bc.CurrentBlock()
	if !reorg && externTd.Cmp(localTd) == 0 {
		reorg = block.Height().Uint64() < currentBlock.Height().Uint64() || (block.Height().Uint64() == currentBlock.Height().Uint64() && rand.Float64() < 0.5)
	}

	var status bool

	if reorg {
		// Reorganise the chain if the parent is not the head block
		if block.PreviousHash() != currentBlock.Hash() {
			if err := bc.reorg(currentBlock, block); err != nil {
				return false, err
			}
		}
		status = true
	}

	bc.WriteBlockAndReceipts(block, receipts)

	// Set new head.
	if status == true {
		bc.WriteLegitimateHashAndHeadBlockHash(block.Height().Uint64(), block.Hash())
		bc.currentBlock.Store(block)
	}
	bc.RemoveFutureBlock(block.Hash())
	return status, nil
}

// WriteBlockWithoutState writes only the block and its metadata to the database,
// but does not write any state.
func (bc *BlockChain) WriteBlockWithoutState(block *types.Block, td *big.Int) error {
	bc.WriteBlockAndTd(block, td)
	return nil
}

func (bc *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain    types.Blocks
		oldChain    types.Blocks
		commonBlock *types.Block
		deletedTxs  types.Transactions
		// deletedLogs []*types.Log
	)

	// first reduce whoever is higher bound
	if oldBlock.Height().Uint64() > newBlock.Height().Uint64() {
		// reduce old chain
		for ; oldBlock != nil && oldBlock.Height().Uint64() != newBlock.Height().Uint64(); oldBlock = bc.GetBlock(oldBlock.PreviousHash()) {
			oldChain = append(oldChain, oldBlock)
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for ; newBlock != nil && newBlock.Height().Uint64() != oldBlock.Height().Uint64(); newBlock = bc.GetBlock(newBlock.PreviousHash()) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return fmt.Errorf("Invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("Invalid new chain")
	}

	for {
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}

		oldChain = append(oldChain, oldBlock)
		newChain = append(newChain, newBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

		oldBlock, newBlock = bc.GetBlock(oldBlock.PreviousHash()), bc.GetBlock(newBlock.PreviousHash())
		if oldBlock == nil {
			return fmt.Errorf("Invalid old chain")
		}
		if newBlock == nil {
			return fmt.Errorf("Invalid new chain")
		}
	}
	// Ensure the user sees large reorgs
	if len(oldChain) > 0 && len(newChain) > 0 {
		logFn := log.Debug
		if len(oldChain) > 63 {
			logFn = log.Warn
		}
		logFn("Chain split detected", "number", commonBlock.Height(), "hash", commonBlock.Hash(),
			"drop", len(oldChain), "dropfrom", oldChain[0].Hash(), "add", len(newChain), "addfrom", newChain[0].Hash())
	} else {
		log.Error("Impossible reorg, please file an issue", "oldnum", oldBlock.Height(), "oldhash", oldBlock.Hash(), "newnum", newBlock.Height(), "newhash", newBlock.Hash())
	}
	// Insert the new chain, taking care of the proper incremental order
	var addedTxs types.Transactions
	for i := len(newChain) - 1; i >= 0; i-- {
		bc.WriteLegitimateHashAndHeadBlockHash(newChain[i].Height().Uint64(), newChain[i].Hash())
		addedTxs = append(addedTxs, newChain[i].Transactions()...)
	}
	// calculate the difference between deleted and added transactions
	diff := txpool.TxDifference(deletedTxs, addedTxs)

	// todo remove diff
	_ = diff
	// for _, tx := range diff {
	// 	rawdb.DeleteTxLookupEntry(bc.db, tx.Hash())
	// }

	if len(oldChain) > 0 {
		go func() {
			// for _, _ := range oldChain {
			// 	// 	bc.chainSideFeed.Send(feed.ForkBlockEvent{Block: block})
			// }
		}()
	}

	return nil
}

// CurrentBlock retrieves the current head block of the canonical chain.
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}

// State returns a new mutable state based on the current HEAD block.
func (bc *BlockChain) State() (*state.StateDB, error) {
	return bc.StateAt(bc.CurrentBlock().StateRoot())
}

// GetCurrentInfo return current info
func (bc *BlockChain) GetCurrentInfo() (*types.Block, *state.StateDB, error) {
	currentBlock := bc.currentBlock.Load().(*types.Block)
	state, err := bc.StateAt(currentBlock.StateRoot())
	return currentBlock, state, err
}

// StateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) StateAt(root utils.Hash) (*state.StateDB, error) {
	return state.New(root, bc.stateCache)
}

// SubscribeChainBlockEvent registers a subscription of Blockfeed.
func (bc *BlockChain) SubscribeChainBlockEvent(ch chan<- feed.BlockEvent) feed.Subscription {
	bc.chainBlockscription = bc.chainHeadFeed.Subscribe(ch)
	return bc.chainBlockscription
}
