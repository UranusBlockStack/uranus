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

package types

import (
	"container/heap"

	"github.com/UranusBlockStack/uranus/common/utils"
)

// Blocks sort by height
type Blocks []*Block

func (s Blocks) Len() int           { return len(s) }
func (s Blocks) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s Blocks) Less(i, j int) bool { return s[i].Height().Uint64() < s[j].Height().Uint64() }

// TxsByNonce  transactions sort by nonce
type TxsByNonce Transactions

func (s TxsByNonce) Len() int           { return len(s) }
func (s TxsByNonce) Less(i, j int) bool { return s[i].data.Nonce < s[j].data.Nonce }
func (s TxsByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxsByPriceToLow transactions sort by price and implements the heap interface
// Price from high to low
type TxsByPriceToLow Transactions

func (s TxsByPriceToLow) Len() int            { return len(s) }
func (s TxsByPriceToLow) Less(i, j int) bool  { return s[i].data.GasPrice.Cmp(s[j].data.GasPrice) > 0 }
func (s TxsByPriceToLow) Swap(i, j int)       { s[i], s[j] = s[j], s[i] }
func (s *TxsByPriceToLow) Push(x interface{}) { *s = append(*s, x.(*Transaction)) }
func (s *TxsByPriceToLow) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TxsByPriceToHigh Price from low to high
type TxsByPriceToHigh []*Transaction

func (t TxsByPriceToHigh) Len() int           { return len(t) }
func (t TxsByPriceToHigh) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t TxsByPriceToHigh) Less(i, j int) bool { return t[i].GasPrice().Cmp(t[j].GasPrice()) < 0 }

// TransactionsByPriceAndNonce represents a set of transactions
type TransactionsByPriceAndNonce struct {
	txs    map[utils.Address]Transactions // Per account nonce-sorted list of transactions
	heads  TxsByPriceToLow                // Next transaction for each unique account (price heap)
	signer Signer                         // Signer for the set of transactions
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve price sorted transactions in a nonce-honouring way.
func NewTransactionsByPriceAndNonce(signer Signer, txs map[utils.Address]Transactions) *TransactionsByPriceAndNonce {
	// Initialize a price based heap with the head transactions
	heads := make(TxsByPriceToLow, 0, len(txs))
	for _, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := accTxs[0].Sender(signer)
		txs[acc] = accTxs[1:]
	}
	heap.Init(&heads)
	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() *Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift() {
	acc, _ := t.heads[0].Sender(t.signer)
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(&t.heads, 0)
	} else {
		heap.Pop(&t.heads)
	}
}

// Pop removes the best transactiont.
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}
