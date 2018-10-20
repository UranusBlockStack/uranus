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

package bloom

import (
	"math/big"
	"testing"
)

type bytesBacked interface {
	Bytes() []byte
}

func BloomLookup(bin Bloom, topic bytesBacked) bool {
	tbloom := bin.Big()
	cmp := Bloom9(topic.Bytes()[:])

	return tbloom.And(tbloom, cmp).Cmp(cmp) == 0
}

func TestBloom(t *testing.T) {
	positive := []string{
		"test1",
		"test2",
		"test3",
		"hello",
		"world",
	}

	negative := []string{
		"tes",
		"lo",
		"ld",
	}

	var bloom Bloom
	for _, data := range positive {
		bloom.Add(new(big.Int).SetBytes([]byte(data)))
	}

	for _, data := range positive {
		if !BloomLookup(bloom, new(big.Int).SetBytes([]byte(data))) {
			t.Error("expected", data, "to test true")
		}
	}
	for _, data := range negative {
		if BloomLookup(bloom, new(big.Int).SetBytes([]byte(data))) {
			t.Error("did not expect", data, "to test true")
		}
	}
}
