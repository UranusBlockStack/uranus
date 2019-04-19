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

package main

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	database "github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/db/leveldb"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/store"
	"github.com/UranusBlockStack/uranus/core/svm"
	"github.com/UranusBlockStack/uranus/core/types"
)

// const
const (
	DefaultNonce             = uint64(1)
	KeyStateRootHash         = "STATE_ROOT_HASH"
	keyGlobalContractAddress = "GLOBAL_CONTRACT_ADDRESS"
)

var prefixFuncHash = []byte("FH-")

func getGlobalContractAddress(db database.Database) utils.Address {
	hexAddr, err := db.GetString(keyGlobalContractAddress)
	if err != nil {
		return utils.EmptyAddress
	}

	return utils.HexMustToAddres(hexAddr)
}

func setGlobalContractAddress(db database.Database, hexAddr string) {
	if err := db.PutString(keyGlobalContractAddress, hexAddr); err != nil {
		panic(err)
	}
}

func setContractCompilationOutput(db database.Database, contractAddress []byte, output *solCompileOutput) {
	key := append(prefixFuncHash, contractAddress...)

	if err := db.Put(key, utils.SerializePanic(output)); err != nil {
		panic(err)
	}
}

func getContractCompilationOutput(db database.Database, contractAddress []byte) *solCompileOutput {
	key := append(prefixFuncHash, contractAddress...)

	value, err := db.Get(key)
	if err != nil {
		return nil
	}

	output := solCompileOutput{}
	if err = utils.Deserialize(value, &output); err != nil {
		panic(err)
	}

	return &output
}

func getFromAddress(statedb *state.Statedb) utils.Address {
	if len(account) == 0 {
		from := *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(from)
		statedb.SetBalance(from, utils.SeeleToFan)
		statedb.SetNonce(from, DefaultNonce)
		return from
	}

	from, err := utils.HexToAddress(account)
	if err != nil {
		fmt.Println("Invalid account address,", err.Error())
		return utils.EmptyAddress
	}

	return from
}

func ensurePrefix(str, prefix string) string {
	if strings.HasPrefix(str, prefix) {
		return str
	}

	return prefix + str
}

// preprocessContract creates the contract tx dependent state DB, blockchain store
func preprocessContract() (database.Database, *state.Statedb, store.BlockchainStore, func(), error) {
	db, err := leveldb.NewLevelDB(defaultDir)
	if err != nil {
		os.RemoveAll(defaultDir)
		return nil, nil, nil, func() {}, err
	}

	hash := utils.EmptyHash
	str, err := db.GetString(KeyStateRootHash)
	if err != nil {
		hash = utils.EmptyHash
	} else {
		h, err := utils.HexToHash(str)
		if err != nil {
			db.Close()
			return nil, nil, nil, func() {}, err
		}
		hash = h
	}

	statedb, err := state.NewStatedb(hash, db)
	if err != nil {
		db.Close()
		return nil, nil, nil, func() {}, err
	}

	return db, statedb, store.NewBlockchainDatabase(db), func() {
		batch := db.NewBatch()
		hash, err := statedb.Commit(batch)
		if err != nil {
			fmt.Println("Failed to commit state DB,", err.Error())
			return
		}

		if err := batch.Commit(); err != nil {
			fmt.Println("Failed to commit batch,", err.Error())
			return
		}

		db.PutString(KeyStateRootHash, hash.Hex())
		db.Close()
	}, nil
}

// Create the contract or call the contract
func processContract(statedb *state.Statedb, bcStore store.BlockchainStore, tx *types.Transaction) (*types.Receipt, error) {
	// A test block header
	header := &types.BlockHeader{
		PreviousBlockHash: crypto.MustHash("block previous hash"),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         crypto.MustHash("state root hash"),
		TxHash:            crypto.MustHash("tx root hash"),
		ReceiptHash:       crypto.MustHash("receipt root hash"),
		Difficulty:        big.NewInt(38),
		Height:            666,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		ExtraData:         make([]byte, 0),
	}

	ctx := &svm.Context{
		Tx:          tx,
		Statedb:     statedb,
		BlockHeader: header,
		BcStore:     bcStore,
	}
	return svm.Process(ctx)
}
