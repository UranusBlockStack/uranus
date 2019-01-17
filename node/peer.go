// Copyright 2018 The uranus Authors
// This file is part of the uranus library.
//
// The uranus library is free software: you can redistribute it and/or modify
// it under the quits of the GNU Lesser General Public License as published by
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

package node

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/node/protocols"
	"github.com/UranusBlockStack/uranus/p2p"
	"gopkg.in/fatih/set.v0"
)

const (
	maxExistedTxs    = 32768
	maxExistedBlocks = 1024
	maxQueuedTxs     = 128
	maxQueuedProps   = 4
	maxQueuedAnns    = 4

	handshakeTimeout = 5 * time.Second
)

type propEvent struct {
	block *types.Block
	td    *big.Int
}

type peer struct {
	id      string
	version int
	*p2p.Peer
	rw p2p.MsgReadWriter

	head utils.Hash
	td   *big.Int
	lock sync.RWMutex

	existedTxs       *set.Set
	existedBlocks    *set.Set
	existedConfirmed *set.Set
	quit             chan struct{}
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:             p,
		version:          version,
		rw:               rw,
		id:               fmt.Sprintf("%x", p.ID().Bytes()[:8]),
		existedTxs:       set.New(),
		existedBlocks:    set.New(),
		existedConfirmed: set.New(),
		quit:             make(chan struct{}),
	}
}

func (p *peer) close() {
	close(p.quit)
}

type PeerInfo struct {
	Version    int      `json:"version"`
	Difficulty *big.Int `json:"difficulty"`
	Head       string   `json:"head"`
}

func (p *peer) Info() *PeerInfo {
	hash, td := p.Head()

	return &PeerInfo{
		Version:    p.version,
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

func (p *peer) Head() (hash utils.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

func (p *peer) SetHead(hash utils.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

func (p *peer) MarkBlock(hash utils.Hash) {
	for p.existedBlocks.Size() >= maxExistedBlocks {
		p.existedBlocks.Pop()
	}
	p.existedBlocks.Add(hash)
}

func (p *peer) MarkTransaction(hash utils.Hash) {
	for p.existedTxs.Size() >= maxExistedTxs {
		p.existedTxs.Pop()
	}
	p.existedTxs.Add(hash)
}

func (p *peer) MarkConfirmed(hash utils.Hash) {
	for p.existedConfirmed.Size() >= maxExistedTxs {
		p.existedConfirmed.Pop()
	}
	p.existedConfirmed.Add(hash)
}

func (p *peer) SendTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		p.existedTxs.Add(tx.Hash())
	}
	return p2p.SendMessage(p.rw, TxMsg, txs)
}

func (p *peer) SendBlockHashes(hashes []utils.Hash) error {
	return p2p.SendMessage(p.rw, BlockHashesMsg, hashes)
}

func (p *peer) SendBlocks(blocks []*types.Block) error {
	return p2p.SendMessage(p.rw, BlocksMsg, blocks)
}

func (p *peer) SendNewBlockHashes(hashes []utils.Hash) error {
	for _, hash := range hashes {
		p.existedBlocks.Add(hash)
	}
	return p2p.SendMessage(p.rw, NewBlockHashesMsg, hashes)
}

func (p *peer) SendConfirmed(confirmed *types.Confirmed) error {
	p.existedConfirmed.Add(confirmed.Hash())
	return p2p.SendMessage(p.rw, ConfirmedMsg, confirmed)
}

func (p *peer) SendNewBlock(block *types.Block, td *big.Int) error {
	p.existedBlocks.Add(block.Hash())
	return p2p.SendMessage(p.rw, NewBlockMsg, []interface{}{block, td})
}

func (p *peer) RequestHashes(from utils.Hash) error {
	return p2p.SendMessage(p.rw, GetBlockHashesMsg, getBlockHashesData{from, uint64(protocols.MaxHashFetch)})
}

func (p *peer) RequestBlocks(hashes []utils.Hash) error {
	return p2p.SendMessage(p.rw, GetBlocksMsg, hashes)
}

func (p *peer) RequestHashesFromNumber(from uint64, count int) error {
	return p2p.SendMessage(p.rw, GetBlockHashesFromNumberMsg, getBlockHashesFromNumberData{from, uint64(count)})
}

func (p *peer) Handshake(network uint64, td *big.Int, head utils.Hash, genesis utils.Hash) error {
	var status statusData
	errc := make(chan error, 2)
	go func() {
		errc <- p2p.SendMessage(p.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(p.version),
			NetworkID:       network,
			TD:              td,
			CurrentBlock:    head,
			GenesisBlock:    genesis,
		})
	}()
	go func() {
		errc <- p.readStatus(network, &status, genesis)
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return fmt.Errorf("time out")
		}
	}
	p.td, p.head = status.TD, status.CurrentBlock
	return nil
}

func (p *peer) readStatus(network uint64, status *statusData, genesis utils.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return fmt.Errorf("first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}

	if err := msg.DecodePayload(&status); err != nil {
		return fmt.Errorf("status msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return fmt.Errorf("genesis %x (!= %x)", status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkID != network {
		return fmt.Errorf("network %d (!= %d)", status.NetworkID, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return fmt.Errorf("version %d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}
