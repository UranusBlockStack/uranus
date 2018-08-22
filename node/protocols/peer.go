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
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/UranusBlockStack/uranus/common/utils"

	"gopkg.in/fatih/set.v0"
)

type relativeHashFetcherFn func(utils.Hash) error
type absoluteHashFetcherFn func(uint64, int) error
type blockFetcherFn func([]utils.Hash) error

type peer struct {
	id           string
	head         utils.Hash
	idle         int32
	rep          int32
	capacity     int32
	started      time.Time
	ignored      *set.Set
	getRelHashes relativeHashFetcherFn
	getAbsHashes absoluteHashFetcherFn
	getBlocks    blockFetcherFn
}

func newPeer(id string, head utils.Hash, getRelHashes relativeHashFetcherFn, getAbsHashes absoluteHashFetcherFn, getBlocks blockFetcherFn) *peer {
	return &peer{
		id:           id,
		head:         head,
		capacity:     1,
		getRelHashes: getRelHashes,
		getAbsHashes: getAbsHashes,
		getBlocks:    getBlocks,
		ignored:      set.New(),
	}
}

func (p *peer) Reset() {
	atomic.StoreInt32(&p.idle, 0)
	atomic.StoreInt32(&p.capacity, 1)
	p.ignored.Clear()
}

func (p *peer) Fetch(request *fetchRequest) error {
	if !atomic.CompareAndSwapInt32(&p.idle, 0, 1) {
		return errors.New("already fetching blocks from peer")
	}
	p.started = time.Now()

	hashes := make([]utils.Hash, 0, len(request.Hashes))
	for hash := range request.Hashes {
		hashes = append(hashes, hash)
	}
	go p.getBlocks(hashes)

	return nil
}

func (p *peer) SetIdle() {
	scale := 2.0
	if time.Since(p.started) > blockSoftTTL {
		scale = 0.5
		if time.Since(p.started) > blockHardTTL {
			scale = 1 / float64(MaxBlockFetch)
		}
	}
	for {
		prev := atomic.LoadInt32(&p.capacity)
		next := int32(math.Max(1, math.Min(float64(MaxBlockFetch), float64(prev)*scale)))

		if atomic.CompareAndSwapInt32(&p.capacity, prev, next) {
			if next == 1 {
				p.Demote()
			}
			break
		}
	}
	atomic.StoreInt32(&p.idle, 0)
}

func (p *peer) Capacity() int {
	return int(atomic.LoadInt32(&p.capacity))
}

func (p *peer) Promote() {
	atomic.AddInt32(&p.rep, 1)
}

func (p *peer) Demote() {
	for {
		prev := atomic.LoadInt32(&p.rep)
		next := prev / 2

		if atomic.CompareAndSwapInt32(&p.rep, prev, next) {
			return
		}
	}
}

type peerSet struct {
	peers map[string]*peer
	lock  sync.RWMutex
}

func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

func (ps *peerSet) Reset() {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	for _, peer := range ps.peers {
		peer.Reset()
	}
}

func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[p.id]; ok {
		return errors.New("peer is already registered")
	}
	ps.peers[p.id] = p
	return nil
}

func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errors.New("peer is not registered")
	}
	delete(ps.peers, id)
	return nil
}

func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

func (ps *peerSet) AllPeers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		list = append(list, p)
	}
	return list
}

func (ps *peerSet) IdlePeers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if atomic.LoadInt32(&p.idle) == 0 {
			list = append(list, p)
		}
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if atomic.LoadInt32(&list[i].rep) < atomic.LoadInt32(&list[j].rep) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	return list
}
