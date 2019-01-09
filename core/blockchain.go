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
	"sync"
	"sync/atomic"
	"time"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	exec "github.com/UranusBlockStack/uranus/core/executor"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/state"
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
	*ledger.Ledger
	config              *params.ChainConfig
	vmConfig            *vm.Config
	genesisBlock        *types.Block
	currentBlock        atomic.Value
	stateCache          state.Database // State database to reuse between imports (contains state cache)
	chainBlockFeed      feed.Feed
	chainBlockscription feed.Subscription
	validator           *blockValidator.Validator

	sideBlockFeed      feed.Feed
	SideBlockscription feed.Subscription

	executor *exec.Executor
	engine   consensus.Engine

	chainmu sync.RWMutex
	quit    chan struct{} // blockchain quit channel
}

// NewBlockChain returns a fully initialised block chain using information available in the database.
func NewBlockChain(cfg *ledger.Config, chainCfg *params.ChainConfig, statedb state.Database, db db.Database, engine consensus.Engine, vmCfg *vm.Config) (*BlockChain, error) {
	stateCache := statedb
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
		engine:     engine,
		quit:       make(chan struct{}),
	}
	bc.executor = exec.NewExecutor(chainCfg, ledger, bc, engine)

	// check chain before blockchian service start.
	if err := bc.preCheck(); err != nil {
		return nil, err
	}

	go bc.loop()
	return bc, nil
}

func (bc *BlockChain) SetAddActionInterface(tp exec.ITxPool) {
	bc.executor.SetTxPool(tp)
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
			if bc.chainBlockscription != nil {
				bc.chainBlockscription.Unsubscribe()
			}
			log.Info("blockchain service stop.")
			return
		}
	}
}

// Stop stops the blockchain service.
func (bc *BlockChain) Stop() {
	if bc.chainBlockscription != nil {
		bc.chainBlockscription.Unsubscribe()
	}
	close(bc.quit)
	log.Info("Blockchain manager stopped")
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
		bc.PostEvent(event)
	}

}

func (bc *BlockChain) PostEvent(event interface{}) {
	switch ev := event.(type) {
	case feed.BlockAndLogsEvent:
		bc.chainBlockFeed.Send(ev)

	case feed.ForkBlockEvent:
		bc.sideBlockFeed.Send(ev)
	}
}

func (bc *BlockChain) InsertChain(blocks types.Blocks) (int, error) {
	for i := 1; i < len(blocks); i++ {
		if blocks[i].Height().Uint64() != blocks[i-1].Height().Uint64()+1 || blocks[i].PreviousHash() != blocks[i-1].Hash() {
			log.Error("Non contiguous block insert", "height", blocks[i].Height(), "hash", blocks[i].Hash(),
				"parent", blocks[i].PreviousHash(), "prevheight", blocks[i-1].Height(), "prevhash", blocks[i-1].Hash())
			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, blocks[i-1].Height().Uint64(),
				blocks[i-1].Hash().Bytes()[:4], i, blocks[i].Height().Uint64(), blocks[i].Hash().Bytes()[:4], blocks[i].PreviousHash().Bytes()[:4])
		}
	}

	var n int
	for _, blk := range blocks {
		if bc.HasBlock(blk.Hash()) {
			continue
		}
		event, _, err := bc.insertChain(blk)
		if err == nil {
			n++
		} else {
			return n, err
		}
		bc.PostEvent(event)
	}
	return n, nil
}

func (bc *BlockChain) insertChain(block *types.Block) (interface{}, []*types.Log, error) {
	err := bc.validator.ValidateHeader(bc, block.BlockHeader(), true)
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

	dposContext, err := types.NewDposContextFromProto(state.Database().TrieDB(), parent.BlockHeader().DposContext)
	if err != nil {
		return nil, nil, nil, err
	}

	block.DposContext = dposContext

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
	dposcontext *types.DposContext,
	gp *utils.GasPool, statedb *state.StateDB, header *types.BlockHeader,
	tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
	return bc.executor.ExecTransaction(author, dposcontext, gp, statedb, header, tx, usedGas, cfg)
}

// ExecActions execute actions
func (bc *BlockChain) ExecActions(statedb *state.StateDB, actions []*types.Action) {
	bc.executor.ExecActions(statedb, actions)
}

//WriteBlockWithState write the block to the chain and get the status.
func (bc *BlockChain) WriteBlockWithState(block *types.Block, receipts types.Receipts, state *state.StateDB) (bool, error) {
	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()
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

	if _, err := block.DposContext.CommitTo(triedb); err != nil {
		return false, err
	}

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
			status = true
		}
	}

	bc.WriteBlockAndReceipts(block, receipts)
	if !status && !reorg {
		// Set new head.
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
	)

	// first reduce whoever is higher bound
	if oldBlock.Height().Uint64() > newBlock.Height().Uint64() {
		// reduce old chain
		for ; oldBlock != nil && oldBlock.Height().Uint64() != newBlock.Height().Uint64(); oldBlock = bc.GetBlock(oldBlock.PreviousHash()) {
			oldChain = append(oldChain, oldBlock)

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
		logFn := log.Debugf
		if len(oldChain) > 63 {
			logFn = log.Warnf
		}
		logFn("Chain split detected height: %v, hash: %v, drop: %v, dropfrom: %v, add: %v, addfrom: %v", commonBlock.Height().Uint64(), commonBlock.Hash(), len(oldChain), oldChain[0].Hash(), len(newChain), newChain[0].Hash())
	} else {
		log.Errorf("Impossible reorg, please file an issue oldnum: %v, oldhash: %v, newheight: %v, newhash: %v", oldBlock.Height(), oldBlock.Hash(), newBlock.Height(), newBlock.Hash())
	}

	// Insert the new chain, taking care of the proper incremental order
	for i := len(newChain) - 1; i >= 0; i-- {
		bc.WriteLegitimateHashAndHeadBlockHash(newChain[i].Height().Uint64(), newChain[i].Hash())
		bc.currentBlock.Store(newChain[i])
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
	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()
	currentBlock := bc.currentBlock.Load().(*types.Block)
	state, err := bc.StateAt(currentBlock.StateRoot())
	return currentBlock, state, err
}

// StateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) StateAt(root utils.Hash) (*state.StateDB, error) {
	return state.New(root, bc.stateCache)
}

// SubscribeChainBlockEvent registers a subscription of Blockfeed.
func (bc *BlockChain) SubscribeChainBlockEvent(ch chan<- feed.BlockAndLogsEvent) feed.Subscription {
	bc.chainBlockscription = bc.chainBlockFeed.Subscribe(ch)
	return bc.chainBlockscription
}

// SubscribeSideBlockEvent registers a subscription of Blockfeed.
func (bc *BlockChain) SubscribeSideBlockEvent(ch chan<- feed.ForkBlockEvent) feed.Subscription {
	bc.SideBlockscription = bc.sideBlockFeed.Subscribe(ch)
	return bc.SideBlockscription
}

func (bc *BlockChain) Config() *params.ChainConfig {
	return bc.config
}
