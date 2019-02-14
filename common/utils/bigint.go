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

package utils

import (
	"math/big"
)

// Big marshals/unmarshals as a JSON string with 0x prefix.
// The zero value marshals as "0x0".
//
// Negative integers are not supported at this time. Attempting to marshal them will
// return an error. Values larger than 256bits are rejected by Unmarshal but will be
// marshaled without error.
type Big big.Int

// ToInt converts b to a big.Int.
func (b *Big) ToInt() *big.Int {
	return (*big.Int)(b)
}

// String returns the hex encoding of b.
func (b *Big) String() string {
	return EncodeBig(b.ToInt())
}

// Add z = x+y; return z
func (z *Big) Add(x *Big, y *Big) *Big {
	z.ToInt().Add(x.ToInt(), y.ToInt())
	return z
}

// Sub z = x = y; return z
func (z *Big) Sub(x *Big, y *Big) *Big {
	z.ToInt().Sub(x.ToInt(), y.ToInt())
	return z
}

// Sadd z+=y, return z
func (z *Big) Sadd(y *Big) *Big {
	z.Add(z, y)
	return z
}

// Ssub z-= y; return z
func (z *Big) Ssub(y *Big) *Big {
	z.Sub(z, y)
	return z
}
