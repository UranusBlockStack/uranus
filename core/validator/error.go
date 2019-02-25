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
	"errors"
	"fmt"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/bloom"
	"github.com/UranusBlockStack/uranus/common/utils"
)

var (
	// ErrKnownBlock is returned when a block to import is already known locally.
	ErrKnownBlock = errors.New("block already known")
	// ErrPrunedAncestor is returned when validating a block requires an ancestor
	// that is known, but the state of which is not available.
	ErrPrunedAncestor = errors.New("pruned ancestor")
	// ErrUnknownAncestor is returned when validating a block requires an ancestor
	// that is unknown.
	ErrUnknownAncestor = errors.New("unknown ancestor")
	// ErrBlockTime timestamp less than or equal to parent's
	ErrBlockTime = errors.New("timestamp less than or equal to parent's")
	// ErrInvalidNumber is returned if a block's number doesn't equal it's parent's
	// plus one.
	ErrInvalidNumber = errors.New("invalid block number")
	// ErrFutureBlock is returned when a block's timestamp is in the future according
	// to the current node.
	ErrFutureBlock = errors.New("block in the future")
	// ErrExtraDataTooLong is returned when extra-data too long
	ErrExtraDataTooLong = func(actual, expected uint64) error {
		return fmt.Errorf("extra-data too long: %d > %d", actual, expected)
	}
	// ErrDifficulty is returned invalid difficulty
	ErrDifficulty = func(actual, expected *big.Int) error {
		return fmt.Errorf("invalid difficulty: have %v, want %v", actual, expected)
	}
	// ErrGasLimitTooBig is returned invalid gasLimit,the gas limit is > 2^63-1
	ErrGasLimitTooBig = func(actual, expected uint64) error {
		return fmt.Errorf("invalid max gaslimit: have %v, max %v", actual, expected)
	}
	// ErrGasUsed is returned invalid gasUsed
	ErrGasUsed = func(actual, expected uint64) error {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", actual, expected)
	}
	// ErrGasLimit is returned invalid gaslimit
	ErrGasLimit = func(actual, expected, extra uint64) error {
		return fmt.Errorf("invalid gaslimit: have %d, want %d += %d", actual, expected, extra)
	}
	// ErrTxsRootHash is returned invalid txs root hash
	ErrTxsRootHash = func(actual, expected utils.Hash) error {
		return fmt.Errorf("transaction txs root hash mismatch: have %x, want %x", actual, expected)
	}

	// ErrReceiptRootHash is returned invalid receiptroot hash
	ErrReceiptRootHash = func(actual, expected utils.Hash) error {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", actual, expected)
	}
	// ErrStateRootHash is returned invalid stateroot hash
	ErrStateRootHash = func(actual, expected utils.Hash) error {
		return fmt.Errorf("transaction state root hash mismatch: have %x, want %x", actual, expected)
	}
	// ErrDposRootHash is returned invalid stateroot hash
	ErrDposRootHash = func(actual, expected utils.Hash) error {
		return fmt.Errorf("dpos state root hash mismatch: have %s, want %s", actual.String(), expected.String())
	}
	// ErrLogsBloom is returned invalid logs bloom
	ErrLogsBloom = func(actual, expected bloom.Bloom) error {
		return fmt.Errorf("invalid logs bloom (remote: %x  local: %x)", actual, expected)
	}
	// ErrLocalGasUsed is returned invalid local gas used
	ErrLocalGasUsed = func(actual, expected uint64) error {
		return fmt.Errorf("invalid gas used (remote: %d local: %d)", actual, expected)
	}
)
