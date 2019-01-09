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
	"github.com/stretchr/testify/assert"
)

func TestIter(t *testing.T) {
	trie, _ := New(utils.Hash{}, NewDatabase(db.NewMemDatabase()))
	vals := []struct{ k, v string }{
		{"hello", "world"},
		{"key", "value"},
		{"uranus", "coin"},
		{"123456", "7890"},
		{"iterator key", "iterator key"},
	}
	all := make(map[string]string)
	for _, val := range vals {
		all[val.k] = val.v
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Commit(nil)

	found := make(map[string]string)
	it := NewIterator(trie.NodeIterator(nil))

	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range all {
		it.Next()
		assert.Equal(t, v, found[k])
	}
}

func TestPrefixIterator(t *testing.T) {
	trie, _ := New(utils.Hash{}, NewDatabase(db.NewMemDatabase()))
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"dog", "puppy"},
		{"somethingveryoddindeedthis is", "myothernodedata"},
		{"drive", "car"},
		{"dollar", "cny"},
		{"dxracer", "chair"},
	}
	all := make(map[string]string)
	for _, val := range vals {
		all[val.k] = val.v
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Commit(nil)

	expect := map[string]string{
		"doge":   "coin",
		"dog":    "puppy",
		"dollar": "cny",
		"do":     "verb",
	}

	found := make(map[string]string)
	it := NewIterator(trie.PrefixIterator([]byte("do")))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range found {
		if expect[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, v, expect[k])
		}
	}
	for k, v := range expect {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, found[k], v)
		}
	}

	expect = map[string]string{
		"doge": "coin",
		"dog":  "puppy",
	}
	found = make(map[string]string)
	it = NewIterator(trie.PrefixIterator([]byte("dog")))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range found {
		if expect[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, v, expect[k])
		}
	}
	for k, v := range expect {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, found[k], v)
		}
	}

	found = make(map[string]string)
	it = NewIterator(trie.PrefixIterator([]byte("test")))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}
	if len(found) > 0 {
		t.Errorf("iterator value count mismatch: got %v want %v", len(found), 0)
	}

	expect = map[string]string{
		"do":     "verb",
		"ether":  "wookiedoo",
		"horse":  "stallion",
		"shaman": "horse",
		"doge":   "coin",
		"dog":    "puppy",
		"somethingveryoddindeedthis is": "myothernodedata",
		"drive":   "car",
		"dollar":  "cny",
		"dxracer": "chair",
	}
	found = make(map[string]string)
	it = NewIterator(trie.PrefixIterator(nil))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}
	for k, v := range found {
		if expect[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, v, expect[k])
		}
	}
	for k, v := range expect {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", k, found[k], v)
		}
	}
}
