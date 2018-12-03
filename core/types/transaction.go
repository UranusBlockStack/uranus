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
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/params"
)

// TxType transaction type
type TxType uint8

const (
	Binary TxType = iota
	LoginCandidate
	LogoutCandidate
	Delegate
	UnDelegate
)

var (
	ErrInvalidSig     = errors.New("invalid transaction v, r, s values")
	errNoSigner       = errors.New("missing signing methods")
	ErrInvalidType    = errors.New("invalid transaction type")
	ErrInvalidAddress = errors.New("invalid transaction payload address")
	ErrInvalidAction  = errors.New("invalid transaction payload action")
)

// Transaction transaction
type Transaction struct {
	data txdata

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	Type      TxType           `json:"type"`
	Nonce     uint64           `json:"nonce"`
	GasPrice  *big.Int         `json:"gasPrice"`
	GasLimit  uint64           `json:"gas"`
	Tos       []*utils.Address `json:"tos" rlp:"nil"`
	Value     *big.Int         `json:"value"`
	Payload   []byte           `json:"payload"`
	Signature []byte           `json:"signature"`
}

// NewTransaction new transaction
func NewTransaction(txType TxType, nonce uint64, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte, tos ...*utils.Address) *Transaction {
	if len(data) > 0 {
		data = utils.CopyBytes(data)
	}
	d := txdata{
		Type:     txType,
		Nonce:    nonce,
		GasLimit: gasLimit,
		GasPrice: new(big.Int),
		Tos:      tos,
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

// Validate Valid the transaction when the type isn't the binary
func (tx *Transaction) Validate() error {
	switch tx.Type() {
	case Binary:
		if len(tx.Tos()) > 1 {
			return errors.New("binary transaction tos need not greater than 1")
		}
	case Delegate:
		fallthrough
	case UnDelegate:
		if uint64(len(tx.Tos())) > params.MaxVotes || tx.Tos() == nil {
			return fmt.Errorf("tos was required but not greater than %v", params.MaxVotes)
		}
	case LoginCandidate:
		fallthrough
	case LogoutCandidate:
		if tx.Tos() != nil {
			return errors.New("LoginCandidate and LogoutCandidate tx.tos wasn't required")
		}
	default:
		return ErrInvalidType
	}
	return nil
}

func (tx *Transaction) Signature() []byte  { return utils.CopyBytes(tx.data.Signature) }
func (tx *Transaction) Payload() []byte    { return utils.CopyBytes(tx.data.Payload) }
func (tx *Transaction) Gas() uint64        { return tx.data.GasLimit }
func (tx *Transaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.GasPrice) }
func (tx *Transaction) Value() *big.Int    { return new(big.Int).Set(tx.data.Value) }
func (tx *Transaction) Nonce() uint64      { return tx.data.Nonce }
func (tx *Transaction) Type() TxType       { return tx.data.Type }
func (tx *Transaction) Tos() []*utils.Address {
	if tx.data.Tos == nil {
		return nil
	}
	return tx.data.Tos
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
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(utils.StorageSize(rlp.ListSize(size)))
	}
	return err
}

// Hash hashes the RLP encoding of tx. It uniquely identifies the transaction.
func (tx *Transaction) Hash() utils.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(utils.Hash)
	}
	hash := rlpHash(tx)
	tx.hash.Store(hash)
	return hash
}

// Size returns the true RLP encoded storage size of the transaction
func (tx *Transaction) Size() utils.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(utils.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	size := utils.StorageSize(c)
	tx.size.Store(size)
	return size
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
	if sender := tx.from.Load(); sender != nil {
		return sender.(utils.Address), nil
	}
	r, s, v, err := signer.SignatureValues(tx, tx.data.Signature)
	if err != nil {
		return utils.Address{}, err
	}
	addr, err := recoverPlain(signer.Hash(tx), r, s, v, false)
	if err != nil {
		return utils.Address{}, err
	}
	tx.from.Store(addr)
	return addr, nil
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
