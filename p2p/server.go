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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/p2p/discover"
)

const (
	connReadTimeout       = 30 * time.Second
	connWriteTimeout      = 20 * time.Second
	defaultMaxPeers       = 100
	defaultMaxAcceptConns = 50
	defaultDialTimeout    = 15 * time.Second
)

var errServerExit = errors.New("server exit")

// Config server options.
type Config struct {
	PrivateKey     *ecdsa.PrivateKey
	MaxPeers       int `mapstructure:"p2p-maxpeers"`
	MaxAcceptConns int
	Name           string
	NetworkID      uint64
	BootNodeStrs   []string
	BootNodes      []*discover.Node
	ListenAddr     string `mapstructure:"p2p-listenaddr"`
	Protocols      []*Protocol
}

type peerOpFunc func(map[discover.NodeID]*Peer)

// Server manages all peer connections.
type Server struct {
	Config
	sync.RWMutex

	dialer Dialer

	wg      sync.WaitGroup
	running bool
	quit    chan struct{}

	ntab         *discover.Table
	listener     net.Listener
	ourHandshake *ProtoHandshake
	lastLookup   time.Time

	posthandshake chan *conn
	addpeer       chan *conn
	delpeer       chan *Peer
	peerOp        chan peerOpFunc
	peerOpDone    chan struct{}
	addnode       chan *discover.Node
	removenode    chan *discover.Node
}

func (srv *Server) Start() (err error) {
	srv.Lock()
	defer srv.Unlock()
	if srv.running {
		return errors.New("already running")
	}
	srv.running = true
	log.Infof("Starting P2P networking")

	if srv.PrivateKey == nil {
		return fmt.Errorf("privateKey is nil")
	}

	srv.quit = make(chan struct{})
	srv.posthandshake = make(chan *conn)
	srv.addpeer = make(chan *conn)
	srv.delpeer = make(chan *Peer)
	srv.peerOp = make(chan peerOpFunc)
	srv.peerOpDone = make(chan struct{})
	srv.addnode = make(chan *discover.Node)
	srv.removenode = make(chan *discover.Node)
	srv.dialer = &TCPDialer{&net.Dialer{Timeout: defaultDialTimeout}}

	var (
		conn     *net.UDPConn
		realaddr *net.UDPAddr
	)

	addr, err := net.ResolveUDPAddr("udp", srv.ListenAddr)
	if err != nil {
		return err
	}
	conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	realaddr = conn.LocalAddr().(*net.UDPAddr)

	// node table

	cfg := discover.Config{
		PrivateKey:   srv.PrivateKey,
		AnnounceAddr: realaddr,
		Bootnodes:    srv.BootNodes,
		Unhandled:    make(chan discover.ReadPacket, 100),
	}
	ntab, err := discover.ListenUDP(conn, cfg)
	if err != nil {
		return err
	}
	srv.ntab = ntab

	dialerTasks := NewDialerManager(srv.MaxPeers, srv.BootNodes, srv.ntab)

	srv.ourHandshake = &ProtoHandshake{Version: 0, Name: srv.Name, ID: discover.PubkeyID(&srv.PrivateKey.PublicKey)}
	for _, p := range srv.Protocols {
		srv.ourHandshake.Protocols = append(srv.ourHandshake.Protocols, p.Key())
	}
	if srv.ListenAddr != "" {
		listener, err := net.Listen("tcp", srv.ListenAddr)
		if err != nil {
			return err
		}
		laddr := listener.Addr().(*net.TCPAddr)
		srv.ListenAddr = laddr.String()
		srv.listener = listener
		srv.wg.Add(1)
		go srv.listenLoop()
	}

	srv.wg.Add(1)
	go srv.run(dialerTasks)
	srv.running = true
	return nil
}

func (srv *Server) Stop() {
	srv.Lock()
	defer srv.Unlock()
	if !srv.running {
		return
	}
	srv.running = false
	if srv.listener != nil {
		srv.listener.Close()
	}
	close(srv.quit)
	srv.wg.Wait()
}

func (srv *Server) Peers() []*Peer {
	var ps []*Peer
	select {
	case srv.peerOp <- func(peers map[discover.NodeID]*Peer) {
		for _, p := range peers {
			ps = append(ps, p)
		}
	}:
		<-srv.peerOpDone
	case <-srv.quit:
	}
	return ps
}

func (srv *Server) PeerCount() int {
	var count int
	select {
	case srv.peerOp <- func(ps map[discover.NodeID]*Peer) { count = len(ps) }:
		<-srv.peerOpDone
	case <-srv.quit:
	}
	return count
}

func (srv *Server) AddPeer(node *discover.Node) {
	select {
	case srv.addnode <- node:
	case <-srv.quit:
	}
}

func (srv *Server) RemovePeer(node *discover.Node) {
	select {
	case srv.removenode <- node:
	case <-srv.quit:
	}
}

func (srv *Server) Self() *discover.Node {
	srv.Lock()
	defer srv.Unlock()

	if !srv.running {
		return &discover.Node{IP: net.ParseIP("0.0.0.0")}
	}

	if srv.ntab == nil {
		if srv.listener == nil {
			return &discover.Node{IP: net.ParseIP("0.0.0.0"), ID: discover.PubkeyID(&srv.PrivateKey.PublicKey)}
		}
		addr := srv.listener.Addr().(*net.TCPAddr)
		return &discover.Node{
			ID:  discover.PubkeyID(&srv.PrivateKey.PublicKey),
			IP:  addr.IP,
			TCP: uint16(addr.Port),
		}
	}
	return srv.ntab.Self()
}

type dialer interface {
	NewTasks(running int, peers map[discover.NodeID]*Peer) []Task
	Done(Task, time.Time)
	AddStatic(n *discover.Node)
	RemoveStatic(n *discover.Node)
}

func (srv *Server) run(dialstate dialer) {
	defer srv.wg.Done()
	var (
		maxActiveDialTasks = 16
		peers              = make(map[discover.NodeID]*Peer)
		taskdone           = make(chan Task, maxActiveDialTasks)
		runningTasks       []Task
		queuedTasks        []Task
	)

	delTask := func(t Task) {
		for i := range runningTasks {
			if runningTasks[i] == t {
				runningTasks = append(runningTasks[:i], runningTasks[i+1:]...)
				break
			}
		}
	}
	startTasks := func(ts []Task) (rest []Task) {
		i := 0
		for ; len(runningTasks) < maxActiveDialTasks && i < len(ts); i++ {
			t := ts[i]
			go func() { t.Do(srv); taskdone <- t }()
			runningTasks = append(runningTasks, t)
		}
		return ts[i:]
	}
	scheduleTasks := func() {
		queuedTasks = append(queuedTasks[:0], startTasks(queuedTasks)...)
		if len(runningTasks) < maxActiveDialTasks {
			nt := dialstate.NewTasks(len(runningTasks)+len(queuedTasks), peers)
			queuedTasks = append(queuedTasks, startTasks(nt)...)
		}
	}

	go func() {
		ticker := time.NewTicker(time.Second * 30)
		for {
			select {
			case <-srv.quit:
				return
			case <-ticker.C:
				srv.peerOp <- func(peers map[discover.NodeID]*Peer) {
					ns := []string{}
					for _, p := range peers {
						ns = append(ns, p.Name())
					}
					log.Infof("Peers %v [%v]", len(peers), strings.Join(ns, ","))
				}
				<-srv.peerOpDone
			}
		}
	}()

running:
	for {
		scheduleTasks()
		select {
		case <-srv.quit:
			break running
		case op := <-srv.peerOp:
			op(peers)
			srv.peerOpDone <- struct{}{}
		case t := <-taskdone:
			dialstate.Done(t, time.Now())
			delTask(t)
		case c := <-srv.posthandshake:
			select {
			case c.cont <- srv.encHandshakeChecks(peers, c):
			case <-srv.quit:
				break running
			}
		case n := <-srv.addnode:
			log.Debugf("Adding static node %v", n.String())
			dialstate.AddStatic(n)
		case n := <-srv.removenode:
			log.Debugf("Removing static node %v", n.String())
			if p, ok := peers[n.ID]; ok {
				p.Disconnect("remove")
			}
			dialstate.RemoveStatic(n)
		case c := <-srv.addpeer:
			err := srv.protoHandshakeChecks(peers, c)
			if err == nil {
				p := NewPeer(c, srv.Protocols)
				go srv.runPeer(p)
				peers[c.id] = p
			}
			select {
			case c.cont <- err:
			case <-srv.quit:
				break running
			}
		case pd := <-srv.delpeer:
			delete(peers, pd.ID())
		}
	}

	if srv.ntab != nil {
		srv.ntab.Close()
	}

	for _, p := range peers {
		p.Disconnect("quit")
	}
	for len(peers) > 0 {
		p := <-srv.delpeer
		delete(peers, p.ID())
	}
}

func (srv *Server) protoHandshakeChecks(peers map[discover.NodeID]*Peer, c *conn) error {
	return srv.encHandshakeChecks(peers, c)
}

func (srv *Server) encHandshakeChecks(peers map[discover.NodeID]*Peer, c *conn) error {
	switch {
	case peers[c.id] != nil:
		return errors.New("already connected")
	case c.id == srv.Self().ID:
		return errors.New("connected to self")
	default:
		return nil
	}
}

type tempError interface {
	Temporary() bool
}

func (srv *Server) listenLoop() {
	defer srv.wg.Done()

	tokens := defaultMaxAcceptConns
	if srv.MaxAcceptConns > 0 {
		tokens = srv.MaxAcceptConns
	}
	slots := make(chan struct{}, tokens)
	for i := 0; i < tokens; i++ {
		slots <- struct{}{}
	}

	for {
		<-slots
		var (
			fd  net.Conn
			err error
		)
		for {
			fd, err = srv.listener.Accept()
			if tempErr, ok := err.(tempError); ok && tempErr.Temporary() {
				continue
			} else if err != nil {
				return
			}
			break
		}

		go func() {
			srv.SetupConn(fd, nil)
			slots <- struct{}{}
		}()
	}
}

func (srv *Server) SetupConn(fd net.Conn, dest *discover.Node) error {
	self := srv.Self()
	if self == nil {
		return errors.New("shutdown")
	}
	c := &conn{fd: fd, cont: make(chan error)}
	err := srv.setupConn(c, dest)
	if err != nil {
		c.close(err)
	}
	return err
}

func (srv *Server) setupConn(c *conn, dest *discover.Node) error {
	srv.Lock()
	running := srv.running
	srv.Unlock()
	if !running {
		return errServerExit
	}
	var err error
	if c.id, err = c.doEncHandshake(srv.PrivateKey, dest); err != nil {
		return err
	}

	if dest != nil && c.id != dest.ID {
		return fmt.Errorf("unexpected identity")
	}
	err = srv.checkpoint(c, srv.posthandshake)
	if err != nil {
		return err
	}
	// Run the protocol handshake
	phs, err := c.doProtoHandshake(srv.ourHandshake)
	if err != nil {
		return err
	}
	if phs.ID != c.id {
		return fmt.Errorf("unexpected identity")
	}
	c.protocols, c.name = phs.Protocols, phs.Name
	err = srv.checkpoint(c, srv.addpeer)
	if err != nil {
		return err
	}

	return nil
}

func (srv *Server) checkpoint(c *conn, stage chan<- *conn) error {
	select {
	case stage <- c:
	case <-srv.quit:
		return errServerExit
	}
	select {
	case err := <-c.cont:
		return err
	case <-srv.quit:
		return errServerExit
	}
}

func (srv *Server) runPeer(p *Peer) {
	p.run()
	srv.delpeer <- p
}

type NodeInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Enode string `json:"enode"`
	IP    string `json:"ip"`
	Ports struct {
		Discovery int `json:"discovery"`
		Listener  int `json:"listener"`
	} `json:"ports"`
	ListenAddr string                 `json:"listenAddr"`
	Protocols  map[string]interface{} `json:"protocols"`
}

func (srv *Server) NodeInfo() *NodeInfo {
	node := srv.Self()
	info := &NodeInfo{
		Name:       srv.Name,
		Enode:      node.String(),
		ID:         node.ID.String(),
		IP:         node.IP.String(),
		ListenAddr: srv.ListenAddr,
		Protocols:  make(map[string]interface{}),
	}
	info.Ports.Discovery = int(node.UDP)
	info.Ports.Listener = int(node.TCP)
	for _, proto := range srv.Protocols {
		if _, ok := info.Protocols[proto.Name]; !ok {
			nodeInfo := interface{}("unknown")
			if query := proto.NodeInfo; query != nil {
				nodeInfo = proto.NodeInfo()
			}
			info.Protocols[proto.Name] = nodeInfo
		}
	}
	return info
}

type PeerInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Caps    []string `json:"caps"`
	Network struct {
		LocalAddress  string `json:"localAddress"`
		RemoteAddress string `json:"remoteAddress"`
	} `json:"network"`
	Protocols map[string]interface{} `json:"protocols"`
}

func (srv *Server) PeersInfo() []*PeerInfo {
	infos := make([]*PeerInfo, 0, srv.PeerCount())
	for _, peer := range srv.Peers() {
		if peer != nil {
			infos = append(infos, peer.PeerInfo())
		}
	}

	return infos
}
