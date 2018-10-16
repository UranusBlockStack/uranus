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
	"testing"

	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/utils"
)

func TestEmptySync(t *testing.T) {
	// test utils.Hash{}
	testdb := NewDatabase(db.NewMemDatabase())
	empty1, _ := New(utils.Hash{}, testdb)
	if req := NewSync(empty1.Hash(), db.NewMemDatabase(), nil).Missing(1); len(req) != 0 {
		t.Errorf(" content requested for empty trie: %v", req)
	}

	// test emptyRoot
	testdb = NewDatabase(db.NewMemDatabase())
	empty2, _ := New(emptyRoot, testdb)
	if req := NewSync(empty2.Hash(), db.NewMemDatabase(), nil).Missing(1); len(req) != 0 {
		t.Errorf("tcontent requested for empty trie: %v", req)
	}
}

func TestIterativeSyncIndividual(t *testing.T) { testIterativeSync(t, 1) }
func TestIterativeSyncBatched(t *testing.T)    { testIterativeSync(t, 100) }

func testIterativeSync(t *testing.T, batch int) {
	srcDb, srcTrie, srcData := makeTestTrie()

	diskdb := db.NewMemDatabase()
	trieMemdb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	queue := append([]utils.Hash{}, sched.Missing(batch)...)
	for len(queue) > 0 {
		results := make([]SyncResult, len(queue))
		for i, hash := range queue {
			data, err := srcDb.Node(hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results[i] = SyncResult{hash, data}
		}
		if _, index, err := sched.Process(results); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		if index, err := sched.Commit(diskdb); err != nil {
			t.Fatalf("failed to commit data #%d: %v", index, err)
		}
		queue = append(queue[:0], sched.Missing(batch)...)
	}
	checkTrieContents(t, trieMemdb, srcTrie.Root(), srcData)
}

func TestIterativeRandomSyncIndividual(t *testing.T) { testIterativeRandomSync(t, 1) }
func TestIterativeRandomSyncBatched(t *testing.T)    { testIterativeRandomSync(t, 100) }

func testIterativeRandomSync(t *testing.T, batch int) {
	srcDb, srcTrie, srcData := makeTestTrie()
	diskdb := db.NewMemDatabase()
	trieMemdb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	queue := make(map[utils.Hash]struct{})
	for _, hash := range sched.Missing(batch) {
		queue[hash] = struct{}{}
	}
	for len(queue) > 0 {
		results := make([]SyncResult, 0, len(queue))
		for hash := range queue {
			data, err := srcDb.Node(hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results = append(results, SyncResult{hash, data})
		}
		if _, index, err := sched.Process(results); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		if index, err := sched.Commit(diskdb); err != nil {
			t.Fatalf("failed to commit data #%d: %v", index, err)
		}
		queue = make(map[utils.Hash]struct{})
		for _, hash := range sched.Missing(batch) {
			queue[hash] = struct{}{}
		}
	}
	checkTrieContents(t, trieMemdb, srcTrie.Root(), srcData)
}

func TestIterativeDelayedSync(t *testing.T) {
	srcDb, srcTrie, srcData := makeTestTrie()

	diskdb := db.NewMemDatabase()
	trieMemdb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	queue := append([]utils.Hash{}, sched.Missing(10000)...)
	for len(queue) > 0 {
		results := make([]SyncResult, len(queue)/2+1)
		for i, hash := range queue[:len(results)] {
			data, err := srcDb.Node(hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results[i] = SyncResult{hash, data}
		}
		if _, index, err := sched.Process(results); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		if index, err := sched.Commit(diskdb); err != nil {
			t.Fatalf("failed to commit data #%d: %v", index, err)
		}
		queue = append(queue[len(results):], sched.Missing(10000)...)
	}
	checkTrieContents(t, trieMemdb, srcTrie.Root(), srcData)
}

func TestIterativeRandomDelayedSync(t *testing.T) {
	srcDb, srcTrie, srcData := makeTestTrie()

	diskdb := db.NewMemDatabase()
	trieMemdb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	queue := make(map[utils.Hash]struct{})
	for _, hash := range sched.Missing(10000) {
		queue[hash] = struct{}{}
	}
	for len(queue) > 0 {
		results := make([]SyncResult, 0, len(queue)/2+1)
		for hash := range queue {
			data, err := srcDb.Node(hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results = append(results, SyncResult{hash, data})

			if len(results) >= cap(results) {
				break
			}
		}
		if _, index, err := sched.Process(results); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		if index, err := sched.Commit(diskdb); err != nil {
			t.Fatalf("failed to commit data #%d: %v", index, err)
		}
		for _, result := range results {
			delete(queue, result.Hash)
		}
		for _, hash := range sched.Missing(10000) {
			queue[hash] = struct{}{}
		}
	}
	checkTrieContents(t, trieMemdb, srcTrie.Root(), srcData)
}

func TestDuplicateAvoidanceSync(t *testing.T) {
	srcDb, srcTrie, srcData := makeTestTrie()
	diskdb := db.NewMemDatabase()
	trieMemdb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	queue := append([]utils.Hash{}, sched.Missing(0)...)
	requested := make(map[utils.Hash]struct{})

	for len(queue) > 0 {
		results := make([]SyncResult, len(queue))
		for i, hash := range queue {
			data, err := srcDb.Node(hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			if _, ok := requested[hash]; ok {
				t.Errorf("hash %x already requested once", hash)
			}
			requested[hash] = struct{}{}

			results[i] = SyncResult{hash, data}
		}
		if _, index, err := sched.Process(results); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		if index, err := sched.Commit(diskdb); err != nil {
			t.Fatalf("failed to commit data #%d: %v", index, err)
		}
		queue = append(queue[:0], sched.Missing(0)...)
	}
	checkTrieContents(t, trieMemdb, srcTrie.Root(), srcData)
}

func TestIncompleteSync(t *testing.T) {
	// Create a random trie to copy
	srcDb, srcTrie, _ := makeTestTrie()

	// Create a destination trie and sync with the scheduler
	diskdb := db.NewMemDatabase()
	triedb := NewDatabase(diskdb)
	sched := NewSync(srcTrie.Hash(), diskdb, nil)

	added := []utils.Hash{}
	queue := append([]utils.Hash{}, sched.Missing(1)...)
	for len(queue) > 0 {
		// Fetch a batch of trie nodes
		results := make([]SyncResult, len(queue))
		for i, hash := range queue {
			data, err := srcDb.Node(hash)
			if err != nil {
				t.Fatalf("failed to retrieve node data for %x: %v", hash, err)
			}
			results[i] = SyncResult{hash, data}
		}
		// Process each of the trie nodes
		if _, index, err := sched.Process(results); err != nil {
			t.Fatalf("failed to process result #%d: %v", index, err)
		}
		if index, err := sched.Commit(diskdb); err != nil {
			t.Fatalf("failed to commit data #%d: %v", index, err)
		}
		for _, result := range results {
			added = append(added, result.Hash)
		}
		// Check that all known sub-tries in the synced trie are complete
		for _, root := range added {
			if err := checkTrieConsistency(triedb, root); err != nil {
				t.Fatalf("trie inconsistent: %v", err)
			}
		}
		// Fetch the next batch to retrieve
		queue = append(queue[:0], sched.Missing(1)...)
	}
	// Sanity check that removing any node from the database is detected
	for _, node := range added[1:] {
		key := node.Bytes()
		value, _ := diskdb.Get(key)

		diskdb.Delete(key)
		if err := checkTrieConsistency(triedb, added[0]); err == nil {
			t.Fatalf("trie inconsistency not caught, missing: %x", key)
		}
		diskdb.Put(key, value)
	}
}

func makeTestTrie() (*Database, *Trie, map[string][]byte) {
	trieMemdb := NewDatabase(db.NewMemDatabase())
	trie, _ := New(utils.Hash{}, trieMemdb)

	// Fill it with some arbitrary data
	content := make(map[string][]byte)
	for i := byte(0); i < 255; i++ {
		// Map the same data under multiple ks
		k, v := utils.LeftPadBytes([]byte{1, i}, 32), []byte{i}
		content[string(k)] = v
		trie.Update(k, v)

		k, v = utils.LeftPadBytes([]byte{2, i}, 32), []byte{i}
		content[string(k)] = v
		trie.Update(k, v)

		// Add some other data to inflate the trie
		for j := byte(3); j < 13; j++ {
			k, v = utils.LeftPadBytes([]byte{j, i}, 32), []byte{j, i}
			content[string(k)] = v
			trie.Update(k, v)
		}
	}
	trie.Commit(nil)
	return trieMemdb, trie, content
}

func checkTrieContents(t *testing.T, db *Database, root []byte, content map[string][]byte) {
	trie, err := New(utils.BytesToHash(root), db)
	if err != nil {
		t.Fatalf("failed to create trie at %x: %v", root, err)
	}
	if err := checkTrieConsistency(db, utils.BytesToHash(root)); err != nil {
		t.Fatalf("inconsistent trie at %x: %v", root, err)
	}
	for k, v := range content {
		have := trie.Get([]byte(k))
		utils.AssertEquals(t, have, v)
	}
}

func checkTrieConsistency(db *Database, root utils.Hash) error {
	trie, err := New(root, db)
	if err != nil {
		return nil
	}
	it := trie.NodeIterator(nil)
	for it.Next(true) {
	}
	return it.Error()
}
