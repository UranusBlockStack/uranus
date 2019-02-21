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
	"sync"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/consensus"
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
	"github.com/UranusBlockStack/uranus/server/forecast"
	"github.com/UranusBlockStack/uranus/wallet"
)

// Uranus implements the service.
type Uranus struct {
	config      *UranusConfig
	chainConfig *params.ChainConfig

	miner      *miner.UMiner
	engine     consensus.Engine
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
	chainCfg, statedb, _, err := ledger.SetupGenesis(config.Genesis, ledger.NewChain(chainDb))
	if err != nil {
		return nil, err
	}

	cjson, _ := json.Marshal(chainCfg)
	log.Infof("chain config %v", string(cjson))

	mux := &feed.TypeMux{}
	uranus := &Uranus{
		config:       config,
		chainDb:      chainDb,
		chainConfig:  chainCfg,
		shutdownChan: make(chan bool),
	}

	uranus.wallet = wallet.NewWallet(ctx.ResolvePath("keystore"))

	// engine
	cpu := cpuminer.NewCpuMiner()
	_ = cpu
	dpos.Option.BlockInterval = chainCfg.BlockInterval
	dpos.Option.BlockRepeat = chainCfg.BlockRepeat
	dpos.Option.MaxValidatorSize = chainCfg.MaxValidatorSize
	dpos.Option.MinStartQuantity = chainCfg.MinStartQuantity
	if chainCfg.DelayEpcho > 0 {
		dpos.Option.DelayEpcho = chainCfg.DelayEpcho
	}

	if chainCfg.MaxConfirmedNum > 0 {
		dpos.Option.MaxConfirmedNum = chainCfg.MaxConfirmedNum
	}

	dpos := dpos.NewDpos(mux, chainDb, statedb, uranus.wallet.SignHash, "coinbase")

	// blockchain
	log.Debugf("Initialised chain configuration: %v", chainCfg)
	uranus.blockchain, err = core.NewBlockChain(config.LedgerConfig, uranus.chainConfig, statedb, chainDb, dpos, &vm.Config{})
	if err != nil {
		return nil, err
	}
	// txpool
	uranus.txPool = txpool.New(config.TxPoolConfig, uranus.chainConfig, uranus.blockchain)

	uranus.blockchain.SetAddActionInterface(uranus.txPool)

	dpos.Init(uranus.blockchain)
	// miner
	uranus.miner = miner.NewUranusMiner(mux, uranus.chainConfig, checkMinerConfig(uranus.config.MinerConfig, uranus.wallet), &MinerBakend{u: uranus}, dpos, uranus.chainDb)
	uranus.engine = dpos
	//dpos.MintLoop(uranus.miner, uranus.blockchain)

	// api
	uranus.uranusAPI = &APIBackend{u: uranus}
	uranus.uranusAPI.gp = forecast.NewForecast(uranus.uranusAPI.BlockByHeight, forecast.DefaultConfig)

	uranus.protocolManager, _ = node.NewProtocolManager(mux, uranus.chainConfig, uranus.txPool, uranus.blockchain, uranus.chainDb, uranus.engine)

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
			Namespace: "Dpos",
			Version:   "0.0.1",
			Service:   rpcapi.NewDposAPI(u.uranusAPI),
		},
	}
}

// Start implements node.Service, starting all internal goroutines.
func (u *Uranus) Start(p2p *p2p.Server) error {
	log.Info("start uranus service...")
	// start p2p
	u.protocolManager.Start(p2p.MaxPeers)
	// start miner
	if u.config.StartMiner {
		u.miner.Start()
	}
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
