// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"math/big"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

func NoopCanTransfer(db StateDB, from utils.Address, balance *big.Int) bool {
	return true
}
func NoopTransfer(db StateDB, from, to utils.Address, amount *big.Int) {}

type NoopEVMCallContext struct{}

func (NoopEVMCallContext) Call(caller ContractRef, addr utils.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (NoopEVMCallContext) CallCode(caller ContractRef, addr utils.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	return nil, nil
}
func (NoopEVMCallContext) Create(caller ContractRef, data []byte, gas, value *big.Int) ([]byte, utils.Address, error) {
	return nil, utils.Address{}, nil
}
func (NoopEVMCallContext) DelegateCall(me ContractRef, addr utils.Address, data []byte, gas *big.Int) ([]byte, error) {
	return nil, nil
}

type NoopStateDB struct{}

func (NoopStateDB) CreateAccount(utils.Address)                                     {}
func (NoopStateDB) SubBalance(utils.Address, *big.Int)                              {}
func (NoopStateDB) AddBalance(utils.Address, *big.Int)                              {}
func (NoopStateDB) GetBalance(utils.Address) *big.Int                               { return nil }
func (NoopStateDB) GetNonce(utils.Address) uint64                                   { return 0 }
func (NoopStateDB) SetNonce(utils.Address, uint64)                                  {}
func (NoopStateDB) GetCodeHash(utils.Address) utils.Hash                            { return utils.Hash{} }
func (NoopStateDB) GetCode(utils.Address) []byte                                    { return nil }
func (NoopStateDB) SetCode(utils.Address, []byte)                                   {}
func (NoopStateDB) GetCodeSize(utils.Address) int                                   { return 0 }
func (NoopStateDB) AddRefund(uint64)                                                {}
func (NoopStateDB) GetRefund() uint64                                               { return 0 }
func (NoopStateDB) GetState(utils.Address, utils.Hash) utils.Hash                   { return utils.Hash{} }
func (NoopStateDB) SetState(utils.Address, utils.Hash, utils.Hash)                  {}
func (NoopStateDB) Suicide(utils.Address) bool                                      { return false }
func (NoopStateDB) HasSuicided(utils.Address) bool                                  { return false }
func (NoopStateDB) Exist(utils.Address) bool                                        { return false }
func (NoopStateDB) Empty(utils.Address) bool                                        { return false }
func (NoopStateDB) RevertToSnapshot(int)                                            {}
func (NoopStateDB) Snapshot() int                                                   { return 0 }
func (NoopStateDB) AddLog(*types.Log)                                               {}
func (NoopStateDB) AddPreimage(utils.Hash, []byte)                                  {}
func (NoopStateDB) ForEachStorage(utils.Address, func(utils.Hash, utils.Hash) bool) {}
