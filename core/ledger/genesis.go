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
	"fmt"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
)

// Genesis specifies the header fields, state of a genesis block.
type Genesis struct {
	Config       *params.ChainConfig `json:"config"`
	Nonce        uint64              `json:"nonce"`
	Timestamp    uint64              `json:"timestamp"`
	ExtraData    []byte              `json:"extraData"`
	GasLimit     uint64              `json:"gasLimit"   `
	Difficulty   *big.Int            `json:"difficulty" `
	Mixhash      utils.Hash          `json:"mixHash"`
	Miner        utils.Address       `json:"miner"`
	Height       uint64              `json:"height"`
	GasUsed      uint64              `json:"gasUsed"`
	PreviousHash utils.Hash          `json:"previousHash"`
}

// DefaultGenesis returns the nurans main net genesis block.
func DefaultGenesis() *Genesis {
	extraData, _ := utils.Decode("uranus gensis block")
	return &Genesis{
		Config:     params.DefaultChainConfig,
		Nonce:      1,
		ExtraData:  extraData,
		GasLimit:   params.GenesisGasLimit,
		Difficulty: big.NewInt(1000000),
	}
}

//SetupGenesis The returned chain configuration is never nil.
func SetupGenesis(genesis *Genesis, chain *Chain) (*params.ChainConfig, utils.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return nil, utils.Hash{}, errGenesisNoConfig
	}
	stored := chain.getLegitimateHash(0)
	if (stored == utils.Hash{}) {
		if genesis == nil {
			log.Info("Writing default genesis block")
			genesis = DefaultGenesis()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(chain)
		if err != nil {
			return nil, utils.Hash{}, err
		}
		return genesis.Config, block.Hash(), nil
	}

	return chain.getChainConfig(stored), stored, nil
}

// Commit writes the block and state of a genesis specification to the database.
func (g *Genesis) Commit(chain *Chain) (*types.Block, error) {
	block := g.ToBlock(chain)
	if block.Height().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with Height > 0")
	}
	chain.putTd(block.Hash(), g.Difficulty)
	chain.putBlock(block)
	chain.putReceipts(block.Hash(), nil)
	chain.putLegitimateHash(block.Height().Uint64(), block.Hash())
	chain.putHeadBlockHash(block.Hash())
	chain.putHeadHeaderHash(block.Hash())
	chain.putChainConfig(block.Hash(), g.Config)

	return block, nil
}

// ToBlock creates the genesis block and writes state.
func (g *Genesis) ToBlock(chain *Chain) *types.Block {
	statedb, _ := state.New(utils.Hash{}, state.NewDatabase(chain.db))
	root := statedb.IntermediateRoot(false)
	head := &types.BlockHeader{
		Height:       new(big.Int).SetUint64(g.Height),
		Nonce:        types.EncodeNonce(g.Nonce),
		TimeStamp:    new(big.Int).SetUint64(g.Timestamp),
		PreviousHash: g.PreviousHash,
		ExtraData:    g.ExtraData,
		GasLimit:     g.GasLimit,
		GasUsed:      g.GasUsed,
		Difficulty:   g.Difficulty,
		Miner:        g.Miner,
		StateRoot:    root,
	}
	statedb.Commit(false)

	statedb.Database().TrieDB().Commit(root, true)
	return types.NewBlock(head, nil, nil)
}
