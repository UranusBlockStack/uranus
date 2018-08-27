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

package rpcapi

import (
	"github.com/UranusBlockStack/uranus/common/utils"
)

// WalletAPI exposes methods for the RPC interface
type WalletAPI struct {
	b Backend
}

// NewWalletAPI creates a new RPC service with methods specific for the wallet.
func NewWalletAPI(b Backend) *WalletAPI {
	return &WalletAPI{b}
}

// NewAccount create a new account, and generate keyfile in keystore dir.
func (w *WalletAPI) NewAccount(passphrase string, reply *utils.Address) error {
	account, err := w.b.NewAccount(passphrase)
	if err != nil {
		return err
	}
	*reply = account.Address
	return nil
}

type DeleteArgs struct {
	Address    utils.Address
	Passphrase string
}

// Delete delete a account by address and passphrase, at the same time, delete keyfile.
func (w *WalletAPI) Delete(args DeleteArgs, reply *bool) error {
	if err := w.b.Delete(args.Address, args.Passphrase); err != nil {
		return err
	}
	*reply = true
	return nil
}

type UpdateArgs struct {
	Address       utils.Address
	Passphrase    string
	NewPassphrase string
}

// Update a account passphrase, at the same time, update keyfile.
func (w *WalletAPI) Update(args UpdateArgs, reply *bool) error {
	if err := w.b.Update(args.Address, args.Passphrase, args.NewPassphrase); err != nil {
		return err
	}
	*reply = true
	return nil
}

// Accounts list all wallet account.
func (w *WalletAPI) Accounts(ignore string, reply *[]utils.Address) error {
	account, err := w.b.Accounts()
	if err != nil {
		return err
	}
	*reply = account
	return nil
}

type ImportRawKeyArgs struct {
	PrivKeyHex string
	Passphrase string
}

// ImportRawKey  import raw PrivateKey into walet.
func (w *WalletAPI) ImportRawKey(args ImportRawKeyArgs, reply *utils.Address) error {
	addr, err := w.b.ImportRawKey(args.PrivKeyHex, args.Passphrase)
	if err != nil {
		return nil
	}

	*reply = addr
	return nil
}
