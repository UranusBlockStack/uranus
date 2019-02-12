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

package cpuminer

import (
	"errors"
	"math/big"
	"runtime"
	"sync"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/math"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
)

var (
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

type CpuMiner struct {
}

func NewCpuMiner() *CpuMiner {
	return &CpuMiner{}
}

func (cm *CpuMiner) Seal(chain consensus.IChainReader, block *types.Block, stop <-chan struct{}, threads int, updateHashes chan uint64) (*types.Block, error) {
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})
	found := make(chan *types.Block)

	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra logic around local/remote
	}
	log.Infof("miner start with %d threads", threads)

	var pend sync.WaitGroup
	for i := 0; i < threads; i++ {
		pend.Add(1)
		factor := math.MaxUint64 / uint64(threads)
		seed := uint64(factor * uint64(i))
		max := uint64(math.MaxUint64)
		if i != threads-1 {
			max = factor * (uint64(i) + 1)
		}
		go func(id int, nonce, max uint64) {
			defer pend.Done()
			cm.mine(block, id, nonce, max, abort, found, updateHashes)
		}(i, seed, max)
	}
	// Wait until sealing is terminated or a nonce is found
	var result *types.Block
	select {
	case <-stop:
		// Outside abort, stop all miner threads
		close(abort)
	case result = <-found:
		// One of the threads found a block, abort all others
		log.Infof("miner block[%+v] finish", result.Height().Uint64())
		close(abort)
	}
	// Wait for all miners to terminate and return the block
	pend.Wait()

	return result, nil
}

func (cm *CpuMiner) mine(block *types.Block, id int, seed uint64, max uint64, abort chan struct{}, found chan *types.Block, updateHashes chan uint64) {
	var nonce = seed
	var hashInt big.Int
	var caltimes = uint64(0)
	var header = block.BlockHeader()
	target := GetMiningTarget(header.Difficulty)
miner:
	for {
		select {
		case <-abort:
			log.Info("nonce finding aborted")
			break miner
		default:
			caltimes++
			if caltimes == 0x7FFF {
				updateHashes <- caltimes
				caltimes = 0
			}
			header.Nonce = types.EncodeNonce(nonce)
			hash := header.Hash()
			hashInt.SetBytes(hash.Bytes())
			// found
			// log.Debugf("target: %v, hashInt: %v ", target.String(), hashInt.String())
			if hashInt.Cmp(target) <= 0 {
				select {
				case <-abort:
					break miner
				case found <- block.WithSeal(header):
				}
				break miner
			}
			// outage
			if nonce == max {
				log.Warnf("nonce finding outage nonce: %v, max: %v", nonce, max)
				break miner
			}
			nonce++
		}
	}
}

// GetMiningTarget returns the mining target for the specified difficulty.
func GetMiningTarget(difficulty *big.Int) *big.Int {
	return new(big.Int).Div(maxUint256, difficulty)
}

// Author returning the header's miner as the proof-of-work verified author of the block.
func (cm *CpuMiner) Author(header *types.BlockHeader) (utils.Address, error) {
	return header.Miner, nil
}

// CalcDifficulty returns the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (cm *CpuMiner) CalcDifficulty(chain consensus.IChainReader, config *params.ChainConfig, time uint64, parentHeader *types.BlockHeader) *big.Int {
	// diff = parentDiff + parentDiff / 1024 * max (1 - (blockTime - parentTime) / 10, -99)
	parentDifficult := parentHeader.Difficulty
	parentTime := parentHeader.TimeStamp.Uint64()
	if parentHeader.Height.Int64() == 0 {
		return parentDifficult
	}
	big1 := big.NewInt(1)
	big99 := big.NewInt(-99)
	big1024 := big.NewInt(1024)

	interval := (time - parentTime) / 10
	var x *big.Int
	x = big.NewInt(int64(interval))
	x.Sub(big1, x)
	if x.Cmp(big99) < 0 {
		x = big99
	}
	var y = new(big.Int).Set(parentDifficult)
	y.Div(parentDifficult, big1024)

	var result = big.NewInt(0)
	result.Mul(x, y)
	result.Add(parentDifficult, result)
	return result
}

// func (cm *CpuMiner) VerifyHeader(chain consensus.IChainReader, header *types.BlockHeader) error {
// 	if header == nil || header.Height == nil {
// 		return consensus.ErrUnknownBlock
// 	}
// 	// Short circuit if the header is known, or it's parent not
// 	if chain.GetBlockByHash(header.Hash()) != nil {
// 		return nil
// 	}
// 	parent := chain.GetBlockByHash(header.PreviousHash)
// 	if parent == nil {
// 		return consensus.ErrUnknownAncestor
// 	}

// 	if header.TimeStamp.Cmp(big.NewInt(time.Now().Unix())) > 0 {
// 		return consensus.ErrFutureBlock
// 	}
// 	parentHeader := parent.BlockHeader()
// 	// Verify the block's difficulty based in it's timestamp and parent's difficulty
// 	expected := cm.CalcDifficulty(chain.Config(), header.TimeStamp.Uint64(), parentHeader)
// 	if expected.Cmp(header.Difficulty) != 0 {
// 		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
// 	}
// 	// Verify that the gas limit is <= 2^63-1
// 	if big.NewInt(int64(header.GasLimit)).Cmp(math.MaxBig63) > 0 {
// 		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, math.MaxBig63)
// 	}
// 	// Verify that the gasUsed is <= gasLimit
// 	if big.NewInt(int64(header.GasUsed)).Cmp(big.NewInt(int64(header.GasLimit))) > 0 {
// 		return fmt.Errorf("invalid gasUsed: have %v, gasLimit %v", header.GasUsed, header.GasLimit)
// 	}

// 	// Verify that the gas limit remains within allowed bounds
// 	diff := new(big.Int).Set(big.NewInt(int64(parentHeader.GasLimit)))
// 	diff = diff.Sub(diff, big.NewInt(int64(header.GasLimit)))
// 	diff.Abs(diff)

// 	limit := new(big.Int).Set(big.NewInt(int64(parentHeader.GasLimit)))
// 	limit = limit.Div(limit, big.NewInt(int64(params.GasLimitBoundDivisor)))

// 	if diff.Cmp(limit) >= 0 || big.NewInt(int64(header.GasLimit)).Cmp(big.NewInt(int64(params.MinGasLimit))) < 0 {
// 		return fmt.Errorf("invalid gas limit: have %v, want %v += %v", header.GasLimit, parentHeader.GasLimit, limit)
// 	}

// 	if diff := new(big.Int).Sub(header.Height, parentHeader.Height); diff.Cmp(big.NewInt(1)) != 0 {
// 		return consensus.ErrInvalidNumber
// 	}
// 	return nil
// }

// VerifySeal  checking whether the given block satisfies the PoW difficulty requirements.
func (cm *CpuMiner) VerifySeal(chain consensus.IChainReader, header *types.BlockHeader) error {
	var hashInt big.Int
	hash := header.Hash()
	hashInt.SetBytes(hash.Bytes())
	if hashInt.Cmp(header.Difficulty) <= 0 {
		return nil
	}
	return errors.New("invalid proof-of-work")
}

// Finalize .
func (cm *CpuMiner) Finalize(chain consensus.IChainReader, header *types.BlockHeader, state *state.StateDB, txs []*types.Transaction, actions []*types.Action, receipts []*types.Receipt, dposContext *types.DposContext) (*types.Block, error) {
	// Accumulate block rewards and commit the final state root
	state.AddBalance(header.Miner, params.BlockReward)
	header.StateRoot = state.IntermediateRoot(true)

	return types.NewBlock(header, txs, actions, receipts), nil
}
