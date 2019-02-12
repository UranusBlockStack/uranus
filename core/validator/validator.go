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

package validator

import (
	"fmt"
	"math/big"

	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
)

// Validator responsible for validating block headers, blocks and processed state.
type Validator struct {
	ledger *ledger.Ledger
	engine consensus.Engine
}

// New returns a new block validator
func New(ledger *ledger.Ledger, engine consensus.Engine) *Validator {
	return &Validator{ledger, engine}
}

// ValidateHeader verifies the the block header
func (v *Validator) ValidateHeader(chain consensus.IChainReader, header *types.BlockHeader, seal bool) error {
	var parent *types.BlockHeader

	if v.ledger.GetHeader(header.Hash()) != nil {
		return nil
	}

	if parent = v.ledger.GetHeader(header.PreviousHash); parent == nil {
		return ErrUnknownAncestor
	}

	// check extra data len
	if uint64(len(header.ExtraData)) > params.MaxExtraDataSize+65 {
		return ErrExtraDataTooLong(uint64(len(header.ExtraData)), params.MaxExtraDataSize+65)
	}

	// check timestamp
	if header.TimeStamp.Cmp(parent.TimeStamp) <= 0 {
		return ErrBlockTime
	}

	// Verify the block's difficulty based in it's timestamp and parent's difficulty
	expected := v.engine.CalcDifficulty(chain, chain.Config(), header.TimeStamp.Uint64(), parent)
	if expected.Cmp(header.Difficulty) != 0 {
		return ErrDifficulty(header.Difficulty, expected)
	}

	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return ErrGasLimitTooBig(header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return ErrGasUsed(header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return ErrGasLimit(header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Height, parent.Height); diff.Cmp(big.NewInt(1)) != 0 {
		return ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := v.engine.VerifySeal(chain, header); err != nil {
			return err
		}
	}
	return nil
}

// ValidateTxs verifies the the block header's transaction root before already validated header
func (v *Validator) ValidateTxs(block *types.Block) error {
	// Check whether the block's known, and if not, that it's linkable
	if v.ledger.HasBlock(block.Hash()) && v.ledger.HasState(block.StateRoot()) {
		return ErrKnownBlock
	}

	parent := v.ledger.GetBlock(block.PreviousHash())
	if parent == nil {
		return ErrUnknownAncestor
	} else {
		if !v.ledger.HasState(parent.StateRoot()) {
			if !v.ledger.HasBlock(parent.Hash()) {
				return ErrUnknownAncestor
			}
			return ErrPrunedAncestor
		}
	}

	// check txs root hash
	header := block.BlockHeader()

	if root := types.DeriveRootHash(block.Transactions()); root != header.TransactionsRoot {
		return ErrTxsRootHash(root, header.TransactionsRoot)
	}

	return nil
}

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used gas, the receipt roots and the state root
// itself.
func (v *Validator) ValidateState(
	block, parent *types.Block,
	statedb *state.StateDB,
	receipts types.Receipts,
	deleteEmptyObjects bool,
	usedGas uint64) error {

	header := block.BlockHeader()
	if block.GasUsed() != usedGas {
		return fmt.Errorf("invalid gas used (remote: %d local: %d)", block.GasUsed(), usedGas)
	}
	// check the received block's bloom with the one derived from the generated receipts For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.LogsBloom {
		return ErrLogsBloom(header.LogsBloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, R1]]))
	receiptSha := types.DeriveRootHash(receipts)
	if receiptSha != header.ReceiptsRoot {
		return ErrReceiptRootHash(header.ReceiptsRoot, receiptSha)
	}
	// check the state root against the received state root and throw an error if they don't match.
	if stateSha := statedb.IntermediateRoot(deleteEmptyObjects); header.StateRoot != stateSha {
		return ErrStateRootHash(header.StateRoot, stateSha)
	}
	if dposSha := block.DposContext.ToProto().Root(); dposSha != header.DposContext.Root() {
		return ErrDposRootHash(dposSha, header.DposContext.Root())
	}
	return nil
}
