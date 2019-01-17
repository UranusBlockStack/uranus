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
	"fmt"
	"math/big"
	"sync"

	"github.com/UranusBlockStack/uranus/common/utils"
)

type peerSet struct {
	sync.RWMutex

	peers  map[string]*peer
	closed bool
}

func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

func (ps *peerSet) Register(p *peer) error {
	ps.Lock()
	defer ps.Unlock()

	if ps.closed {
		return fmt.Errorf("closed")
	}
	if _, ok := ps.peers[p.id]; ok {
		return fmt.Errorf("registered")
	}
	ps.peers[p.id] = p

	return nil
}

func (ps *peerSet) Unregister(id string) error {
	ps.Lock()
	defer ps.Unlock()

	p, ok := ps.peers[id]
	if !ok {
		return fmt.Errorf("unregistered")
	}
	delete(ps.peers, id)
	p.close()

	return nil
}

func (ps *peerSet) Peer(id string) *peer {
	ps.RLock()
	defer ps.RUnlock()

	return ps.peers[id]
}

func (ps *peerSet) Len() int {
	ps.RLock()
	defer ps.RUnlock()

	return len(ps.peers)
}

func (ps *peerSet) PeersWithoutBlock(hash utils.Hash) []*peer {
	ps.RLock()
	defer ps.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.existedBlocks.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithoutTx(hash utils.Hash) []*peer {
	ps.RLock()
	defer ps.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.existedTxs.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) PeersWithoutConfirmed(hash utils.Hash) []*peer {
	ps.RLock()
	defer ps.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.existedConfirmed.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (ps *peerSet) BestPeer() *peer {
	ps.RLock()
	defer ps.RUnlock()

	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

func (ps *peerSet) Close() {
	ps.Lock()
	defer ps.Unlock()

	for _, p := range ps.peers {
		p.Disconnect("close")
	}
	ps.closed = true
}
