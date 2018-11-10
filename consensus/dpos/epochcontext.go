package dpos

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
)

type EpochContext struct {
	TimeStamp   int64
	DposContext *types.DposContext
	Statedb     *state.StateDB
}

func (ec *EpochContext) lookupValidator(now int64) (validator utils.Address, err error) {
	validator = utils.Address{}
	offset := now % epochInterval
	// if offset%blockInterval != 0 {
	// 	return utils.Address{}, ErrInvalidMintBlockTime
	// }
	offset /= blockInterval

	validators, err := ec.DposContext.GetValidators()
	if err != nil {
		return utils.Address{}, err
	}
	validatorSize := len(validators)
	if validatorSize == 0 {
		return utils.Address{}, errors.New("failed to lookup validator")
	}
	offset %= int64(validatorSize)
	return validators[offset], nil
}

func (ec *EpochContext) tryElect(genesis, parent *types.BlockHeader) error {
	genesisEpoch := genesis.TimeStamp.Int64() / epochInterval
	prevEpoch := parent.TimeStamp.Int64() / epochInterval
	currentEpoch := ec.TimeStamp / epochInterval

	prevEpochIsGenesis := prevEpoch == genesisEpoch
	if prevEpochIsGenesis && prevEpoch < currentEpoch {
		prevEpoch = currentEpoch - 1
	}

	prevEpochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(prevEpochBytes, uint64(prevEpoch))
	iter := mtp.NewIterator(ec.DposContext.MintCntTrie().PrefixIterator(prevEpochBytes))
	for i := prevEpoch; i < currentEpoch; i++ {
		// if prevEpoch is not genesis, kickout not active candidate
		if !prevEpochIsGenesis && iter.Next() {
			if err := ec.kickoutValidator(prevEpoch); err != nil {
				return err
			}
		}
		votes, err := ec.CountVotes()
		if err != nil {
			return err
		}
		candidates := sortableAddresses{}
		for candidate, cnt := range votes {
			candidates = append(candidates, &sortableAddress{candidate, cnt})
		}
		// if len(candidates) < safeSize {
		// 	return errors.New("too few candidates")
		// }
		sort.Sort(candidates)
		if len(candidates) > maxValidatorSize {
			candidates = candidates[:maxValidatorSize]
		}

		// shuffle candidates
		seed := int64(binary.LittleEndian.Uint32(crypto.Keccak512(parent.Hash().Bytes()))) + i
		r := rand.New(rand.NewSource(seed))
		for i := len(candidates) - 1; i > 0; i-- {
			j := int(r.Int31n(int32(i + 1)))
			candidates[i], candidates[j] = candidates[j], candidates[i]
		}
		sortedValidators := make([]utils.Address, 0)
		for _, candidate := range candidates {
			sortedValidators = append(sortedValidators, candidate.address)
		}

		epochTrie, _ := types.NewEpochTrie(utils.Hash{}, ec.DposContext.DB())
		ec.DposContext.SetEpoch(epochTrie)
		ec.DposContext.SetValidators(sortedValidators)
		log.Info("Come to new epoch", "prevEpoch", i, "nextEpoch", i+1)
	}
	return nil
}

// CountVotes
func (ec *EpochContext) CountVotes() (votes map[utils.Address]*big.Int, err error) {
	votes = map[utils.Address]*big.Int{}
	delegateTrie := ec.DposContext.DelegateTrie()
	candidateTrie := ec.DposContext.CandidateTrie()
	statedb := ec.Statedb

	iterCandidate := mtp.NewIterator(candidateTrie.NodeIterator(nil))
	existCandidate := iterCandidate.Next()
	if !existCandidate {
		return votes, errors.New("no candidates")
	}
	for existCandidate {
		candidate := iterCandidate.Value
		candidateAddr := utils.BytesToAddress(candidate)
		delegateIterator := mtp.NewIterator(delegateTrie.PrefixIterator(candidate))
		existDelegator := delegateIterator.Next()
		if !existDelegator {
			votes[candidateAddr] = new(big.Int)
			existCandidate = iterCandidate.Next()
			continue
		}
		for existDelegator {
			delegator := delegateIterator.Value
			score, ok := votes[candidateAddr]
			if !ok {
				score = new(big.Int)
			}
			delegatorAddr := utils.BytesToAddress(delegator)
			weight := statedb.GetBalance(delegatorAddr)
			score.Add(score, weight)
			votes[candidateAddr] = score
			existDelegator = delegateIterator.Next()
		}
		existCandidate = iterCandidate.Next()
	}
	return votes, nil
}

func (ec *EpochContext) kickoutValidator(epoch int64) error {
	validators, err := ec.DposContext.GetValidators()
	if err != nil {
		return fmt.Errorf("failed to get validator: %s", err)
	}
	if len(validators) == 0 {
		return errors.New("no validator could be kickout")
	}

	epochDuration := epochInterval
	// First epoch duration may lt epoch interval,
	// while the first block time wouldn't always align with epoch interval,
	// so caculate the first epoch duartion with first block time instead of epoch interval,
	// prevent the validators were kickout incorrectly.
	if ec.TimeStamp-timeOfFirstBlock < epochInterval {
		epochDuration = ec.TimeStamp - timeOfFirstBlock
	}

	needKickoutValidators := sortableAddresses{}
	for _, validator := range validators {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, uint64(epoch))
		key = append(key, validator.Bytes()...)
		cnt := int64(0)
		if cntBytes := ec.DposContext.MintCntTrie().Get(key); cntBytes != nil {
			cnt = int64(binary.BigEndian.Uint64(cntBytes))
		}
		if cnt < epochDuration/blockInterval/maxValidatorSize/2 {
			// not active validators need kickout
			needKickoutValidators = append(needKickoutValidators, &sortableAddress{validator, big.NewInt(cnt)})
		}
	}
	// no validators need kickout
	needKickoutValidatorCnt := len(needKickoutValidators)
	if needKickoutValidatorCnt <= 0 {
		return nil
	}
	sort.Sort(sort.Reverse(needKickoutValidators))

	candidateCount := 0
	iter := mtp.NewIterator(ec.DposContext.CandidateTrie().NodeIterator(nil))
	for iter.Next() {
		candidateCount++
		if candidateCount >= needKickoutValidatorCnt+safeSize {
			break
		}
	}

	for i, validator := range needKickoutValidators {
		// ensure candidate count greater than or equal to safeSize
		if candidateCount <= safeSize {
			log.Info("No more candidate can be kickout", "prevEpochID", epoch, "candidateCount", candidateCount, "needKickoutCount", len(needKickoutValidators)-i)
			return nil
		}

		if err := ec.DposContext.KickoutCandidate(validator.address); err != nil {
			return err
		}
		// if kickout success, candidateCount minus 1
		candidateCount--
		log.Info("Kickout candidate", "prevEpochID", epoch, "candidate", validator.address.String(), "mintCnt", validator.weight.String())
	}
	return nil
}

type sortableAddress struct {
	address utils.Address
	weight  *big.Int
}
type sortableAddresses []*sortableAddress

func (p sortableAddresses) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p sortableAddresses) Len() int      { return len(p) }
func (p sortableAddresses) Less(i, j int) bool {
	if p[i].weight.Cmp(p[j].weight) < 0 {
		return false
	} else if p[i].weight.Cmp(p[j].weight) > 0 {
		return true
	} else {
		return p[i].address.String() < p[j].address.String()
	}
}

func TryElect(statedb *state.StateDB, dposContext *types.DposContext, timeStamp int64, header *types.BlockHeader, uranus consensus.IUranus) error {
	epochContext := &EpochContext{
		Statedb:     statedb,
		DposContext: dposContext,
		TimeStamp:   timeStamp,
	}
	parent := uranus.GetBlockByHash(header.PreviousHash)
	if timeOfFirstBlock == 0 {
		if firstBlock := uranus.GetBlockByHeight(1); firstBlock != nil {
			timeOfFirstBlock = firstBlock.BlockHeader().TimeStamp.Int64()
		}
	}
	genesis := uranus.GetBlockByHeight(0)
	err := epochContext.tryElect(genesis.BlockHeader(), parent.BlockHeader())
	if err != nil {
		return fmt.Errorf("got error when elect next epoch, err: %v", err)
	}

	//update mint count trie
	updateMintCnt(parent.BlockHeader().TimeStamp.Int64(), header.TimeStamp.Int64(), header.Miner, dposContext)
	return nil
}
