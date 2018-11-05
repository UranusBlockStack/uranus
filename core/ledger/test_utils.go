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

package ledger

import (
	"math/big"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

func testBlock() *types.Block {

	add1 := utils.BytesToAddress([]byte{0x11})
	add2 := utils.BytesToAddress([]byte{0x22})
	add3 := utils.BytesToAddress([]byte{0x33})

	tx1 := types.NewTransaction(types.Binary, 1, &add1, big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	tx2 := types.NewTransaction(types.Binary, 2, &add2, big.NewInt(222), 2222, big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	tx3 := types.NewTransaction(types.Binary, 3, &add3, big.NewInt(333), 3333, big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	txs := []*types.Transaction{tx1, tx2, tx3}

	block := types.NewBlock(&types.BlockHeader{Height: big.NewInt(1)}, txs, nil)
	return block
}

func testReceipt() []*types.Receipt {
	receipt1 := &types.Receipt{
		Status:            types.ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*types.Log{
			{Address: utils.BytesToAddress([]byte{0x11})},
			{Address: utils.BytesToAddress([]byte{0x01, 0x11})},
		},
		TransactionHash: utils.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: utils.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         111111,
	}
	receipt2 := &types.Receipt{
		State:             utils.Hash{2}.Bytes(),
		CumulativeGasUsed: 2,
		Logs: []*types.Log{
			{Address: utils.BytesToAddress([]byte{0x22})},
			{Address: utils.BytesToAddress([]byte{0x02, 0x22})},
		},
		TransactionHash: utils.BytesToHash([]byte{0x22, 0x22}),
		ContractAddress: utils.BytesToAddress([]byte{0x02, 0x22, 0x22}),
		GasUsed:         222222,
	}

	receipt3 := &types.Receipt{
		State:             utils.Hash{2}.Bytes(),
		CumulativeGasUsed: 3,
		Logs: []*types.Log{
			{Address: utils.BytesToAddress([]byte{0x33})},
			{Address: utils.BytesToAddress([]byte{0x03, 0x33})},
		},
		TransactionHash: utils.BytesToHash([]byte{0x33, 0x33}),
		ContractAddress: utils.BytesToAddress([]byte{0x03, 0x33, 0x33}),
		GasUsed:         333333,
	}
	receipts := []*types.Receipt{receipt1, receipt2, receipt3}
	return receipts
}
