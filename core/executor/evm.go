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
	"math/big"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
)

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(tx *types.Transaction, bheader *types.BlockHeader, ledger *ledger.Ledger, engine consensus.Engine, author, txFrom *utils.Address) vm.Context {
	var beneficiary, from utils.Address

	if author == nil {
		beneficiary = bheader.Miner
	} else {
		beneficiary = *author
	}

	if txFrom == nil {
		from, _ = tx.Sender(types.Signer{})
	} else {
		from = *txFrom
	}

	vm := vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		Origin:      from,
		Coinbase:    beneficiary,
		BlockNumber: new(big.Int).Set(bheader.Height),
		Time:        new(big.Int).Set(bheader.TimeStamp),
		Difficulty:  new(big.Int).Set(bheader.Difficulty),
		GasLimit:    bheader.GasLimit,
		GasPrice:    new(big.Int).Set(tx.GasPrice()),
	}
	vm.GetHash = func(n uint64) utils.Hash {
		for header := ledger.GetHeader(bheader.PreviousHash); header != nil; header = ledger.GetHeader(header.PreviousHash) {
			if n == header.Height.Uint64()-1 {
				return header.PreviousHash
			}
		}
		return utils.Hash{}
	}

	return vm
}

// CanTransfer checks wether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr utils.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient utils.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
