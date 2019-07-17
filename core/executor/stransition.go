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
	"errors"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/txpool"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
)

var (
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
)

type StateTransition struct {
	gp   *utils.GasPool
	tx   *types.Transaction
	from utils.Address

	gas        uint64
	gasPrice   *big.Int
	initialGas uint64
	value      *big.Int
	data       []byte
	state      vm.StateDB
	evm        *vm.EVM
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(txFrom *utils.Address, evm *vm.EVM, tx *types.Transaction, gp *utils.GasPool) *StateTransition {
	var from utils.Address
	if txFrom == nil {
		from, _ = tx.Sender(types.Signer{})
	} else {
		from = *txFrom
	}
	return &StateTransition{
		gp:       gp,
		evm:      evm,
		from:     from,
		tx:       tx,
		gasPrice: tx.GasPrice(),
		value:    tx.Value(),
		data:     tx.Payload(),
		state:    evm.StateDB,
	}
}

func NewStateTransitionForApi(evm *vm.EVM, from utils.Address, tx *types.Transaction, gp *utils.GasPool) *StateTransition {
	return &StateTransition{
		gp:       gp,
		evm:      evm,
		from:     from,
		tx:       tx,
		gasPrice: tx.GasPrice(),
		value:    tx.Value(),
		data:     tx.Payload(),
		state:    evm.StateDB,
	}
}

// ExecStateTransition computes the new state by applying the given message against the old state within the environment.
func ExecStateTransition(txfrom *utils.Address, evm *vm.EVM, tx *types.Transaction, gp *utils.GasPool) ([]byte, uint64, bool, error) {
	return NewStateTransition(txfrom, evm, tx, gp).TransitionDb()
}

// to returns the recipient of the message.
func (st *StateTransition) tos() []*utils.Address {
	if st.tx == nil || st.tx.Tos() == nil /* contract creation */ {
		return nil
	}
	return st.tx.Tos()
}

func (st *StateTransition) useGas(amount uint64) error {
	if st.gas < amount {
		return vm.ErrOutOfGas
	}
	st.gas -= amount

	return nil
}

func (st *StateTransition) buyGas() error {
	mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.tx.Gas()), st.gasPrice)
	if st.state.GetBalance(st.from).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}
	if err := st.gp.SubGas(st.tx.Gas()); err != nil {
		return err
	}
	st.gas += st.tx.Gas()

	st.initialGas = st.tx.Gas()
	st.state.SubBalance(st.from, mgval)
	return nil
}

func (st *StateTransition) preCheck() error {
	nonce := st.state.GetNonce(st.from)
	if nonce < st.tx.Nonce() {
		return ErrNonceTooHigh
	} else if nonce > st.tx.Nonce() {
		return ErrNonceTooLow
	}
	return st.buyGas()
}

// TransitionDb will transition the state by applying the current message and returning the result including the the used gas.
func (st *StateTransition) TransitionDb() (ret []byte, usedGas uint64, failed bool, err error) {
	if err = st.preCheck(); err != nil {
		return
	}
	sender := vm.AccountRef(st.from)
	contractCreation := st.tx.Tos() == nil

	// Pay intrinsic gas
	gas, err := txpool.IntrinsicGas(st.data, st.tx.Type(), contractCreation)
	if err != nil {
		return nil, 0, false, err
	}
	if err = st.useGas(gas); err != nil {
		return nil, 0, false, err
	}

	var (
		evm   = st.evm
		vmerr error
	)
	if contractCreation {
		ret, _, st.gas, vmerr = evm.Create(sender, st.data, st.gas, st.value)
	} else {
		// Increment the nonce for the next transaction
		st.state.SetNonce(st.from, st.state.GetNonce(sender.Address())+1)
		ret, st.gas, vmerr = evm.Call(sender, *st.tos()[0], st.data, st.gas, st.value)
	}
	if vmerr != nil {
		log.Debugf("VM returned with err: %v ", vmerr)
		if vmerr == vm.ErrInsufficientBalance {
			return nil, 0, false, vmerr
		}
	}
	st.refundGas()
	st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))

	return ret, st.gasUsed(), vmerr != nil, err
}

func (st *StateTransition) refundGas() {
	// Apply refund counter, capped to half of the used gas.
	refund := st.gasUsed() / 2
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund

	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
	st.state.AddBalance(st.from, remaining)

	st.gp.AddGas(st.gas)
}

// gasUsed returns the amount of gas used up by the state transition.
func (st *StateTransition) gasUsed() uint64 {
	return st.initialGas - st.gas
}
