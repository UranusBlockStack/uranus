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
	"path/filepath"
	"testing"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
)

func TestGetAndPutKey(t *testing.T) {
	account, err := genNewAccount()
	if err != nil {
		t.Error(err)
	}

	dir, _ := ioutil.TempDir("", "")

	ks := NewKeyStore(dir)

	fileName := filepath.Join(dir, "keyfile")

	auth := "test"

	if err := ks.PutKey(account, fileName, auth); err != nil {
		t.Error(err)
	}

	newAccount, err := ks.GetKey(account.Address, fileName, auth)
	if err != nil {
		t.Error(err)
	}

	utils.AssertEquals(t, crypto.ByteFromECDSA(account.PrivateKey), crypto.ByteFromECDSA(newAccount.PrivateKey))
	utils.AssertEquals(t, account.Address, newAccount.Address)

}
