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

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenGenesis(t *testing.T) {
	genesis := DefaultGenesis()

	j, err := genesis.MarshalJSON()
	if err != nil {
		panic(fmt.Sprintf("genesis marshal --- %v", err))
	}
	//fmt.Println("genesis", string(j))

	if err := genesis.UnmarshalJSON(j); err != nil {
		panic(fmt.Sprintf("genesis marshal --- %v", err))
	}
	i, _ := genesis.MarshalJSON()
	assert.Equal(t, string(i), string(j))

	s := `{
		"alloc": {
			"0xde1e758511a7c67e7db93d1c23c1060a21db4615": {
			  "balance": 1000
			},
			"0x27dc8de9e9a1cb673543bd5fce89e83af09e228f": {
				"balance": 1100
			},
			"0xd64a66c28a6ae5150af5e7c34696502793b91ae7": {
			"balance": 900
			}
		},
		"config": {
		  "chainID": 72
		},
		"nonce": "0x0000000000000000",
		"difficulty": "0x4000",
		"mixhash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase": "0x0000000000000000000000000000000000000000",
		"timestamp": "0x00",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"extraData": "0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa",
		"gasLimit": "0xffffffff",
		"validator":"0xd64a66c28a6ae5150af5e7c34696502793b91ae7"
	   }
	   `

	if err := json.Unmarshal([]byte(s), genesis); err != nil {
		panic(fmt.Sprintf("genesis unmarshal --- %v", err))
	}
	j, _ = json.Marshal(genesis)
	fmt.Println("genesis", string(j))
}
