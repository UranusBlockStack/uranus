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
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/p2p/discover"
)

const (
	keepAliveInterval = 15 * time.Second
)

// message code
const (
	handshakeMsg = iota
	pingMsg
	pongMsg
	quitMsg
)

// Peer represent a connected remote node
type Peer struct {
	rw         *conn
	running    map[string]*protoRW //protocols
	runningErr chan error
	quit       chan string
	closed     chan struct{}
	wg         sync.WaitGroup
}

// ID return the node id
func (p *Peer) ID() discover.NodeID {
	return p.rw.id
}

// Name return the node name
func (p *Peer) Name() string {
	return p.String()
}

func (p *Peer) String() string {
	return fmt.Sprintf("Peer %x(%v)", p.rw.id[:8], p.RemoteAddr())
}

// Protocols return supported subprotocols of the remote peer.
func (p *Peer) Protocols() []*ProtocolKey {
	return p.rw.protocols
}

// RemoteAddr returns the remote address of connection.
func (p *Peer) RemoteAddr() net.Addr {
	return p.rw.fd.RemoteAddr()
}

// LocalAddr returns the local address of connection.
func (p *Peer) LocalAddr() net.Addr {
	return p.rw.fd.LocalAddr()
}

// Disconnect terminates the peer connection with the given reason.
func (p *Peer) Disconnect(reason string) {
	select {
	case p.quit <- reason:
	case <-p.closed:
	}
}

// NewPeer returns a peer
func NewPeer(conn *conn, protocols []*Protocol) *Peer {
	running := make(map[string]*protoRW)
	for _, protocolKey := range conn.protocols {
		for _, protocol := range protocols {
			if protocol.Name == protocolKey.Name && protocol.Version == protocolKey.Version {
				running[protocol.Name] = &protoRW{
					Protocol: *protocol,
					in:       make(chan *Message),
					w:        conn,
				}
			}
		}
	}
	p := &Peer{
		rw:         conn,
		running:    running,
		runningErr: make(chan error, len(running)+1),
		quit:       make(chan string),
		closed:     make(chan struct{}),
	}
	return p
}

func (p *Peer) pingloop() {
	pingTimer := time.NewTimer(keepAliveInterval)
	defer p.wg.Done()
	defer pingTimer.Stop()
	for {
		select {
		case <-pingTimer.C:
			if err := SendMessage(p.rw, pingMsg, nil); err != nil {
				p.runningErr <- err
				return
			}
			pingTimer.Reset(keepAliveInterval)
		case <-p.closed:
			return
		}
	}
}

func (p *Peer) loop() {
	defer p.wg.Done()
	for {
		msg, err := p.rw.ReadMsg()
		if opErr, ok := err.(*net.OpError); ok && (opErr.Timeout() || opErr.Temporary()) {
			continue
		}
		if err != nil {
			p.runningErr <- err
			return
		}
		msg.ReceivedAt = time.Now()
		if err = p.handle(msg); err != nil {
			p.runningErr <- err
			return
		}
	}
}

func (p *Peer) startProtocols() {
	wstart := make(chan struct{}, 1)
	wstart <- struct{}{}
	for _, proto := range p.running {
		proto := proto
		proto.closed = p.closed
		proto.wstart = wstart
		proto.werr = p.runningErr
		go func() {
			defer p.wg.Done()
			err := proto.Run(p, proto)
			p.runningErr <- err
		}()
	}
}

func (p *Peer) run() (err error) {
	p.wg.Add(2)
	go p.pingloop()
	go p.loop()
	p.wg.Add(len(p.running))
	p.startProtocols()

running:
	for {
		select {
		case err = <-p.runningErr:
			log.Warnf("peer %v close --- %v", p.ID().String(), err)
			break running
		case str := <-p.quit:
			log.Warnf("peer %v close --- %v", p.ID().String(), err)
			err = errors.New(str)
			break running
		}
	}

	close(p.closed)
	p.rw.close(err)
	p.wg.Wait()
	return err
}

func (p *Peer) handle(msg *Message) error {
	switch {
	case msg.Code == pingMsg:
		if err := SendMessage(p.rw, pongMsg, nil); err != nil {
			return err
		}
	case msg.Code == pongMsg:
	case msg.Code == quitMsg:
		var reason string
		msg.EncodePayload(&reason)
		return errors.New(reason)
	default:
		proto, err := p.getProto(msg.Code)
		if err != nil {
			return fmt.Errorf("msg code out of range: %v", msg.Code)
		}
		select {
		case proto.in <- msg:
			return nil
		case <-p.closed:
			return io.EOF
		}
	}
	return nil
}

func (p *Peer) getProto(code uint64) (*protoRW, error) {
	for _, proto := range p.running {
		if code >= proto.Offset && code < proto.Offset+proto.Size {
			return proto, nil
		}
	}
	return nil, fmt.Errorf("invalid message code")
}

func (p *Peer) PeerInfo() *PeerInfo {
	info := &PeerInfo{
		ID:        p.ID().String(),
		Name:      p.Name(),
		Protocols: make(map[string]interface{}),
	}
	info.Network.LocalAddress = p.LocalAddr().String()
	info.Network.RemoteAddress = p.RemoteAddr().String()

	for _, proto := range p.running {
		protoInfo := interface{}("unknown")
		if query := proto.Protocol.PeerInfo; query != nil {
			if metadata := query(p.ID()); metadata != nil {
				protoInfo = metadata
			} else {
				protoInfo = "handshake"
			}
		}
		info.Protocols[proto.Name] = protoInfo
	}
	return info
}

type protoRW struct {
	Protocol
	in     chan *Message
	w      MsgReadWriter
	closed <-chan struct{}
	wstart chan struct{}
	werr   chan error
}

func (rw *protoRW) WriteMsg(msg *Message) (err error) {
	select {
	case <-rw.wstart:
		err = rw.w.WriteMsg(msg)
		if err == nil {
			rw.wstart <- struct{}{}
		} else {
			rw.werr <- err
		}
	case <-rw.closed:
		err = fmt.Errorf("shutting down")
	}
	return err
}

func (rw *protoRW) ReadMsg() (*Message, error) {
	select {
	case msg := <-rw.in:
		return msg, nil
	case <-rw.closed:
		return nil, io.EOF
	}
}
