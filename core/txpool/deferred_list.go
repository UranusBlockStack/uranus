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

package txpool

import (
	"container/heap"
	"math/big"
	"sync"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

type deferredList struct {
	all   map[utils.Hash]*types.Action
	items *timeHeap
	cache []*types.Action
	sync.RWMutex
}

func newDeferredList() *deferredList {
	return &deferredList{
		all:   make(map[utils.Hash]*types.Action),
		items: new(timeHeap),
	}
}

func (d *deferredList) Remove(actions []*types.Action) {
	for _, a := range actions {
		d.deleteAction(a.Hash())
	}
}

func (d *deferredList) Put(a *types.Action) {
	t := &timeActionHash{
		timestamp: new(big.Int).SetBytes(a.GenTimeStamp.Bytes()),
		hash:      a.Hash(),
	}
	heap.Push(d.items, t)
	d.addAction(a)
}

func (d *deferredList) Cap(threshold *big.Int) []*types.Action {
	var actions []*types.Action
	if d.cache != nil {
		for _, a := range d.cache {
			t := &timeActionHash{
				timestamp: a.GenTimeStamp,
				hash:      a.Hash(),
			}
			heap.Push(d.items, t)
		}
	}

	for len(*d.items) > 0 {
		t := heap.Pop(d.items).(*timeActionHash)
		a := d.getAction(t.hash)
		if a == nil {
			continue
		}
		if t.timestamp.Cmp(threshold) >= 0 {
			heap.Push(d.items, t)
			break
		}
		actions = append(actions, a)
	}

	d.cache = make([]*types.Action, len(actions))
	copy(d.cache, actions)
	return actions
}

func (d *deferredList) Range(f func(hash utils.Hash, t *timeActionHash) bool) {
	d.RLock()
	defer d.Unlock()

	for hash, action := range d.all {
		value := &timeActionHash{
			timestamp: action.GenTimeStamp,
			hash:      action.Hash(),
		}
		if !f(hash, value) {
			break
		}
	}
}

func (d *deferredList) getAction(hash utils.Hash) *types.Action {
	d.Lock()
	defer d.Unlock()
	return d.all[hash]
}

func (d *deferredList) addAction(a *types.Action) {
	d.RLock()
	defer d.RUnlock()
	d.all[a.Hash()] = a
}

func (d *deferredList) countAction() int {
	d.RLock()
	defer d.RUnlock()
	return len(d.all)
}

func (d *deferredList) deleteAction(hash utils.Hash) {
	d.Lock()
	defer d.Unlock()
	delete(d.all, hash)
}
