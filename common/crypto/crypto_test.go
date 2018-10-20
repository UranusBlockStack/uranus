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

package crypto

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/UranusBlockStack/uranus/common/utils"
)

var (
	testPrivHex = "9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658"
	testAddrHex = "0xC08B5542D177ac6686946920409741463a15dDdB"
	fileName    = "test_key"
)

func TestKeccak256(t *testing.T) {
	msg := []byte("test")
	exp, _ := hex.DecodeString("9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658")
	hashFunc := func(in []byte) []byte { h := Keccak256Hash(in); return h[:] }
	// test Sha3-256-array
	utils.AssertEquals(t, hashFunc(msg), exp)
}

func TestCreateContractAddr(t *testing.T) {

	key, _ := HexToECDSA(testPrivHex)
	addr := utils.HexToAddress(testAddrHex)
	genAddr := PubkeyToAddress(key.PublicKey)

	utils.AssertEquals(t, genAddr, addr)

	caddr0 := CreateAddress(addr, 0)
	caddr1 := CreateAddress(addr, 1)
	caddr2 := CreateAddress(addr, 2)

	utils.AssertEquals(t, caddr0, utils.HexToAddress("0x85687286F24F7F85b01c9447AA87EbbD854E9a85"))
	utils.AssertEquals(t, caddr1, utils.HexToAddress("0xCAbD645c6CB887D0CE10Ed1A21e806b54F2e7529"))
	utils.AssertEquals(t, caddr2, utils.HexToAddress("0x5aCbe50B03022610a48d66F16bfB0a1A8F5e11e2"))
}

func TestSaveECDSA(t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	if err := SaveECDSA(fileName, key); err != nil {
		t.Fatal(err)
	}

	defer os.Remove(fileName)

	key1, err := LoadECDSA(fileName)
	if err != nil {
		t.Fatal(err)
	}
	utils.AssertEquals(t, PubkeyToAddress(key1.PublicKey), utils.HexToAddress(testAddrHex))

}
func TestLoadECDSAFile(t *testing.T) {
	ioutil.WriteFile(fileName, []byte(testPrivHex), 0600)
	defer os.Remove(fileName)

	key, err := LoadECDSA(fileName)
	if err != nil {
		t.Fatal(err)
	}
	utils.AssertEquals(t, PubkeyToAddress(key.PublicKey), utils.HexToAddress(testAddrHex))

}

func BenchmarkSha3(b *testing.B) {
	a := []byte("hello world")
	for i := 0; i < b.N; i++ {
		Keccak256(a)
	}
}
