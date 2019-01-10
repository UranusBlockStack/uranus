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

package consensus

import (
	"errors"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/params"
)

// Engine TODO
type Engine interface {
	Author(header *types.BlockHeader) (utils.Address, error) //Delete
	CalcDifficulty(config *params.ChainConfig, time uint64, parent *types.BlockHeader) *big.Int

	VerifySeal(chain IChainReader, header *types.BlockHeader) error
	//VerifyHeader(chain IChainReader, header *types.BlockHeader) error

	Seal(chain IChainReader, block *types.Block, stop <-chan struct{}, threads int, updateHashes chan uint64) (*types.Block, error)

	Finalize(chain IChainReader, header *types.BlockHeader, state *state.StateDB, txs []*types.Transaction, actions []*types.Action, receipts []*types.Receipt, dposContext *types.DposContext) (*types.Block, error)
}

type ITxPool interface {
	Pending() (map[utils.Address]types.Transactions, error)
	Actions() []*types.Action
}

type IBlockChain interface {
	PostEvent(event interface{})
	GetCurrentInfo() (*types.Block, *state.StateDB, error)
	WriteBlockWithState(*types.Block, types.Receipts, *state.StateDB) (bool, error)
	ExecActions(statedb *state.StateDB, actions []*types.Action)
	ExecTransaction(*utils.Address, *types.DposContext, *utils.GasPool, *state.StateDB, *types.BlockHeader, *types.Transaction, *uint64, vm.Config) (*types.Receipt, uint64, error)
}

type IChainReader interface {
	Config() *params.ChainConfig
	CurrentBlock() *types.Block
	GetBlockByHeight(uint64) *types.Block
	GetBlockByHash(utils.Hash) *types.Block
}

type IUranus interface {
	ITxPool
	IBlockChain
	IChainReader
}

var (
	ErrUnknownBlock = errors.New("unknown block")

	ErrUnknownAncestor = errors.New("unknown ancestor")

	ErrFutureBlock = errors.New("block in the future")

	ErrInvalidNumber = errors.New("invalid block number")
)
