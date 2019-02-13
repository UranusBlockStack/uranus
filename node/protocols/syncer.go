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
	"errors"
	"math/rand"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

const (
	arriveTimeout = 500 * time.Millisecond
	gatherSlack   = 100 * time.Millisecond
	fetchTimeout  = 5 * time.Second
	maxUncleDist  = 32
	maxQueueDist  = 32
	hashLimit     = 256
	blockLimit    = 64
)

var (
	errTerminated = errors.New("terminated")
)

type blockRequesterFn func([]utils.Hash) error

type blockValidatorFn func(header *types.BlockHeader) error

type blockBroadcasterFn func(block *types.Block, propagate bool)

type chainHeightFn func() uint64

type announce struct {
	hash utils.Hash
	time time.Time

	origin string
	fetch  blockRequesterFn
}

type inject struct {
	origin string
	block  *types.Block
}

type Fetcher struct {
	notify chan *announce
	inject chan *inject
	filter chan chan []*types.Block
	done   chan utils.Hash
	quit   chan struct{}

	announces map[string]int
	announced map[utils.Hash][]*announce
	fetching  map[utils.Hash]*announce

	queue  *prque.Prque
	queues map[string]int
	queued map[utils.Hash]*inject

	getBlock       blockRetrievalFn
	validateBlock  blockValidatorFn
	broadcastBlock blockBroadcasterFn
	chainHeight    chainHeightFn
	insertChain    chainInsertFn
	dropPeer       peerDropFn
}

func NewFetcher(getBlock blockRetrievalFn, validateBlock blockValidatorFn, broadcastBlock blockBroadcasterFn, chainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn) *Fetcher {
	return &Fetcher{
		notify:         make(chan *announce),
		inject:         make(chan *inject),
		filter:         make(chan chan []*types.Block),
		done:           make(chan utils.Hash),
		quit:           make(chan struct{}),
		announces:      make(map[string]int),
		announced:      make(map[utils.Hash][]*announce),
		fetching:       make(map[utils.Hash]*announce),
		queue:          prque.New(),
		queues:         make(map[string]int),
		queued:         make(map[utils.Hash]*inject),
		getBlock:       getBlock,
		validateBlock:  validateBlock,
		broadcastBlock: broadcastBlock,
		chainHeight:    chainHeight,
		insertChain:    insertChain,
		dropPeer:       dropPeer,
	}
}

func (f *Fetcher) Start() {
	go f.loop()
}

func (f *Fetcher) Stop() {
	close(f.quit)
}

func (f *Fetcher) Notify(peer string, hash utils.Hash, time time.Time, fetcher blockRequesterFn) error {
	block := &announce{
		hash:   hash,
		time:   time,
		origin: peer,
		fetch:  fetcher,
	}
	select {
	case f.notify <- block:
		log.Infof("Peer %s: notify block [%x]", peer, hash.Bytes()[:4])
		return nil
	case <-f.quit:
		return errTerminated
	}
}

func (f *Fetcher) Enqueue(peer string, block *types.Block) error {
	op := &inject{
		block:  block,
		origin: peer,
	}

	select {
	case f.inject <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

func (f *Fetcher) Filter(blocks types.Blocks) types.Blocks {
	filter := make(chan []*types.Block)

	select {
	case f.filter <- filter:
	case <-f.quit:
		return nil
	}
	select {
	case filter <- blocks:
	case <-f.quit:
		return nil
	}
	select {
	case blocks := <-filter:
		return blocks
	case <-f.quit:
		return nil
	}
}

func (f *Fetcher) loop() {
	fetch := time.NewTimer(0)
	for {
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > fetchTimeout {
				log.Infof("forget notify block [%x], timeout %v", hash.Bytes()[:4], time.Since(announce.time))
				f.forgetHash(hash)
			}
		}
		height := f.chainHeight()
		for !f.queue.Empty() {
			op := f.queue.PopItem().(*inject)

			number := op.block.Height().Uint64()
			if number > height+1 {
				f.queue.Push(op, -float32(op.block.Height().Uint64()))
				break
			}
			hash := op.block.Hash()
			if number+maxUncleDist < height || f.getBlock(hash) != nil {
				f.forgetBlock(hash)
				break
			}
			f.insert(op.origin, op.block)
		}
		select {
		case <-f.quit:
			return

		case notification := <-f.notify:

			count := f.announces[notification.origin] + 1
			if count > hashLimit {
				log.Infof("Peer %s: exceeded outstanding announces (%d)", notification.origin, hashLimit)
				break
			}
			if _, ok := f.fetching[notification.hash]; ok {
				break
			}
			f.announces[notification.origin] = count
			f.announced[notification.hash] = append(f.announced[notification.hash], notification)
			if len(f.announced) == 1 {
				f.reschedule(fetch)
			}

		case op := <-f.inject:
			f.enqueue(op.origin, op.block)

		case hash := <-f.done:
			f.forgetHash(hash)
			f.forgetBlock(hash)

		case <-fetch.C:
			request := make(map[string][]utils.Hash)

			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arriveTimeout-gatherSlack {
					announce := announces[rand.Intn(len(announces))]
					f.forgetHash(hash)

					if f.getBlock(hash) == nil {
						request[announce.origin] = append(request[announce.origin], hash)
						f.fetching[hash] = announce
					}
				}
			}
			for _, hashes := range request {
				fetcher, hashes := f.fetching[hashes[0]].fetch, hashes
				go func() {
					fetcher(hashes)
				}()
			}
			f.reschedule(fetch)

		case filter := <-f.filter:
			var blocks types.Blocks
			select {
			case blocks = <-filter:
			case <-f.quit:
				return
			}

			explicit, download := []*types.Block{}, []*types.Block{}
			for _, block := range blocks {
				hash := block.Hash()

				if f.fetching[hash] != nil && f.queued[hash] == nil {
					if f.getBlock(hash) == nil {
						explicit = append(explicit, block)
					} else {
						f.forgetHash(hash)
					}
				} else {
					download = append(download, block)
				}
			}

			select {
			case filter <- download:
			case <-f.quit:
				return
			}
			for _, block := range explicit {
				if announce := f.fetching[block.Hash()]; announce != nil {
					f.enqueue(announce.origin, block)
				}
			}
		}
	}
}

func (f *Fetcher) reschedule(fetch *time.Timer) {
	if len(f.announced) == 0 {
		return
	}
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	duration := arriveTimeout - time.Since(earliest)
	if duration <= 0 {
		duration = time.Millisecond
	}
	fetch.Reset(duration)
}

func (f *Fetcher) enqueue(peer string, block *types.Block) {
	hash := block.Hash()

	count := f.queues[peer] + 1
	if count > blockLimit {
		log.Infof("Peer %s: discarded block #%d [%x], exceeded allowance (%d)", peer, block.Height().Uint64(), hash.Bytes()[:4], blockLimit)
		return
	}
	if dist := int64(block.Height().Uint64()) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
		log.Infof("Peer %s: discarded block #%d [%x], distance %d", peer, block.Height().Uint64(), hash.Bytes()[:4], dist)
		return
	}
	if _, ok := f.queued[hash]; !ok {
		op := &inject{
			origin: peer,
			block:  block,
		}
		f.queues[peer] = count
		f.queued[hash] = op
		f.queue.Push(op, -float32(block.Height().Uint64()))

		log.Infof("Peer %s: queued block #%d [%x], total %v", peer, block.Height().Uint64(), hash.Bytes()[:4], f.queue.Size())

	}
}

func (f *Fetcher) insert(peer string, block *types.Block) {
	hash := block.Hash()

	log.Infof("Peer %s: importing block #%d [%x]", peer, block.Height().Uint64(), hash[:4])
	go func() {
		defer func() { f.done <- hash }()

		parent := f.getBlock(block.PreviousHash())
		if parent == nil {
			return
		}
		switch err := f.validateBlock(block.BlockHeader()); err {
		case nil:
			go f.broadcastBlock(block, true)

		default:
			log.Infof("Peer %s: block #%d [%x] verification failed: %v", peer, block.Height().Uint64(), hash[:4], err)
			f.dropPeer(peer)
			return
		}
		if _, err := f.insertChain(types.Blocks{block}); err != nil {
			log.Infof("Peer %s: block #%d [%x] import failed: %v", peer, block.Height().Uint64(), hash[:4], err)
			return
		}

		go f.broadcastBlock(block, false)
	}()
}

func (f *Fetcher) forgetHash(hash utils.Hash) {
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)

	if announce := f.fetching[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.fetching, hash)
	}
}

func (f *Fetcher) forgetBlock(hash utils.Hash) {
	if insert := f.queued[hash]; insert != nil {
		f.queues[insert.origin]--
		if f.queues[insert.origin] == 0 {
			delete(f.queues, insert.origin)
		}
		delete(f.queued, hash)
	}
}
