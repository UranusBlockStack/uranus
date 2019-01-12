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
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/executor"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
)

// UranusAPI exposes methods for the RPC interface
type UranusAPI struct {
	b Backend
}

// NewUranusAPI creates a new RPC service with methods specific for the uranus.
func NewUranusAPI(b Backend) *UranusAPI {
	return &UranusAPI{b}
}

// SuggestGasPrice return suggest gas price.
func (u *UranusAPI) SuggestGasPrice(ignore string, reply *utils.Big) error {
	gasprice, err := u.b.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	*reply = *(*utils.Big)(gasprice)
	return nil
}

type GetBalanceArgs struct {
	Address     utils.Address
	BlockHeight *BlockHeight
}

func (a GetBalanceArgs) getBlockHeight() BlockHeight {
	blockheight := LatestBlockHeight
	if a.BlockHeight != nil {
		blockheight = *a.BlockHeight
	}
	return blockheight
}

// GetBalance returns the amount of wei for the given address in the state of the given block number
func (u *UranusAPI) GetBalance(args GetBalanceArgs, reply *utils.Big) error {
	state, err := u.getState(args.getBlockHeight())
	if err != nil {
		return err
	}
	*reply = *(*utils.Big)(state.GetBalance(args.Address))
	return nil
}

type GetNonceArgs struct {
	GetBalanceArgs
}

// GetNonce returns nonce for the given address
func (u *UranusAPI) GetNonce(args GetNonceArgs, reply *utils.Uint64) error {
	state, err := u.getState(args.getBlockHeight())
	if err != nil {
		return err
	}
	nonce := state.GetNonce(args.Address)
	*reply = (utils.Uint64)(nonce)
	return nil
}

type GetCodeArgs struct {
	GetBalanceArgs
}

// GetCode returns code for the given address
func (u *UranusAPI) GetCode(args GetCodeArgs, reply *utils.Bytes) error {
	state, err := u.getState(args.getBlockHeight())
	if err != nil {
		return err
	}
	code := state.GetCode(args.Address)
	*reply = code
	return nil
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From       utils.Address
	Tos        []*utils.Address
	Gas        *utils.Uint64
	GasPrice   *utils.Big
	Value      *utils.Big
	Nonce      *utils.Uint64
	Data       *utils.Bytes
	TxType     *utils.Uint64
	Passphrase string
}

// check is a helper function that fills in default values for unspecified tx fields.
func (args *SendTxArgs) check(ctx context.Context, b Backend) error {
	if args.TxType == nil {
		args.TxType = new(utils.Uint64)
		*(*uint64)(args.Gas) = uint64(types.Binary)
	}

	if args.Gas == nil {
		args.Gas = new(utils.Uint64)
		*(*uint64)(args.Gas) = 90000
	}
	if args.GasPrice == nil {
		price, err := b.SuggestGasPrice(ctx)
		if err != nil {
			return err
		}
		args.GasPrice = (*utils.Big)(price)
	}
	if args.Value == nil {
		args.Value = new(utils.Big)
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.From)
		if err != nil {
			return err
		}
		args.Nonce = (*utils.Uint64)(&nonce)
	}

	if args.Tos == nil {
		// Contract creation
		var input []byte
		if args.Data != nil {
			input = *args.Data
		}
		if len(input) == 0 {
			return errors.New(`contract creation without any data provided`)
		}
		args.TxType = new(utils.Uint64)
		*(*uint64)(args.Gas) = uint64(types.Binary)
	}
	return nil
}

func (args *SendTxArgs) toTransaction() *types.Transaction {
	var input []byte
	if args.Data != nil {
		input = *args.Data
	}
	if args.Tos == nil {
		return types.NewTransaction(types.TxType(uint64(*args.TxType)), uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input)
	}
	return types.NewTransaction(types.TxType(uint64(*args.TxType)), uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input, args.Tos...)
}

// SignAndSendTransaction sign and send transaction .
func (u *UranusAPI) SignAndSendTransaction(args SendTxArgs, reply *utils.Hash) error {
	if err := args.check(context.Background(), u.b); err != nil {
		return err
	}

	tx, err := u.b.SignTx(args.From, args.toTransaction(), args.Passphrase)
	if err != nil {
		return err
	}

	hash, err := submitTransaction(context.Background(), u.b, tx)
	if err != nil {
		return err
	}

	*reply = hash
	return nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
func (u *UranusAPI) SendRawTransaction(encodedTx utils.Bytes, reply *utils.Hash) error {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return err
	}
	hash, err := submitTransaction(context.Background(), u.b, tx)
	if err != nil {
		return err
	}
	*reply = hash
	return nil
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From        utils.Address
	Tos         []*utils.Address
	Gas         utils.Uint64
	GasPrice    utils.Big
	Value       utils.Big
	Data        utils.Bytes
	TxType      uint8
	BlockHeight *BlockHeight
}

// Call executes the given transaction on the state for the given block number.
func (u *UranusAPI) Call(args CallArgs, reply *utils.Bytes) error {
	blockheight := LatestBlockHeight
	if args.BlockHeight != nil {
		blockheight = *args.BlockHeight
	}
	timeout := 5 * time.Second
	defer func(start time.Time) { log.Debugf("Executing EVM call finished runtime: %v", time.Since(start)) }(time.Now())
	block, err := u.b.BlockByHeight(context.Background(), blockheight)
	if err != nil {
		return err
	}
	state, err := u.b.BlockChain().StateAt(block.StateRoot())
	if err != nil {
		return err
	}

	// Set default gas & gas price if none were set
	gas, gasPrice := uint64(args.Gas), args.GasPrice.ToInt()
	if gas == 0 {
		gas = math.MaxUint64 / 2
	}
	if gasPrice.Sign() == 0 {
		gasPrice = new(big.Int).SetUint64(1e9)
	}

	nonce, err := u.b.GetPoolNonce(context.Background(), args.From)
	if err != nil {
		return err
	}

	tx := types.NewTransaction(types.TxType(args.TxType), nonce, args.Value.ToInt(), gas, gasPrice, args.Data, args.Tos...)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var (
		cancel context.CancelFunc
		ctx    = context.Background()
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	evm, vmError, err := u.b.GetEVM(ctx, args.From, tx, state, block.BlockHeader(), vm.Config{})
	if err != nil {
		return err
	}
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	gp := new(utils.GasPool).AddGas(math.MaxUint64)

	stx := executor.NewStateTransitionForApi(evm, args.From, tx, gp)

	res, _, _, err := stx.TransitionDb()
	if err := vmError(); err != nil {
		return err
	}

	*reply = (utils.Bytes)(res)
	return err
}

func (u *UranusAPI) getState(height BlockHeight) (*state.StateDB, error) {
	block, err := u.b.BlockByHeight(context.Background(), height)
	if err != nil {
		return nil, err
	}
	return u.b.BlockChain().StateAt(block.StateRoot())
}
