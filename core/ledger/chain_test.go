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
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	ldb "github.com/UranusBlockStack/uranus/common/db/leveldb"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"golang.org/x/crypto/sha3"
)

func createTestDB(t *testing.T) (string, *ldb.Database) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary : %v", err)
	}

	ldb, err := ldb.New(dir, 0, 0)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	return dir, ldb
}

func TestTransactionAndReceiptsStorage(t *testing.T) {
	dir, db := createTestDB(t)
	defer os.RemoveAll(dir)
	defer db.Close()

	chain := NewChain(db)

	receipts := testReceipt()

	block := testBlock()

	// Check that no transactions entries are in a pristine database
	for i, tx := range block.Transactions() {
		if txn := chain.getTransaction(tx.Hash()); txn != nil {
			t.Fatalf("tx #%d [%x]: non existent transaction returned: %v", i, tx.Hash(), txn)
		}
	}
	// Insert all the transactions into the database, and verify contents
	chain.putBlock(block)

	for i, tx := range block.Transactions() {
		if txn := chain.getTransaction(tx.Hash()); txn == nil {
			t.Fatalf("tx #%d [%x]: transaction not found", i, tx.Hash())
		} else {
			if tx.Hash() != txn.Tx.Hash() {
				t.Fatalf("tx #%d [%x]: transaction mismatch: have %v, want %v", i, tx.Hash(), txn, tx)
			}
		}
	}

	// Insert the receipt slice into the database and check presence
	chain.putReceipts(block.Hash(), receipts)
	if rs := chain.getReceipts(block.Hash()); len(rs) == 0 {
		t.Fatalf("no receipts returned")
	} else {
		for i := 0; i < len(receipts); i++ {
			rlpHave, _ := rlp.EncodeToBytes(rs[i])
			rlpWant, _ := rlp.EncodeToBytes(receipts[i])

			if !bytes.Equal(rlpHave, rlpWant) {
				t.Fatalf("receipt #%d: receipt mismatch: have %v, want %v", i, rs[i], receipts[i])
			}
		}
	}
	// Delete the receipt slice and check purge
	chain.deleteReceipts(block.Hash())
	if rs := chain.getReceipts(block.Hash()); len(rs) != 0 {
		t.Fatalf("deleted receipts returned: %v", rs)
	}

	// Delete the transactions and check purge
	for i, tx := range block.Transactions() {
		chain.deleteTransaction(tx.Hash())
		if txn := chain.getTransaction(tx.Hash()); txn != nil {
			t.Fatalf("tx #%d [%x]: deleted transaction returned: %v", i, tx.Hash(), txn)
		}
	}
}

// Tests block header storage and retrieval operations.
func TestHeaderStorage(t *testing.T) {
	dir, db := createTestDB(t)
	defer os.RemoveAll(dir)
	defer db.Close()
	chain := NewChain(db)

	// Create a test header to move around the database and make sure it's really new
	header := &types.BlockHeader{Height: big.NewInt(42), ExtraData: []byte("test header")}
	if entry := chain.getHeader(header.Hash()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	// put and verify the header in the database
	chain.putHeader(header)
	if entry := chain.getHeader(header.Hash()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != header.Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, header)
	}

	if entry := chain.getHeader(header.Hash()); entry == nil {
		t.Fatalf("Stored header RLP not found")
	} else {
		hasher := sha3.NewLegacyKeccak256()
		data, _ := rlp.EncodeToBytes(entry)
		hasher.Write(data)

		if hash := utils.BytesToHash(hasher.Sum(nil)); hash != header.Hash() {
			t.Fatalf("Retrieved RLP header mismatch: have %v, want %v", entry, header)
		}
	}
	// Delete the header and verify the execution
	chain.deleteHeader(header.Hash())
	if entry := chain.getHeader(header.Hash()); entry != nil {
		t.Fatalf("Deleted header returned: %v", entry)
	}
}

// Tests block storage and retrieval operations.
func TestBlockStorage(t *testing.T) {
	dir, db := createTestDB(t)
	defer os.RemoveAll(dir)
	defer db.Close()
	chain := NewChain(db)

	// Create a test block to move around the database and make sure it's really new
	block := types.NewBlock(&types.BlockHeader{
		ExtraData:        []byte("test block"),
		TransactionsRoot: utils.Hash{},
		ReceiptsRoot:     utils.Hash{},
	}, nil, nil, nil)
	if entry := chain.getBlock(block.Hash()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	if entry := chain.getHeader(block.Hash()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	if entry := chain.getTransactions(block.Hash()); entry != nil {
		t.Fatalf("Non existent body returned: %v", entry)
	}
	// put and verify the block in the database
	chain.putBlock(block)
	if entry := chain.getBlock(block.Hash()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
	if entry := chain.getHeader(block.Hash()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != block.BlockHeader().Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, block.BlockHeader())
	}
	if txs := chain.getTransactions(block.Hash()); txs == nil {
		t.Fatalf("Stored body not found")
	} else if types.DeriveRootHash(txs.ToTransactions()) != types.DeriveRootHash(block.Transactions()) {
		t.Fatalf("Retrieved body mismatch: have %v, want %v", txs, block.Transactions())
	}

	// Delete the block and verify the execution
	chain.deleteBlock(block.Hash())
	if entry := chain.getBlock(block.Hash()); entry != nil {
		t.Fatalf("Deleted block returned: %v", entry)
	}
	if entry := chain.getHeader(block.Hash()); entry != nil {
		t.Fatalf("Deleted header returned: %v", entry)
	}
	if entry := chain.getTransactions(block.Hash()); entry != nil {
		t.Fatalf("Deleted body returned: %v", entry)
	}
}

// Tests that partial block contents don't get reassembled into full blocks.
func TestPartialBlockStorage(t *testing.T) {
	dir, db := createTestDB(t)
	defer os.RemoveAll(dir)
	defer db.Close()
	chain := NewChain(db)

	block := types.NewBlockWithBlockHeader(&types.BlockHeader{
		ExtraData:        []byte("test block"),
		TransactionsRoot: utils.Hash{},
		ReceiptsRoot:     utils.Hash{},
	})
	// Store a header and check that it's not recognized as a block
	chain.putHeader(block.BlockHeader())
	if entry := chain.getBlock(block.Hash()); entry == nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	chain.deleteHeader(block.Hash())

	// Store a body and check that it's not recognized as a block
	chain.putTransactions(block.Hash(), block.Height().Uint64(), block.Transactions())
	if entry := chain.getBlock(block.Hash()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	chain.deleteTransactions(block.Hash())

	// Store a header and a body separately and check reassembly
	chain.putHeader(block.BlockHeader())
	chain.putTransactions(block.Hash(), block.Height().Uint64(), block.Transactions())

	if entry := chain.getBlock(block.Hash()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
}

// Tests block total difficulty storage and retrieval operations.
func TestTdStorage(t *testing.T) {
	dir, db := createTestDB(t)
	defer os.RemoveAll(dir)
	defer db.Close()
	chain := NewChain(db)

	// Create a test TD to move around the database and make sure it's really new
	hash, td := utils.Hash{}, big.NewInt(314)
	if entry := chain.getTd(hash); entry != nil {
		t.Fatalf("Non existent TD returned: %v", entry)
	}
	// Write and verify the TD in the database
	chain.putTd(hash, td)
	if entry := chain.getTd(hash); entry == nil {
		t.Fatalf("Stored TD not found")
	} else if entry.Cmp(td) != 0 {
		t.Fatalf("Retrieved TD mismatch: have %v, want %v", entry, td)
	}
	// Delete the TD and verify the execution
	chain.deleteTd(hash)
	if entry := chain.getTd(hash); entry != nil {
		t.Fatalf("Deleted TD returned: %v", entry)
	}
}

func TestLegitimateMappingStorage(t *testing.T) {
	dir, db := createTestDB(t)
	defer os.RemoveAll(dir)
	defer db.Close()
	chain := NewChain(db)

	// Create a test legitimate Height and assinged hash to move around
	hash, Height := utils.Hash{0: 0xff}, uint64(314)
	if entry := chain.getLegitimateHash(Height); entry != (utils.Hash{}) {
		t.Fatalf("Non existent legitimate mapping returned: %v", entry)
	}
	// Write and verify the TD in the database
	chain.putLegitimateHash(Height, hash)
	if entry := chain.getLegitimateHash(Height); entry == (utils.Hash{}) {
		t.Fatalf("Stored legitimate mapping not found")
	} else if entry != hash {
		t.Fatalf("Retrieved legitimate mapping mismatch: have %v, want %v", entry, hash)
	}
	// Delete the TD and verify the execution
	chain.deleteLegitimateHash(Height)
	if entry := chain.getLegitimateHash(Height); entry != (utils.Hash{}) {
		t.Fatalf("Deleted legitimate mapping returned: %v", entry)
	}
}

func TestHeadStorage(t *testing.T) {
	dir, db := createTestDB(t)
	defer os.RemoveAll(dir)
	defer db.Close()
	chain := NewChain(db)

	blockFull := types.NewBlockWithBlockHeader(&types.BlockHeader{ExtraData: []byte("test block full")})

	if entry := chain.getHeadBlockHash(); entry != (utils.Hash{}) {
		t.Fatalf("Non head block entry returned: %v", entry)
	}

	// Assign separate entries for the head header and block
	chain.putHeadBlockHash(blockFull.Hash())

	if entry := chain.getHeadBlockHash(); entry != blockFull.Hash() {
		t.Fatalf("Head block hash mismatch: have %v, want %v", entry, blockFull.Hash())
	}

}
