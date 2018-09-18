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
	"github.com/UranusBlockStack/uranus/common/utils"
)

// MinerAPI exposes methods for the RPC interface
type MinerAPI struct {
	b Backend
}

// NewMinerAPI creates a new API definition for admin methods of the node itself.
func NewMinerAPI(b Backend) *MinerAPI {
	return &MinerAPI{b}
}

// Start the miner
func (api *MinerAPI) Start(threads *int32, reply *bool) error {
	nthread := int32(1)
	if threads != nil {
		nthread = *threads
	}
	err := api.b.Start(nthread)
	*reply = err == nil
	return err
}

// Stop the miner
func (api *MinerAPI) Stop(ignore string, reply *bool) error {
	err := api.b.Stop()
	*reply = err == nil
	return err
}

// SetCoinbase the coinbase of the miner
func (api *MinerAPI) SetCoinbase(coinbase utils.Address, reply *bool) error {
	err := api.b.SetCoinbase(coinbase)
	*reply = err == nil
	return err
}
