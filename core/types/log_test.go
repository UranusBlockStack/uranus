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
	"testing"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/stretchr/testify/assert"
)

func TestLogDecodeAndEncode(t *testing.T) {
	log := &LogForStorage{
		Address:          utils.HexToAddress("0xecf8f87f810ecf450940c9f60066b4a7a501d6a7"),
		BlockHash:        utils.HexToHash("0x656c34545f90a730a19008c0e7a7cd4fb3895064b48d6d69761bd5abad681056"),
		BlockHeight:      2019236,
		Data:             []byte("0x000000000000000000000000000000000000000000000001a055690d9db80000"),
		LogIndex:         2,
		TransactionIndex: 3,
		TransactionHash:  utils.HexToHash("0x3b198bfd5d2907285af009e9ae84a0ecd63677110d89d7e030251acb87f6487e"),
		Topics: []utils.Hash{
			utils.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			utils.HexToHash("0x00000000000000000000000080b2c9d7cbbf30a1b0fc8983c647d754c6525615"),
		},
	}

	data, err := rlp.EncodeToBytes(log)
	if err != nil {
		t.Error(err)
	}

	tmpLog := &LogForStorage{}

	err = rlp.DecodeBytes(data, tmpLog)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, log, tmpLog)
}
