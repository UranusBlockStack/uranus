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

package server

import (
	"encoding/json"
	"time"

	"github.com/UranusBlockStack/uranus/consensus/miner"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/txpool"
)

// UranusConfig uranus config
type UranusConfig struct {
	// If nil, the default genesis block is used.
	Genesis *ledger.Genesis

	DBHandles   int
	DBCache     int
	TrieCache   int
	TrieTimeout time.Duration

	StartMiner bool `mapstructure:"miner-start"`

	// Ledger config
	LedgerConfig *ledger.Config

	// Transaction pool options
	TxPoolConfig *txpool.Config

	// miner config
	MinerConfig *miner.Config
}

func (c UranusConfig) String() string {
	cfgJSON, _ := json.Marshal(c)
	return string(cfgJSON)
}
