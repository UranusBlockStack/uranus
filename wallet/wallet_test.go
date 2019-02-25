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

package wallet

import (
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/stretchr/testify/assert"
)

func TestAccounts(t *testing.T) {
	dir, _ := ioutil.TempDir("", "test_keystoredir")
	w := NewWallet(dir)
	account1, err := w.NewAccount("test")
	if err != nil {
		t.Fatal(err)
	}

	account2, err := w.NewAccount("test")
	if err != nil {
		t.Fatal(err)
	}

	var accounts Accounts
	accounts = append(accounts, Account{Address: account1.Address, FileName: account1.FileName})
	accounts = append(accounts, Account{Address: account2.Address, FileName: account2.FileName})

	taccounts, err := w.Accounts()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, accounts, taccounts)
}

func TestImportAndExportRawKey(t *testing.T) {
	dir, _ := ioutil.TempDir("", "test_keystoredir")
	w := NewWallet(dir)

	account, err := w.NewAccount("testPass")
	if err != nil {
		t.Fatal(err)
	}

	keyhex, err := w.ExportRawKey(account.Address, "testPass")
	if err != nil {
		t.Fatal(err)
	}

	addr, err := w.ImportRawKey(keyhex, "testPass")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, account.Address, addr)
}

func TestNewAccount(t *testing.T) {
	dir, _ := ioutil.TempDir("", "test_keystoredir")
	w := NewWallet(dir)
	account, err := w.NewAccount("test")
	if err != nil {
		t.Fatal(err)
	}

	ks := NewKeyStore(dir)
	path := filepath.Join(dir, account.FileName)

	newAccount, err := ks.GetKey(account.Address, path, "test")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, account.Address, newAccount.Address)
	assert.Equal(t, crypto.ByteFromECDSA(account.PrivateKey), crypto.ByteFromECDSA(newAccount.PrivateKey))

}

func TestDelete(t *testing.T) {
	dir, _ := ioutil.TempDir("", "test_keystoredir")
	w := NewWallet(dir)
	account, err := w.NewAccount("test")
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Delete(account, "test"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, account.FileName)

	if utils.FileExists(path) {
		t.Error()
	}

	// test remove nil file
	if err := w.Delete(account, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdate(t *testing.T) {
	dir, _ := ioutil.TempDir("", "test_keystoredir")
	w := NewWallet(dir)
	account, err := w.NewAccount("test")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Update(account, "test", "newTest"); err != nil {
		t.Fatal(err)
	}

	ks := NewKeyStore(dir)
	path := filepath.Join(dir, account.FileName)
	newAccount, err := ks.GetKey(account.Address, path, "newTest")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, account.Address, newAccount.Address)
	assert.Equal(t, crypto.ByteFromECDSA(account.PrivateKey), crypto.ByteFromECDSA(newAccount.PrivateKey))

	newAccount, err = ks.GetKey(account.Address, path, "test")
	assert.Equal(t, err, ErrDecrypt)
}

func TestSignTx(t *testing.T) {

	dir, _ := ioutil.TempDir("", "test_keystoredir")
	w := NewWallet(dir)
	account, err := w.NewAccount("test")
	if err != nil {
		t.Fatal(err)
	}
	to := utils.Address{}
	tx := types.NewTransaction(types.Binary, 0, big.NewInt(100), 1000, big.NewInt(100), nil, &to)

	signTx, err := w.SignTx(account.Address, tx, "test")
	if err != nil {
		t.Fatal(err)
	}

	from, err := signTx.Sender(types.Signer{})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, from, account.Address)
}
