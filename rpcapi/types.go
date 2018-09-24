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
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type BlockHeight int64

const (
	PendingBlockHeight  = BlockHeight(-2)
	LatestBlockHeight   = BlockHeight(-1)
	EarliestBlockHeight = BlockHeight(0)
)

func (bn *BlockHeight) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	switch input {
	case "earliest":
		*bn = EarliestBlockHeight
		return nil
	case "latest":
		*bn = LatestBlockHeight
		return nil
	case "pending":
		*bn = PendingBlockHeight
		return nil
	}

	blckNum, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if blckNum > math.MaxInt64 {
		return fmt.Errorf("Blocknumber too high")
	}

	*bn = BlockHeight(blckNum)
	return nil
}

func (bn BlockHeight) Int64() int64 {
	return (int64)(bn)
}

type RPCTransaction struct {
	BlockHash        utils.Hash     `json:"blockHash"`
	BlockHeight      *utils.Big     `json:"blockHeight"`
	From             utils.Address  `json:"from"`
	Gas              utils.Uint64   `json:"gas"`
	GasPrice         *utils.Big     `json:"gasPrice"`
	Hash             utils.Hash     `json:"hash"`
	Input            utils.Bytes    `json:"input"`
	Nonce            utils.Uint64   `json:"nonce"`
	To               *utils.Address `json:"to"`
	TransactionIndex utils.Uint     `json:"transactionIndex"`
	Value            *utils.Big     `json:"value"`
	Signature        utils.Bytes    `json:"signature"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC representation
func newRPCTransaction(tx *types.Transaction, blockHash utils.Hash, blockHeight uint64, index uint64) *RPCTransaction {
	from, _ := tx.Sender(types.Signer{})

	result := &RPCTransaction{
		From:      from,
		Gas:       utils.Uint64(tx.Gas()),
		GasPrice:  (*utils.Big)(tx.GasPrice()),
		Hash:      tx.Hash(),
		Input:     utils.Bytes(tx.Payload()),
		Nonce:     utils.Uint64(tx.Nonce()),
		To:        tx.To(),
		Value:     (*utils.Big)(tx.Value()),
		Signature: utils.Bytes(tx.Signature()),
	}
	if blockHash != (utils.Hash{}) {
		result.BlockHash = blockHash
		result.BlockHeight = (*utils.Big)(new(big.Int).SetUint64(blockHeight))
		result.TransactionIndex = utils.Uint(index)
	}
	return result
}

// newRPCPendingTransaction returns a pending transaction that will serialize to the RPC representation
func newRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	return newRPCTransaction(tx, utils.Hash{}, 0, 0)
}

// newRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockIndex(b *types.Block, index uint64) *RPCTransaction {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	return newRPCTransaction(txs[index], b.Hash(), b.Height().Uint64(), index)
}

// newRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func newRPCRawTransactionFromBlockIndex(b *types.Block, index uint64) utils.Bytes {
	txs := b.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := rlp.EncodeToBytes(txs[index])
	return blob
}

// newRPCTransactionFromBlockHash returns a transaction that will serialize to the RPC representation.
func newRPCTransactionFromBlockHash(b *types.Block, hash utils.Hash) *RPCTransaction {
	for idx, tx := range b.Transactions() {
		if tx.Hash() == hash {
			return newRPCTransactionFromBlockIndex(b, uint64(idx))
		}
	}
	return nil
}

func RPCMarshalBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	head := b.BlockHeader() // copies the header once
	fields := map[string]interface{}{
		"height":     (*utils.Big)(head.Height),
		"hash":       b.Hash(),
		"parentHash": head.PreviousHash,
		"nonce":      head.Nonce,
		// "logsBloom":        head.LogsBloom,
		"stateRoot":        head.StateRoot,
		"miner":            head.Miner,
		"difficulty":       (*utils.Big)(head.Difficulty),
		"extraData":        utils.Bytes(head.ExtraData),
		"size":             utils.Uint64(b.Size()),
		"gasLimit":         utils.Uint64(head.GasLimit),
		"gasUsed":          utils.Uint64(head.GasUsed),
		"timestamp":        (*utils.Big)(head.TimeStamp),
		"transactionsRoot": head.TransactionsRoot,
		"receiptsRoot":     head.ReceiptsRoot,
	}

	if inclTx {
		formatTx := func(tx *types.Transaction) (interface{}, error) {
			return tx.Hash(), nil
		}
		if fullTx {
			formatTx = func(tx *types.Transaction) (interface{}, error) {
				return newRPCTransactionFromBlockHash(b, tx.Hash()), nil
			}
		}
		txs := b.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields["transactions"] = transactions
	}

	return fields, nil
}
