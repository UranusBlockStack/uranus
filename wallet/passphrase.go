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
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/math"
)

// EncryptKey encrypt the key via auth
func EncryptKey(account Account, auth string) ([]byte, error) {
	salt := getRandBuff(32)
	derivedKey, err := getScryptKey(auth, salt)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := math.PaddedBigBytes(account.PrivateKey.D, 32)

	iv := getRandBuff(aes.BlockSize) // 16
	cipherText, err := AesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)
	cryptoInfo := cryptoJSON{
		Cipher:     "aes-128-ctr",
		CipherText: hex.EncodeToString(cipherText),
		CipherIV:   hex.EncodeToString(iv),
		KDF:        "scrypt",
		KDFSalt:    hex.EncodeToString(salt),
		MAC:        hex.EncodeToString(mac),
	}
	return json.Marshal(&encryptedKey{
		hex.EncodeToString(account.Address[:]),
		cryptoInfo,
	})
}

// DecryptKey returns the decrypted key via auth
func DecryptKey(keyjson []byte, auth string) (*Account, error) {
	k := new(encryptedKey)
	if err := json.Unmarshal(keyjson, k); err != nil {
		return nil, err
	}

	keyBytes, err := decryptKey(k, auth)
	if err != nil {
		return nil, err
	}
	key, err := crypto.ByteToECDSA(keyBytes, true)
	if err != nil {
		return nil, err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return &Account{
		Address:    addr,
		PrivateKey: key,
		FileName:   keyFileName(addr),
	}, nil
}

func decryptKey(keyProtected *encryptedKey, auth string) (keyBytes []byte, err error) {
	if keyProtected.Crypto.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("Cipher not supported: %v", keyProtected.Crypto.Cipher)
	}
	mac, err := hex.DecodeString(keyProtected.Crypto.MAC)
	if err != nil {
		return nil, err
	}
	iv, err := hex.DecodeString(keyProtected.Crypto.CipherIV)
	if err != nil {
		return nil, err
	}
	cipherText, err := hex.DecodeString(keyProtected.Crypto.CipherText)
	if err != nil {
		return nil, err
	}

	if keyProtected.Crypto.KDF != "scrypt" {
		return nil, fmt.Errorf("KDF not supported: %v", keyProtected.Crypto.KDF)

	}

	salt, err := hex.DecodeString(keyProtected.Crypto.KDFSalt)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getScryptKey(auth, salt)
	if err != nil {
		return nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrDecrypt
	}

	return AesCTRXOR(derivedKey[:16], cipherText, iv)
}
