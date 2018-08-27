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

package forecast

import (
	"context"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/rpcapi"
)

// todo change max price
var maxPrice = big.NewInt(500 * 1e9)

type getBlockPricesResult struct {
	price *big.Int
	err   error
}

type bigIntArray []*big.Int

func (s bigIntArray) Len() int           { return len(s) }
func (s bigIntArray) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s bigIntArray) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type GetBlock func(ctx context.Context, height rpcapi.BlockHeight) (*types.Block, error)

// Forecast gas prices based on the content of recent blocks.
type Forecast struct {
	cfg                 *Config
	getBlockFunc        GetBlock
	lastBlockHash       atomic.Value
	lastPrice           atomic.Value
	maxEmpty, maxBlocks int

	fetchLock sync.Mutex
}

// NewForecast returns a new Forecast.
func NewForecast(f GetBlock, cfg *Config) *Forecast {
	forecast := &Forecast{
		cfg:          cfg.check(),
		getBlockFunc: f,
		maxEmpty:     cfg.BlockNum / 2,
		maxBlocks:    cfg.BlockNum * 5,
	}
	forecast.lastPrice.Store(cfg.GasPrice)
	return forecast
}

// SuggestPrice returns the recommended gas price.
func (gpf *Forecast) SuggestPrice(ctx context.Context) (*big.Int, error) {
	lastBlockHash := gpf.lastBlockHash.Load().(utils.Hash)
	lastPrice := gpf.lastPrice.Load().(*big.Int)

	block, err := gpf.getBlockFunc(ctx, rpcapi.LatestBlockHeight)
	if err != nil {
		return nil, err
	}
	blockHash := block.Hash()
	if blockHash == lastBlockHash {
		return lastPrice, nil
	}

	gpf.fetchLock.Lock()
	defer gpf.fetchLock.Unlock()

	// try checking the cache again, maybe the last fetch fetched what we need
	lastBlockHash = gpf.lastBlockHash.Load().(utils.Hash)
	lastPrice = gpf.lastPrice.Load().(*big.Int)
	if blockHash == lastBlockHash {
		return lastPrice, nil
	}

	blockHeight := block.Height().Uint64()
	ch := make(chan getBlockPricesResult, gpf.cfg.BlockNum)
	sent := 0
	exp := 0
	var blockPrices []*big.Int
	for sent < gpf.cfg.BlockNum && blockHeight > 0 {
		go gpf.getBlockPrices(ctx, block, ch)
		sent++
		exp++
		blockHeight--
	}
	maxEmpty := gpf.maxEmpty
	for exp > 0 {
		res := <-ch
		if res.err != nil {
			return lastPrice, res.err
		}
		exp--
		if res.price != nil {
			blockPrices = append(blockPrices, res.price)
			continue
		}
		if maxEmpty > 0 {
			maxEmpty--
			continue
		}
		if blockHeight > 0 && sent < gpf.maxBlocks {
			go gpf.getBlockPrices(ctx, block, ch)
			sent++
			exp++
			blockHeight--
		}
	}
	price := lastPrice
	if len(blockPrices) > 0 {
		sort.Sort(bigIntArray(blockPrices))
		price = blockPrices[(len(blockPrices)-1)*gpf.cfg.Percent/100]
	}
	if price.Cmp(maxPrice) > 0 {
		price = new(big.Int).Set(maxPrice)
	}

	gpf.lastBlockHash.Store(blockHash)
	gpf.lastPrice.Store(price)
	return price, nil
}

// getBlockPrices calculates the lowest transaction gas price in a given block
// and sends it to the result channel. If the block is empty, price is nil.
func (gpf *Forecast) getBlockPrices(ctx context.Context, block *types.Block, ch chan getBlockPricesResult) {
	txs := make([]*types.Transaction, len(block.Transactions()))
	copy(txs, block.Transactions())
	sort.Sort(types.TxsByPriceToHigh(txs))

	for _, tx := range txs {
		sender, err := tx.Sender(types.Signer{})
		if err == nil && sender != block.Miner() {
			ch <- getBlockPricesResult{tx.GasPrice(), nil}
			return
		}
	}
	ch <- getBlockPricesResult{nil, nil}
}
