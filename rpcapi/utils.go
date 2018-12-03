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
	"context"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

// submitTransaction is a helper function that submits tx to txPool and logs a message.
func submitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (utils.Hash, error) {
	if err := b.SendTx(ctx, tx); err != nil {
		return utils.Hash{}, err
	}
	if tx.Tos() == nil {
		from, err := tx.Sender(types.Signer{})
		if err != nil {
			return utils.Hash{}, err
		}
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Infof("Submitted contract creation hash: %v ,contract address: %v", tx.Hash(), addr)
	} else {
		log.Infof("Submitted transaction hash: %v,recipient address: %v", tx.Hash(), tx.Tos())
	}
	return tx.Hash(), nil
}
