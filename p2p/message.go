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
	"fmt"
	"time"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/p2p/discover"
)

// Message defines a p2p message.
type Message struct {
	Code    uint64
	Payload []byte

	ReceivedAt time.Time
}

// MsgReader provides reading of messages.
type MsgReader interface {
	ReadMsg() (*Message, error)
}

// MsgWriter provides writing of messages.
type MsgWriter interface {
	WriteMsg(*Message) error
}

// MsgReadWriter provides reading and writing of messages.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

func SendMessage(w MsgWriter, code uint64, data interface{}) error {
	msg := &Message{
		Code: code,
	}
	if err := msg.EncodePayload(data); err != nil {
		return err
	}
	return w.WriteMsg(msg)
}

// EncodePayload set the RLP content of a message
func (msg *Message) EncodePayload(val interface{}) error {
	payload, err := rlp.EncodeToBytes(val)
	if err != nil {
		return fmt.Errorf("message rlp %#v error --- %s", val, err)
	}
	msg.Payload = payload
	return nil
}

// DecodePayload parses the RLP content of a message
func (msg *Message) DecodePayload(val interface{}) error {
	return rlp.DecodeBytes(msg.Payload, val)
}

// ProtocolKey defines the structure of a protocol id.
type ProtocolKey struct {
	Name    string
	Version uint
}

// Protocol defined a P2P subprotocol.
type Protocol struct {
	Name string

	Version uint

	Offset uint64

	Size uint64

	Run func(peer *Peer, rw MsgReadWriter) error

	NodeInfo func() interface{}

	PeerInfo func(id discover.NodeID) interface{}
}

func (p *Protocol) Key() *ProtocolKey {
	return &ProtocolKey{
		Name:    p.Name,
		Version: p.Version,
	}
}

// ProtoHandshake defines the protocol handshake.
type ProtoHandshake struct {
	Version    uint64
	Name       string
	Protocols  []*ProtocolKey
	ListenPort uint64
	ID         discover.NodeID
}

// ProtocolKeys
type ProtocolKeys []*ProtocolKey

func (ps ProtocolKeys) Len() int      { return len(ps) }
func (ps ProtocolKeys) Swap(i, j int) { ps[i], ps[j] = ps[j], ps[i] }
func (ps ProtocolKeys) Less(i, j int) bool {
	return ps[i].Name < ps[j].Name || (ps[i].Name == ps[j].Name && ps[i].Version < ps[j].Version)
}
