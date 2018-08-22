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
	"testing"

	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
)

func TestRLP(t *testing.T) {
	receipt := &Receipt{
		Status:            ReceiptStatusSuccessful,
		CumulativeGasUsed: 1,
		Logs: []*Log{
			{Address: utils.BytesToAddress([]byte{0x11})},
			{Address: utils.BytesToAddress([]byte{0x01, 0x11})},
		},
		TransactionHash: utils.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: utils.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}
	data, err := rlp.EncodeToBytes((*ReceiptForStorage)(receipt))
	if err != nil {
		t.Error(err)
	}

	tr := &ReceiptForStorage{}

	err = rlp.DecodeBytes(data, tr)
	if err != nil {
		t.Error(err)
	}

	tmpdata, err := rlp.EncodeToBytes(tr)
	if err != nil {
		t.Error(err)
	}

	// ReceiptForStorage
	utils.AssertEquals(t, data, tmpdata)

	data, err = rlp.EncodeToBytes(receipt)
	if err != nil {
		t.Error(err)
	}

	tmpReceipt := &Receipt{}

	err = rlp.DecodeBytes(data, tmpReceipt)
	if err != nil {
		t.Error(err)
	}

	tmpdata, err = rlp.EncodeToBytes(tmpReceipt)
	if err != nil {
		t.Error(err)
	}
	utils.AssertEquals(t, data, tmpdata)
}
