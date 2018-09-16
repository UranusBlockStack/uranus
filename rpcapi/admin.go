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

import "github.com/UranusBlockStack/uranus/p2p"

// AdminAPI exposes methods for the RPC interface
type AdminAPI struct {
	b Backend
}

// NewAdminAPI creates a new API definition for admin methods of the node itself.
func NewAdminAPI(b Backend) *AdminAPI {
	return &AdminAPI{b}
}

// AddPeer requests connecting to a remote node
func (api *AdminAPI) AddPeer(url string, reply *bool) error {
	err := api.b.AddPeer(url)
	*reply = err == nil
	return err
}

// RemovePeer disconnects from a a remote node if the connection exists
func (api *AdminAPI) RemovePeer(url string, reply *bool) error {
	err := api.b.RemovePeer(url)
	*reply = err == nil
	return err
}

// Peers retrieves all the information we know about each individual peer at the protocol granularity.
func (api *AdminAPI) Peers(ignore string, reply *[]*p2p.PeerInfo) (err error) {
	*reply, err = api.b.Peers()
	return err
}

// NodeInfo retrieves all the information we know about the host node at the protocol granularity.
func (api *AdminAPI) NodeInfo(ignore string, reply **p2p.NodeInfo) (err error) {
	*reply, err = api.b.NodeInfo()
	return err
}
