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
	"sync"

	"github.com/UranusBlockStack/uranus/server/forecast"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/consensus/dpos"
	"github.com/UranusBlockStack/uranus/consensus/miner"
	"github.com/UranusBlockStack/uranus/consensus/pow/cpuminer"
	"github.com/UranusBlockStack/uranus/core"
	"github.com/UranusBlockStack/uranus/core/ledger"
	"github.com/UranusBlockStack/uranus/core/txpool"
	"github.com/UranusBlockStack/uranus/core/vm"
	"github.com/UranusBlockStack/uranus/feed"
	"github.com/UranusBlockStack/uranus/node"
	"github.com/UranusBlockStack/uranus/p2p"
	"github.com/UranusBlockStack/uranus/params"
	"github.com/UranusBlockStack/uranus/rpc"
	"github.com/UranusBlockStack/uranus/rpcapi"
	"github.com/UranusBlockStack/uranus/wallet"
)

// Uranus implements the service.
type Uranus struct {
	config      *UranusConfig
	chainConfig *params.ChainConfig

	miner      *miner.UMiner
	blockchain *core.BlockChain
	txPool     *txpool.TxPool
	chainDb    db.Database // Block chain database
	wallet     *wallet.Wallet

	protocolManager *node.ProtocolManager

	uranusAPI *APIBackend

	shutdownChan chan bool // Channel for shutting down
	lock         sync.RWMutex
}

// New creates a new Uranus object
func New(ctx *node.Context, config *UranusConfig) (*Uranus, error) {
	log.Debugf("load uranus config: %s", config)
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}

	// Setup genesis block
	chainCfg, _, err := ledger.SetupGenesis(config.Genesis, ledger.NewChain(chainDb))
	if err != nil {
		return nil, err
	}

	uranus := &Uranus{
		config:       config,
		chainDb:      chainDb,
		chainConfig:  chainCfg,
		shutdownChan: make(chan bool),
	}

	uranus.wallet = wallet.NewWallet(ctx.ResolvePath("keystore"))

	// blockchain
	log.Debugf("Initialised chain configuration: %v", chainCfg)
	uranus.blockchain, err = core.NewBlockChain(config.LedgerConfig, uranus.chainConfig, chainDb, nil, &vm.Config{})
	if err != nil {
		return nil, err
	}
	// txpool
	uranus.txPool = txpool.New(config.TxPoolConfig, uranus.chainConfig, uranus.blockchain)

	mux := &feed.TypeMux{}

	engine := cpuminer.NewCpuMiner()
	dpos := dpos.NewDpos(uranus.wallet.SignHash)
	// miner
	uranus.miner = miner.NewUranusMiner(mux, uranus.chainConfig, checkMinerConfig(uranus.config.MinerConfig, uranus.wallet), &MinerBakend{u: uranus}, dpos)
	_ = dpos
	//dpos.MintLoop(uranus.miner, uranus.blockchain)

	// api
	uranus.uranusAPI = &APIBackend{u: uranus}
	uranus.uranusAPI.gp = forecast.NewForecast(uranus.uranusAPI.BlockByHeight, forecast.DefaultConfig)

	uranus.protocolManager, _ = node.NewProtocolManager(mux, uranus.chainConfig, uranus.txPool, uranus.blockchain, uranus.chainDb, engine)

	return uranus, nil
}

// Protocols implements node.Service.
func (u *Uranus) Protocols() []*p2p.Protocol {
	return u.protocolManager.SubProtocols
}

// APIs return the collection of RPC services the Uranus package offers.
func (u *Uranus) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "Admin",
			Version:   "0.0.1",
			Service:   rpcapi.NewAdminAPI(u.uranusAPI),
		},
		{
			Namespace: "Miner",
			Version:   "0.0.1",
			Service:   rpcapi.NewMinerAPI(u.uranusAPI),
		},
		{
			Namespace: "Uranus",
			Version:   "0.0.1",
			Service:   rpcapi.NewUranusAPI(u.uranusAPI),
		},
		{
			Namespace: "Wallet",
			Version:   "0.0.1",
			Service:   rpcapi.NewWalletAPI(u.uranusAPI),
		},
		{
			Namespace: "TxPool",
			Version:   "0.0.1",
			Service:   rpcapi.NewTransactionPoolAPI(u.uranusAPI),
		},
		{
			Namespace: "BlockChain",
			Version:   "0.0.1",
			Service:   rpcapi.NewBlockChainAPI(u.uranusAPI),
		},
		{
			Namespace: "dpos",
			Version:   "0.0.1",
			Service:   rpcapi.NewDposAPI(u.uranusAPI),
		},
	}
}

// Start implements node.Service, starting all internal goroutines.
func (u *Uranus) Start(p2p *p2p.Server) error {
	log.Info("start uranus service...")
	// start miner
	if u.config.StartMiner {
		u.miner.Start()
	}
	// start p2p
	u.protocolManager.Start(p2p.MaxPeers)
	u.uranusAPI.srv = p2p
	return nil
}

// Stop implements node.Service, terminating all internal goroutine
func (u *Uranus) Stop() error {
	u.miner.Stop()
	u.txPool.Stop()
	u.chainDb.Close()
	u.protocolManager.Stop()
	close(u.shutdownChan)
	return nil
}

// BlockChain returns blcokchain.
func (u *Uranus) BlockChain() *core.BlockChain { return u.blockchain }
