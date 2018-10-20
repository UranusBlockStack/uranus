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

package types

import (
	"math/big"
	"testing"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/utils"
)

var (
	testPrivHex = "9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658"
	testAddrHex = "0xC08B5542D177ac6686946920409741463a15dDdB"
	testmsg     = rlpHash([]byte("test"))
)

func TestRecoverPlain(t *testing.T) {
	key, _ := crypto.HexToECDSA(testPrivHex)
	expAddr := utils.HexToAddress(testAddrHex)
	signature, err := crypto.Sign(testmsg.Bytes(), key)
	if err != nil {
		t.Fatal(err)
	}
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:64])
	v := new(big.Int).SetBytes([]byte{signature[64] + 27})

	addr, err := recoverPlain(testmsg, r, s, v, false)
	if err != nil {
		t.Fatal(err)
	}
	utils.AssertEquals(t, addr, expAddr)

}
