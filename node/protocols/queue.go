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
	"fmt"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

var (
	blockCacheLimit = 8 * MaxBlockFetch
)

type fetchRequest struct {
	Peer   *peer
	Hashes map[utils.Hash]int
	Time   time.Time
}

type queue struct {
	hashPool    map[utils.Hash]int
	hashQueue   *prque.Prque
	hashCounter int

	pendPool map[string]*fetchRequest

	blockPool   map[utils.Hash]uint64
	blockCache  []*Block
	blockOffset uint64
	lock        sync.RWMutex
}

func newQueue() *queue {
	return &queue{
		hashPool:   make(map[utils.Hash]int),
		hashQueue:  prque.New(),
		pendPool:   make(map[string]*fetchRequest),
		blockPool:  make(map[utils.Hash]uint64),
		blockCache: make([]*Block, blockCacheLimit),
	}
}

func (q *queue) Reset() {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.hashPool = make(map[utils.Hash]int)
	q.hashQueue.Reset()
	q.hashCounter = 0

	q.pendPool = make(map[string]*fetchRequest)

	q.blockPool = make(map[utils.Hash]uint64)
	q.blockOffset = 0
	q.blockCache = make([]*Block, blockCacheLimit)
}

func (q *queue) Size() (int, int) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.hashPool), len(q.blockPool)
}

func (q *queue) Pending() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return q.hashQueue.Size()
}

func (q *queue) InFlight() int {
	q.lock.RLock()
	defer q.lock.RUnlock()

	return len(q.pendPool)
}

func (q *queue) Throttle() bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	pending := 0
	for _, request := range q.pendPool {
		pending += len(request.Hashes)
	}
	return pending >= len(q.blockCache)-len(q.blockPool)
}

func (q *queue) Has(hash utils.Hash) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	if _, ok := q.hashPool[hash]; ok {
		return true
	}
	if _, ok := q.blockPool[hash]; ok {
		return true
	}
	return false
}

func (q *queue) Insert(hashes []utils.Hash, fifo bool) []utils.Hash {
	q.lock.Lock()
	defer q.lock.Unlock()

	inserts := make([]utils.Hash, 0, len(hashes))
	for _, hash := range hashes {
		if old, ok := q.hashPool[hash]; ok {
			_ = old
			log.Warnf("Hash %x already scheduled at index %v", hash, old)
			continue
		}
		q.hashCounter = q.hashCounter + 1
		inserts = append(inserts, hash)

		q.hashPool[hash] = q.hashCounter
		if fifo {
			q.hashQueue.Push(hash, -float32(q.hashCounter)) // Lowest gets schedules first
		} else {
			q.hashQueue.Push(hash, float32(q.hashCounter)) // Highest gets schedules first
		}
	}
	return inserts
}

func (q *queue) GetHeadBlock() *Block {
	q.lock.RLock()
	defer q.lock.RUnlock()

	if len(q.blockCache) == 0 {
		return nil
	}
	return q.blockCache[0]
}

func (q *queue) GetBlock(hash utils.Hash) *Block {
	q.lock.RLock()
	defer q.lock.RUnlock()

	index, ok := q.blockPool[hash]
	if !ok {
		return nil
	}
	if q.blockOffset <= index && index < q.blockOffset+uint64(len(q.blockCache)) {
		return q.blockCache[index-q.blockOffset]
	}
	return nil
}

func (q *queue) TakeBlocks() []*Block {
	q.lock.Lock()
	defer q.lock.Unlock()

	blocks := []*Block{}
	for _, block := range q.blockCache {
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		delete(q.blockPool, block.RawBlock.Hash())
	}

	copy(q.blockCache, q.blockCache[len(blocks):])
	for k, n := len(q.blockCache)-len(blocks), len(q.blockCache); k < n; k++ {
		q.blockCache[k] = nil
	}
	q.blockOffset += uint64(len(blocks))

	return blocks
}

func (q *queue) Reserve(p *peer, count int) *fetchRequest {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.hashQueue.Empty() {
		return nil
	}
	if _, ok := q.pendPool[p.id]; ok {
		return nil
	}
	space := len(q.blockCache) - len(q.blockPool)
	for _, request := range q.pendPool {
		space -= len(request.Hashes)
	}
	send := make(map[utils.Hash]int)
	skip := make(map[utils.Hash]int)

	for proc := 0; proc < space && len(send) < count && !q.hashQueue.Empty(); proc++ {
		hash, priority := q.hashQueue.Pop()
		if p.ignored.Has(hash) {
			skip[hash.(utils.Hash)] = int(priority)
		} else {
			send[hash.(utils.Hash)] = int(priority)
		}
	}
	for hash, index := range skip {
		q.hashQueue.Push(hash, float32(index))
	}
	if len(send) == 0 {
		return nil
	}
	request := &fetchRequest{
		Peer:   p,
		Hashes: send,
		Time:   time.Now(),
	}
	q.pendPool[p.id] = request

	return request
}

func (q *queue) Cancel(request *fetchRequest) {
	q.lock.Lock()
	defer q.lock.Unlock()

	for hash, index := range request.Hashes {
		q.hashQueue.Push(hash, float32(index))
	}
	delete(q.pendPool, request.Peer.id)
}

func (q *queue) Expire(timeout time.Duration) []string {
	q.lock.Lock()
	defer q.lock.Unlock()

	peers := []string{}
	for id, request := range q.pendPool {
		if time.Since(request.Time) > timeout {
			for hash, index := range request.Hashes {
				q.hashQueue.Push(hash, float32(index))
			}
			peers = append(peers, id)
		}
	}
	for _, id := range peers {
		delete(q.pendPool, id)
	}
	return peers
}

func (q *queue) Deliver(id string, blocks []*types.Block) (err error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	request := q.pendPool[id]
	if request == nil {
		return errors.New("no fetches pending")
	}
	delete(q.pendPool, id)

	if len(blocks) == 0 {
		for hash := range request.Hashes {
			request.Peer.ignored.Add(hash)
		}
	}
	errs := make([]error, 0)
	for _, block := range blocks {
		hash := block.Hash()
		if _, ok := request.Hashes[hash]; !ok {
			errs = append(errs, fmt.Errorf("non-requested block %x", hash))
			continue
		}
		index := int(int64(block.Height().Uint64()) - int64(q.blockOffset))
		if index >= len(q.blockCache) || index < 0 {
			return errInvalidChain
		}
		q.blockCache[index] = &Block{
			RawBlock:   block,
			OriginPeer: id,
		}
		delete(request.Hashes, hash)
		delete(q.hashPool, hash)
		q.blockPool[hash] = block.Height().Uint64()
	}
	for hash, index := range request.Hashes {
		q.hashQueue.Push(hash, float32(index))
	}
	if len(errs) != 0 {
		if len(errs) == len(blocks) {
			return errors.New("stale delivery")
		}
		return fmt.Errorf("multiple failures: %v", errs)
	}
	return nil
}

func (q *queue) Prepare(offset uint64) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.blockOffset < offset {
		q.blockOffset = offset
	}
}
