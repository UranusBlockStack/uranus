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
)

type MinerBakend struct {
	u *Uranus
}

// Pending returns txpool pending transactions
func (m *MinerBakend) Pending() (map[utils.Address]types.Transactions, error) {
	return m.u.txPool.Pending()
}

// GetCurrentInfo get blockchain current info
func (m *MinerBakend) GetCurrentInfo() (*types.Block, *state.StateDB, error) {
	return m.u.blockchain.GetCurrentInfo()
}

// WriteBlockWithState write block and state in chain
func (m *MinerBakend) WriteBlockWithState(block *types.Block, receipts types.Receipts, state *state.StateDB) (bool, error) {
	return m.u.blockchain.WriteBlockWithState(block, receipts, state)
}

// ExecTransaction exectue transaction return receipt
func (m *MinerBakend) ExecTransaction(author *utils.Address,
	gp *utils.GasPool, statedb *state.StateDB, header *types.BlockHeader,
	tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
	return m.u.blockchain.ExecTransaction(author, gp, statedb, header, tx, usedGas, cfg)
}
