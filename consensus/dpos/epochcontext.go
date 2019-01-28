package dpos

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
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
	offset := now % Option.epochInterval()
	// if offset%Option.BlockInterval != 0 {
	// 	return utils.Address{}, ErrInvalidMintBlockTime
	// }
	offset /= Option.BlockInterval * Option.BlockRepeat

	validators, err := ec.DposContext.GetValidators()
	if err != nil {
		return utils.Address{}, err
	}
	validatorSize := int64(len(validators))

	if validatorSize == 0 {
		return utils.Address{}, errors.New("failed to lookup validator")
	}
	if !ec.DposContext.IsDpos() {
		offset %= int64(validatorSize)
	} else if offset >= validatorSize {
		return utils.Address{}, nil
	}
	return validators[offset], nil
}

func (ec *EpochContext) tryElect(genesis, parent *types.BlockHeader) error {
	genesisEpoch := genesis.TimeStamp.Int64() / Option.epochInterval()
	prevEpoch := parent.TimeStamp.Int64() / Option.epochInterval()
	currentEpoch := ec.TimeStamp / Option.epochInterval()
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
		votes, total, err := ec.CountVotes()
		if err != nil {
			return err
		}
		if int64(len(votes)) < Option.consensusSize() || total.Cmp(Option.MinStartQuantity) < 0 {
			//log.Warn("dpos not activated")
			return nil
		}
		candidates := sortableAddresses{}
		for candidate, cnt := range votes {
			candidates = append(candidates, &sortableAddress{candidate, cnt})
		}
		sort.Sort(candidates)
		if int64(len(candidates)) > Option.MaxValidatorSize {
			candidates = candidates[:Option.MaxValidatorSize]
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
		firstEpcho := timeOfFirstBlock / Option.epochInterval()
		log.Infof("Come to new epoch prevEpoch %v nextEpoch %v, validators %v", i-firstEpcho+1, i-firstEpcho+2, sortedValidators)

	}
	return nil
}

// CountVotes
func (ec *EpochContext) CountVotes() (votes map[utils.Address]*big.Int, total *big.Int, err error) {
	votes = map[utils.Address]*big.Int{}
	delegateTrie := ec.DposContext.DelegateTrie()
	candidateTrie := ec.DposContext.CandidateTrie()
	statedb := ec.Statedb

	iterCandidate := mtp.NewIterator(candidateTrie.NodeIterator(nil))
	existCandidate := iterCandidate.Next()
	if !existCandidate {
		return votes, total, errors.New("no candidates")
	}
	total = big.NewInt(0)
	for existCandidate {
		candidateInfo := &types.CandidateInfo{}
		rlp.DecodeBytes(iterCandidate.Value, candidateInfo)
		candidateAddr := candidateInfo.Addr
		delegateIterator := mtp.NewIterator(delegateTrie.PrefixIterator(candidateInfo.Addr.Bytes()))
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
			weight := statedb.GetLockedBalance(delegatorAddr)
			total = new(big.Int).Add(total, weight)
			score.Add(score, weight)
			votes[candidateAddr] = score
			existDelegator = delegateIterator.Next()
		}
		votes[candidateAddr] = new(big.Int).Mul(votes[candidateAddr], big.NewInt(int64(candidateInfo.Weight)))
		existCandidate = iterCandidate.Next()
	}
	return votes, total, nil
}

func (ec *EpochContext) kickoutValidator(epoch int64) error {
	validators, err := ec.DposContext.GetValidators()
	if err != nil {
		return fmt.Errorf("failed to get validator: %s", err)
	}
	if len(validators) == 0 {
		return errors.New("no validator could be kickout")
	}

	epochDuration := Option.epochInterval()
	// First epoch duration may lt epoch interval,
	// while the first block time wouldn't always align with epoch interval,
	// so caculate the first epoch duartion with first block time instead of epoch interval,
	// prevent the validators were kickout incorrectly.
	if ec.TimeStamp-timeOfFirstBlock < Option.epochInterval() {
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
		// degrade && upgrade
		candidate, err := ec.DposContext.CandidateTrie().TryGet(validator.Bytes())
		if err != nil {
			return err
		}
		if candidate == nil {
			return errors.New("invalid candidate to delegate")
		}
		candidateInfo := &types.CandidateInfo{}
		if err := rlp.DecodeBytes(candidate, candidateInfo); err != nil {
			return err
		}
		if cnt < epochDuration/Option.BlockInterval/Option.MaxValidatorSize/2 {
			if candidateInfo.Weight > 10 {
				candidateInfo.Weight -= 10
				candidateInfo.DegradeTime = uint64(time.Now().Unix())
			}
		} else {
			if candidateInfo.Weight < 100 {
				candidateInfo.Weight += 10
				candidateInfo.DegradeTime = uint64(time.Now().Unix())
			}
		}
		val, err := rlp.EncodeToBytes(candidateInfo)
		if err != nil {
			return err
		}
		if err := ec.DposContext.CandidateTrie().TryUpdate(validator.Bytes(), val); err != nil {
			return err
		}
	}
	// no validators need kickout
	needKickoutValidatorCnt := int64(len(needKickoutValidators))
	if needKickoutValidatorCnt <= 0 {
		return nil
	}
	sort.Sort(sort.Reverse(needKickoutValidators))

	candidateCount := int64(0)
	iter := mtp.NewIterator(ec.DposContext.CandidateTrie().NodeIterator(nil))
	for iter.Next() {
		candidateCount++
		if candidateCount >= needKickoutValidatorCnt+Option.consensusSize() {
			break
		}
	}

	for i, validator := range needKickoutValidators {
		// ensure candidate count greater than or equal to Option.consensusSize()
		if candidateCount <= Option.consensusSize() {
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
