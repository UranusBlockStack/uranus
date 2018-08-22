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

package rpcapi

import (
	"context"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
)

type Backend interface {
	ChainConfig() *params.ChainConfig
	CurrentBlock() *types.Block
	BlockByHeight(ctx context.Context, height BlockHeight) (*types.Block, error)
	BlockByHash(ctx context.Context, blockHash utils.Hash) (*types.Block, error)
	GetReceipts(ctx context.Context, blockHash utils.Hash) (types.Receipts, error)
	GetReceipt(ctx context.Context, txHash utils.Hash) (*types.Receipt, error)
	GetLogs(ctx context.Context, blockHash utils.Hash) ([][]*types.Log, error)
	GetTd(blockHash utils.Hash) *big.Int
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	GetPoolTransactions() (types.Transactions, error)
	GetPoolTransaction(txHash utils.Hash) *types.Transaction
	GetPoolNonce(ctx context.Context, addr utils.Address) (uint64, error)
	TxPoolStats() (pending int, queued int)
	TxPoolContent() (map[utils.Address]types.Transactions, map[utils.Address]types.Transactions)
	GetTransaction(txHash utils.Hash) *types.StorageTx
}
