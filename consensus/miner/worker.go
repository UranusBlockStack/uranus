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

package miner

import (
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/executor"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/params"
)

type Work struct {
	config *params.ChainConfig
	Height uint64
	Block  *types.Block
	signer types.Signer

	txs      []*types.Transaction
	receipts []*types.Receipt
	gasUsed  *uint64
	state    *state.StateDB
	tcount   int // tx count in cycle
}

func NewWork(blk *types.Block, height uint64, state *state.StateDB) *Work {
	return &Work{
		Block:   blk,
		Height:  height,
		state:   state,
		gasUsed: new(uint64),
		signer:  types.Signer{},
	}
}

func (w *Work) applyTransactions(blockchain consensus.IBlockChain, txs *types.TransactionsByPriceAndNonce) error {
	gp := new(utils.GasPool).AddGas(w.Block.BlockHeader().GasLimit)
	var (
		coalescedLogs []*types.Log
	)
	for {
		// If we don't have enough gas for any further transactions then we're done
		if gp.Gas() < params.TxGas {
			log.Debugf("Not enough gas for further transactions gp: %v", gp)
			break
		}
		// Retrieve the next transaction and abort if all done
		tx := txs.Peek()
		if tx == nil {
			break
		}

		from, _ := tx.Sender(w.signer)

		// Check whether the tx is replay protected.
		ok, err := tx.Protected(w.signer)
		if err != nil || ok {
			log.Debugf("Ignoring reply protected transaction hash: %v", tx.Hash())
			txs.Pop()
			continue
		}

		// Start executing the transaction
		w.state.Prepare(tx.Hash(), utils.Hash{}, w.tcount)

		err, logs := w.commitTransaction(tx, blockchain, w.gasUsed, gp)
		switch err {
		case utils.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Debugf("Gas limit exceeded for current block sender: %v", from)
			txs.Pop()

		case executor.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Debugf("Skipping transaction with low nonce sender: %v, nonce: %v", from, tx.Nonce())
			txs.Shift()

		case executor.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Debugf("Skipping account with hight nonce sender: %v, nonce: %v", from, tx.Nonce())
			txs.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debugf("Transaction failed, account skipped hash: %v, err: %v", tx.Hash(), err)
			txs.Shift()
		}
	}
	return nil
}

func (w *Work) commitTransaction(tx *types.Transaction, bc consensus.IBlockChain, GasUsed *uint64, gp *utils.GasPool) (error, []*types.Log) {
	snap := w.state.Snapshot()
	receipt, _, err := bc.ExecTransaction(&w.Block.BlockHeader().Miner, gp, w.state, w.Block.BlockHeader(), tx, GasUsed, vm.Config{})
	if err != nil {
		w.state.RevertToSnapshot(snap)
		return err, nil
	}
	w.txs = append(w.txs, tx)
	w.receipts = append(w.receipts, receipt)

	return nil, receipt.Logs
}
