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

func TestInsert(t *testing.T) {
	trie, _ := New(utils.Hash{}, NewDatabase(db.NewMemDatabase()))

	trie.Update([]byte("hello"), []byte("world"))
	trie.Update([]byte("key"), []byte("value"))
	trie.Update([]byte("123"), []byte("456"))

	exp := utils.HexToHash("0xf4308bfc802bcf37deca66392ce128d03455aff790312e9157801af6dc6e4ded")
	root := trie.Hash()
	utils.AssertEquals(t, root, exp)

	trie, _ = New(utils.Hash{}, NewDatabase(db.NewMemDatabase()))

	trie.Update([]byte("A"), []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	exp = utils.HexToHash("d23786fb4a010da3ce639d66d5e904a11dbc02746d1ce25029e53290cabf28ab")
	root, err := trie.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	utils.AssertEquals(t, root, exp)
}

func TestGet(t *testing.T) {
	trie, _ := New(utils.Hash{}, NewDatabase(db.NewMemDatabase()))
	trie.Update([]byte("hello"), []byte("world"))
	trie.Update([]byte("key"), []byte("value"))
	trie.Update([]byte("123"), []byte("456"))

	for i := 0; i < 2; i++ {
		res := trie.Get([]byte("hello"))
		utils.AssertEquals(t, res, []byte("world"))

		unknown := trie.Get([]byte("unknown"))
		if unknown != nil {
			t.Errorf("expected nil got %x", unknown)
		}
		if i == 1 {
			return
		}
		trie.Commit(nil)
	}
}

func TestDelete(t *testing.T) {
	trie, _ := New(utils.Hash{}, NewDatabase(db.NewMemDatabase()))

	vals := []struct{ k, v string }{
		{"hello", "world"},
		{"key", "value"},
		{"uranus", "coin"},
		{"123456", "7890"},
		{"iterator key", "iterator key"},
	}
	for _, val := range vals {
		if val.v != "" {
			trie.Update([]byte(val.k), []byte(val.v))
		} else {
			trie.Delete([]byte(val.k))
		}
	}

	hash := trie.Hash()
	exp := utils.HexToHash("0x5eea7814396f5a29818fca46237889c0ae95ba6a8821594c1e8e5c83fdf20a9e")
	utils.AssertEquals(t, hash, exp)
}
