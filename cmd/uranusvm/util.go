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
	"math"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	database "github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/db/leveldb"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/executor"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/params"
	jww "github.com/spf13/jwalterweatherman"
)

// const
const (
	DefaultNonce             = uint64(1)
	KeyStateRootHash         = "STATE_ROOT_HASH"
	keyGlobalContractAddress = "GLOBAL_CONTRACT_ADDRESS"
)

var prefixFuncHash = []byte("FH-")

func getGlobalContractAddress(db database.Database) utils.Address {
	byteAddr, err := db.Get([]byte(keyGlobalContractAddress))
	if err != nil {
		return utils.EmptyAddress
	}

	return utils.BytesToAddress(byteAddr)
}

func setGlobalContractAddress(db database.Database, addr utils.Address) {
	if err := db.Put([]byte(keyGlobalContractAddress), addr.Bytes()); err != nil {
		jww.ERROR.Println("setGlobalContractAddress failed ", err.Error())
	}
}

func setContractCompilationOutput(db database.Database, contractAddress []byte, output *solCompileOutput) {
	key := append(prefixFuncHash, contractAddress...)
	byteOutput, err := rlp.Serialize(output)
	if err != nil {
		jww.ERROR.Println(err)

	}
	if err := db.Put(key, byteOutput); err != nil {
		jww.ERROR.Println(err)
	}
}

func getContractCompilationOutput(db database.Database, contractAddress []byte) *solCompileOutput {
	key := append(prefixFuncHash, contractAddress...)

	value, err := db.Get(key)
	if err != nil {
		return nil
	}

	output := solCompileOutput{}
	if err = rlp.Deserialize(value, &output); err != nil {
		jww.ERROR.Println(err)
	}

	return &output
}

func getFromAddress(statedb *state.StateDB) utils.Address {
	if len(account) == 0 {
		from := *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(from)
		statedb.SetBalance(from, new(big.Int).SetUint64(math.MaxUint64))
		statedb.SetNonce(from, DefaultNonce)
		return from
	}

	return utils.HexToAddress(account)
}

func ensurePrefix(str, prefix string) string {
	if strings.HasPrefix(str, prefix) {
		return str
	}

	return prefix + str
}

// preprocessContract creates the contract tx dependent state DB, blockchain store
func preprocessContract() (database.Database, *state.StateDB, *executor.Executor, func(), error) {
	db, err := leveldb.New(defaultDir, 0, 0)
	if err != nil {
		os.RemoveAll(defaultDir)
		return nil, nil, nil, func() {}, err
	}

	hash := utils.Hash{}
	b, err := db.Get([]byte(KeyStateRootHash))
	if err != nil {
		hash = utils.Hash{}
	} else {
		hash = utils.BytesToHash(b)
	}

	statedb, err := state.New(hash, state.NewDatabase(db))
	if err != nil {
		db.Close()
		return nil, nil, nil, func() {}, err
	}

	exec := executor.NewExecutor(params.DefaultChainConfig, nil, nil, nil)
	return db, statedb, exec, func() {
		hash, err := statedb.Commit(true)
		if err != nil {
			jww.ERROR.Println("Failed to commit state DB,", err.Error())
			return
		}

		statedb.Database().TrieDB().Commit(hash, false)
		db.Put([]byte(KeyStateRootHash), hash.Bytes())
		db.Close()
	}, nil
}

// Create the contract or call the contract
func processContract(from utils.Address, statedb *state.StateDB, exec *executor.Executor, tx *types.Transaction) ([]byte, *types.Receipt, error) {
	gasUsed := uint64(0)
	gp := new(utils.GasPool).AddGas(tx.Gas())
	header := &types.BlockHeader{
		PreviousHash:     utils.BytesToHash([]byte("block previous hash")),
		Miner:            *crypto.MustGenerateRandomAddress(),
		StateRoot:        utils.BytesToHash([]byte("state root hash")),
		TransactionsRoot: utils.BytesToHash([]byte("tx root hash")),
		ReceiptsRoot:     utils.BytesToHash([]byte("receipt root hash")),
		Difficulty:       big.NewInt(38),
		Height:           big.NewInt(666),
		GasLimit:         tx.Gas(),
		TimeStamp:        big.NewInt(time.Now().Unix()),
		ExtraData:        make([]byte, 0),
	}

	result, receipt, _, err := exec.ExecTransaction(nil, &from, nil, gp, statedb, header, tx, &gasUsed, vm.Config{})
	if err != nil {
		return nil, nil, err
	}
	return result, receipt, nil
}
