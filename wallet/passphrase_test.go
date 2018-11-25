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
	"testing"

	"github.com/UranusBlockStack/uranus/common/utils"
)

func TestKeyEncryptDecrypt(t *testing.T) {
	keyjson, err := ioutil.ReadFile("test-scrypt.json")
	if err != nil {
		t.Fatal(err)
	}

	password := "foo"
	address := utils.HexToAddress("20d218714ade0e9cd1b0d5777e0fce5dac3cfd56")

	if _, err := DecryptKey(keyjson, password+"bad"); err == nil {
		t.Errorf("  json key decrypted with bad password")
	}

	account, err := DecryptKey(keyjson, password)
	if err != nil {
		t.Errorf(" json key failed to decrypt: %v", err)
	}
	if account.Address != address {
		t.Errorf(" key address mismatch: have %x, want %x", account.Address, address)
	}
	password += "new data appended"
	if keyjson, err = EncryptKey(*account, password); err != nil {
		t.Errorf(" failed to recrypt key %v", err)
	}
}
