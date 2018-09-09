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
