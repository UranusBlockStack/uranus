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
	"fmt"
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

	addresses := []utils.Address{account1.Address, account2.Address}

	taddresses, err := w.Accounts()
	if err != nil {
		t.Fatal(err)
	}

	_ = taddresses
	_ = addresses
	// todo sort
	// assert.Equal(t, addresses, taddresses)
}

func TestImportRawKey(t *testing.T) {
	dir, _ := ioutil.TempDir("", "test_keystoredir")
	w := NewWallet(dir)
	account, err := genNewAccount()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(utils.BytesToHex(crypto.ByteFromECDSA(account.PrivateKey)))

	addr, err := w.ImportRawKey(utils.BytesToHex(crypto.ByteFromECDSA(account.PrivateKey)), "test")
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
	fileName := filepath.Join(dir, account.Address.Hex()+".json")

	newAccount, err := ks.GetKey(account.Address, fileName, "test")
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

	if err := w.Delete(account.Address, "test"); err != nil {
		t.Fatal(err)
	}
	fileName := filepath.Join(dir, account.Address.Hex()+".json")

	if utils.FileExists(fileName) {
		t.Error()
	}

	// test remove nil file
	if err := w.Delete(account.Address, "test"); err != nil {
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
	if err := w.Update(account.Address, "test", "newTest"); err != nil {
		t.Fatal(err)
	}

	ks := NewKeyStore(dir)
	fileName := filepath.Join(dir, account.Address.Hex()+".json")
	newAccount, err := ks.GetKey(account.Address, fileName, "newTest")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, account.Address, newAccount.Address)
	assert.Equal(t, crypto.ByteFromECDSA(account.PrivateKey), crypto.ByteFromECDSA(newAccount.PrivateKey))

	newAccount, err = ks.GetKey(account.Address, fileName, "test")
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
	tx := types.NewTransaction(0, &to, big.NewInt(100), 1000, big.NewInt(100), nil)

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
