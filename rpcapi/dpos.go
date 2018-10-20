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

package rpcapi

import (
	"context"
	"fmt"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

// DposAPI exposes methods for the RPC interface
type DposAPI struct {
	b Backend
}

// NewDposAPI creates a new API definition for dpos methods of the node itself.
func NewDposAPI(b Backend) *DposAPI {
	return &DposAPI{b}
}

// GetValidators retrieves the list of the validators at specified block
func (api *DposAPI) GetValidators(number *BlockHeight, reply *[]utils.Address) error {
	var block *types.Block
	if number == nil || *number == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *number)
	}
	if block == nil {
		return fmt.Errorf("not found block %v", *number)
	}
	header := block.BlockHeader()

	// TODO
	_ = header

	return nil
}

// GetVoters retrieves the list of the voters at specified block
func (api *DposAPI) GetVoters(number *BlockHeight, reply *[]utils.Address) error {
	var block *types.Block
	if number == nil || *number == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *number)
	}
	if block == nil {
		return fmt.Errorf("not found block %v", *number)
	}
	header := block.BlockHeader()

	// TODO
	_ = header

	return nil
}

// GetDelegators retrieves the list of the delegators at specified block
func (api *DposAPI) GetDelegators(number *BlockHeight, reply *[]utils.Address) error {
	var block *types.Block
	if number == nil || *number == LatestBlockHeight {
		block = api.b.CurrentBlock()
	} else {
		block, _ = api.b.BlockByHeight(context.Background(), *number)
	}
	if block == nil {
		return fmt.Errorf("not found block %v", *number)
	}
	header := block.BlockHeader()

	// TODO
	_ = header

	return nil
}
