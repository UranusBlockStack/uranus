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
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"io"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
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

func genNewAccount() (Account, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return Account{}, err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return Account{
		PrivateKey: key,
		Address:    addr,
		FileName:   keyFileName(addr),
	}, nil
}

var keyFileSuffix = `.json`

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyAddr utils.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:])) + keyFileSuffix
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

func noKeyFile(fi os.FileInfo) bool {
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}
