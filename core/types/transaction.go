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
	"crypto/ecdsa"
	"io"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

// Transaction transaction
type Transaction struct {
	data txdata
}

type txdata struct {
	Nonce     uint64         `json:"nonce"    gencodec:"required"`
	GasPrice  *big.Int       `json:"gasPrice" gencodec:"required"`
	GasLimit  uint64         `json:"gas"      gencodec:"required"`
	To        *utils.Address `json:"to"       rlp:"nil"`
	Value     *big.Int       `json:"value"    gencodec:"required"`
	Payload   []byte         `json:"payload"    gencodec:"required"`
	Signature []byte         `json:"signature"    gencodec:"required"`
}

// NewTransaction new transaction
func NewTransaction(nonce uint64, to utils.Address, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = utils.CopyBytes(data)
	}
	d := txdata{
		Nonce:    nonce,
		GasLimit: gasLimit,
		GasPrice: new(big.Int),
		To:       &to,
		Value:    new(big.Int),
		Payload:  data,
	}
	if value != nil {
		d.Value.Set(value)
	}
	if gasPrice != nil {
		d.GasPrice.Set(gasPrice)
	}

	return &Transaction{data: d}
}
func (tx *Transaction) Signature() []byte  { return utils.CopyBytes(tx.data.Signature) }
func (tx *Transaction) Payload() []byte    { return utils.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() uint64        { return tx.data.GasLimit }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.GasPrice) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.data.Value) }
func (tx *Transaction) Nonce() uint64      { return tx.data.Nonce }
func (tx *Transaction) To() *utils.Address {
	if tx.data.To == nil {
		return nil
	}
	to := *tx.data.To
	return &to
}

// Cost returns value + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.GasPrice, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Value)
	return total
}

// EncodeRLP implements rlp.Encoder
func (tx *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *Transaction) DecodeRLP(s *rlp.Stream) error {
	err := s.Decode(&tx.data)
	return err
}

// Hash hashes the RLP encoding of tx. It uniquely identifies the transaction.
func (tx *Transaction) Hash() utils.Hash {
	return rlpHash(tx)
}

// Size returns the true RLP encoded storage size of the transaction
func (tx *Transaction) Size() utils.StorageSize {

	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	return utils.StorageSize(c)
}

// WithSignature returns a new transaction with the given signature.
func (tx *Transaction) WithSignature(signature []byte) {
	tx.data.Signature = utils.CopyBytes(signature)
}

// SignTx signs the transaction using the given signer and private key
func (tx *Transaction) SignTx(s Signer, prv *ecdsa.PrivateKey) error {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return err
	}
	tx.WithSignature(sig)
	return nil
}

// Sender sender address of the transaction using the given signer
func (tx *Transaction) Sender(signer Signer) (utils.Address, error) {
	r, s, v, err := signer.SignatureValues(tx, tx.data.Signature)
	if err != nil {
		return utils.Address{}, err
	}
	return recoverPlain(signer.Hash(tx), r, s, v, false)
}

// ChainID returns which chain id this transaction was signed for (if at all)
func (tx *Transaction) ChainID(signer Signer) (*big.Int, error) {
	_, _, v, err := signer.SignatureValues(tx, tx.data.Signature)
	if err != nil {
		return nil, err
	}
	return chainID(v), err
}

// Protected returns whether the transaction is protected from replay protection.
func (tx *Transaction) Protected(signer Signer) (bool, error) {
	_, _, v, err := signer.SignatureValues(tx, tx.data.Signature)
	if err != nil {
		return false, err
	}
	return isProtectedV(v), nil
}

type StorageTx struct {
	BlockHash   utils.Hash
	BlockHeight uint64
	TxIndex     uint64
	Tx          *Transaction
}

func NewStorageTx(blockHash utils.Hash, blockHeight, txIndex uint64, tx *Transaction) *StorageTx {
	return &StorageTx{
		BlockHash:   blockHash,
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		Tx:          tx,
	}
}

type StorageTxs []*StorageTx

func (s StorageTxs) ToTransactions() Transactions {
	var txs = make(Transactions, len(s))
	for i, stx := range s {
		txs[i] = stx.Tx
	}
	return txs
}
