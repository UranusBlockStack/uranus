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

// deriveSigner makes a *best* guess about which signer to use.

package types

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
)

func recoverPlain(sighash utils.Hash, R, S, Vb *big.Int) (utils.Address, error) {
	if Vb.BitLen() > 8 {
		return utils.Address{}, fmt.Errorf("invalid chain id for signer --- bit length %d > 8", Vb.BitLen())
	}
	V := byte(Vb.Uint64() - 27)
	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the snature
	pub, err := crypto.EcrecoverToByte(sighash[:], sig)
	if err != nil {
		return utils.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return utils.Address{}, errors.New("invalid public key")
	}
	var addr utils.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

// chainID derives the chain id from the given v parameter
func chainID(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

// isProtectedV returns whether is protected from replay protection.
func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28
	}
	// anything not 27 or 28 are considered unprotected
	return true
}

// Signer encapsulates transaction signature handling.
type Signer struct{}

// SignatureValues returns signature values. This signature  needs to be in the [R || S || V] format where V is 0 or 1.
func (s Signer) SignatureValues(signature []byte) (r, sb, v *big.Int, err error) {
	if len(signature) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(signature)))
	}
	r = new(big.Int).SetBytes(signature[:32])
	sb = new(big.Int).SetBytes(signature[32:64])
	v = new(big.Int).SetBytes([]byte{signature[64] + 27})
	return r, sb, v, nil
}

// Hash returns the hash to be signed by the sender.
func (s Signer) Hash(tx *Transaction) utils.Hash {
	return rlpHash([]interface{}{
		tx.data.Type,
		tx.data.Nonce,
		tx.data.GasPrice,
		tx.data.GasLimit,
		tx.data.Tos,
		tx.data.Value,
		tx.data.Payload,
	})
}
