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

package main

import (
	"time"

	"github.com/UranusBlockStack/uranus/common/fdlimit"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/txpool"
	"github.com/UranusBlockStack/uranus/node"
	"github.com/UranusBlockStack/uranus/p2p"
	"github.com/UranusBlockStack/uranus/server"
)

var startConfig = defaultStartConfig()

func defaultStartConfig() *StartConfig {
	return &StartConfig{
		LogConfig:    log.DefaultConfig,
		NodeConfig:   defaultNodeConfig(),
		UranusConfig: defaultUranusConfig(),
	}
}

func defaultUranusConfig() *server.UranusConfig {
	return &server.UranusConfig{
		Genesis:      ledger.DefaultGenesis(),
		DBHandles:    dbHandles(),
		DBCache:      512,
		TrieCache:    256,
		TrieTimeout:  60 * time.Minute,
		TxPoolConfig: defaultTxPoolConfig(),
	}
}

func defaultNodeConfig() *node.Config {
	return &node.Config{
		Name: Identifier,
		Host: "localhost",
		Port: 8000,
		Cors: []string{},
		P2P:  defaultP2PConfig(),
	}
}

func defaultP2PConfig() *p2p.Config {
	// todo add p2p config-
	return &p2p.Config{
		ListenAddr: ":7090",
		MaxPeers:   25,
	}
}

func defaultTxPoolConfig() *txpool.Config {
	return &txpool.Config{
		PriceLimit:      1,
		PriceBump:       10,
		AccountSlots:    16,
		GlobalSlots:     4096,
		AccountQueue:    64,
		GlobalQueue:     1024,
		TimeoutDuration: 3 * time.Hour,
	}
}

func dbHandles() int {
	limit, err := fdlimit.Current()
	if err != nil {
		log.Errorf("Failed to retrieve file descriptor allowance: %v", err)
	}
	if limit < 2048 {
		if err := fdlimit.Raise(2048); err != nil {
			log.Errorf("Failed to raise file descriptor allowance: %v", err)
		}
	}
	if limit > 2048 {
		limit = 2048
	}
	return limit / 2
}
