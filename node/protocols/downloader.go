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

package protocols

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/feed"
	"gopkg.in/fatih/set.v0"
)

var (
	MinHashFetch    = 512
	MaxHashFetch    = 512
	MaxBlockFetch   = 128
	hashTTL         = 5 * time.Second
	blockSoftTTL    = 3 * time.Second
	blockHardTTL    = 3 * blockSoftTTL
	crossCheckCycle = time.Second

	maxQueuedHashes = 256 * 1024
	maxBannedHashes = 4096
	maxBlockProcess = 256
)

var (
	errBusy             = errors.New("busy")
	errUnknownPeer      = errors.New("peer is unknown or unhealthy")
	errBadPeer          = errors.New("action from bad peer ignored")
	errStallingPeer     = errors.New("peer is stalling")
	errBannedHead       = errors.New("peer head hash already banned")
	errNoPeers          = errors.New("no peers to keep download active")
	errPendingQueue     = errors.New("pending items in queue")
	errTimeout          = errors.New("timeout")
	errEmptyHashSet     = errors.New("empty hash set by peer")
	errPeersUnavailable = errors.New("no peers available or all peers tried for block download process")
	errAlreadyInPool    = errors.New("hash already in pool")
	errInvalidChain     = errors.New("retrieved hash chain is invalid")
	errCrossCheckFailed = errors.New("block cross-check failed")
	errCancelHashFetch  = errors.New("hash fetching canceled (requested)")
	errCancelBlockFetch = errors.New("block downloading canceled (requested)")
	errNoSyncActive     = errors.New("no sync active")
)

type hashCheckFn func(utils.Hash) bool
type blockRetrievalFn func(utils.Hash) *types.Block
type headRetrievalFn func() *types.Block
type chainInsertFn func(types.Blocks) (int, error)
type peerDropFn func(id string)
type getTdFn func(utils.Hash) *big.Int

type blockPack struct {
	peerID string
	blocks []*types.Block
}

type hashPack struct {
	peerID string
	hashes []utils.Hash
}

type crossCheck struct {
	expire time.Time
	parent utils.Hash
}

type Downloader struct {
	mux *feed.TypeMux

	queue  *queue
	peers  *peerSet
	checks map[utils.Hash]*crossCheck
	banned *set.Set

	interrupt int32

	importStart time.Time
	importQueue []*Block
	importDone  int
	importLock  sync.Mutex

	hasBlock    hashCheckFn
	getBlock    blockRetrievalFn
	headBlock   headRetrievalFn
	insertChain chainInsertFn
	dropPeer    peerDropFn
	gettd       getTdFn

	synchronising int32
	processing    int32
	notified      int32

	newPeerCh chan *peer
	hashCh    chan hashPack
	blockCh   chan blockPack
	processCh chan bool

	cancelCh   chan struct{}
	cancelLock sync.RWMutex
}

type Block struct {
	RawBlock   *types.Block
	OriginPeer string
}

func NewDownloader(mux *feed.TypeMux, hasBlock hashCheckFn, getBlock blockRetrievalFn, headBlock headRetrievalFn, gettd getTdFn, insertChain chainInsertFn, dropPeer peerDropFn) *Downloader {
	downloader := &Downloader{
		mux:         mux,
		queue:       newQueue(),
		peers:       newPeerSet(),
		hasBlock:    hasBlock,
		getBlock:    getBlock,
		headBlock:   headBlock,
		gettd:       gettd,
		insertChain: insertChain,
		dropPeer:    dropPeer,
		newPeerCh:   make(chan *peer, 1),
		hashCh:      make(chan hashPack, 1),
		blockCh:     make(chan blockPack, 1),
		processCh:   make(chan bool, 1),
	}
	downloader.banned = set.New()
	return downloader
}

func (d *Downloader) Stats() (pending int, cached int, importing int, estimate time.Duration) {
	pending, cached = d.queue.Size()

	d.importLock.Lock()
	defer d.importLock.Unlock()

	for len(d.importQueue) > 0 && d.hasBlock(d.importQueue[0].RawBlock.Hash()) {
		d.importQueue = d.importQueue[1:]
		d.importDone++
	}
	importing = len(d.importQueue)

	estimate = 0
	if d.importDone > 0 {
		estimate = time.Since(d.importStart) / time.Duration(d.importDone) * time.Duration(pending+cached+importing)
	}
	return
}

func (d *Downloader) Synchronising() bool {
	return atomic.LoadInt32(&d.synchronising) > 0
}

func (d *Downloader) RegisterPeer(id string, version int, head utils.Hash, getRelHashes relativeHashFetcherFn, getAbsHashes absoluteHashFetcherFn, getBlocks blockFetcherFn) error {
	if d.banned.Has(head) {
		log.Infof("Register rejected, head hash banned: %v", id)
		return errBannedHead
	}
	log.Infof("Registering peer %v", id)
	if err := d.peers.Register(newPeer(id, head, getRelHashes, getAbsHashes, getBlocks)); err != nil {
		log.Infof("Register failed: %v", err)
		return err
	}
	return nil
}

func (d *Downloader) UnregisterPeer(id string) error {
	log.Infof("Unregistering peer %v", id)
	if err := d.peers.Unregister(id); err != nil {
		log.Infof("Unregister failed: %v", err)
		return err
	}
	return nil
}

func (d *Downloader) Synchronise(id string, head utils.Hash, td *big.Int) error {
	log.Infof("Attempting synchronisation: %v, head 0x%x, TD %v", id, head[:4], td)

	err := d.synchronise(id, head, td)
	switch err {
	case nil:
		log.Infof("Synchronisation completed")

	case errBusy:
		log.Debugf("Synchronisation already in progress")

	case errTimeout, errBadPeer, errStallingPeer, errBannedHead, errEmptyHashSet, errPeersUnavailable, errInvalidChain, errCrossCheckFailed:
		log.Errorf("Removing peer %v: %v", id, err)
		d.dropPeer(id)

	case errPendingQueue:
		log.Errorf("Synchronisation aborted: %v", err)

	default:
		log.Errorf("Synchronisation failed: %v", err)
	}
	return err
}

func (d *Downloader) synchronise(id string, hash utils.Hash, td *big.Int) error {
	if !atomic.CompareAndSwapInt32(&d.synchronising, 0, 1) {
		return errBusy
	}
	defer atomic.StoreInt32(&d.synchronising, 0)

	if d.banned.Has(hash) {
		return errBannedHead
	}
	if atomic.CompareAndSwapInt32(&d.notified, 0, 1) {
		log.Info("Block synchronisation started")
	}
	if _, cached := d.queue.Size(); cached > 0 && d.queue.GetHeadBlock() != nil {
		return errPendingQueue
	}
	d.queue.Reset()
	d.peers.Reset()
	d.checks = make(map[utils.Hash]*crossCheck)

	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelLock.Unlock()

	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeer(p, hash, td)
}

func (d *Downloader) Has(hash utils.Hash) bool {
	return d.queue.Has(hash)
}

func (d *Downloader) syncWithPeer(p *peer, hash utils.Hash, td *big.Int) (err error) {
	d.mux.Post(StartEvent{})
	defer func() {
		if err != nil {
			log.Errorf("downloading canceled: findAncestor %v", err)
			d.cancel()
			d.mux.Post(FailedEvent{err})
		} else {
			d.mux.Post(DoneEvent{})
		}
	}()

	number, err := d.findAncestor(p)
	if err != nil {
		return err
	}
	errc := make(chan error, 2)
	go func() { errc <- d.fetchHashes(p, td, number+1) }()
	go func() { errc <- d.fetchBlocks(number + 1) }()

	if err := <-errc; err != nil {
		log.Errorf("downloading canceled: fetchBlocks or fetchHashes %v", err)
		d.cancel()
		<-errc
		return err
	}
	return <-errc
}

func (d *Downloader) cancel() {
	d.cancelLock.Lock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
		default:
			close(d.cancelCh)
		}
	}
	d.cancelLock.Unlock()

	d.queue.Reset()
}

func (d *Downloader) Terminate() {
	atomic.StoreInt32(&d.interrupt, 1)
	d.cancel()
}

func (d *Downloader) findAncestor(p *peer) (uint64, error) {
	log.Infof("%v: looking for common ancestor", p.id)

	head := d.headBlock().Height().Uint64()
	from := int64(head) - int64(MaxHashFetch)
	if from < 0 {
		from = 0
	}
	go p.getAbsHashes(uint64(from), MaxHashFetch)

	number, hash := uint64(0), utils.Hash{}
	timeout := time.After(hashTTL)

	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return 0, errCancelHashFetch

		case hashPack := <-d.hashCh:
			if hashPack.peerID != p.id {
				log.Infof("Received hashes from incorrect peer(%s)", hashPack.peerID)
				break
			}
			hashes := hashPack.hashes
			if len(hashes) == 0 {
				log.Infof("%v: empty head hash set", p.id)
				return 0, errEmptyHashSet
			}
			finished = true
			for i := len(hashes) - 1; i >= 0; i-- {
				if d.hasBlock(hashes[i]) {
					number, hash = uint64(from)+uint64(i), hashes[i]
					break
				}
			}

		case <-d.blockCh:

		case <-timeout:
			log.Infof("%v: head hash timeout", p.id)
			return 0, errTimeout
		}
	}
	if !utils.EmptyHash(hash) {
		log.Infof("%v: common ancestor: #%d [%x]", p.id, number, hash[:4])
		return number, nil
	}
	start, end := uint64(0), head
	for start+1 < end {
		check := (start + end) / 2

		timeout := time.After(hashTTL)
		go p.getAbsHashes(uint64(check), 1)

		for arrived := false; !arrived; {
			select {
			case <-d.cancelCh:
				return 0, errCancelHashFetch

			case hashPack := <-d.hashCh:
				if hashPack.peerID != p.id {
					log.Infof("Received hashes from incorrect peer(%s)", hashPack.peerID)
					break
				}
				hashes := hashPack.hashes
				if len(hashes) != 1 {
					log.Infof("%v: invalid search hash set (%d)", p.id, len(hashes))
					return 0, errBadPeer
				}
				arrived = true

				block := d.getBlock(hashes[0])
				if block == nil {
					end = check
					break
				}
				if block.Height().Uint64() != check {
					log.Infof("%v: non requested hash #%d [%x], instead of #%d", p.id, block.Height().Uint64(), block.Hash().Bytes()[:4], check)
					return 0, errBadPeer
				}
				start = check

			case <-d.blockCh:

			case <-timeout:
				log.Infof("%v: search hash timeout", p.id)
				return 0, errTimeout
			}
		}
	}
	return start, nil
}

func (d *Downloader) fetchHashes(p *peer, td *big.Int, from uint64) error {
	log.Infof("%v: downloading hashes from #%d", p.id, from)

	timeout := time.NewTimer(0)
	<-timeout.C
	defer timeout.Stop()

	getHashes := func(from uint64) {
		log.Infof("%v: fetching %d hashes from #%d", p.id, MaxHashFetch, from)

		go p.getAbsHashes(from, MaxHashFetch)
		timeout.Reset(hashTTL)
	}
	getHashes(from)
	gotHashes := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelHashFetch

		case hashPack := <-d.hashCh:
			if hashPack.peerID != p.id {
				log.Infof("Received hashes from incorrect peer(%s)", hashPack.peerID)
				break
			}
			timeout.Stop()

			if len(hashPack.hashes) == 0 {
				log.Infof("%v: no available hashes", p.id)

				select {
				case d.processCh <- false:
				case <-d.cancelCh:
				}

				if !gotHashes && td.Cmp(d.gettd(d.headBlock().Hash())) > 0 {
					return errStallingPeer
				}
				return nil
			}
			gotHashes = true

			log.Infof("%v: inserting %d hashes from #%d", p.id, len(hashPack.hashes), from)

			inserts := d.queue.Insert(hashPack.hashes, true)
			if len(inserts) != len(hashPack.hashes) {
				log.Errorf("%v: stale hashes", p.id)
				return errBadPeer
			}
			cont := d.queue.Pending() < maxQueuedHashes
			select {
			case d.processCh <- cont:
			default:
			}
			if !cont {
				return nil
			}
			from += uint64(len(hashPack.hashes))
			getHashes(from)

		case <-timeout.C:
			log.Infof("%v: hash request timed out", p.id)
			return errTimeout
		}
	}
}

func (d *Downloader) fetchBlocks(from uint64) error {
	log.Infof("Downloading blocks from #%d", from)
	defer log.Infof("Block download terminated from #%d", from)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	d.queue.Prepare(from)
	finished := false

	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case blockPack := <-d.blockCh:

			if peer := d.peers.Peer(blockPack.peerID); peer != nil {
				err := d.queue.Deliver(blockPack.peerID, blockPack.blocks)
				switch err {
				case nil:
					if len(blockPack.blocks) == 0 {
						peer.Demote()
						peer.SetIdle()
						log.Infof("%s: no blocks delivered", peer.id)
						break
					}
					peer.Promote()
					peer.SetIdle()
					log.Infof("%s: delivered %d blocks", peer.id, len(blockPack.blocks))
					go d.process()

				case errInvalidChain:
					return err

				case errNoFetchesPending:
					peer.Demote()
					peer.SetIdle()
					log.Infof("%s: out of bound delivery", peer.id)

				case errStaleDelivery:
					peer.Demote()
					log.Infof("%s: stale delivery", peer.id)

				default:
					peer.Demote()
					peer.SetIdle()
					log.Infof("%s: delivery partially failed: %v", peer.id, err)
					go d.process()
				}
			}
			select {
			case update <- struct{}{}:
			default:
			}

		case cont := <-d.processCh:
			if !cont {
				finished = true
			}
			select {
			case update <- struct{}{}:
			default:
			}

		case <-ticker.C:
			select {
			case update <- struct{}{}:
			default:
			}

		case <-update:
			if d.peers.Len() == 0 {
				return errNoPeers
			}
			for _, pid := range d.queue.Expire(blockHardTTL) {
				if peer := d.peers.Peer(pid); peer != nil {
					peer.Demote()
					log.Infof("%s: block delivery timeout", peer.id)
				}
			}
			if d.queue.Pending() == 0 {
				if d.queue.InFlight() == 0 && finished {
					log.Infof("Block fetching completed")
					return nil
				}
				break
			}
			for _, peer := range d.peers.IdlePeers() {
				if d.queue.Throttle() {
					break
				}
				request := d.queue.Reserve(peer, peer.Capacity())
				if request == nil {
					continue
				}
				if err := peer.Fetch(request); err != nil {
					log.Infof("%v: fetch failed, rescheduling", peer)
					d.queue.Cancel(request)
				}
			}
			if !d.queue.Throttle() && d.queue.InFlight() == 0 {
				return errPeersUnavailable
			}
		}
	}
}
func (d *Downloader) banBlocks(peerID string, head utils.Hash) error {
	log.Infof("Banning a batch out of %d blocks from %s", d.queue.Pending(), peerID)

	peer := d.peers.Peer(peerID)
	if peer == nil {
		return nil
	}
	request := d.queue.Reserve(peer, MaxBlockFetch)
	if request == nil {
		return nil
	}
	if err := peer.Fetch(request); err != nil {
		return err
	}
	timeout := time.After(blockHardTTL)
	for {
		select {
		case <-d.cancelCh:
			return errCancelBlockFetch

		case <-timeout:
			return errTimeout

		case <-d.hashCh:

		case blockPack := <-d.blockCh:
			blocks := blockPack.blocks

			if len(blocks) == 1 {
				block := blocks[0]
				if _, ok := d.checks[block.Hash()]; ok {
					delete(d.checks, block.Hash())
					break
				}
			}
			if blockPack.peerID != peerID {
				break
			}
			if len(blocks) == 0 {
				return errors.New("no blocks returned to ban")
			}

			if bytes.Compare(blocks[0].Hash().Bytes(), head.Bytes()) != 0 {
				return errors.New("head block not the banned one")
			}
			index := 0
			for _, block := range blocks[1:] {
				if bytes.Compare(block.PreviousHash().Bytes(), blocks[index].Hash().Bytes()) != 0 {
					break
				}
				index++
			}
			d.banned.Add(blocks[index].Hash())
			for d.banned.Size() > maxBannedHashes {
				var evacuate utils.Hash

				d.banned.Each(func(item interface{}) bool {
					evacuate = item.(utils.Hash)
					return false
				})
				d.banned.Remove(evacuate)
			}
			log.Infof("Banned %d blocks from: %s", index+1, peerID)
			return nil
		}
	}
}

func (d *Downloader) process() {
	if !atomic.CompareAndSwapInt32(&d.processing, 0, 1) {
		return
	}

	defer func() {
		if atomic.LoadInt32(&d.interrupt) == 0 && d.queue.GetHeadBlock() != nil {
			d.process()
		}
	}()

	defer func() {
		d.importLock.Lock()
		d.importQueue = nil
		d.importDone = 0
		d.importLock.Unlock()

		atomic.StoreInt32(&d.processing, 0)
	}()
	for {
		blocks := d.queue.TakeBlocks()
		if len(blocks) == 0 {
			return
		}
		d.importLock.Lock()
		d.importStart = time.Now()
		d.importQueue = blocks
		d.importDone = 0
		d.importLock.Unlock()

		log.Infof("Inserting chain with %d blocks (#%v - #%v)", len(blocks), blocks[0].RawBlock.Height(), blocks[len(blocks)-1].RawBlock.Height())
		for len(blocks) != 0 {
			if atomic.LoadInt32(&d.interrupt) == 1 {
				return
			}
			max := int(math.Min(float64(len(blocks)), float64(maxBlockProcess)))
			raw := make(types.Blocks, 0, max)
			for _, block := range blocks[:max] {
				raw = append(raw, block.RawBlock)
			}
			index, err := d.insertChain(raw)
			if err != nil {
				d.dropPeer(blocks[index].OriginPeer)
				log.Errorf("downloading canceled: insertChain %v %v %v", raw[0].Height(), raw[0].Hash().String(), err)
				d.cancel()
				return
			}
			blocks = blocks[max:]
		}
	}
}

func (d *Downloader) DeliverBlocks(id string, blocks []*types.Block) error {
	if atomic.LoadInt32(&d.synchronising) == 0 {
		return errNoSyncActive
	}
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()

	select {
	case d.blockCh <- blockPack{id, blocks}:
		return nil

	case <-cancel:
		return errNoSyncActive
	}
}

func (d *Downloader) DeliverHashes(id string, hashes []utils.Hash) error {
	if atomic.LoadInt32(&d.synchronising) == 0 {
		return errNoSyncActive
	}
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()

	select {
	case d.hashCh <- hashPack{id, hashes}:
		return nil

	case <-cancel:
		return errNoSyncActive
	}
}

type DoneEvent struct{}
type StartEvent struct{}
type FailedEvent struct{ Err error }

var (
	errNoFetchesPending = errors.New("no fetches pending")
	errStaleDelivery    = errors.New("stale delivery")
)
