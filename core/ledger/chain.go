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

package ledger

import (
	"bytes"
	"encoding/json"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
)

type Chain struct {
	db db.Database
}

// NewChain return Chain store schema
func NewChain(db db.Database) *Chain {
	return &Chain{db: db}
}

func (c *Chain) getBlock(hash utils.Hash) *types.Block {
	header := c.getHeader(hash)
	if header == nil {
		return nil
	}
	txs := c.getTransactions(hash)
	if txs != nil {
		return types.NewBlockWithBlockHeader(header).WithTxs(txs.ToTransactions())
	}
	return types.NewBlockWithBlockHeader(header)
}

func (c *Chain) putBlock(b *types.Block) {
	c.putHeader(b.BlockHeader())
	c.putTransactions(b.Hash(), b.Height().Uint64(), b.Transactions())
}

func (c *Chain) deleteBlock(blockHash utils.Hash) {
	c.deleteReceipts(blockHash)
	c.deleteHeader(blockHash)
	c.deleteTransactions(blockHash)
	c.deleteTd(blockHash)
}

// header

func (c *Chain) HasHeader(blockHash utils.Hash) bool {
	if has, err := c.db.Has(keyHeader(blockHash)); !has || err != nil {
		return false
	}
	return true
}

func (c *Chain) getHeader(blockHash utils.Hash) *types.BlockHeader {
	data, err := c.db.Get(keyHeader(blockHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get header RLP hash: %v, err: %v", blockHash, err)
	}
	if len(data) == 0 {
		return nil
	}
	header := new(types.BlockHeader)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		log.Errorf("Invalid block header RLP hash: %v, err: %v", blockHash, err)
		return nil
	}
	return header
}

func (c *Chain) putHeader(header *types.BlockHeader) {
	if err := c.db.Put(keyHeaderHeight(header.Hash()), utils.EncodeUint64ToByte(header.Height.Uint64())); err != nil {
		log.Errorf("Invalid block header RLP hash: %v, err: %v", header.Hash(), err)
	}
	// put the encoded header
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		log.Fatalf("Failed to RLP encode header err: %v", err)
	}
	if err := c.db.Put(keyHeader(header.Hash()), data); err != nil {
		log.Fatalf("Failed to store header err: %v", err)
	}
}

func (c *Chain) deleteHeader(blockHash utils.Hash) {
	if err := c.db.Delete(keyHeader(blockHash)); err != nil {
		log.Fatalf("Failed to delete header err: %v", err)
	}
	if err := c.db.Delete(keyHeaderHeight(blockHash)); err != nil {
		log.Fatalf("Failed to delete hash to number mapping err: %v", err)
	}
}

// transaction

func (c *Chain) hastransactions(txHash utils.Hash) bool {
	if has, err := c.db.Has(keyTxHashs(txHash)); !has || err != nil {
		return false
	}
	return true
}

func (c *Chain) getTransactions(txHash utils.Hash) types.StorageTxs {
	data, err := c.db.Get(keyTxHashs(txHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get transactions hashs RLP hash: %v, err: %v", txHash, err)
	}
	if len(data) == 0 {
		return nil
	}
	hashs := new([]utils.Hash)
	if err := rlp.Decode(bytes.NewReader(data), hashs); err != nil {
		log.Errorf("Invalid block transactions hashs RLP hash: %v, err: %v", txHash, err)
		return nil
	}

	txs := []*types.StorageTx{}
	for _, v := range *hashs {
		tx := c.getTransaction(v)
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	return txs
}

func (c *Chain) getTransaction(txHash utils.Hash) *types.StorageTx {
	data, err := c.db.Get(keyTransacton(txHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get transaction RLP hash: %v, err: %v", txHash, err)
	}
	if len(data) == 0 {
		return nil
	}
	tx := new(types.StorageTx)
	if err := rlp.Decode(bytes.NewReader(data), tx); err != nil {
		log.Errorf("Invalid block transaction RLP hash: %v,err: %v", txHash, err)
		return nil
	}
	return tx
}

func (c *Chain) putTransactions(blockHash utils.Hash, blockHeight uint64, txs types.Transactions) {
	hashs := make([]utils.Hash, len(txs))

	for k, v := range txs {
		hashs[k] = v.Hash()
		c.putTransaction(v.Hash(), blockHash, uint64(k), blockHeight, v)
	}

	data, err := rlp.EncodeToBytes(hashs)
	if err != nil {
		log.Fatalf("Failed to RLP encode transactions err: %v", err)
	}

	if err := c.db.Put(keyTxHashs(blockHash), data); err != nil {
		log.Fatalf("Failed to store block transactions err: %v", err)
	}
}

func (c *Chain) putTransaction(txHash, blockHash utils.Hash, txIndex, blockHeight uint64, tx *types.Transaction) {
	data, err := rlp.EncodeToBytes(types.NewStorageTx(blockHash, blockHeight, txIndex, tx))
	if err != nil {
		log.Fatalf("Failed to RLP encode transaction err: %v", err)
	}

	if err := c.db.Put(keyTransacton(txHash), data); err != nil {
		log.Fatalf("Failed to store block transaction err: %v", err)
	}
}

func (c *Chain) deleteTransactions(blockHash utils.Hash) {
	data, err := c.db.Get(keyTxHashs(blockHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get transactions hashs RLP hash: %v,err: %v", blockHash, err)
	}
	if len(data) == 0 {
		return
	}
	hashs := new([]utils.Hash)
	if err := rlp.Decode(bytes.NewReader(data), hashs); err != nil {
		log.Errorf("Invalid block transactions hashs RLP hash: %v,err: %v", blockHash, err)
		return
	}

	for _, v := range *hashs {
		c.deleteTransaction(v)
	}

	if err := c.db.Delete(keyTxHashs(blockHash)); err != nil {
		log.Fatalf("Failed to delete block transactions err: %v", err)
	}
}

func (c *Chain) deleteTransaction(txHash utils.Hash) {
	if err := c.db.Delete(keyTransacton(txHash)); err != nil {
		log.Fatalf("Failed to delete block transactions err: %v", err)
	}
}

// receipts

func (c *Chain) getReceipts(blockHash utils.Hash) types.Receipts {
	data, err := c.db.Get(keyTxHashs(blockHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get transactions hashs RLP hash: %v,err: %v", blockHash, err)
	}
	if len(data) == 0 {
		return nil
	}
	hashs := new([]utils.Hash)
	if err := rlp.Decode(bytes.NewReader(data), hashs); err != nil {
		log.Errorf("Invalid block transactions hashs RLP hash: %v,err: %v", blockHash, err)
		return nil
	}

	receipts := types.Receipts{}
	for _, v := range *hashs {
		r := c.getReceipt(v)
		if r != nil {
			receipts = append(receipts, r)
		}
	}
	return receipts
}

func (c *Chain) getReceipt(txHash utils.Hash) *types.Receipt {
	// Retrieve the flattened receipt slice
	data, err := c.db.Get(keyReceipt(txHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get receipts RLP hash: %v,err: %v", txHash, err)
	}
	if len(data) == 0 {
		return nil
	}
	storageReceipt := new(types.ReceiptForStorage)
	if err := rlp.DecodeBytes(data, &storageReceipt); err != nil {
		log.Errorf("Invalid receipt array RLP hash: %v,err: %v", txHash, err)
		return nil
	}
	return (*types.Receipt)(storageReceipt)
}

func (c *Chain) putReceipts(blockHash utils.Hash, receipts types.Receipts) {
	data, err := c.db.Get(keyTxHashs(blockHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get transactions hashs RLP hash: %v,err: %v", blockHash, err)
	}
	if len(data) == 0 {
		return
	}
	hashs := new([]utils.Hash)
	if err := rlp.Decode(bytes.NewReader(data), hashs); err != nil {
		log.Errorf("Invalid block transactions hashs RLP hash: %v,err: %v", blockHash, err)
		return
	}

	for k, v := range *hashs {
		c.putReceipt(v, (*types.ReceiptForStorage)(receipts[k]))
	}
}

func (c *Chain) putReceipt(txHash utils.Hash, receipt *types.ReceiptForStorage) {
	bytes, err := rlp.EncodeToBytes(receipt)
	if err != nil {
		log.Fatalf("Failed to encode block receipts err: %v", err)
	}
	// Store the flattened receipt slice
	if err := c.db.Put(keyReceipt(txHash), bytes); err != nil {
		log.Fatalf("Failed to store block receipts err: %v", err)
	}
}

func (c *Chain) deleteReceipts(blockHash utils.Hash) {
	data, err := c.db.Get(keyTxHashs(blockHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get transactions hashs RLP hash: %v,err: %v", blockHash, err)
	}
	if len(data) == 0 {
		return
	}
	hashs := new([]utils.Hash)
	if err := rlp.Decode(bytes.NewReader(data), hashs); err != nil {
		log.Errorf("Invalid block transactions hashs RLP hash: %v,err: %v", blockHash, err)
		return
	}

	for _, v := range *hashs {
		c.deleteReceipt(v)
	}

}

func (c *Chain) deleteReceipt(txHash utils.Hash) {
	if err := c.db.Delete(keyReceipt(txHash)); err != nil {
		log.Fatalf("Failed to delete block receipts err: %v", err)
	}
}

// td

func (c *Chain) getTd(blockHash utils.Hash) *big.Int {
	data, err := c.db.Get(keyTD(blockHash))
	if err != nil && err != ErrLDBNotFound {
		log.Fatalf("Failed to get td RLP block hash: %v,err: %v", blockHash, err)
	}
	if len(data) == 0 {
		return nil
	}
	td := new(big.Int)
	if err := rlp.Decode(bytes.NewReader(data), td); err != nil {
		log.Errorf("Invalid block total difficulty RLP block hash: %v,err: %v", blockHash, err)
		return nil
	}
	return td
}

func (c *Chain) putTd(blockHash utils.Hash, td *big.Int) {
	data, err := rlp.EncodeToBytes(td)
	if err != nil {
		log.Fatalf("Failed to RLP encode block total difficulty err: %v", err)
	}
	if err := c.db.Put(keyTD(blockHash), data); err != nil {
		log.Fatalf("Failed to store block total difficulty err: %v", err)
	}
}

func (c *Chain) deleteTd(blockHash utils.Hash) {
	if err := c.db.Delete(keyTD(blockHash)); err != nil {
		log.Fatalf("Failed to delete block total difficulty err: %v", err)
	}
}

// legitimate

func (c *Chain) getLegitimateHash(height uint64) utils.Hash {
	data, _ := c.db.Get(append(keyLegitimate, utils.EncodeUint64ToByte(height)...))
	if len(data) == 0 {
		return utils.Hash{}
	}
	return utils.BytesToHash(data)
}

func (c *Chain) putLegitimateHash(height uint64, hash utils.Hash) {
	if err := c.db.Put(append(keyLegitimate, utils.EncodeUint64ToByte(height)...), hash.Bytes()); err != nil {
		log.Fatalf("Failed to store height to hash mapping err: %v", err)
	}
}

func (c *Chain) deleteLegitimateHash(height uint64) {
	if err := c.db.Delete(append(keyLegitimate, utils.EncodeUint64ToByte(height)...)); err != nil {
		log.Fatalf("Failed to delete height to hash mapping err: %v", err)
	}
}

func (c *Chain) getHeaderHeight(blockHash utils.Hash) *uint64 {
	data, _ := c.db.Get(append(keyHeaderHeight(blockHash)))
	if len(data) == 0 {
		return nil
	}
	height, _ := utils.DecodeUint64(string(data))
	return &height
}

func (c *Chain) getHeadBlockHash() utils.Hash {
	data, _ := c.db.Get(keyLastBlock)
	if len(data) == 0 {
		return utils.Hash{}
	}
	return utils.BytesToHash(data)
}

func (c *Chain) putHeadBlockHash(blockHash utils.Hash) {
	if err := c.db.Put(keyLastBlock, blockHash.Bytes()); err != nil {
		log.Fatalf("Failed to store last block's hash err: %v", err)
	}
}

func (c *Chain) getChainConfig(hash utils.Hash) *params.ChainConfig {
	data, _ := c.db.Get(append(keyChainConfig, hash.Bytes()...))
	if len(data) == 0 {
		return nil
	}
	var config params.ChainConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Error("Invalid Chain config JSON", "hash", hash, "err", err)
		return nil
	}
	return &config
}

func (c *Chain) putChainConfig(hash utils.Hash, cfg *params.ChainConfig) {
	if cfg == nil {
		return
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		log.Fatal("Failed to JSON encode Chain config", "err", err)
	}
	if err := c.db.Put(append(keyChainConfig, hash.Bytes()...), data); err != nil {
		log.Fatal("Failed to store Chain config", "err", err)
	}
}
