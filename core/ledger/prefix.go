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

package ledger

import "github.com/UranusBlockStack/uranus/common/utils"

var (
	// key
	keyDBVersion   = []byte("DBVersion")
	keyChainConfig = []byte("chainConfig")
	keyLegitimate  = []byte("legitimate")
	keyLastHeader  = []byte("LastHeader")
	keyLastBlock   = []byte("LastBlock")

	keyTD = func(hash utils.Hash) []byte { return append([]byte("td"), hash.Bytes()...) }

	keyHeader       = func(hash utils.Hash) []byte { return append([]byte("h"), hash.Bytes()...) }
	keyHeaderHash   = func(number uint64) []byte { return append([]byte("hh"), utils.EncodeUint64ToByte(number)...) }
	keyHeaderHeight = func(hash utils.Hash) []byte { return append([]byte("hn"), hash.Bytes()...) }

	keyBlock      = func(hash utils.Hash) []byte { return append([]byte("b"), hash.Bytes()...) }
	keyTxHashs    = func(hash utils.Hash) []byte { return append([]byte("txhs"), hash.Bytes()...) }
	keyReceipt    = func(hash utils.Hash) []byte { return append([]byte("r"), hash.Bytes()...) }
	keyTransacton = func(hash utils.Hash) []byte { return append([]byte("tx"), hash.Bytes()...) }
)
