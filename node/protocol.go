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

package node

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core"
	"github.com/UranusBlockStack/uranus/core/txpool"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/feed"
	"github.com/UranusBlockStack/uranus/node/protocols"
	"github.com/UranusBlockStack/uranus/p2p"
	"github.com/UranusBlockStack/uranus/p2p/discover"
	"github.com/UranusBlockStack/uranus/params"
)

var baseProtocolName = "uransus"

var baseProtocolVersion uint = 1

var maxMsgSize = 10 * 1024 * 1024

const (
	StatusMsg                   = iota + 1000 //1000
	NewBlockHashesMsg                         //1001
	TxMsg                                     //1002
	GetBlockHashesMsg                         //1003
	BlockHashesMsg                            //1004
	GetBlocksMsg                              //1005
	BlocksMsg                                 //1006
	NewBlockMsg                               //1007
	GetBlockHashesFromNumberMsg               //1008
	ConfirmedMsg                              //1009
)

type statusData struct {
	ProtocolVersion uint32
	NetworkID       uint64
	TD              *big.Int
	CurrentBlock    utils.Hash
	GenesisBlock    utils.Hash
}

type newBlockData struct {
	Block *types.Block
	TD    *big.Int
}

type getBlockHashesData struct {
	Hash   utils.Hash
	Amount uint64
}

type getBlockHashesFromNumberData struct {
	Number uint64
	Amount uint64
}

type ProtocolManager struct {
	networkId   uint64
	txpool      *txpool.TxPool
	blockchain  *core.BlockChain
	chainconfig *params.ChainConfig
	maxPeers    int

	txsCh         chan feed.NewTxsEvent
	txsSub        feed.Subscription
	minedBlockSub *feed.TypeMuxSubscription
	downloader    *protocols.Downloader
	fetcher       *protocols.Fetcher
	peers         *peerSet
	SubProtocols  []*p2p.Protocol
	newPeerCh     chan *peer
	txsyncCh      chan *txsync
	quitSync      chan struct{}
	noMorePeers   chan struct{}
	wg            sync.WaitGroup
	eventMux      *feed.TypeMux

	acceptTxs uint32
}

func NewProtocolManager(mux *feed.TypeMux, config *params.ChainConfig, txpool *txpool.TxPool, blockchain *core.BlockChain, chaindb db.Database, engine consensus.Engine) (*ProtocolManager, error) {
	manager := &ProtocolManager{
		eventMux:    mux,
		txpool:      txpool,
		blockchain:  blockchain,
		chainconfig: config,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		noMorePeers: make(chan struct{}),
		txsyncCh:    make(chan *txsync),
		quitSync:    make(chan struct{}),
		acceptTxs:   1,
	}

	manager.SubProtocols = make([]*p2p.Protocol, 0)
	manager.SubProtocols = append(manager.SubProtocols, &p2p.Protocol{
		Name:    baseProtocolName,
		Version: baseProtocolVersion,
		Offset:  1000,
		Size:    1000,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			peer := manager.newPeer(int(baseProtocolVersion), p, rw)
			select {
			case manager.newPeerCh <- peer:
				manager.wg.Add(1)
				defer manager.wg.Done()
				return manager.handle(peer)
			case <-manager.quitSync:
				return fmt.Errorf("quit")
			}
		},
		NodeInfo: func() interface{} {
			return manager.NodeInfo()
		},
		PeerInfo: func(id discover.NodeID) interface{} {
			if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
				return p.Info()
			}
			return nil
		},
	})

	validator := func(header *types.BlockHeader) error {
		return engine.VerifySeal(blockchain, header)
	}

	heighter := func() uint64 {
		return blockchain.CurrentBlock().Height().Uint64()
	}
	inserter := func(blocks types.Blocks) (int, error) {
		atomic.StoreUint32(&manager.acceptTxs, 1)
		return manager.blockchain.InsertChain(blocks)
	}

	manager.downloader = protocols.NewDownloader(manager.eventMux, manager.blockchain.HasBlock, manager.blockchain.GetBlockByHash, manager.blockchain.CurrentBlock, manager.blockchain.GetTd, inserter, manager.removePeer)
	manager.fetcher = protocols.NewFetcher(manager.blockchain.GetBlockByHash, validator, manager.BroadcastBlock, heighter, inserter, manager.removePeer)
	return manager, nil
}

func (pm *ProtocolManager) removePeer(id string) {
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debugf("Removing uranus peer %v", id)
	pm.downloader.UnregisterPeer(id)
	if err := pm.peers.Unregister(id); err != nil {
		log.Errorf("Peer removal %v failed --- %v", id, err)
	}
	if peer != nil {
		peer.Peer.Disconnect("remove")
	}
}

func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers

	pm.txsCh = make(chan feed.NewTxsEvent, txChanSize)
	pm.txsSub = pm.txpool.SubscribeNewTxsEvent(pm.txsCh)
	go pm.txBroadcastLoop()

	pm.minedBlockSub = pm.eventMux.Subscribe(feed.NewMiner{}, feed.NewMinedBlockEvent{}, feed.NewConfirmedEvent{})
	go pm.minedBroadcastLoop()

	go pm.syncer()
	go pm.txsyncLoop()
}

func (pm *ProtocolManager) Stop() {
	log.Info("Stopping uranus protocol")

	pm.txsSub.Unsubscribe()
	pm.minedBlockSub.Unsubscribe()

	pm.noMorePeers <- struct{}{}

	close(pm.quitSync)

	pm.peers.Close()

	pm.wg.Wait()

	log.Info("uranus protocol stopped")
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, rw)
}

func (pm *ProtocolManager) handle(p *peer) error {
	if pm.peers.Len() >= pm.maxPeers {
		return fmt.Errorf("too many peer")
	}
	log.Debugf("uranus peer connected name %v", p.Name())

	var (
		genesis = pm.blockchain.GetBlockByHeight(0)
		head    = pm.blockchain.CurrentBlock().BlockHeader()
		hash    = head.Hash()
		td      = pm.blockchain.GetTd(hash)
	)
	if err := p.Handshake(pm.networkId, td, hash, genesis.Hash()); err != nil {
		log.Errorf("uranus handshake failed --- %v", err)
		return err
	}

	if err := pm.peers.Register(p); err != nil {
		log.Errorf("uranus peer registration failed --- %v", err)
		return err
	}
	defer pm.removePeer(p.id)

	if err := pm.downloader.RegisterPeer(p.id, p.version, p.head, p.RequestHashes, p.RequestHashesFromNumber, p.RequestBlocks); err != nil {
		return err
	}
	pm.syncTransactions(p)

	for {
		if err := pm.handleMsg(p); err != nil {
			log.Warnf("uranus message handling failed --- %v", err)
			return err
		}
	}
}

func (pm *ProtocolManager) handleMsg(p *peer) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}

	switch msg.Code {
	case StatusMsg:
		return fmt.Errorf("uncontrolled status message")

	case GetBlockHashesMsg:
		var request getBlockHashesData
		if err := msg.DecodePayload(&request); err != nil {
			return fmt.Errorf("%v: %v", msg, err)
		}
		if request.Amount > uint64(protocols.MaxHashFetch) {
			request.Amount = uint64(protocols.MaxHashFetch)
		}

		getBlockHashesFromHash := func(hash utils.Hash, max uint64) (hashes []utils.Hash) {
			block := pm.blockchain.GetBlockByHash(hash)
			if block == nil {
				return
			}
			for i := uint64(0); i < max; i++ {
				block = pm.blockchain.GetBlockByHash(block.PreviousHash())
				if block == nil {
					break
				}
				hashes = append(hashes, block.Hash())
				if block.Height().Cmp(big.NewInt(0)) <= 0 {
					break
				}
			}
			return
		}

		hashes := getBlockHashesFromHash(request.Hash, request.Amount)
		if len(hashes) == 0 {
			log.Infof("invalid block hash %x", request.Hash)
		}
		return p.SendBlockHashes(hashes)
	case GetBlockHashesFromNumberMsg:
		var request getBlockHashesFromNumberData
		if err := msg.DecodePayload(&request); err != nil {
			return fmt.Errorf("%v: %v", msg, err)
		}
		if request.Amount > uint64(protocols.MaxHashFetch) {
			request.Amount = uint64(protocols.MaxHashFetch)
		}
		last := pm.blockchain.GetBlockByHeight(request.Number + request.Amount - 1)
		if last == nil {
			last = pm.blockchain.CurrentBlock()
			request.Amount = last.Height().Uint64() - request.Number + 1
		}
		if last.Height().Uint64() < request.Number {
			return p.SendBlockHashes(nil)
		}

		block := last
		hashes := []utils.Hash{last.Hash()}
		for i := uint64(0); i < request.Amount-1; i++ {
			block = pm.blockchain.GetBlockByHash(block.PreviousHash())
			if block == nil {
				break
			}

			hashes = append(hashes, block.Hash())

			if block.Height().Cmp(big.NewInt(0)) <= 0 {
				break
			}
		}
		for i := 0; i < len(hashes)/2; i++ {
			hashes[i], hashes[len(hashes)-1-i] = hashes[len(hashes)-1-i], hashes[i]
		}

		return p.SendBlockHashes(hashes)
	case BlockHashesMsg:
		buf := bytes.NewBuffer(msg.Payload)
		msgStream := rlp.NewStream(buf, uint64(buf.Len()))

		var hashes []utils.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			break
		}

		err := pm.downloader.DeliverHashes(p.id, hashes)
		if err != nil {
			log.Error(err)
		}

	case GetBlocksMsg:
		buf := bytes.NewBuffer(msg.Payload)
		msgStream := rlp.NewStream(buf, uint64(buf.Len()))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		var (
			hash   utils.Hash
			bytes  utils.StorageSize
			hashes []utils.Hash
			blocks []*types.Block
		)
		for {
			err := msgStream.Decode(&hash)
			if err == rlp.EOL {
				break
			} else if err != nil {
				return fmt.Errorf("msg %v: %v", msg, err)
			}
			hashes = append(hashes, hash)

			if block := pm.blockchain.GetBlockByHash(hash); block != nil {
				blocks = append(blocks, block)
				bytes += block.Size()
				if len(blocks) >= protocols.MaxBlockFetch {
					break
				}
			}
		}

		return p.SendBlocks(blocks)

	case BlocksMsg:
		buf := bytes.NewBuffer(msg.Payload)
		msgStream := rlp.NewStream(buf, uint64(buf.Len()))

		var blocks []*types.Block
		if err := msgStream.Decode(&blocks); err != nil {
			blocks = nil
		}
		for _, block := range blocks {
			block.ReceivedAt = msg.ReceivedAt
		}
		if blocks := pm.fetcher.Filter(blocks); len(blocks) > 0 {
			pm.downloader.DeliverBlocks(p.id, blocks)
		}

	case NewBlockHashesMsg:
		buf := bytes.NewBuffer(msg.Payload)
		msgStream := rlp.NewStream(buf, uint64(buf.Len()))

		var hashes []utils.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			break
		}

		for _, hash := range hashes {
			p.MarkBlock(hash)
		}
		unknown := make([]utils.Hash, 0, len(hashes))
		for _, hash := range hashes {
			if !pm.blockchain.HasBlock(hash) {
				unknown = append(unknown, hash)
			}
		}
		for _, hash := range unknown {
			pm.fetcher.Notify(p.id, hash, time.Now(), p.RequestBlocks)
		}

	case NewBlockMsg:
		var request newBlockData
		if err := msg.DecodePayload(&request); err != nil {
			return fmt.Errorf("%v: %v", msg, err)
		}

		request.Block.ReceivedAt = msg.ReceivedAt

		p.MarkBlock(request.Block.Hash())
		p.head = request.Block.Hash()

		pm.fetcher.Enqueue(p.id, request.Block)

		// Assuming the block is importable by the peer, but possibly not yet done so,
		// calculate the head hash and TD that the peer truly must have.
		var (
			trueHead = request.Block.PreviousHash()
			trueTD   = new(big.Int).Sub(request.TD, request.Block.Difficulty())
		)
		// Update the peers total difficulty if better than the previous
		if _, td := p.Head(); trueTD.Cmp(td) > 0 {
			p.SetHead(trueHead, trueTD)

			// Schedule a sync if above ours. Note, this will not fire a sync for a gap of
			// a singe block (as the true TD is below the propagated block), however this
			// scenario should easily be covered by the fetcher.
			currentBlock := pm.blockchain.CurrentBlock()
			if trueTD.Cmp(pm.blockchain.GetTd(currentBlock.Hash())) > 0 {
				go pm.synchronise(p)
			}
		}

	case TxMsg:
		if atomic.LoadUint32(&pm.acceptTxs) == 0 {
			break
		}

		var txs []*types.Transaction
		if err := msg.DecodePayload(&txs); err != nil {
			return fmt.Errorf("msg %v: %v", msg, err)
		}
		if pm.txpool.AddTxsChan(txs) {
			for i, tx := range txs {
				if tx == nil {
					return fmt.Errorf("transaction %d is nil", i)
				}
				p.MarkTransaction(tx.Hash())
			}
		}
	case ConfirmedMsg:
		confirmed := types.Confirmed{}
		if err := msg.DecodePayload(&confirmed); err != nil {
			return fmt.Errorf("msg %v: %v", msg, err)
		}
		pm.eventMux.Post(confirmed)
		pm.BroadcastConfirmed(&confirmed)
	default:
		return fmt.Errorf("invalide message %v", msg.Code)
	}
	return nil
}

func (pm *ProtocolManager) BroadcastConfirmed(confirmed *types.Confirmed) {
	for _, peer := range pm.peers.PeersWithoutConfirmed(confirmed.Hash()) {
		peer.SendConfirmed(confirmed)
	}
}

func (pm *ProtocolManager) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := pm.peers.PeersWithoutBlock(hash)

	if propagate {
		var td *big.Int
		if parent := pm.blockchain.GetBlock(block.PreviousHash()); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), pm.blockchain.GetTd(block.PreviousHash()))
		} else {
			log.Errorf("Propagating dangling block height %v hash %v", block.Height(), hash)
			return
		}
		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {
			peer.SendNewBlock(block, td)
		}
		return
	}
	if pm.blockchain.HasBlock(hash) {
		for _, peer := range peers {
			peer.SendNewBlockHashes([]utils.Hash{hash})
		}
	}
}

func (pm *ProtocolManager) BroadcastTxs(txs types.Transactions) {
	var txset = make(map[*peer]types.Transactions)

	for _, tx := range txs {
		peers := pm.peers.PeersWithoutTx(tx.Hash())
		for _, peer := range peers {
			txset[peer] = append(txset[peer], tx)
		}
		log.Infof("Broadcast transaction hash %v recipients %v", tx.Hash(), len(peers))
	}
	for peer, txs := range txset {
		peer.SendTransactions(txs)
	}
}

func (pm *ProtocolManager) minedBroadcastLoop() {
	for obj := range pm.minedBlockSub.Chan() {
		switch ev := obj.Data.(type) {
		case feed.NewMinedBlockEvent:
			pm.BroadcastBlock(ev.Block, true)
			pm.BroadcastBlock(ev.Block, false)
		case feed.NewConfirmedEvent:
			pm.BroadcastConfirmed(ev.Confirmed)
		case feed.NewMiner:
			atomic.StoreUint32(&pm.acceptTxs, 1)
		}
	}
}

func (pm *ProtocolManager) txBroadcastLoop() {
	for {
		select {
		case feed := <-pm.txsCh:
			pm.BroadcastTxs(feed.Txs)

		case <-pm.txsSub.Err():
			return
		}
	}
}

type NodeInfo struct {
	Difficulty *big.Int   `json:"difficulty"`
	Genesis    utils.Hash `json:"genesis"`
	Head       utils.Hash `json:"head"`
}

func (pm *ProtocolManager) NodeInfo() *NodeInfo {
	currentBlock := pm.blockchain.CurrentBlock()
	return &NodeInfo{
		Difficulty: pm.blockchain.GetTd(currentBlock.Hash()),
		Genesis:    pm.blockchain.GetBlockByHeight(0).Hash(),
		Head:       currentBlock.Hash(),
	}
}

type txsync struct {
	p   *peer
	txs []*types.Transaction
}

func (pm *ProtocolManager) syncTransactions(p *peer) {
	var txs types.Transactions
	pending, _ := pm.txpool.Pending()
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	if len(txs) == 0 {
		return
	}
	select {
	case pm.txsyncCh <- &txsync{p, txs}:
	case <-pm.quitSync:
	}
}

func (pm *ProtocolManager) txsyncLoop() {
	var (
		pending = make(map[discover.NodeID]*txsync)
		sending = false
		pack    = new(txsync)
		done    = make(chan error, 1)
	)

	send := func(s *txsync) {
		size := utils.StorageSize(0)
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs) && size < txsyncPackSize; i++ {
			pack.txs = append(pack.txs, s.txs[i])
			size += s.txs[i].Size()
		}
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.p.ID())
		}
		log.Infof("Sending batch of transactions count %v", len(pack.txs))
		sending = true
		go func() { done <- pack.p.SendTransactions(pack.txs) }()
	}

	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-pm.txsyncCh:
			pending[s.p.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			if err != nil {
				delete(pending, pack.p.ID())
			}
			if s := pick(); s != nil {
				send(s)
			}
		case <-pm.quitSync:
			return
		}
	}
}

func (pm *ProtocolManager) syncer() {
	pm.fetcher.Start()
	defer pm.fetcher.Stop()
	defer pm.downloader.Terminate()

	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSync.Stop()

	for {
		select {
		case <-pm.newPeerCh:
			if pm.peers.Len() < minDesiredPeerCount {
				break
			}
			go pm.synchronise(pm.peers.BestPeer())

		case <-forceSync.C:
			go pm.synchronise(pm.peers.BestPeer())

		case <-pm.noMorePeers:
			return
		}
	}
}

func (pm *ProtocolManager) synchronise(peer *peer) {
	if peer == nil {
		return
	}

	currentBlock := pm.blockchain.CurrentBlock()
	td := pm.blockchain.GetTd(currentBlock.Hash())
	_, pTd := peer.Head()
	if pTd.Cmp(td) <= 0 {
		return
	}

	if err := pm.downloader.Synchronise(peer.id, peer.head, peer.td); err != nil {
		return
	}
	atomic.StoreUint32(&pm.acceptTxs, 1)
	if head := pm.blockchain.CurrentBlock(); head.Height().Uint64() > 0 {
		go pm.BroadcastBlock(head, false)
	}
}

const (
	softResponseLimit   = 2 * 1024 * 1024
	txChanSize          = 4096
	minDesiredPeerCount = 5
	forceSyncCycle      = 10 * time.Second
	txsyncPackSize      = 100 * 1024
)
