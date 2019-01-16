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
	"errors"
	"fmt"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/math"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus/dpos"
	"github.com/UranusBlockStack/uranus/core"
	"github.com/UranusBlockStack/uranus/core/executor"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/p2p"
	"github.com/UranusBlockStack/uranus/p2p/discover"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"github.com/UranusBlockStack/uranus/server/forecast"
	"github.com/UranusBlockStack/uranus/wallet"
)

// APIBackend implements node all apis.
type APIBackend struct {
	u   *Uranus
	gp  *forecast.Forecast
	srv *p2p.Server
}

// BlockChain return core blockchian.
func (api *APIBackend) BlockChain() *core.BlockChain {
	return api.u.BlockChain()
}

// CurrentBlock returns blockchain current block.
func (api *APIBackend) CurrentBlock() *types.Block {
	return api.u.blockchain.CurrentBlock()
}

//BlockByHeight returns block by block height.
func (api *APIBackend) BlockByHeight(ctx context.Context, height rpcapi.BlockHeight) (*types.Block, error) {
	// Pending block is only known by the miner
	if height == rpcapi.PendingBlockHeight {
		block := api.u.miner.PendingBlock()
		if block != nil {
			return block, nil
		}
		return nil, errors.New("Miner no have pending block")
	}
	// Otherwise resolve and return the block
	if height == rpcapi.LatestBlockHeight {
		return api.u.blockchain.CurrentBlock(), nil
	}

	if height < -2 {
		return nil, errors.New("block height must >= -2")
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

// NewAccount creates a new account
func (api *APIBackend) NewAccount(passphrase string) (wallet.Account, error) {
	return api.u.wallet.NewAccount(passphrase)
}

// Delete removes the speciified account
func (api *APIBackend) Delete(address utils.Address, passphrase string) error {
	acc, err := api.u.wallet.Find(address)
	if err != nil {
		return err
	}
	return api.u.wallet.Delete(acc, passphrase)
}

// Update update the specified account
func (api *APIBackend) Update(address utils.Address, passphrase, newPassphrase string) error {
	acc, err := api.u.wallet.Find(address)
	if err != nil {
		return err
	}
	return api.u.wallet.Update(acc, passphrase, newPassphrase)
}

// SignTx sign the specified transaction
func (api *APIBackend) SignTx(addr utils.Address, tx *types.Transaction, passphrase string) (*types.Transaction, error) {
	return api.u.wallet.SignTx(addr, tx, passphrase)
}

// Accounts list all wallet accounts.
func (api *APIBackend) Accounts() (wallet.Accounts, error) {
	return api.u.wallet.Accounts()
}

// ImportRawKey import raw key intfo wallet.
func (api *APIBackend) ImportRawKey(privkey string, passphrase string) (utils.Address, error) {
	return api.u.wallet.ImportRawKey(privkey, passphrase)
}

// ExportRawKey return key hex.
func (api *APIBackend) ExportRawKey(addr utils.Address, passphrase string) (string, error) {
	return api.u.wallet.ExportRawKey(addr, passphrase)
}

// SuggestGasPrice suggest gas price
func (api *APIBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return api.gp.SuggestPrice(ctx)
}

func (api *APIBackend) GetEVM(ctx context.Context, from utils.Address, tx *types.Transaction, state *state.StateDB, bheader *types.BlockHeader, vmCfg vm.Config) (*vm.EVM, func() error, error) {

	state.SetBalance(from, math.MaxBig256)
	vmError := func() error { return nil }

	context := vm.Context{
		CanTransfer: executor.CanTransfer,
		Transfer:    executor.Transfer,
		Origin:      from,
		Coinbase:    utils.Address{},
		BlockNumber: new(big.Int).Set(bheader.Height),
		Time:        new(big.Int).Set(bheader.TimeStamp),
		Difficulty:  new(big.Int).Set(bheader.Difficulty),
		GasLimit:    bheader.GasLimit,
		GasPrice:    new(big.Int).Set(tx.GasPrice()),
	}
	context.GetHash = func(n uint64) utils.Hash {
		for header := api.u.BlockChain().Ledger.GetHeader(bheader.PreviousHash); header != nil; header = api.u.BlockChain().Ledger.GetHeader(header.PreviousHash) {
			if n == header.Height.Uint64()-1 {
				return header.PreviousHash
			}
		}
		return utils.Hash{}
	}

	return vm.NewEVM(context, state, api.u.chainConfig, vmCfg), vmError, nil
}

func (api *APIBackend) AddPeer(url string) error {
	node, err := discover.ParseNode(url)
	if err != nil {
		return fmt.Errorf("invalid enode: %v", err)
	}
	api.srv.AddPeer(node)
	return nil
}
func (api *APIBackend) RemovePeer(url string) error {
	node, err := discover.ParseNode(url)
	if err != nil {
		return fmt.Errorf("invalid enode: %v", err)
	}
	api.srv.RemovePeer(node)
	return nil
}

func (api *APIBackend) Peers() ([]*p2p.PeerInfo, error) {
	return api.srv.PeersInfo(), nil
}

func (api *APIBackend) NodeInfo() (*p2p.NodeInfo, error) {
	return api.srv.NodeInfo(), nil
}

func (api *APIBackend) Start(threads int32) error {
	api.u.miner.SetThreads(threads)
	return api.u.miner.Start()
}
func (api *APIBackend) Stop() error {
	api.u.miner.Stop()
	return nil
}
func (api *APIBackend) SetCoinbase(address utils.Address) error {
	api.u.miner.SetCoinBase(address)
	return nil
}
func (api *APIBackend) GetConfirmedBlockNumber() (*big.Int, error) {
	return api.u.engine.(*dpos.Dpos).GetConfirmedBlockNumber()
}

func (api *APIBackend) GetBFTConfirmedBlockNumber() (*big.Int, error) {
	return api.u.engine.(*dpos.Dpos).GetBFTConfirmedBlockNumber()
}
