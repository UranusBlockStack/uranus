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
	"crypto/ecdsa"

	"github.com/UranusBlockStack/uranus/common/utils"
)

type encryptedKey struct {
	Address string     `json:"addr"`
	Crypto  cryptoJSON `json:"crypto"`
}

type cryptoJSON struct {
	Cipher     string `json:"cipher"`
	CipherText string `json:"ciphertext"`
	CipherIV   string `json:"iv"`
	KDF        string `json:"kdf"`
	KDFSalt    string `json:"kdfsalt"`
	MAC        string `json:"mac"`
}

type lockAccount struct {
	account    *Account
	passphrase string
}

type Account struct {
	Address    utils.Address
	PrivateKey *ecdsa.PrivateKey
}
