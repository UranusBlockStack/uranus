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

	"github.com/UranusBlockStack/uranus/common/utils"
)

// TransactionPoolAPI exposes methods for the RPC interface
type TransactionPoolAPI struct {
	b Backend
}

// NewTransactionPoolAPI creates a new RPC service with methods specific for the transaction pool.
func NewTransactionPoolAPI(b Backend) *TransactionPoolAPI {
	return &TransactionPoolAPI{b}
}

// Content returns the transactions contained within the transaction pool.
func (s *TransactionPoolAPI) Content(ignore string, reply *map[string]map[string]map[string]*RPCTransaction) error {
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
func (s *TransactionPoolAPI) Status(ignore string, reply *map[string]utils.Uint) error {
	pending, queue := s.b.TxPoolStats()
	*reply = map[string]utils.Uint{
		"pending": utils.Uint(pending),
		"queued":  utils.Uint(queue),
	}
	return nil
}
