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
	Validators   []utils.Address     `json:"validators"`
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
		Validators: []utils.Address{
			utils.HexToAddress(params.GenesisCandidate),
		},
	}
}

//SetupGenesis The returned chain configuration is never nil.
func SetupGenesis(genesis *Genesis, chain *Chain) (*params.ChainConfig, state.Database, utils.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return nil, nil, utils.Hash{}, errGenesisNoConfig
	}
	stored := chain.getLegitimateHash(0)
	if (stored == utils.Hash{}) {
		if genesis == nil {
			log.Info("Writing default genesis block")
			genesis = DefaultGenesis()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, statedb, err := genesis.Commit(chain)
		if err != nil {
			return nil, nil, utils.Hash{}, err
		}
		return genesis.Config, statedb, block.Hash(), nil
	}

	return chain.getChainConfig(stored), state.NewDatabase(chain.db), stored, nil
}

// Commit writes the block and state of a genesis specification to the database.
func (g *Genesis) Commit(chain *Chain) (*types.Block, state.Database, error) {
	block, statedb := g.ToBlock(chain)
	if block.Height().Sign() != 0 {
		return nil, statedb, fmt.Errorf("can't commit genesis block with Height > 0")
	}
	chain.putTd(block.Hash(), g.Difficulty)
	chain.putBlock(block)
	chain.putReceipts(block.Hash(), nil)
	chain.putLegitimateHash(block.Height().Uint64(), block.Hash())
	chain.putHeadBlockHash(block.Hash())
	chain.putChainConfig(block.Hash(), g.Config)

	return block, statedb, nil
}

// ToBlock creates the genesis block and writes state.
func (g *Genesis) ToBlock(chain *Chain) (*types.Block, state.Database) {
	statedb, _ := state.New(utils.Hash{}, state.NewDatabase(chain.db))
	root := statedb.IntermediateRoot(false)
	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), &types.DposContextProto{})
	if err != nil {
		panic(err)
	}
	dposContext.SetValidators(g.Validators)
	for _, validator := range g.Validators {
		dposContext.DelegateTrie().TryUpdate(append(validator.Bytes(), validator.Bytes()...), validator.Bytes())
		dposContext.CandidateTrie().TryUpdate(validator.Bytes(), validator.Bytes())
	}
	dposContextProto := dposContext.ToProto()
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
		DposContext:  dposContextProto,
	}

	// add dposcontext
	if _, err := dposContext.CommitTo(statedb.Database().TrieDB()); err != nil {
		panic(err)
	}
	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true)

	return types.NewBlock(head, nil, nil), statedb.Database()
}
