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
	"time"
)

type ChainConfig struct {
	ChainID             *big.Int `json:"chainId"`
	MinDelegateState    *big.Int `json:"minDelegateState"`
	MinDelegatePercent  *big.Int `json:"minDelegatePercent"`
	MinDelegateDuration int64    `json:"minDelegateDuration"`
	BlockInterval       int64    `json:"blockInterval"`
	BlockRepeat         int64    `json:"blockRepeat"`
	MaxValidatorSize    int64    `json:"epchoValidators"`
	DelayEpcho          int64    `json:"delayepcho"`
	MaxConfirmedNum     int64    `json:"maxConfirmedNum"`
	GenesisCandidate    string   `json:"candiate"`
	MinStartQuantity    *big.Int `json:"startQuantity"`
	MaxVotes            int64    `json:"votes"`
	DelayDuration       int64    `json:"refund"`
}

// String implements fmt.Stringer.
func (c ChainConfig) String() string {
	cfgJSON, _ := json.Marshal(c)
	return string(cfgJSON)
}

var TestChainConfig = &ChainConfig{
	ChainID:             big.NewInt(0),
	MinDelegateState:    new(big.Int).Mul(big.NewInt(1000), big.NewInt(18)),
	MinDelegatePercent:  big.NewInt(10),
	MinDelegateDuration: int64(time.Hour / time.Second),
	GenesisCandidate:    "0x970e8128ab834e8eac17ab8e3812f010678cf791",
	MinStartQuantity:    big.NewInt(100),
	MaxVotes:            30,
	DelayDuration:       72 * 3600,
	BlockInterval:       int64(3000 * time.Millisecond),
	BlockRepeat:         12,
	MaxValidatorSize:    3,
}
var DefaultChainConfig = &ChainConfig{
	ChainID:             big.NewInt(1),
	MinDelegateState:    new(big.Int).Mul(big.NewInt(1000), big.NewInt(18)),
	MinDelegatePercent:  big.NewInt(10),
	MinDelegateDuration: int64(time.Hour / time.Second),
	GenesisCandidate:    "0x970e8128ab834e8eac17ab8e3812f010678cf791",
	MinStartQuantity:    big.NewInt(100),
	MaxVotes:            30,
	DelayDuration:       72 * 3600,
	BlockInterval:       int64(500 * time.Millisecond),
	BlockRepeat:         12,
	MaxValidatorSize:    3,
}
