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
	"errors"
	"math/big"

	"github.com/UranusBlockStack/uranus/common/math"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/params"
)

func (g Genesis) MarshalJSON() ([]byte, error) {
	type genesisJ struct {
		Config       *params.ChainConfig   `json:"config"`
		Nonce        math.HexOrDecimal64   `json:"nonce"`
		Timestamp    math.HexOrDecimal64   `json:"timestamp"`
		ExtraData    utils.Bytes           `json:"extraData"`
		GasLimit     math.HexOrDecimal64   `json:"gasLimit"`
		Difficulty   *math.HexOrDecimal256 `json:"difficulty"`
		Mixhash      utils.Hash            `json:"mixHash"`
		Miner        utils.Address         `json:"miner"`
		Height       math.HexOrDecimal64   `json:"height"`
		GasUsed      math.HexOrDecimal64   `json:"gasUsed"`
		PreviousHash utils.Hash            `json:"previousHash"`
		Validator    utils.Address         `json:"validator"`
		Alloc        GenesisAlloc          `json:"alloc"`
	}

	var enc genesisJ
	enc.Config = g.Config
	enc.Nonce = math.HexOrDecimal64(g.Nonce)
	enc.Timestamp = math.HexOrDecimal64(g.Timestamp)
	enc.ExtraData = g.ExtraData
	enc.GasLimit = math.HexOrDecimal64(g.GasLimit)
	enc.Difficulty = (*math.HexOrDecimal256)(g.Difficulty)
	enc.Mixhash = g.Mixhash
	enc.Miner = g.Miner
	enc.Height = math.HexOrDecimal64(g.Height)
	enc.GasUsed = math.HexOrDecimal64(g.GasUsed)
	enc.PreviousHash = g.PreviousHash
	if g.Alloc != nil {
		enc.Alloc = make(map[utils.Address]GenesisAccount, len(g.Alloc))
		for k, v := range g.Alloc {
			enc.Alloc[k] = v
		}
	}
	return json.Marshal(&enc)
}

func (g *Genesis) UnmarshalJSON(input []byte) error {
	type genesisJ struct {
		Config       *params.ChainConfig   `json:"config"`
		Nonce        *math.HexOrDecimal64  `json:"nonce"`
		Timestamp    *math.HexOrDecimal64  `json:"timestamp"`
		ExtraData    utils.Bytes           `json:"extraData"`
		GasLimit     *math.HexOrDecimal64  `json:"gasLimit"`
		Difficulty   *math.HexOrDecimal256 `json:"difficulty"`
		Mixhash      *utils.Hash           `json:"mixHash"`
		Miner        *utils.Address        `json:"miner"`
		Height       *math.HexOrDecimal64  `json:"height"`
		GasUsed      *math.HexOrDecimal64  `json:"gasUsed"`
		PreviousHash *utils.Hash           `json:"previousHash"`
		Alloc        GenesisAlloc          `json:"alloc"`
	}
	var dec genesisJ
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Config != nil {
		g.Config = dec.Config
	}
	if dec.Nonce != nil {
		g.Nonce = uint64(*dec.Nonce)
	}
	if dec.Timestamp != nil {
		g.Timestamp = uint64(*dec.Timestamp)
	}
	if dec.ExtraData != nil {
		g.ExtraData = dec.ExtraData
	}
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Genesis")
	}
	g.GasLimit = uint64(*dec.GasLimit)
	if dec.Difficulty == nil {
		return errors.New("missing required field 'difficulty' for Genesis")
	}
	g.Difficulty = (*big.Int)(dec.Difficulty)
	if dec.Mixhash != nil {
		g.Mixhash = *dec.Mixhash
	}
	if dec.Miner != nil {
		g.Miner = *dec.Miner
	}

	if dec.Height != nil {
		g.Height = uint64(*dec.Height)
	}
	if dec.GasUsed != nil {
		g.GasUsed = uint64(*dec.GasUsed)
	}
	if dec.PreviousHash != nil {
		g.PreviousHash = *dec.PreviousHash
	}
	if dec.Alloc != nil {
		g.Alloc = make(GenesisAlloc, len(dec.Alloc))
		for k, v := range dec.Alloc {
			g.Alloc[utils.Address(k)] = v
		}
	}
	return nil
}
