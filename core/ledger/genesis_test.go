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
	"os"
	"testing"

	mdb "github.com/UranusBlockStack/uranus/common/db/memorydb"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/params"
	"github.com/stretchr/testify/assert"
)

func TestDefaultGenesis(t *testing.T) {
	block, _ := DefaultGenesis().ToBlock(NewChain(mdb.New()))
	assert.Equal(t, block.Hash().Hex(), "0x437ccc55c2785859205318c2906d955e2cb35b76cb46f88e1df1611732f4d09c")
}

func TestSetupGenesisBlock(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(c *Chain) (*params.ChainConfig, state.Database, utils.Hash, error)
		wantConfig *params.ChainConfig
		wantHash   utils.Hash
		wantErr    error
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(c *Chain) (*params.ChainConfig, state.Database, utils.Hash, error) {
				return SetupGenesis(new(Genesis), c)
			},
			wantErr:    errGenesisNoConfig,
			wantConfig: nil,
		},
		{
			name: "no block in DB, genesis == nil",
			fn: func(c *Chain) (*params.ChainConfig, state.Database, utils.Hash, error) {
				return SetupGenesis(nil, c)
			},
			wantHash:   utils.HexToHash("0x437ccc55c2785859205318c2906d955e2cb35b76cb46f88e1df1611732f4d09c"),
			wantConfig: params.DefaultChainConfig,
		},
		{
			name: "genesis block in DB, genesis == nil",
			fn: func(c *Chain) (*params.ChainConfig, state.Database, utils.Hash, error) {
				DefaultGenesis().Commit(c)
				return SetupGenesis(nil, c)
			},
			wantHash:   utils.HexToHash("0x437ccc55c2785859205318c2906d955e2cb35b76cb46f88e1df1611732f4d09c"),
			wantConfig: params.DefaultChainConfig,
		},
	}
	for _, test := range tests {
		dir, db := createTestDB(t)
		defer os.RemoveAll(dir)
		defer db.Close()

		Chaindb := NewChain(db)
		config, _, hash, err := test.fn(Chaindb)

		assert.Equal(t, test.wantErr, err)
		assert.Equal(t, test.wantConfig, config)
		if hash != test.wantHash {
			t.Errorf("%s: returned hash %s, want %s", test.name, hash.Hex(), test.wantHash.Hex())
		} else if err == nil {
			// Check database content.
			stored := Chaindb.getBlock(test.wantHash)
			if stored.Hash() != test.wantHash {
				t.Errorf("%s: block in DB has hash %s, want %s", test.name, stored.Hash(), test.wantHash)
			}
		}

	}

}
