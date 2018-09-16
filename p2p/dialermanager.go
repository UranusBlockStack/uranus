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

package p2p

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/p2p/discover"
)

// Dialer is used to connect to node in the network
type Dialer interface {
	Dial(*discover.Node) (net.Conn, error)
}

// TCPDialer implements the Dialer interface
type TCPDialer struct {
	*net.Dialer
}

// Dial create TCP connections to node in the network
func (t *TCPDialer) Dial(dest *discover.Node) (net.Conn, error) {
	addr := &net.TCPAddr{IP: dest.IP, Port: int(dest.TCP)}
	return t.Dialer.Dial("tcp", addr.String())
}

// Task is used to done for discover
type Task interface {
	Do(*Server) error
}

// DialerTask is generated for node to dial.
type DialerTask struct {
	dest *discover.Node
}

// DiscoverTask runs discovery operations for lookup.
type DiscoverTask struct {
	results []*discover.Node
}

// DialerManager schedules dials and discovery lookups.
type DialerManager struct {
	dialerLimit int                                // max dialers
	ntab        *discover.Table                    // discover nodes
	bootnodes   map[discover.NodeID]*discover.Node // nodes default or static

	dialing map[discover.NodeID]*discover.Node
	lookup  map[discover.NodeID]*discover.Node
	//TODO
	dialingExp map[discover.NodeID]time.Time
}

// NewDialerManager
func NewDialerManager(maxdialers int, bootnodes []*discover.Node, ntab *discover.Table) *DialerManager {
	dm := &DialerManager{
		dialerLimit: maxdialers,
		ntab:        ntab,
		bootnodes:   make(map[discover.NodeID]*discover.Node),
		dialing:     make(map[discover.NodeID]*discover.Node),
		lookup:      make(map[discover.NodeID]*discover.Node),
		dialingExp:  make(map[discover.NodeID]time.Time),
	}
	for _, node := range bootnodes {
		dm.bootnodes[node.ID] = node
	}
	return dm
}

func (dm *DialerManager) canDial(n *discover.Node, peers map[discover.NodeID]*Peer) error {
	_, ok := dm.dialing[n.ID]
	switch {
	case ok:
		return errors.New("dialed")
	case peers[n.ID] != nil:
		return errors.New("connected")
	case dm.ntab != nil && n.ID == dm.ntab.Self().ID:
		return errors.New("self")
	}
	return nil
}

func (dm *DialerManager) NewTasks(runningTasks int, peers map[discover.NodeID]*Peer) (tasks []Task) {
	//bootnodes
	for _, node := range dm.bootnodes {
		if err := dm.canDial(node, peers); err != nil {
			continue
		}
		task := &DialerTask{dest: node}
		dm.dialing[node.ID] = node
		tasks = append(tasks, task)
	}

	needDialers := dm.dialerLimit
	for _, p := range peers {
		if _, ok := dm.bootnodes[p.rw.id]; ok {
			continue
		}
		needDialers--
	}
	for id := range dm.dialing {
		if _, ok := dm.bootnodes[id]; ok {
			continue
		}
		needDialers--
	}

	if needDialers <= 0 {
		return
	}

	// Use random nodes from the table for half of the necessary need dials.
	randomCandidates := needDialers / 2
	if randomCandidates > 0 {
		randomNodes := make([]*discover.Node, randomCandidates)
		n := dm.ntab.ReadRandomNodes(randomNodes)
		for index, node := range randomNodes {
			if index >= n {
				break
			}
			if index+1 >= randomCandidates {
				break
			}
			if needDialers <= 0 {
				return
			}
			if err := dm.canDial(node, peers); err != nil {
				continue
			}
			task := &DialerTask{dest: node}
			dm.dialing[node.ID] = node
			tasks = append(tasks, task)
			needDialers--
		}
	}

	if len(dm.lookup) < needDialers {
		tasks = append(tasks, &DiscoverTask{})
	}

	for _, node := range dm.lookup {
		if needDialers <= 0 {
			return
		}
		if err := dm.canDial(node, peers); err != nil {
			continue
		}
		task := &DialerTask{dest: node}
		dm.dialing[node.ID] = node
		tasks = append(tasks, task)
		needDialers--
	}
	return
}

func (dm *DialerManager) Done(t Task, now time.Time) {
	switch t := t.(type) {
	case *DialerTask:
		delete(dm.dialing, t.dest.ID)
		delete(dm.lookup, t.dest.ID)
	case *DiscoverTask:
		for _, node := range t.results {
			dm.lookup[node.ID] = node
		}
	}
}

func (dm *DialerManager) AddStatic(n *discover.Node) {
	if _, ok := dm.bootnodes[n.ID]; ok {
		return
	}
	dm.bootnodes[n.ID] = n
}

func (dm *DialerManager) RemoveStatic(n *discover.Node) {
	delete(dm.bootnodes, n.ID)
}

func (t *DialerTask) Do(srv *Server) error {
	if t.dest.Incomplete() {
		if srv.ntab == nil {
			return fmt.Errorf("Resolving node %s failed ---- discovery is disabled", t.dest.ID)
		}

		resolved := srv.ntab.Resolve(t.dest.ID)
		if resolved == nil {
			return fmt.Errorf("Resolving node %s failed", t.dest.ID)
		}
		t.dest = resolved
		log.Debugf("Resolved node %s, addr %#v", t.dest.ID, &net.TCPAddr{IP: t.dest.IP, Port: int(t.dest.TCP)})
	}

	fd, err := srv.dialer.Dial(t.dest)
	if err != nil {
		return err
	}
	return srv.SetupConn(fd, t.dest)
}

func (t *DiscoverTask) Do(srv *Server) error {
	next := srv.lastLookup.Add(10 * time.Second)
	if now := time.Now(); now.Before(next) {
		time.Sleep(next.Sub(now))
	}
	var target discover.NodeID
	rand.Read(target[:])
	t.results = srv.ntab.Lookup(target)
	srv.lastLookup = time.Now()
	return nil
}
