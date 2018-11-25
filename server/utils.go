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
	"runtime"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus/pow"
	"github.com/UranusBlockStack/uranus/node"
	"github.com/UranusBlockStack/uranus/params"
	"github.com/UranusBlockStack/uranus/wallet"
)

// CreateDB creates the chain database.
func CreateDB(ctx *node.Context, config *UranusConfig, name string) (db.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DBCache, config.DBHandles)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func checkMinerConfig(cfg *pow.Config, wallet *wallet.Wallet) *pow.Config {
	// extra data
	if uint64(len([]byte(cfg.ExtraData))) > params.MaxExtraDataSize {
		log.Warnf("Miner extra data exceed limit extra: %v, limit:%v", cfg.ExtraData, params.MaxExtraDataSize)
		cfg.ExtraData = ""
	}

	// threads
	if cfg.MinerThreads <= 0 {
		cfg.MinerThreads = 1
	} else if cfg.MinerThreads > runtime.NumCPU() {
		cfg.MinerThreads = runtime.NumCPU()
	}

	// coinbase
	accounts, err := wallet.Accounts()
	if len(accounts) == 0 {
		if err != nil {
			log.Error(err)
		}
		if !utils.IsHexAddr(cfg.CoinBaseAddr) || (utils.HexToAddress(cfg.CoinBaseAddr) == (utils.Address{})) {
			account, err := wallet.NewAccount("coinbase")
			if err != nil {
				log.Warnf("generate conbase account failed: %v", err)
				return cfg
			}
			log.Warnf("CoinBase automatically configured address: %v, passphrase: %v", account.Address, "coinbase")
			cfg.CoinBaseAddr = account.Address.Hex()
		}
		return cfg
	}

	cfg.CoinBaseAddr = accounts[0].Address.Hex()
	log.Infof("Coinbase addr: %v", cfg.CoinBaseAddr)
	return cfg
}
