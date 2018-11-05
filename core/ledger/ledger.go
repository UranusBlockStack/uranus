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
	"fmt"
	"math/big"
	"time"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

// Ledger represents the ledger in blockchain
type Ledger struct {
	cache    *cache
	chain    *Chain
	hasState func(hash utils.Hash) bool
}

// New creates a new ledger.
func New(cachecfg *Config, db db.Database, hasState func(hash utils.Hash) bool) *Ledger {
	if cachecfg == nil {
		cachecfg = new(Config)
	}
	return &Ledger{
		newCache(cachecfg),
		NewChain(db),
		hasState,
	}
}

// CheckLastBlock check the last block is right or reset chain with genesis block.
func (l *Ledger) CheckLastBlock(genesis *types.Block) *types.Block {
	// Restore the last known head block
	head := l.chain.getHeadBlockHash()
	if head == (utils.Hash{}) {
		log.Warn("Empty database, reseting chain")
		l.ResetChain(genesis)
		return genesis
	}
	// Make sure the entire head block is available
	cur := l.GetBlockByHash(head)
	if cur == nil {
		log.Warn("Head block missing, resetting chain", "hash", head)
		l.ResetChain(genesis)
		return genesis
	}
	return cur
}

// ResetChain reset chain with genesis block
func (l *Ledger) ResetChain(genesis *types.Block) {
	l.RewindChain(0)
	l.chain.putBlock(genesis)
	// clear cache
	l.cache.cleanAll()
	l.chain.putHeadBlockHash(genesis.Hash())
	return
}

// RewindChain rewind the chain, deleting all block
func (l *Ledger) RewindChain(height uint64) {
	log.Warnf("Rewinding chain target: %v", height)
	hash := l.chain.getHeadBlockHash()
	block := l.chain.getBlock(hash)
	if block.Height().Uint64() >= height {
		l.chain.deleteBlock(block.Hash())
		l.RewindChain(height)
	}
	return
}

// WriteLegitimateHashAndHeadBlockHash
func (l *Ledger) WriteLegitimateHashAndHeadBlockHash(height uint64, hash utils.Hash) {
	l.chain.putLegitimateHash(height, hash)
	l.chain.putHeadBlockHash(hash)
}

func (l *Ledger) WriteBlockAndReceipts(block *types.Block, receipts types.Receipts) {
	l.chain.putBlock(block)
	l.chain.putReceipts(block.Hash(), receipts)
}

// WriteBlockAndTd serializes a block and td into the database, header and txs separately.
func (l *Ledger) WriteBlockAndTd(block *types.Block, td *big.Int) {
	l.chain.putBlock(block)
	l.chain.putTd(block.Hash(), td)
}

func (l *Ledger) DeleteBlock(blockHash utils.Hash) {
	l.cache.blockCache.Remove(blockHash)
	l.chain.deleteBlock(blockHash)
}

// GetReceipts return Receipts by block hash.
func (l *Ledger) GetReceipts(blockHash utils.Hash) types.Receipts {
	return l.chain.getReceipts(blockHash)
}

// GetReceipt return Receipt by transaction hash.
func (l *Ledger) GetReceipt(txHash utils.Hash) *types.Receipt {
	return l.chain.getReceipt(txHash)
}

// GetFutureBlock get all future from ledger cache .
func (l *Ledger) GetFutureBlock() types.Blocks {
	return l.cache.getAllFutureBlock()
}

//PutFutureBlock put a block in future cache
func (l *Ledger) PutFutureBlock(block *types.Block) error {
	max := big.NewInt(time.Now().Unix() + maxTimeFutureBlocks)
	if block.Time().Cmp(max) > 0 {
		return fmt.Errorf("future block: %v > %v", block.Time(), max)
	}
	l.cache.futureBlockAdd(block.Hash(), block)
	return nil
}

// RemoveFutureBlock remove future block by hash
func (l *Ledger) RemoveFutureBlock(hash utils.Hash) {
	l.cache.removeFutureBlock(hash)
}

// HasFutureBlock checks if a future block in cache.
func (l *Ledger) HasFutureBlock(hash utils.Hash) bool {
	return l.cache.futureBlocks.Contains(hash)
}

// HasHeader checks if a block header is present in the database or not, caching it if present.
func (l *Ledger) HasHeader(hash utils.Hash, height uint64) bool {
	return l.chain.getHeader(hash) != nil
}

// HasState checks if state trie is fully present in the database or not.
func (l *Ledger) HasState(hash utils.Hash) bool {
	return l.hasState(hash)
}

// HasBlock checks if a block is fully present in the database or not.
func (l *Ledger) HasBlock(hash utils.Hash) bool {
	return l.GetBlock(hash) != nil
}

// GetHeader retrieves a header from the database by hash,
func (l *Ledger) GetHeader(hash utils.Hash) *types.BlockHeader {
	block := l.cache.getBlock(hash)
	if block != nil {
		return block.BlockHeader()
	}
	return l.chain.getHeader(hash)
}

// GetDB get db database
func (l *Ledger) GetDB() db.Database {
	return l.chain.db
}

// WriteTd stores a block's total difficulty into the database, also caching it along the way.
func (l *Ledger) WriteTd(hash utils.Hash, td *big.Int) {
	l.chain.putTd(hash, td)
	l.cache.tdAdd(hash, td)
}

// GetTd retrieves a block from the database by hash,caching it if found.
func (l *Ledger) GetTd(hash utils.Hash) *big.Int {
	if td := l.cache.getTd(hash); td != nil {
		return td
	}
	td := l.chain.getTd(hash)
	if td == nil {
		return nil
	}
	l.cache.tdAdd(hash, td)
	return td
}

// GetBlock retrieves a block from the database by hash,caching it if found.
func (l *Ledger) GetBlock(hash utils.Hash) *types.Block {
	// Short circuit if the block's already in the cache, retrieve otherwise
	var block *types.Block
	if block = l.cache.getBlock(hash); block != nil {
		return block
	}
	if block = l.chain.getBlock(hash); block == nil {
		return nil
	}
	// Cache the found block for next time and return
	l.cache.blockAdd(block.Hash(), block)
	return block
}

// GetBlockByHash retrieves a block from the database by hash, caching it if found.
func (l *Ledger) GetBlockByHash(blockHash utils.Hash) *types.Block {
	return l.GetBlock(blockHash)
}

// GetBlockByHeight retrieves a block from the database by height.
func (l *Ledger) GetBlockByHeight(height uint64) *types.Block {
	hash := l.chain.getLegitimateHash(height)
	if hash == (utils.Hash{}) {
		return nil
	}
	return l.GetBlock(hash)
}

// GetTransactionByHash retrieves a transaction from the database by hash, caching it if found.
func (l *Ledger) GetTransactionByHash(txHash utils.Hash) *types.StorageTx {
	return l.chain.getTransaction(txHash)
}
