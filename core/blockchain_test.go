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

package core

import (
	"testing"

	"github.com/UranusBlockStack/uranus/consensus/pow/cpuminer"
	"github.com/stretchr/testify/assert"
)

func TestTheLastBlock(t *testing.T) {
	cpum := cpuminer.NewCpuMiner()
	_, blockchain, err := newLegitimate(cpum, 0)
	if err != nil {
		t.Fatalf("failed to create pristine chain: %v", err)
	}
	defer blockchain.Stop()

	blocks := makeBlocks(blockchain.CurrentBlock(), 1, cpum, blockchain.GetDB(), 0)

	if _, err := blockchain.InsertChain(blocks); err != nil {
		t.Fatalf("Failed to insert block: %v", err)
	}

	assert.Equal(t, blocks[len(blocks)-1].Hash().Hex(), blockchain.GetHeadBlockHash().Hex())
}
