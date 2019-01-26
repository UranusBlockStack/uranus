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

import (
	"context"
	"fmt"

	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/types"
)

// BlockChainAPI exposes methods for the RPC interface
type BlockChainAPI struct {
	b Backend
}

// NewBlockChainAPI creates a new RPC service with methods specific for the blockchain.
func NewBlockChainAPI(b Backend) *BlockChainAPI {
	return &BlockChainAPI{b}
}

type GetBlockByHeightArgs struct {
	BlockHeight *BlockHeight
	FullTx      bool
}

// GetBlockByHeight returns the requested block.
func (s *BlockChainAPI) GetBlockByHeight(args GetBlockByHeightArgs, reply *map[string]interface{}) error {
	blockheight := LatestBlockHeight
	if args.BlockHeight != nil {
		blockheight = *args.BlockHeight
	}
	block, err := s.b.BlockByHeight(context.Background(), blockheight)
	if block != nil {
		response, err := s.rpcOutputBlock(block, true, args.FullTx)
		if err == nil && blockheight == PendingBlockHeight {
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		*reply = response
		return err
	}
	if block == nil {
		return fmt.Errorf("not found")
	}
	return err
}

type GetBlockByHashArgs struct {
	BlockHash utils.Hash
	FullTx    bool
}

// GetBlockByHash returns the requested block.
func (s *BlockChainAPI) GetBlockByHash(args GetBlockByHashArgs, reply *map[string]interface{}) error {
	block, err := s.b.BlockByHash(context.Background(), args.BlockHash)
	if block != nil {
		response, err := s.rpcOutputBlock(block, true, args.FullTx)
		if err != nil {
			return err
		}
		*reply = response
		return nil
	}
	if block == nil {
		return fmt.Errorf("not found")
	}
	return err
}

// GetTransactionByHash returns the transaction for the given hash
func (s *BlockChainAPI) GetTransactionByHash(Hash utils.Hash, reply *RPCTransaction) error {
	if stx := s.b.GetTransaction(Hash); stx != nil {
		*reply = *newRPCTransaction(stx.Tx, stx.BlockHash, stx.BlockHeight, stx.TxIndex)
		return nil

	}
	if tx := s.b.GetPoolTransaction(Hash); tx != nil {
		*reply = *newRPCPendingTransaction(tx)
		return nil
	} else if tx == nil {
		return fmt.Errorf("not found")
	}
	return nil
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (s *BlockChainAPI) GetTransactionReceipt(Hash utils.Hash, reply *map[string]interface{}) error {
	stx := s.b.GetTransaction(Hash)
	if stx == nil {
		return fmt.Errorf("not found")
	}
	receipt, err := s.b.GetReceipt(context.Background(), Hash)
	if err != nil {
		return err
	}
	if receipt == nil {
		return fmt.Errorf("not found")
	}
	from, _ := stx.Tx.Sender(types.Signer{})
	fields := map[string]interface{}{
		"blockHash":         stx.BlockHash,
		"blockHeight":       utils.Uint64(stx.BlockHeight),
		"root":              utils.Bytes(receipt.State),
		"status":            utils.Uint(receipt.Status),
		"transactionHash":   Hash,
		"transactionIndex":  utils.Uint64(stx.TxIndex),
		"from":              from,
		"tos":               stx.Tx.Tos(),
		"gasUsed":           utils.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": utils.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		// 	"logsBloom":         receipt.LogsBloom,
	}

	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (utils.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	*reply = fields
	return nil
}

func (s *BlockChainAPI) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields, err := RPCMarshalBlock(b, inclTx, fullTx)
	if err != nil {
		return nil, err
	}
	fields["totalDifficulty"] = (*utils.Big)(s.b.GetTd(b.Hash()))
	return fields, err
}
