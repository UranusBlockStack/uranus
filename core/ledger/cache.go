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

	lockcache "github.com/UranusBlockStack/uranus/common/cache"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

type cache struct {
	tdCache, txsCache, blockCache, futureBlocks *lockcache.Cache
}

func newCache(cfg *Config) *cache {
	cfg.check()
	tdCache, _ := lockcache.New(cfg.tdCacheLimit)
	txsCache, _ := lockcache.New(cfg.txsCacheLimit)
	blockCache, _ := lockcache.New(cfg.blockCacheLimit)
	futureBlocks, _ := lockcache.New(cfg.futureBlockLimit)
	return &cache{
		tdCache,
		txsCache,
		blockCache,
		futureBlocks,
	}
}

func (c *cache) getTd(hash utils.Hash) *big.Int {
	if td, ok := c.tdCache.Get(hash); ok {
		return td.(*big.Int)
	}
	return nil
}

func (c *cache) tdAdd(hash utils.Hash, td *big.Int) bool {
	return c.tdCache.Add(hash, td)
}

func (c *cache) getTxs(hash utils.Hash) types.Transactions {
	if txs, ok := c.txsCache.Get(hash); ok {
		return txs.(types.Transactions)
	}
	return nil
}

func (c *cache) txsAdd(hash utils.Hash, txs types.Transactions) bool {
	return c.txsCache.Add(hash, txs)
}

func (c *cache) getBlock(hash utils.Hash) *types.Block {
	if block, ok := c.blockCache.Get(hash); ok {
		return block.(*types.Block)
	}
	return nil
}

func (c *cache) blockAdd(hash utils.Hash, block *types.Block) bool {
	return c.blockCache.Add(hash, block)
}

func (c *cache) getFutureBlock(hash utils.Hash) *types.Block {
	if block, ok := c.futureBlocks.Get(hash); ok {
		return block.(*types.Block)
	}
	return nil
}

func (c *cache) getAllFutureBlock() types.Blocks {
	blocks := make([]*types.Block, 0, c.futureBlocks.Len())
	for _, hash := range c.futureBlocks.Keys() {
		if block, exist := c.futureBlocks.Peek(hash); exist {
			blocks = append(blocks, block.(*types.Block))
		}
	}
	return blocks
}

func (c *cache) futureBlockAdd(hash utils.Hash, block *types.Block) bool {
	return c.futureBlocks.Add(hash, block)
}

func (c *cache) removeFutureBlock(hash utils.Hash) {
	c.futureBlocks.Remove(hash)
}

func (c *cache) cleanAll() {
	c.txsCache.Purge()
	c.tdCache.Purge()
	c.blockCache.Purge()
	c.futureBlocks.Purge()
}
