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
	"testing"

	"github.com/UranusBlockStack/uranus/common/utils"
)

var (
	key, _ = GenerateKey()

	testmsg     = Keccak256([]byte("test"))
	testsig, _  = Sign(testmsg, key)
	testpubkey  = FromECDSAPub(&key.PublicKey)
	testpubkeyc = CompressPubkey(&key.PublicKey)
)

func TestSignAndEcrecover(t *testing.T) {
	key, _ := HexToECDSA(testPrivHex)
	addr := utils.HexToAddress(testAddrHex)

	sig, err := Sign(testmsg, key)
	if err != nil {
		t.Errorf("Sign error: %s", err)
	}
	recoveredPub, err := EcrecoverToByte(testmsg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	pubKey := ByteToECDSAPub(recoveredPub)
	utils.AssertEquals(t, PubkeyToAddress(*pubKey), addr)

	recoveredPub2, err := EcrecoverToPub(testmsg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	utils.AssertEquals(t, PubkeyToAddress(*recoveredPub2), addr)

}

func TestInvalidSign(t *testing.T) {
	if _, err := Sign(make([]byte, 1), nil); err == nil {
		t.Errorf("expected sign with hash 1 byte to error")
	}
	if _, err := Sign(make([]byte, 33), nil); err == nil {
		t.Errorf("expected sign with hash 33 byte to error")
	}
}

func TestVerifySignature(t *testing.T) {
	sig := testsig[:len(testsig)-1] // remove recovery id

	if !VerifySignature(testpubkey, testmsg, sig) {
		t.Errorf("can't verify signature with uncompressed key")
	}
	if !VerifySignature(testpubkeyc, testmsg, sig) {
		t.Errorf("can't verify signature with compressed key")
	}
	if VerifySignature(nil, testmsg, sig) {
		t.Errorf("signature valid with no key")
	}
	if VerifySignature(testpubkey, nil, sig) {
		t.Errorf("signature valid with no message")
	}
	if VerifySignature(testpubkey, testmsg, nil) {
		t.Errorf("nil signature valid")
	}
	if VerifySignature(testpubkey, testmsg, append(utils.CopyBytes(sig), 1, 2, 3)) {
		t.Errorf("signature valid with extra bytes at the end")
	}
	if VerifySignature(testpubkey, testmsg, sig[:len(sig)-2]) {
		t.Errorf("signature valid even though it's incomplete")
	}
	wrongkey := utils.CopyBytes(testpubkey)
	wrongkey[10]++
	if VerifySignature(wrongkey, testmsg, sig) {
		t.Errorf("signature valid with with wrong public key")
	}
}

func BenchmarkEcrecoverSignature(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := EcrecoverToByte(testmsg, testsig); err != nil {
			b.Fatal("ecrecover error", err)
		}
	}
}

func BenchmarkVerifySignature(b *testing.B) {
	sig := testsig[:len(testsig)-1] // remove recovery id
	for i := 0; i < b.N; i++ {
		if !VerifySignature(testpubkey, testmsg, sig) {
			b.Fatal("verify error")
		}
	}
}

func BenchmarkDecompressPubkey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := DecompressPubkey(testpubkeyc); err != nil {
			b.Fatal(err)
		}
	}
}
