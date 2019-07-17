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
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/feed"
	"github.com/UranusBlockStack/uranus/params"
)

type MinerBakend struct {
	u *Uranus
}

func (m *MinerBakend) Actions() []*types.Action {
	return m.u.txPool.Actions()
}

// Pending returns txpool pending transactions
func (m *MinerBakend) Pending() (map[utils.Address]types.Transactions, error) {
	return m.u.txPool.Pending()
}

func (m *MinerBakend) PostEvent(event interface{}) {
	m.u.blockchain.PostEvent(event)
}

// GetCurrentInfo get blockchain current info
func (m *MinerBakend) GetCurrentInfo() (*types.Block, *state.StateDB, error) {
	return m.u.blockchain.GetCurrentInfo()
}

// GetHeader retrieves a header from the database by hash,
func (m *MinerBakend) GetHeader(hash utils.Hash) *types.BlockHeader {
	return m.u.blockchain.GetHeader(hash)
}

// WriteBlockWithState write block and state in chain
func (m *MinerBakend) WriteBlockWithState(block *types.Block, receipts types.Receipts, state *state.StateDB) (bool, error) {
	return m.u.blockchain.WriteBlockWithState(block, receipts, state)
}

func (m *MinerBakend) ExecActions(statedb *state.StateDB, actions []*types.Action) {
	m.u.blockchain.ExecActions(statedb, actions)
}

// ExecTransaction exectue transaction return receipt
func (m *MinerBakend) ExecTransaction(author *utils.Address, dposcontext *types.DposContext,
	gp *utils.GasPool, statedb *state.StateDB, header *types.BlockHeader,
	tx *types.Transaction, usedGas *uint64, cfg vm.Config) ([]byte, *types.Receipt, uint64, error) {
	return m.u.blockchain.ExecTransaction(author, dposcontext, gp, statedb, header, tx, usedGas, cfg)
}

func (m *MinerBakend) Config() *params.ChainConfig {
	return m.u.blockchain.Config()
}
func (m *MinerBakend) CurrentBlock() *types.Block {
	return m.u.blockchain.CurrentBlock()
}
func (m *MinerBakend) GetBlockByHeight(height uint64) *types.Block {
	return m.u.blockchain.GetBlockByHeight(height)
}
func (m *MinerBakend) GetBlockByHash(hash utils.Hash) *types.Block {
	return m.u.blockchain.GetBlockByHash(hash)
}

func (m *MinerBakend) SubscribeChainBlockEvent(ch chan<- feed.BlockAndLogsEvent) feed.Subscription {
	return m.u.blockchain.SubscribeChainBlockEvent(ch)
}

func (m *MinerBakend) SubscribeNewTxsEvent(ch chan<- feed.NewTxsEvent) feed.Subscription {
	return m.u.txPool.SubscribeNewTxsEvent(ch)
}
