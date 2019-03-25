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

package mtp

import (
	"fmt"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
)

// TrieWarp wraps a trie with key hashing.
type TrieWarp struct {
	trie             Trie
	hashKeyBuf       [utils.HashLength]byte
	secKeyCache      map[string][]byte
	secKeyCacheOwner *TrieWarp // Pointer to self, replace the key cache on mismatch
}

// NewTrieWarp creates a trie with an existing root node from a backing database and optional intermediate in-memory node pool.
func NewTrieWarp(root utils.Hash, db *Database, cachelimit uint16) (*TrieWarp, error) {
	if db == nil {
		panic("mtp.NewTrieWarp called without a database")
	}
	trie, err := New(root, db)
	if err != nil {
		return nil, err
	}
	trie.SetCacheLimit(cachelimit)
	return &TrieWarp{trie: *trie}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *TrieWarp) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
func (t *TrieWarp) TryGet(key []byte) ([]byte, error) {
	return t.trie.TryGet(t.hashKey(key))
}

// Update associates key with value in the trie.
func (t *TrieWarp) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie.
func (t *TrieWarp) TryUpdate(key, value []byte) error {
	hk := t.hashKey(key)
	err := t.trie.TryUpdate(hk, value)
	if err != nil {
		return err
	}
	t.getSecKeyCache()[string(hk)] = utils.CopyBytes(key)
	return nil
}

// Delete removes any existing value for key from the trie.
func (t *TrieWarp) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
func (t *TrieWarp) TryDelete(key []byte) error {
	hk := t.hashKey(key)
	delete(t.getSecKeyCache(), string(hk))
	return t.trie.TryDelete(hk)
}

// GetKey returns the sha3 preimage of a hashed key that was
func (t *TrieWarp) GetKey(shaKey []byte) []byte {
	if key, ok := t.getSecKeyCache()[string(shaKey)]; ok {
		return key
	}
	key, _ := t.trie.db.preimage(utils.BytesToHash(shaKey))
	return key
}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
func (t *TrieWarp) Commit(onleaf LeafCallback) (root utils.Hash, err error) {
	// Write all the pre-images to the actual disk database
	if len(t.getSecKeyCache()) > 0 {
		t.trie.db.lock.Lock()
		for hk, key := range t.secKeyCache {
			t.trie.db.insertPreimage(utils.BytesToHash([]byte(hk)), key)
		}
		t.trie.db.lock.Unlock()

		t.secKeyCache = make(map[string][]byte)
	}
	// Commit the trie to its intermediate node database
	return t.trie.Commit(onleaf)
}

// Hash returns the root hash of TrieWarp.
func (t *TrieWarp) Hash() utils.Hash {
	return t.trie.Hash()
}

// Root returns the root hash of TrieWarp.
// Deprecated: use Hash instead.
func (t *TrieWarp) Root() []byte {
	return t.trie.Root()
}

// Copy returns a copy of TrieWarp.
func (t *TrieWarp) Copy() *TrieWarp {
	cpy := *t
	return &cpy
}

func (t *TrieWarp) Prove(key []byte, fromLevel uint, proofDb db.Writer) error {
	return t.trie.Prove(key, fromLevel, proofDb)
}

// NodeIterator returns an iterator that returns nodes of the underlying trie.
func (t *TrieWarp) NodeIterator(start []byte) NodeIterator {
	return t.trie.NodeIterator(start)
}

// hashKey returns the hash of key as an ephemeral buffer.
func (t *TrieWarp) hashKey(key []byte) []byte {
	h := newHasher(0, 0, nil)
	h.sha.Reset()
	h.sha.Write(key)
	buf := h.sha.Sum(t.hashKeyBuf[:0])
	returnHasherToPool(h)
	return buf
}

// getSecKeyCache returns the current secure key cache, creating a new one if ownership changed.
func (t *TrieWarp) getSecKeyCache() map[string][]byte {
	if t != t.secKeyCacheOwner {
		t.secKeyCacheOwner = t
		t.secKeyCache = make(map[string][]byte)
	}
	return t.secKeyCache
}
