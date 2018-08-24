package cpuminer

import (
	"math/big"
	"runtime"
	"sync"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/math"
	"github.com/UranusBlockStack/uranus/core/types"
)

var (
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

type CpuMiner struct {
	quit chan struct{}
	mu   sync.Mutex
}

func NewCpuMiner() *CpuMiner {
	return &CpuMiner{
		quit: make(chan struct{}),
	}
}

func (cm *CpuMiner) Mine(block *types.Block, stop <-chan struct{}, threads int, updateHashes chan uint64) (*types.Block, error) {
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
			cm.MineBlock(block, id, nonce, max, abort, found, updateHashes)
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

func (cm *CpuMiner) MineBlock(block *types.Block, id int, seed uint64, max uint64, abort chan struct{}, found chan *types.Block, updateHashes chan uint64) {
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
		case <-cm.quit:
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

func (cm *CpuMiner) Stop() {
	close(cm.quit)
}

// GetMiningTarget returns the mining target for the specified difficulty.
func GetMiningTarget(difficulty *big.Int) *big.Int {
	return new(big.Int).Div(maxUint256, difficulty)
}

// GetDifficult adjust difficult by parent info
func GetDifficult(time uint64, parentHeader *types.BlockHeader) *big.Int {
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
