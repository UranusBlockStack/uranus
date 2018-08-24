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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"

	"io"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"golang.org/x/crypto/scrypt"
)

const (
	scryptN     = 2
	scryptP     = 1
	scryptR     = 8
	scryptDKLen = 32
)

func getRandBuff(n int) []byte {
	buff := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, buff)
	if err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	return buff
}

// use scrypt to calculate auth key
func getScryptKey(auth string, salt []byte) ([]byte, error) {
	authArray := []byte(auth)
	return scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
}

// AesCTRXOR encrypts data with CTRXOR
func AesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func writeKeyFile(file string, content []byte) error {
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}

	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

func genNewAccount() (*Account, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	account := &Account{
		PrivateKey: key,
		Address:    addr,
	}

	return account, nil
}
