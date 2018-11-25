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
	"path/filepath"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
)

// KeyStore definition
type KeyStore struct {
	keyStoreDir string
}

// NewKeyStore new a KeyStore instance
func NewKeyStore(keydir string) *KeyStore {
	_, err := utils.OpenDir(keydir)
	if err != nil {
		log.Warn("Func NewKeyStore open key store dir failed: %v", err)
	}
	return &KeyStore{keyStoreDir: keydir}
}

// GetKey returns the key by the specified addr
func (ks KeyStore) GetKey(addr utils.Address, filename, auth string) (*Account, error) {
	keyjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	Account, err := DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	if Account.Address != addr {
		return nil, fmt.Errorf("address content mismatch: have  %x, want %x", Account.Address, addr)
	}
	return Account, nil
}

// PutKey stores the specified key
func (ks KeyStore) PutKey(account Account, path string, auth string) error {
	keyjson, err := EncryptKey(account, auth)
	if err != nil {
		return err
	}

	return writeKeyFile(path, keyjson)
}

// JoinPath returns the abs path of the keystore file
func (ks KeyStore) JoinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(ks.keyStoreDir, filename)
}
