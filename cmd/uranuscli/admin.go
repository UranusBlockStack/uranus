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

package main

import (
	"github.com/UranusBlockStack/uranus/p2p"
	"github.com/spf13/cobra"
)

var listPeersCmd = &cobra.Command{
	Use:   "listPeers",
	Short: "List all connected peers.",
	Long:  `List all connected peers.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := []*p2p.PeerInfo{}
		ClientCall("Admin.Peers", nil, &result)
		printJSONList(result)
	},
}

var addPeerCmd = &cobra.Command{
	Use:   "addPeer <nodeurl>",
	Short: "Connecting to a remote node.",
	Long:  `Connecting to a remote node.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		ClientCall("Admin.AddPeer", args[0], &result)
		printJSON(result)
	},
}

var removePeerCmd = &cobra.Command{
	Use:   "removePeer <nodeurl>",
	Short: "Disconnects from a a remote node if the connection exists.",
	Long:  `Disconnects from a a remote node if the connection exists.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var result bool
		ClientCall("Admin.RemovePeer", args[0], &result)
		printJSON(result)
	},
}
var nodeInfoCmd = &cobra.Command{
	Use:   "nodeInfo ",
	Short: "retrieves all the information we know about the host node at the protocol granularity",
	Long:  `retrieves all the information we know about the host node at the protocol granularity`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		result := &p2p.NodeInfo{}
		ClientCall("Admin.NodeInfo", nil, &result)
		printJSON(result)
	},
}
