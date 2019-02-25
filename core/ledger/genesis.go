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
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/math"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
)

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[utils.Address]GenesisAccount

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code    []byte                    `json:"code,omitempty"`
	Storage map[utils.Hash]utils.Hash `json:"storage,omitempty"`
	Balance math.HexOrDecimal256      `json:"balance" gencodec:"required"`
	Nonce   uint64                    `json:"nonce,omitempty"`
}

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
	Alloc        GenesisAlloc        `json:"alloc"`
}

// DefaultGenesis returns the nurans main net genesis block.
func DefaultGenesis() *Genesis {
	gtime, _ := time.Parse("2006-01-02 15:04:05", "2019-01-15 00:00:00")
	extraData, _ := utils.Decode("uranus gensis block")
	return &Genesis{
		Config:     params.DefaultChainConfig,
		Nonce:      1,
		ExtraData:  extraData,
		GasLimit:   params.GenesisGasLimit,
		Timestamp:  uint64(gtime.UnixNano()),
		Difficulty: big.NewInt(0),
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
	if genesis != nil {
		log.Warnf("genesis alreay exist, ingore setup genesis")
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
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, (*big.Int)(&account.Balance))
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}

	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), &types.DposContextProto{})
	if err != nil {
		panic(err)
	}
	validator := utils.HexToAddress(g.Config.GenesisCandidate)
	dposContext.SetValidators([]utils.Address{validator})
	dposContext.DelegateTrie().TryUpdate(append(validator.Bytes(), validator.Bytes()...), validator.Bytes())
	candidateInfo := &types.CandidateInfo{
		Addr:   validator,
		Weight: 100,
	}
	val, _ := rlp.EncodeToBytes(candidateInfo)
	dposContext.CandidateTrie().TryUpdate(validator.Bytes(), val)

	triedb := statedb.Database().TrieDB()
	if _, err := dposContext.CommitTo(triedb); err != nil {
		panic(err)
	}
	root, err := statedb.Commit(false)
	if err != nil {
		panic(err)
	}

	if err := triedb.Commit(root, false); err != nil {
		panic(err)
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
	return types.NewBlock(head, nil, nil, nil), statedb.Database()
}
