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
	"fmt"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

// PublicTransactionPoolAPI exposes methods for the RPC interface
type PublicTransactionPoolAPI struct {
	b Backend
}

// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     utils.Address  `json:"from"`
	To       *utils.Address `json:"to"`
	Gas      *utils.Uint64  `json:"gas"`
	GasPrice *utils.Big     `json:"gasPrice"`
	Value    *utils.Big     `json:"value"`
	Nonce    *utils.Uint64  `json:"nonce"`
	Data     *utils.Bytes   `json:"data"`
	Input    *utils.Bytes   `json:"input"`
}

// NewPublicTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewPublicTransactionPoolAPI(b Backend) *PublicTransactionPoolAPI {
	return &PublicTransactionPoolAPI{b}
}

// Content returns the transactions contained within the transaction pool.
func (s *PublicTransactionPoolAPI) Content(ignore string, reply *map[string]map[string]map[string]*RPCTransaction) error {
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction),
		"queued":  make(map[string]map[string]*RPCTransaction),
	}
	pending, queue := s.b.TxPoolContent()

	// Flatten the pending transactions
	for account, txs := range pending {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		dump := make(map[string]*RPCTransaction)
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = newRPCPendingTransaction(tx)
		}
		content["queued"][account.Hex()] = dump
	}

	*reply = content
	return nil
}

// Status returns the number of pending and queued transaction in the pool.
func (s *PublicTransactionPoolAPI) Status(ignore string, reply *map[string]utils.Uint) error {
	pending, queue := s.b.TxPoolStats()
	*reply = map[string]utils.Uint{
		"pending": utils.Uint(pending),
		"queued":  utils.Uint(queue),
	}
	return nil
}

// SendRawTransaction will add the signed transaction to the transaction pool.
func (s *PublicTransactionPoolAPI) SendRawTransaction(encodedTx utils.Bytes, reply *utils.Hash) error {
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return err
	}
	hash, err := submitTransaction(context.Background(), s.b, tx)
	if err != nil {
		return nil
	}

	*reply = hash
	return nil
}

// submitTransaction is a helper function that submits tx to txPool and logs a message.
func submitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (utils.Hash, error) {
	if err := b.SendTx(ctx, tx); err != nil {
		return utils.Hash{}, err
	}
	if tx.To() == nil {
		from, err := tx.Sender(types.Signer{})
		if err != nil {
			return utils.Hash{}, err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Infof("Submitted contract creation hash: %v ,contract address: %v", tx.Hash(), addr)
	} else {
		log.Infof("Submitted transaction hash: %v,recipient address: %v", tx.Hash(), tx.To())
	}
	return tx.Hash(), nil
}
