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

package types

import (
	"io"
	"unsafe"

	"github.com/UranusBlockStack/uranus/common/bloom"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

var (
	// ReceiptStatusFailed is the status code of a transaction if execution failed.
	ReceiptStatusFailed = uint64(0)
	// ReceiptStatusFailedRLP is rlp bytes of a transaction if execution failed.
	ReceiptStatusFailedRLP = []byte{}

	// ReceiptStatusSuccessful is the status code of a transaction if execution succeeded.
	ReceiptStatusSuccessful = uint64(1)
	// ReceiptStatusSuccessfulRLP is rlp bytes of a transaction if execution succeeded.
	ReceiptStatusSuccessfulRLP = []byte{0x01}
)

// Receipt represents the results of a transaction.
type Receipt struct {
	// Consensus fields
	State             []byte      `json:"root"`
	Status            uint64      `json:"status"`
	CumulativeGasUsed uint64      `json:"cumulativeGasUsed" gencodec:"required"`
	LogsBloom         bloom.Bloom `json:"logsBloom"         gencodec:"required"`
	Logs              []*Log      `json:"logs"              gencodec:"required"`

	// Derived fields. (don't reorder!)
	TransactionHash utils.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress utils.Address `json:"contractAddress"`
	GasUsed         uint64        `json:"gasUsed" gencodec:"required"`
}

// NewReceipt creates a transaction receipt.
func NewReceipt(root []byte, failed bool, cumulativeGasUsed uint64) *Receipt {
	r := &Receipt{State: utils.CopyBytes(root), CumulativeGasUsed: cumulativeGasUsed}
	if failed {
		r.Status = ReceiptStatusFailed
	} else {
		r.Status = ReceiptStatusSuccessful
	}
	return r
}

type rlpReceipt struct {
	State             []byte
	Status            uint64
	CumulativeGasUsed uint64
	LogsBloom         bloom.Bloom
	Logs              []*Log
}

// EncodeRLP implements rlp.Encoder
func (r *Receipt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &rlpReceipt{r.State, r.Status, r.CumulativeGasUsed, r.LogsBloom, r.Logs})
}

// DecodeRLP implements rlp.Decoder
func (r *Receipt) DecodeRLP(s *rlp.Stream) error {
	dec := &rlpReceipt{}
	if err := s.Decode(dec); err != nil {
		return err
	}
	r.State = dec.State
	r.Status = dec.Status
	r.CumulativeGasUsed, r.LogsBloom, r.Logs = dec.CumulativeGasUsed, dec.LogsBloom, dec.Logs
	return nil
}

// Size returns the approximate memory used by all internal contents.
func (r *Receipt) Size() utils.StorageSize {
	size := utils.StorageSize(unsafe.Sizeof(*r)) + utils.StorageSize(len(r.State))

	size += utils.StorageSize(len(r.Logs)) * utils.StorageSize(unsafe.Sizeof(Log{}))
	for _, log := range r.Logs {
		size += utils.StorageSize(len(log.Topics)*utils.HashLength + len(log.Data))
	}
	return size
}

// ReceiptForStorage is a wrapper around a Receipt that flattens and parses the entire content of a log including non-consensus fields.
type ReceiptForStorage Receipt

type rlpReceiptStorage struct {
	State             []byte
	Status            uint64
	CumulativeGasUsed uint64
	LogsBloom         bloom.Bloom
	TransactionHash   utils.Hash
	ContractAddress   utils.Address
	Logs              []*LogForStorage
	GasUsed           uint64
}

// EncodeRLP implements rlp.Encoder, and flattens all content fields of a receipt into an RLP stream.
func (r *ReceiptForStorage) EncodeRLP(w io.Writer) error {
	enc := &rlpReceiptStorage{
		State:             r.State,
		Status:            r.Status,
		CumulativeGasUsed: r.CumulativeGasUsed,
		LogsBloom:         r.LogsBloom,
		TransactionHash:   r.TransactionHash,
		ContractAddress:   r.ContractAddress,
		Logs:              make([]*LogForStorage, len(r.Logs)),
		GasUsed:           r.GasUsed,
	}
	for i, log := range r.Logs {
		enc.Logs[i] = (*LogForStorage)(log)
	}
	return rlp.Encode(w, enc)
}

func (r *ReceiptForStorage) DecodeRLP(s *rlp.Stream) error {
	var dec rlpReceiptStorage
	if err := s.Decode(&dec); err != nil {
		return err
	}

	// Assign the consensus fields
	r.State = dec.State
	r.Status = dec.Status
	r.CumulativeGasUsed, r.LogsBloom = dec.CumulativeGasUsed, dec.LogsBloom
	r.Logs = make([]*Log, len(dec.Logs))
	for i, log := range dec.Logs {
		r.Logs[i] = (*Log)(log)
	}
	// Assign the implementation fields
	r.TransactionHash, r.ContractAddress, r.GasUsed = dec.TransactionHash, dec.ContractAddress, dec.GasUsed
	return nil
}
