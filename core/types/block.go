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
	"encoding/binary"
	"io"
	"math/big"
	"time"
	"unsafe"

	"github.com/UranusBlockStack/uranus/common/bloom"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

// A MinerNonce is a 64-bit hash which proves (combined with the mix-hash) that a sufficient amount of computation has been carried out on a block.
type MinerNonce [8]byte

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) MinerNonce {
	var n MinerNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64 returns the integer value of a block nonce.
func (n MinerNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n MinerNonce) MarshalText() ([]byte, error) {
	return utils.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *MinerNonce) UnmarshalText(input []byte) error {
	return utils.UnmarshalFixedText("BlockNonce", input, n[:])
}

// BlockHeader represents a block header in blockchain.
type BlockHeader struct {
	PreviousHash     utils.Hash    `json:"previousHash"       gencodec:"required"`
	Miner            utils.Address `json:"miner"            gencodec:"required"`
	StateRoot        utils.Hash    `json:"stateRoot"        gencodec:"required"`
	TransactionsRoot utils.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptsRoot     utils.Hash    `json:"receiptsRoot"     gencodec:"required"`
	LogsBloom        bloom.Bloom   `json:"logsBloom"        gencodec:"required"`
	Difficulty       *big.Int      `json:"difficulty"       gencodec:"required"`
	Height           *big.Int      `json:"height"           gencodec:"required"`
	GasLimit         uint64        `json:"gasLimit"         gencodec:"required"`
	GasUsed          uint64        `json:"gasUsed"          gencodec:"required"`
	TimeStamp        *big.Int      `json:"timestamp"        gencodec:"required"`
	ExtraData        []byte        `json:"extraData"        gencodec:"required"`
	Nonce            MinerNonce    `json:"nonce"            gencodec:"required"`
}

// Hash returns the block hash of the header
func (h *BlockHeader) Hash() utils.Hash {
	return rlpHash(h)
}

// HashNoNonce returns the hash which is used as input for the proof-of-work search.
func (h *BlockHeader) HashNoNonce() utils.Hash {
	return rlpHash([]interface{}{
		h.PreviousHash,
		h.Miner,
		h.StateRoot,
		h.TransactionsRoot,
		h.ReceiptsRoot,
		h.LogsBloom,
		h.Difficulty,
		h.Height,
		h.GasLimit,
		h.GasUsed,
		h.TimeStamp,
		h.ExtraData,
	})
}

// Size returns the approximate memory used by all internal contents.
func (h *BlockHeader) Size() utils.StorageSize {
	return utils.StorageSize(unsafe.Sizeof(*h)) + utils.StorageSize(len(h.ExtraData)+(h.Difficulty.BitLen()+h.Height.BitLen()+h.TimeStamp.BitLen())/8)
}

// CopyBlockHeader creates a deep copy of a block header
func CopyBlockHeader(h *BlockHeader) *BlockHeader {
	cpy := *h
	if cpy.TimeStamp = new(big.Int); h.TimeStamp != nil {
		cpy.TimeStamp.Set(h.TimeStamp)
	}
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Height = new(big.Int); h.Height != nil {
		cpy.Height.Set(h.Height)
	}
	if len(h.ExtraData) > 0 {
		cpy.ExtraData = make([]byte, len(h.ExtraData))
		copy(cpy.ExtraData, h.ExtraData)
	}
	return &cpy
}

// Block represents an entire block in the blockchain.
type Block struct {
	header       *BlockHeader
	transactions []*Transaction

	// These fields are used to track inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// NewBlock creates a new block.
func NewBlock(header *BlockHeader, txs []*Transaction, receipts []*Receipt) *Block {
	b := &Block{header: CopyBlockHeader(header)}

	if len(txs) == 0 {
		b.header.TransactionsRoot = DeriveRootHash(Transactions(txs))
	} else {
		b.header.TransactionsRoot = DeriveRootHash(Transactions(txs))
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptsRoot = DeriveRootHash(Receipts(receipts))
	} else {
		b.header.ReceiptsRoot = DeriveRootHash(Receipts(receipts))
		b.header.LogsBloom = CreateBloom(receipts)
	}

	return b
}

// NewBlockWithBlockHeader creates a block with the given header data.
func NewBlockWithBlockHeader(header *BlockHeader) *Block {
	return &Block{header: CopyBlockHeader(header)}
}

type rlpBlock struct {
	BlockHeader *BlockHeader
	Txs         []*Transaction
}

// EncodeRLP implements rlp.Encoder
func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rlpBlock{
		BlockHeader: b.header,
		Txs:         b.transactions,
	})
}

// DecodeRLP implements rlp.Decoder
func (b *Block) DecodeRLP(s *rlp.Stream) error {
	rb := &rlpBlock{}
	if err := s.Decode(rb); err != nil {
		return err
	}
	b.header, b.transactions = rb.BlockHeader, rb.Txs
	return nil
}

func (b *Block) Transactions() Transactions { return b.transactions }
func (b *Block) Transaction(hash utils.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}
func (b *Block) Height() *big.Int             { return new(big.Int).Set(b.header.Height) }
func (b *Block) GasLimit() uint64             { return b.header.GasLimit }
func (b *Block) GasUsed() uint64              { return b.header.GasUsed }
func (b *Block) Difficulty() *big.Int         { return new(big.Int).Set(b.header.Difficulty) }
func (b *Block) Time() *big.Int               { return new(big.Int).Set(b.header.TimeStamp) }
func (b *Block) Nonce() uint64                { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() bloom.Bloom           { return b.header.LogsBloom }
func (b *Block) Miner() utils.Address         { return b.header.Miner }
func (b *Block) StateRoot() utils.Hash        { return b.header.StateRoot }
func (b *Block) PreviousHash() utils.Hash     { return b.header.PreviousHash }
func (b *Block) TransactionsRoot() utils.Hash { return b.header.TransactionsRoot }
func (b *Block) ReceiptsRoot() utils.Hash     { return b.header.ReceiptsRoot }
func (b *Block) ExtraData() []byte            { return utils.CopyBytes(b.header.ExtraData) }
func (b *Block) BlockHeader() *BlockHeader    { return CopyBlockHeader(b.header) }

// Hash returns the keccak256 hash of b's header.
func (b *Block) Hash() utils.Hash {
	return b.header.Hash()
}

func (b *Block) HashNoNonce() utils.Hash {
	return b.header.HashNoNonce()
}

// Size returns the true RLP encoded storage size of the block
func (b *Block) Size() utils.StorageSize {
	c := writeCounter(0)
	rlp.Encode(&c, b)
	return utils.StorageSize(c)
}

// WithTxs  a block with the given txs data.
func (b *Block) WithTxs(txs []*Transaction) *Block {
	b.transactions = txs
	return b
}

// WithSeal returns a new block with the data from b but the header replaced with the sealed one.
func (b *Block) WithSeal(header *BlockHeader) *Block {
	cpy := *header
	return &Block{
		header:       &cpy,
		transactions: b.transactions,
	}
}

// WithStateRoot  a block with the given state root.
func (b *Block) WithStateRoot(root utils.Hash) *Block {
	b.header.StateRoot = root
	return b
}
