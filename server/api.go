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

package server

import (
	"context"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
	"github.com/UranusBlockStack/uranus/rpcapi"
)

// API implements node all apis.
type APIBackend struct {
	u *Uranus
}

// ChainConfig returns the active chain configuration.
func (api *APIBackend) ChainConfig() *params.ChainConfig {
	return api.u.chainConfig
}

// CurrentBlock returns blockchain current block.
func (api *APIBackend) CurrentBlock() *types.Block {
	return api.u.blockchain.CurrentBlock()
}

//BlockByHeight returns block by block height.
func (api *APIBackend) BlockByHeight(ctx context.Context, height rpcapi.BlockHeight) (*types.Block, error) {
	// Pending block is only known by the miner
	if height == rpcapi.PendingBlockHeight {
	}
	// Otherwise resolve and return the block
	if height == rpcapi.LatestBlockHeight {
		return api.u.blockchain.CurrentBlock(), nil
	}
	return api.u.blockchain.GetBlockByHeight(uint64(height)), nil
}

// BlockByHash returns Block by block hash.
func (api *APIBackend) BlockByHash(ctx context.Context, blockHash utils.Hash) (*types.Block, error) {
	return api.u.blockchain.GetBlockByHash(blockHash), nil
}

// GetReceipts returns receipte by block hash.
func (api *APIBackend) GetReceipts(ctx context.Context, blockHash utils.Hash) (types.Receipts, error) {
	return api.u.blockchain.GetReceipts(blockHash), nil
}

// GetReceipt returns receipte by tx hash.
func (api *APIBackend) GetReceipt(ctx context.Context, txHash utils.Hash) (*types.Receipt, error) {
	return api.u.blockchain.GetReceipt(txHash), nil
}

// GetLogs return Logs by block hash.
func (api *APIBackend) GetLogs(ctx context.Context, blockHash utils.Hash) ([][]*types.Log, error) {
	receipts := api.u.blockchain.GetReceipts(blockHash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

// GetTd get total difficulty by block hash.
func (api *APIBackend) GetTd(blockHash utils.Hash) *big.Int {
	return api.u.blockchain.GetTd(blockHash)
}

// SendTx send signed transaction to txpool.
func (api *APIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return api.u.txPool.AddTx(signedTx)
}

// GetPoolTransactions get txpool pending transactions.
func (api *APIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := api.u.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

// GetPoolTransaction get tansaction by hash from txpool.
func (api *APIBackend) GetPoolTransaction(txHash utils.Hash) *types.Transaction {
	return api.u.txPool.Get(txHash)
}

// GetTransaction get tansaction by hash from chain db.
func (api *APIBackend) GetTransaction(txHash utils.Hash) *types.StorageTx {
	return api.u.blockchain.GetTransactionByHash(txHash)
}

// GetPoolNonce get txpool nonce by address.
func (api *APIBackend) GetPoolNonce(ctx context.Context, addr utils.Address) (uint64, error) {
	return api.u.txPool.State().GetNonce(addr), nil
}

// TxPoolStats get transaction pool stats.
func (api *APIBackend) TxPoolStats() (pending int, queued int) {
	return api.u.txPool.Stats()
}

// TxPoolContent get transaction pool content.
func (api *APIBackend) TxPoolContent() (map[utils.Address]types.Transactions, map[utils.Address]types.Transactions) {
	return api.u.txPool.Content()
}
