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

package miner

import (
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/consensus/dpos"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/feed"
	"github.com/UranusBlockStack/uranus/node/protocols"
	"github.com/UranusBlockStack/uranus/params"
)

type Config struct {
	CoinBaseAddr string `mapstructure:"miner-coinbase"`
	MinerThreads int    `mapstructure:"miner-threads"`
	ExtraData    string `mapstructure:"miner-extradata"`
}

const (
	// hpsUpdateSecs is the number of seconds to wait in between each
	// update to the hashes per second monitor.
	hpsUpdateSecs  = 10
	hashUpdateSecs = 15
)

type Result struct {
	work  *Work
	block *types.Block
}

type UMiner struct {
	mu               sync.Mutex
	wg               sync.WaitGroup
	mining           int32
	canStart         int32
	threads          int32
	stopCh           chan struct{}
	quitCurrentOp    chan struct{}
	speedMonitorQuit chan struct{}

	workCh       chan *Work
	recvCh       chan *Result
	updateHashes chan uint64
	uranus       consensus.IUranus
	db           db.Database

	extraData   []byte
	coinbase    utils.Address
	currentWork *Work
	engine      consensus.Engine
	config      *params.ChainConfig

	mux *feed.TypeMux
}

func NewUranusMiner(mux *feed.TypeMux, config *params.ChainConfig, minerCfg *Config, uranus consensus.IUranus, engine consensus.Engine, db db.Database) *UMiner {
	coinbase := utils.HexToAddress(minerCfg.CoinBaseAddr)
	uminer := &UMiner{
		mux:              mux,
		config:           config,
		uranus:           uranus,
		mining:           0,
		canStart:         1,
		threads:          int32(minerCfg.MinerThreads),
		stopCh:           make(chan struct{}),
		speedMonitorQuit: make(chan struct{}),
		workCh:           make(chan *Work),
		recvCh:           make(chan *Result),
		updateHashes:     make(chan uint64),
		extraData:        []byte(minerCfg.ExtraData),
		coinbase:         coinbase,
		engine:           engine,
		db:               db,
	}
	go uminer.loop()
	return uminer
}

func (m *UMiner) loop() {
	events := m.mux.Subscribe(protocols.StartEvent{}, protocols.DoneEvent{}, protocols.FailedEvent{})
	minning := int32(0)
out:
	for ev := range events.Chan() {
		switch ev.Data.(type) {
		case protocols.StartEvent:
			atomic.StoreInt32(&m.canStart, 0)
			minning = atomic.LoadInt32(&m.mining)
			if minning == 1 {
				log.Warnf("Mining operation maybe aborted due to sync operation")
				m.Stop()
			}
		case protocols.DoneEvent, protocols.FailedEvent:
			atomic.StoreInt32(&m.canStart, 1)
			if minning == 1 {
				log.Warnf("Mining operation maybe start due to sync done or sync failed")
				if err := m.Start(); err != nil {
					log.Errorf("Mining operation start failed --- %v", err)
				}
			}
			events.Unsubscribe()
			break out
		}
	}
}

func (m *UMiner) Start() error {
	if atomic.LoadInt32(&m.canStart) == 0 {
		log.Info("Can not start miner when syncing")
		return fmt.Errorf("node is syncing now")
	}
	if atomic.LoadInt32(&m.mining) == 1 {
		log.Info("Miner is running")
		return fmt.Errorf("miner is running")
	}
	m.stopCh = make(chan struct{})
	m.speedMonitorQuit = make(chan struct{})
	m.workCh = make(chan *Work)
	m.recvCh = make(chan *Result)

	// CAS to ensure only 1 mining goroutine.
	if !atomic.CompareAndSwapInt32(&m.mining, 0, 1) {
		log.Warn("Another goroutine has already started to mine")
		return nil
	}

	m.wg.Add(4)
	go m.Wait()
	go m.Update()
	go m.SpeedMonitor()
	go m.mintLoop()

	// if err := m.prepareNewBlock(); err != nil { // try to prepare the first block
	// 	log.Warnf("mining prepareNewBlock err: %v", err)
	// 	atomic.StoreInt32(&m.mining, 0)
	// 	return err
	// }

	log.Info("Miner is started.")
	return nil
}

func (m *UMiner) Stop() {
	if !atomic.CompareAndSwapInt32(&m.mining, 1, 0) {
		return
	}
	// notify all threads to terminate
	if m.stopCh != nil {
		close(m.stopCh)
	}

	// wait for all threads to terminate
	close(m.speedMonitorQuit)
	close(m.recvCh)
	close(m.workCh)
	m.recvCh = nil
	m.workCh = nil

	m.wg.Wait()
	log.Info("Miner is stopped.")
}

func (m *UMiner) Wait() {
	defer m.wg.Done()
out:
	for {
		select {
		case result, ok := <-m.recvCh:
			if !ok || result == nil {
				continue
			}
			_, err := m.uranus.WriteBlockWithState(result.block, result.work.receipts, result.work.state)
			if err != nil {
				log.Errorf("failed to write the block and state, for %s", err.Error())
				break
			}

			m.uranus.PostEvent(feed.BlockAndLogsEvent{Block: result.block})
			m.mux.Post(feed.NewMinedBlockEvent{
				Block: result.block,
			})
		case <-m.stopCh:
			break out
		}
	}
	log.Debug("miner wait block thread quit ...")
}

func (m *UMiner) Update() {
	defer m.wg.Done()
out:
	for {
		select {
		case work, ok := <-m.workCh:
			if !ok && work == nil {
				break out
			}
			m.mu.Lock()
			if m.quitCurrentOp != nil {
				close(m.quitCurrentOp)
			}
			m.quitCurrentOp = make(chan struct{})
			go m.GenerateBlocks(work, m.quitCurrentOp)
			m.mu.Unlock()
		case <-m.stopCh:
			break out
		}
	}
	log.Debug("miner update to generate block thread quit ...")
}

func (m *UMiner) GenerateBlocks(work *Work, quit <-chan struct{}) {
	header := m.currentWork.Block.BlockHeader()
	header.GasUsed = *m.currentWork.gasUsed

	block, err := m.engine.Finalize(m.uranus, header, work.state, work.txs, work.receipts, work.dposContext)
	block.DposContext = work.dposContext
	if err != nil {
		log.Warnf("Block sealing failed: %v", err)
		m.recvCh <- nil
	} else {
		work.Block = block
		if result, err := m.engine.Seal(m.uranus, work.Block, quit, int(m.threads), m.updateHashes); result != nil {
			log.Infof("Successfully sealed new block number: %v, hash: %v, diff: %v", result.Height(), result.Hash(), result.Difficulty())
			m.recvCh <- &Result{work, result}
		} else {
			if err != nil {
				log.Warnf("Block sealing failed: %v", err)
			}
			m.recvCh <- nil
		}
	}
}

func (m *UMiner) prepareNewBlock() error {
	timestamp := time.Now().Unix()
	parent, stateDB, err := m.uranus.GetCurrentInfo()
	if err != nil {
		return fmt.Errorf("failed to get current info, %s", err)
	}

	if parent.BlockHeader().TimeStamp.Cmp(new(big.Int).SetInt64(timestamp)) >= 0 {
		timestamp = parent.BlockHeader().TimeStamp.Int64() + 1
	}
	// this will ensure we're not going off too far in the future
	if now := time.Now().Unix(); timestamp > now+1 {
		wait := time.Duration(timestamp-now) * time.Second
		log.Infof("Mining too far in the future, waiting for %s", wait)
		time.Sleep(wait)
	}

	height := parent.BlockHeader().Height
	difficult := m.engine.CalcDifficulty(m.uranus.Config(), uint64(timestamp), parent.BlockHeader())
	log.Debugf("block_height: %+v, difficult: %+v, hash: %v", parent.Height().Uint64(), difficult.Uint64(), parent.Hash())
	header := &types.BlockHeader{
		PreviousHash: parent.Hash(),
		Miner:        m.coinbase,
		Height:       height.Add(height, big.NewInt(1)),
		TimeStamp:    big.NewInt(timestamp),
		GasLimit:     calcGasLimit(parent),
		Difficulty:   difficult,
		ExtraData:    m.extraData,
	}
	var dposContext *types.DposContext = nil
	if _, ok := m.engine.(*dpos.Dpos); ok {
		var err error
		dposContext, err = types.NewDposContextFromProto(stateDB.Database().TrieDB(), parent.BlockHeader().DposContext)
		if err != nil {
			return err
		}
	}

	log.Debugf("miner a block with coinbase %v", m.coinbase)
	m.currentWork = NewWork(types.NewBlockWithBlockHeader(header), parent.Height().Uint64(), stateDB, dposContext)

	pending, err := m.uranus.Pending()
	if err != nil {
		log.Errorf("Failed to fetch pending transactions: %v", err)
		return fmt.Errorf("Failed to fetch pending transactions, err: %s", err.Error())
	}

	txs := types.NewTransactionsByPriceAndNonce(m.currentWork.signer, pending)
	err = m.currentWork.applyTransactions(m.uranus, txs)
	if err != nil {
		return fmt.Errorf("failed to apply transaction %s", err)
	}

	log.Infof("committing a new task to engine, height: %v, difficult: %v", header.Height, header.Difficulty)
	m.PushWork(m.currentWork)
	return nil
}

func (m *UMiner) PushWork(work *Work) {
	if m.workCh != nil {
		m.workCh <- work
	}
}

func (m *UMiner) SetCoinBase(addr utils.Address) {
	m.mu.Lock()
	m.coinbase = addr
	m.mu.Unlock()
	//m.prepareNewBlock()
}

func (m *UMiner) GetCoinBase() utils.Address {
	return m.coinbase
}

func (m *UMiner) SpeedMonitor() {
	var hashesPerSec float64
	var totalHashes uint64
	ticker := time.NewTicker(time.Second * hpsUpdateSecs)
	defer ticker.Stop()
	defer m.wg.Done()

out:
	for {
		select {
		// Periodic updates from the workers with how many hashes they
		// have performed.
		case numHashes := <-m.updateHashes:
			totalHashes += numHashes

			// Time to update the hashes per second.
		case <-ticker.C:
			curHashesPerSec := float64(totalHashes) / hpsUpdateSecs
			if hashesPerSec == 0 {
				hashesPerSec = curHashesPerSec
			}
			hashesPerSec = (hashesPerSec + curHashesPerSec) / 2
			totalHashes = 0
			if hashesPerSec != 0 {
				log.Debugf("Hash speed: %6.0f kilohashes/s",
					hashesPerSec/1000)
			}

		case <-m.speedMonitorQuit:
			break out
		}
	}

	log.Debug("CPU miner speed monitor quit")
}

func (m *UMiner) SetThreads(cnt int32) {
	m.mu.Lock()
	m.threads = cnt
	m.mu.Unlock()

	//m.prepareNewBlock()
}

func (m *UMiner) PendingBlock() *types.Block {
	if m.currentWork == nil {
		return nil
	}
	return m.currentWork.Block
}

func calcGasLimit(parent *types.Block) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor
	// decay = parentGasLimit / 1024 -1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// however, if we're now below the target (TargetGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if limit < params.GenesisGasLimit {
		limit = parent.GasLimit() + decay
		if limit > params.GenesisGasLimit {
			limit = params.GenesisGasLimit
		}
	}
	return limit
}

func (m *UMiner) mintLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(time.Second)
	sub := m.mux.Subscribe(feed.NewMinedBlockEvent{})
	if _, ok := m.engine.(*dpos.Dpos); !ok {
		ticker.Stop()
		defer sub.Unsubscribe()
	} else {
		defer ticker.Stop()
		sub.Unsubscribe()
	}

	for {
		select {
		case now := <-ticker.C:
			if err := m.engine.(*dpos.Dpos).CheckValidator(m.uranus.CurrentBlock(), m.coinbase, now.Unix()); err != nil {
				switch err {
				case dpos.ErrWaitForPrevBlock,
					dpos.ErrMintFutureBlock,
					dpos.ErrInvalidMintBlockTime:
					log.Debugf("Failed to mint the block, while %v", err)
				case dpos.ErrInvalidBlockValidator:
					log.Warnf("Failed to mint the block, while %v", err)
				default:
					log.Errorf("Failed to mint the block, err %v", err)
				}
				continue
			}
			if err := m.prepareNewBlock(); err != nil {
				log.Warnf("prepareNewBlock err: %v", err)
			}
		case <-sub.Chan():
			if err := m.prepareNewBlock(); err != nil {
				log.Warnf("prepareNewBlock err: %v", err)
			}
		case <-m.stopCh:
			return

		}
	}
}
