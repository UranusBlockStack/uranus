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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	lockcache "github.com/UranusBlockStack/uranus/common/cache"
	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

const (
	defaultAccountCacheLimit = 1000
)

type Wallet struct {
	ks           *KeyStore
	accountCache *lockcache.Cache
}

// NewWallet initialize wallet.
func NewWallet(ksdir string) *Wallet {
	accountCache, _ := lockcache.New(defaultAccountCacheLimit)
	wallet := &Wallet{
		ks:           NewKeyStore(ksdir),
		accountCache: accountCache,
	}
	return wallet
}

// Accounts list all account.
func (w *Wallet) Accounts() (Accounts, error) {
	accounts := Accounts{}
	rd, err := ioutil.ReadDir(w.ks.keyStoreDir)
	if err != nil {
		return nil, err
	}
	for _, fi := range rd {
		if !noKeyFile(fi) {
			addrHex := strings.TrimSuffix(fi.Name(), keyFileSuffix)
			addr := utils.HexToAddress(addrHex[len(addrHex)-40:])
			accounts = append(accounts, Account{
				Address:  addr,
				FileName: fi.Name(),
			})
		}
	}
	sort.Sort(accounts)
	return accounts, nil
}

// ImportRawKey import key in to key store.
func (w *Wallet) ImportRawKey(privkey string, passphrase string) (utils.Address, error) {
	key, err := crypto.HexToECDSA(privkey)
	if err != nil {
		return utils.Address{}, err
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)

	fileName := keyFileName(addr)
	path := filepath.Join(w.ks.keyStoreDir, fileName)
	// check if key exist
	if utils.FileExists(path) {
		return addr, nil
	}
	account := Account{
		Address:    addr,
		PrivateKey: key,
		FileName:   fileName,
	}
	if err := w.ks.PutKey(account, path, passphrase); err != nil {
		return utils.Address{}, err
	}

	w.accountCache.Add(account.Address, &lockAccount{passphrase: passphrase, account: &account})
	return account.Address, nil
}

// NewAccount creates a new account
func (w *Wallet) NewAccount(passphrase string) (Account, error) {
	account, err := genNewAccount()
	if err != nil {
		return Account{}, err
	}

	path := filepath.Join(w.ks.keyStoreDir, account.FileName)
	if err := w.ks.PutKey(account, path, passphrase); err != nil {
		return Account{}, err
	}
	w.accountCache.Add(account.Address, &lockAccount{passphrase: passphrase, account: &account})
	return account, nil
}

// Find resolves the given account into a unique entry in the keystore.
func (w *Wallet) Find(addr utils.Address) (Account, error) {
	if la, ok := w.accountCache.Get(addr); ok {
		return *la.(*lockAccount).account, nil
	}

	rd, err := ioutil.ReadDir(w.ks.keyStoreDir)
	if err != nil {
		return Account{}, err
	}

	for _, fi := range rd {
		if !noKeyFile(fi) {
			addrHex := strings.TrimSuffix(fi.Name(), keyFileSuffix)
			address := utils.HexToAddress(addrHex[len(addrHex)-40:])
			if address == addr {
				return Account{
					Address:  addr,
					FileName: fi.Name(),
				}, nil
			}
		}
	}

	return Account{}, ErrNoMatch
}

// Delete removes the speciified account
func (w *Wallet) Delete(account Account, passphrase string) error {
	path := filepath.Join(w.ks.keyStoreDir, account.FileName)

	if !utils.FileExists(path) {
		return nil
	}
	_, err := w.ks.GetKey(account.Address, path, passphrase)
	if err != nil {
		return err
	}

	w.accountCache.Remove(account.Address)
	return os.Remove(path)
}

// Update update the specified account
func (w *Wallet) Update(account Account, passphrase, newPassphrase string) error {
	path := filepath.Join(w.ks.keyStoreDir, account.FileName)

	if !utils.FileExists(path) {
		return ErrNoMatch
	}
	newaccount, err := w.ks.GetKey(account.Address, path, passphrase)
	if err != nil {
		return err
	}
	newaccount.FileName = account.FileName
	w.accountCache.Add(account.Address, &lockAccount{passphrase: newPassphrase, account: newaccount})
	return w.ks.PutKey(*newaccount, path, newPassphrase)
}

// SignTx sign the specified transaction
func (w *Wallet) SignTx(addr utils.Address, tx *types.Transaction, passphrase string) (*types.Transaction, error) {
	if la, ok := w.accountCache.Get(addr); ok {
		if la.(*lockAccount).passphrase == passphrase {
			if err := tx.SignTx(types.Signer{}, la.(*lockAccount).account.PrivateKey); err != nil {
				return nil, err
			}
			return tx, nil
		}
		return nil, ErrDecrypt
	}

	path := filepath.Join(w.ks.keyStoreDir, keyFileName(addr))
	account, err := w.ks.GetKey(addr, path, passphrase)
	if err != nil {
		return nil, err
	}

	if err := tx.SignTx(types.Signer{}, account.PrivateKey); err != nil {
		return nil, err
	}

	w.accountCache.Add(addr, &lockAccount{passphrase: passphrase, account: account})
	return tx, nil
}

func (w *Wallet) SignHash(addr utils.Address, hash []byte) ([]byte, error) {
	var prv *ecdsa.PrivateKey
	passphrase := "coinbase"

	fileName := filepath.Join(w.ks.keyStoreDir, addr.Hex()+keyFileSuffix)
	account, err := w.ks.GetKey(addr, fileName, passphrase)
	if err != nil {
		return nil, err
	}
	prv = account.PrivateKey
	return crypto.Sign(hash[:], prv)
}
