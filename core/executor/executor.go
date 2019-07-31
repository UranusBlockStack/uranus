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

package executor

import (
	"fmt"
	"math/big"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/txpool"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/params"
)

type ITxPool interface {
	AddAction(a *types.Action)
}

// Executor is a transactions executor
type Executor struct {
	config *params.ChainConfig // Chain configuration options
	ledger *ledger.Ledger      // ledger
	tp     ITxPool
	chain  consensus.IChainReader
	engine consensus.Engine
}

// NewExecutor initialises a new Executor.
func NewExecutor(config *params.ChainConfig, l *ledger.Ledger, chain consensus.IChainReader, engine consensus.Engine) *Executor {
	return &Executor{
		config: config,
		ledger: l,
		chain:  chain,
		engine: engine,
	}
}

func (e *Executor) SetTxPool(tp ITxPool) {
	e.tp = tp
}

// ExecBlock execute block
func (e *Executor) ExecBlock(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.BlockHeader()
		allLogs  []*types.Log
		gp       = new(utils.GasPool).AddGas(block.GasLimit())
	)

	e.ExecActions(statedb, block.Actions())

	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		_, receipt, _, err := e.ExecTransaction(nil, nil, block.DposCtx(), gp, statedb, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}

	e.engine.Finalize(e.chain, header, statedb, block.Transactions(), block.Actions(), receipts, block.DposCtx())
	return receipts, allLogs, *usedGas, nil
}

// ExecActions execute actions
func (e *Executor) ExecActions(statedb *state.StateDB, actions []*types.Action) {
	for _, a := range actions {
		lb := statedb.GetUnLockedBalance(a.Sender)
		statedb.AddBalance(a.Sender, lb)
		statedb.SetUnLockedBalance(a.Sender, big.NewInt(0))
	}
}

// ExecTransaction attempts to execute a transaction to the given state database and uses the input parameters for its environment.
func (e *Executor) ExecTransaction(author, txFrom *utils.Address,
	dposContext *types.DposContext,
	gp *utils.GasPool, statedb *state.StateDB, header *types.BlockHeader,
	tx *types.Transaction, usedGas *uint64, cfg vm.Config) ([]byte, *types.Receipt, uint64, error) {
	var (
		gas    uint64
		failed bool
		err    error
		result []byte
	)

	if tx.Type() == types.Binary {

		// Create a new context to be used in the EVM environment
		context := NewEVMContext(tx, header, e.ledger, e.engine, author, txFrom)
		// Create a new environment which holds all relevant informationabout the transaction and calling mechanisms.
		vmenv := vm.NewEVM(context, statedb, e.config, cfg)
		// Apply the transaction to the current state (included in the env)
		result, gas, failed, err = ExecStateTransition(txFrom, vmenv, tx, gp)
		if err != nil {
			return nil, nil, 0, err
		}
	} else {
		var vmerr error
		gas, failed, vmerr = e.applyDposMessage(header.TimeStamp, dposContext, tx, statedb, gp)
		if vmerr == vm.ErrInsufficientBalance {
			return nil, nil, 0, vmerr
		}
	}

	root := statedb.IntermediateRoot(true).Bytes()
	*usedGas += gas

	receipt := types.NewReceipt(root, failed, *usedGas)
	receipt.TransactionHash = tx.Hash()
	receipt.GasUsed = gas
	// create contract
	if tx.Tos() == nil {
		var from utils.Address
		if txFrom == nil {
			from, _ = tx.Sender(types.Signer{})
		} else {
			from = *txFrom
		}
		receipt.ContractAddress = crypto.CreateAddress(from, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.LogsBloom = types.CreateBloom(types.Receipts{receipt})
	return result, receipt, gas, err
}

func (e *Executor) applyDposMessage(timestamp *big.Int, dposContext *types.DposContext, tx *types.Transaction, statedb *state.StateDB, gp *utils.GasPool) (uint64, bool, error) {
	gas, _ := txpool.IntrinsicGas(tx.Payload(), tx.Type(), false)
	from, _ := tx.Sender(types.Signer{})
	feeval := new(big.Int).Mul(new(big.Int).SetUint64(gas), tx.GasPrice())
	if statedb.GetBalance(from).Cmp(feeval) < 0 {
		return gas, true, errInsufficientBalanceForGas
	}
	statedb.SubBalance(from, feeval)
	statedb.SetNonce(from, tx.Nonce()+1)
	if err := gp.SubGas(gas); err != nil {
		return 0, true, err
	}

	snapshot := statedb.Snapshot()
	dpossnapshot := dposContext.Snapshot()
	switch tx.Type() {
	case types.LoginCandidate:
		if err := dposContext.BecomeCandidate(from); err != nil {
			dpossnapshot.RevertToSnapShot(dpossnapshot)
			statedb.RevertToSnapshot(snapshot)
			return gas, true, err
		}
	case types.LogoutCandidate:
		if err := dposContext.KickoutCandidate(from); err != nil {
			dpossnapshot.RevertToSnapShot(dpossnapshot)
			statedb.RevertToSnapshot(snapshot)
			return gas, true, err
		}
	case types.Delegate:
		if tx.Value().Sign() > 0 {
			if new(big.Int).Sub(statedb.GetBalance(from), tx.Value()).Sign() < 0 {
				return gas, true, fmt.Errorf("balance insufficient")
			}
			statedb.SubBalance(from, tx.Value())
			statedb.SetLockedBalance(from, new(big.Int).Add(statedb.GetLockedBalance(from), tx.Value()))
			statedb.SetDelegateTimestamp(from, timestamp)
		}

		err := dposContext.Delegate(from, tx.Tos())
		if err != nil {
			dpossnapshot.RevertToSnapShot(dpossnapshot)
			statedb.RevertToSnapshot(snapshot)
			return gas, true, err
		}
	case types.UnDelegate:
		if tx.Value().Sign() > 0 {
			ttimestamp := statedb.GetDelegateTimestamp(from).Int64()
			tt := time.Unix(ttimestamp/int64(time.Second), ttimestamp%int64(time.Second))
			t := time.Unix(timestamp.Int64()/int64(time.Second), timestamp.Int64()%int64(time.Second))
			if d := t.Sub(tt); d < time.Duration(60*int64(time.Second)*e.config.MinDelegateDuration) {
				return gas, true, fmt.Errorf("min delegate duration insufficient, %s < %s", d, time.Duration(60*int64(time.Second)*e.config.MinDelegateDuration))
			}

			if new(big.Int).Sub(statedb.GetLockedBalance(from), tx.Value()).Sign() < 0 {
				return gas, true, fmt.Errorf("lockedbalance insufficient")
			}
			statedb.SetLockedBalance(from, new(big.Int).Sub(statedb.GetLockedBalance(from), tx.Value()))
			statedb.SetUnLockedBalance(from, new(big.Int).Add(statedb.GetUnLockedBalance(from), tx.Value()))
			statedb.SetUnDelegateTimestamp(from, timestamp)
		}

		if statedb.GetLockedBalance(from).Sign() == 0 {
			err := dposContext.UnDelegate(from)
			if err != nil {
				dpossnapshot.RevertToSnapShot(dpossnapshot)
				statedb.RevertToSnapshot(snapshot)
				return gas, true, err
			}
		}
	case types.Redeem:
		ttimestamp := statedb.GetUnDelegateTimestamp(from).Int64()
		tt := time.Unix(ttimestamp/int64(time.Second), ttimestamp%int64(time.Second))
		t := time.Unix(timestamp.Int64()/int64(time.Second), timestamp.Int64()%int64(time.Second))
		if t.Before(tt.Add(time.Duration(e.chain.Config().DelayDuration * int64(time.Second)))) {
			return gas, true, fmt.Errorf("duration insufficient, after %v", tt.Add(time.Duration(e.chain.Config().DelayDuration*int64(time.Second))))
		}
		statedb.AddBalance(from, statedb.GetUnLockedBalance(from))
		statedb.SetUnLockedBalance(from, big.NewInt(0))
	}
	return gas, false, nil
}

func (e *Executor) addAction(sender utils.Address, tx *types.Transaction) {
	a := types.NewAction(tx.Hash(), sender, big.NewInt(time.Now().Unix()), big.NewInt(e.chain.Config().DelayDuration))
	e.tp.AddAction(a)
}
