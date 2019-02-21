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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/rlp"
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

// GetLatestBlockHeight returns the latest block height.
func (s *BlockChainAPI) GetLatestBlockHeight(ingore string, reply *utils.Uint64) error {
	h := s.b.CurrentBlock().Height()
	*reply = (utils.Uint64)(h.Uint64())
	return nil
}

type GetLatestBlockArgs struct {
	FullTx bool
}

// GetLatestBlock returns the latest block.
func (s *BlockChainAPI) GetLatestBlock(args GetLatestBlockArgs, reply *map[string]interface{}) error {
	block := s.b.CurrentBlock()
	response, err := s.rpcOutputBlock(block, true, args.FullTx)
	if err == nil {
		for _, field := range []string{"hash", "nonce", "miner"} {
			response[field] = nil
		}
	}
	*reply = response
	return err
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

// ExportBlocksArgs export blocks
type ExportBlocksArgs struct {
	FileName    string
	FirstHeight *BlockHeight
	LastHeight  *BlockHeight
}

// ExportBlocks export block
func (s *BlockChainAPI) ExportBlocks(args ExportBlocksArgs, reply *map[string]interface{}) error {

	fh, err := os.OpenFile(args.FileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer fh.Close()

	var writer io.Writer = fh
	if strings.HasSuffix(args.FileName, ".gz") {
		writer = gzip.NewWriter(writer)
		defer writer.(*gzip.Writer).Close()
	}

	if args.FirstHeight == nil {
		firtHeight := BlockHeight(1)
		args.FirstHeight = &firtHeight
	}
	if args.LastHeight == nil {
		lastHeight := BlockHeight(s.b.CurrentBlock().Height().Int64())
		args.LastHeight = &lastHeight
	}
	return s.b.BlockChain().ExportN(writer, uint64(args.FirstHeight.Int64()), uint64(args.LastHeight.Int64()))

}

// ImportBlocks import blocks
func (s *BlockChainAPI) ImportBlocks(filename string, reply *map[string]interface{}) error {
	interrupt := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	go func() {
		if _, ok := <-interrupt; ok {
			log.Info("Interrupted during import, stopping at next batch")
		}
		close(stop)
	}()

	checkInterrupt := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}

	log.Infof("Importing blockchain file %v", filename)
	fh, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fh.Close()
	var reader io.Reader = fh
	if strings.HasSuffix(filename, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return err
		}
	}
	stream := rlp.NewStream(reader, 0)

	importBatchSize := 2500

	n := 0
	for batch := 0; ; batch++ {
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		i := 0
		blocks := make([]*types.Block, 0)
		for ; i < importBatchSize; i++ {
			var b types.Block
			if err := stream.Decode(&b); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("at block %d: %v", n, err)
			}
			// don't import first block
			if b.Height().Uint64() == 0 {
				i--
				continue
			}
			blocks = append(blocks, &b)
			n++
		}
		if i == 0 {
			break
		}
		// Import the batch.
		if checkInterrupt() {
			return fmt.Errorf("interrupted")
		}
		if _, err := s.b.BlockChain().InsertChain(blocks); err != nil {
			return fmt.Errorf("invalid block %d: %v", n, err)
		}
	}
	return nil
}
