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

package params

import (
	"encoding/json"
	"math/big"
)

type ChainConfig struct {
	ChainID *big.Int
}

// String implements fmt.Stringer.
func (c ChainConfig) String() string {
	cfgJSON, _ := json.Marshal(c)
	return string(cfgJSON)
}

var TestChainConfig = &ChainConfig{ChainID: big.NewInt(0)}
var DefaultChainConfig = &ChainConfig{ChainID: big.NewInt(1)}

// Rules evm rules
type Rules struct {
	ChainId                                   *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158 bool
	IsByzantium                               bool
}

func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainId := c.ChainID
	if chainId == nil {
		chainId = new(big.Int)
	}
	return Rules{ChainId: new(big.Int).Set(chainId),
		IsHomestead: IsHomestead(num),
		IsEIP150:    IsEIP150(num),
		IsEIP155:    IsEIP155(num),
		IsEIP158:    IsEIP158(num),
		IsByzantium: IsByzantium(num)}
}

func IsHomestead(num *big.Int) bool {
	return isForked(big.NewInt(1150000), num)
}

func IsEIP150(num *big.Int) bool {
	return isForked(big.NewInt(2463000), num)
}

func IsEIP155(num *big.Int) bool {
	return isForked(big.NewInt(2675000), num)
}

func IsEIP158(num *big.Int) bool {
	return isForked(big.NewInt(2675000), num)
}

func IsByzantium(num *big.Int) bool {
	return isForked(big.NewInt(4370000), num)
}

func IsConstantinople(num *big.Int) bool {
	return isForked(nil, num)
}

func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func MainGasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case IsEIP158(num):
		return GasTableEIP158
	case IsEIP150(num):
		return GasTableEIP150
	default:
		return GasTableHomestead
	}
}
