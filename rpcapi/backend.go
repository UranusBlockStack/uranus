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
	"github.com/UranusBlockStack/uranus/core"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/p2p"
	"github.com/UranusBlockStack/uranus/wallet"
)

// Backend for rpc api
type Backend interface {
	// blockchain backend
	BlockChain() *core.BlockChain
	CurrentBlock() *types.Block
	BlockByHeight(ctx context.Context, height BlockHeight) (*types.Block, error)
	BlockByHash(ctx context.Context, blockHash utils.Hash) (*types.Block, error)
	GetReceipts(ctx context.Context, blockHash utils.Hash) (types.Receipts, error)
	GetReceipt(ctx context.Context, txHash utils.Hash) (*types.Receipt, error)
	GetLogs(ctx context.Context, blockHash utils.Hash) ([][]*types.Log, error)
	GetTd(blockHash utils.Hash) *big.Int
	GetTransaction(txHash utils.Hash) *types.StorageTx
	// txpool backend
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	GetPoolTransactions() (types.Transactions, error)
	GetPoolTransaction(txHash utils.Hash) *types.Transaction
	GetPoolNonce(ctx context.Context, addr utils.Address) (uint64, error)
	TxPoolStats() (pending int, queued int)
	TxPoolContent() (map[utils.Address]types.Transactions, map[utils.Address]types.Transactions)
	// wallet backend
	NewAccount(passphrase string) (*wallet.Account, error)
	Delete(address utils.Address, passphrase string) error
	Update(address utils.Address, passphrase, newPassphrase string) error
	SignTx(addr utils.Address, tx *types.Transaction, passphrase string) (*types.Transaction, error)
	Accounts() ([]utils.Address, error)
	ImportRawKey(privkey string, passphrase string) (utils.Address, error)
	// forecast backend
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	// evm
	GetEVM(ctx context.Context, from utils.Address, tx *types.Transaction, state *state.StateDB, bheader *types.BlockHeader, vmCfg vm.Config) (*vm.EVM, func() error, error)

	// p2p
	AddPeer(url string) error
	RemovePeer(url string) error
	Peers() ([]*p2p.PeerInfo, error)
	NodeInfo() (*p2p.NodeInfo, error)

	//miner
	Start(int32) error
	Stop() error
	SetCoinbase(utils.Address) error

	GetConfirmedBlockNumber() (*big.Int, error)
}
