package dpos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/crypto/sha3"
	"github.com/UranusBlockStack/uranus/common/db"
	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/mtp"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/feed"
	"github.com/UranusBlockStack/uranus/params"
	lru "github.com/hashicorp/golang-lru"
)

var Option = &option{
	BlockInterval:    int64(500 * time.Millisecond), // 500 ms
	BlockRepeat:      12,
	MaxValidatorSize: 3,
	MinStartQuantity: big.NewInt(1000),
}

func (opt *option) consensusSize() int64 {
	return opt.MaxValidatorSize*2/3 + 1
}

func (opt *option) epochInterval() int64 {
	return opt.BlockInterval * opt.BlockRepeat * opt.MaxValidatorSize
}

type option struct {
	BlockInterval    int64
	BlockRepeat      int64
	MaxValidatorSize int64
	MinStartQuantity *big.Int
}

const (
	extraSeal = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
)

var (
	errMissingSignature           = errors.New("extra-data 65 byte suffix signature missing")
	ErrInvalidTimestamp           = errors.New("invalid timestamp")
	ErrWaitForPrevBlock           = errors.New("wait for last block arrived")
	ErrMintFutureBlock            = errors.New("mint the future block")
	ErrMismatchSignerAndValidator = errors.New("mismatch block signer and validator")
	ErrInvalidBlockValidator      = errors.New("invalid block validator")
	ErrInvalidMintBlockTime       = errors.New("invalid time to mint the block")
	ErrNilBlockHeader             = errors.New("nil block header returned")
)
var (
	timeOfFirstBlock   = int64(0)
	confirmedBlockHead = []byte("confirmed-block-head")
)

type SignerFn func(utils.Address, []byte) ([]byte, error)
type Dpos struct {
	eventMux             *feed.TypeMux
	chainDb              db.Database
	db                   state.Database
	signFn               SignerFn
	confirmedBlockHeader *types.BlockHeader
	bftConfirmeds        *lru.Cache
	coinbase             utils.Address
}

func NewDpos(eventMux *feed.TypeMux, chainDb db.Database, db state.Database, signFn SignerFn) *Dpos {
	d := &Dpos{
		eventMux: eventMux,
		chainDb:  chainDb,
		db:       db,
		signFn:   signFn,
	}
	return d
}
func (dpos *Dpos) Init(chain consensus.IChainReader) {
	dpos.confirmedBlockHeader, _ = dpos.loadConfirmedBlockHeader(chain)
	go func() {
		dpos.bftConfirmeds, _ = lru.New(int(chain.Config().MaxValidatorSize))
		sub := dpos.eventMux.Subscribe(types.Confirmed{})
		for ev := range sub.Chan() {
			switch ev.Data.(type) {
			case types.Confirmed:
				confirmed := ev.Data.(types.Confirmed)
				dpos.handleConfirmed(chain, &confirmed)
			default:
			}
		}
	}()
}

func (dpos *Dpos) handleConfirmed(chain consensus.IChainReader, confirmed *types.Confirmed) {
	if confirmed.IsValidate() {
		if blk := chain.GetBlockByHeight(confirmed.BlockHeight); blk != nil && bytes.Compare(blk.Hash().Bytes(), confirmed.BlockHash.Bytes()) == 0 {
			dpos.bftConfirmeds.Add(confirmed.Address, confirmed.BlockHeight)
		}
	} else {
		// TODO
	}
}

func prevSlot(now int64) int64 {
	return int64((now-Option.BlockInterval/10)/Option.BlockInterval) * Option.BlockInterval
}

func nextSlot(now int64) int64 {
	return int64((now+Option.BlockInterval-Option.BlockInterval/10)/Option.BlockInterval) * Option.BlockInterval
}

// update counts in MintCntTrie for the miner of newBlock
func updateMintCnt(parentBlockTime, currentBlockTime int64, validator utils.Address, dposContext *types.DposContext) {
	currentMintCntTrie := dposContext.MintCntTrie()
	currentEpoch := parentBlockTime / Option.epochInterval()
	currentEpochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(currentEpochBytes, uint64(currentEpoch))

	cnt := int64(1)
	newEpoch := currentBlockTime / Option.epochInterval()
	// still during the currentEpochID
	if currentEpoch == newEpoch {
		iter := mtp.NewIterator(currentMintCntTrie.NodeIterator(currentEpochBytes))

		// when current is not genesis, read last count from the MintCntTrie
		if iter.Next() {
			cntBytes := currentMintCntTrie.Get(append(currentEpochBytes, validator.Bytes()...))

			// not the first time to mint
			if cntBytes != nil {
				cnt = int64(binary.BigEndian.Uint64(cntBytes)) + 1
			}
		}
	}

	newCntBytes := make([]byte, 8)
	newEpochBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(newEpochBytes, uint64(newEpoch))
	binary.BigEndian.PutUint64(newCntBytes, uint64(cnt))
	dposContext.MintCntTrie().TryUpdate(append(newEpochBytes, validator.Bytes()...), newCntBytes)
}

func sigHash(header *types.BlockHeader) (hash utils.Hash) {
	hasher := sha3.NewKeccak256()
	rlp.Encode(hasher, []interface{}{
		header.PreviousHash,
		header.Miner,
		header.StateRoot,
		header.TransactionsRoot,
		header.ReceiptsRoot,
		header.LogsBloom,
		header.Difficulty,
		header.Height,
		header.GasLimit,
		header.GasUsed,
		header.TimeStamp,
		header.ExtraData[:len(header.ExtraData)-extraSeal], // Yes, this will panic if extra is too short
		header.Nonce,
		header.DposContext.Root(),
	})
	hasher.Sum(hash[:0])
	return hash
}

func ecrecover(header *types.BlockHeader) (utils.Address, error) {
	if len(header.ExtraData) < extraSeal {
		return utils.Address{}, errMissingSignature
	}

	signature := header.ExtraData[len(header.ExtraData)-extraSeal:]
	pubkey, err := crypto.EcrecoverToByte(sigHash(header).Bytes(), signature)
	if err != nil {
		return utils.Address{}, err
	}
	var signer utils.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	return signer, nil
}

func (d *Dpos) Seal(chain consensus.IChainReader, block *types.Block, stop <-chan struct{}, threads int, updateHashes chan uint64) (*types.Block, error) {
	header := block.BlockHeader()
	if header == nil || header.Height == nil {
		return nil, consensus.ErrUnknownBlock
	}
	header.ExtraData = append(header.ExtraData, make([]byte, extraSeal)...)
	block = block.WithSeal(header)

	now := time.Now().UnixNano()
	delay := nextSlot(now) - now
	if delay > 0 {
		select {
		case <-stop:
			return nil, nil
		case <-time.After(time.Duration(delay) * time.Second):
		}
	}

	// time's up, sign the block
	sighash, err := d.signFn(header.Miner, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	copy(header.ExtraData[len(header.ExtraData)-extraSeal:], sighash)
	d.updateConfirmedBlockHeader(chain, block.DposCtx().IsDpos())
	return block.WithSeal(header), nil
}

func (d *Dpos) SetCoinBase(addr utils.Address) {
	d.coinbase = addr
}

func (d *Dpos) Author(header *types.BlockHeader) (utils.Address, error) {
	return header.Miner, nil
}

func (d *Dpos) CalcDifficulty(config *params.ChainConfig, time uint64, parent *types.BlockHeader) *big.Int {
	return big.NewInt(1)
}

func (d *Dpos) VerifySeal(chain consensus.IChainReader, header *types.BlockHeader) error {
	if header == nil || header.Height == nil {
		return consensus.ErrUnknownBlock
	}

	parent := chain.GetBlockByHash(header.PreviousHash)
	if parent == nil || bytes.Compare(parent.Hash().Bytes(), header.PreviousHash.Bytes()) != 0 {
		return consensus.ErrUnknownAncestor
	}

	signer, err := ecrecover(header)
	if err != nil {
		return err
	}

	if bytes.Compare(signer.Bytes(), header.Miner.Bytes()) != 0 {
		return ErrMismatchSignerAndValidator
	}

	statedb, err := state.New(chain.CurrentBlock().StateRoot(), d.db)
	if err != nil {
		return err
	}
	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), parent.BlockHeader().DposContext)
	if err != nil {
		return err
	}
	epochContext := &EpochContext{DposContext: dposContext}
	validator, err := epochContext.lookupValidator(header.TimeStamp.Int64())
	if err != nil {
		return err
	}
	if bytes.Compare(validator.Bytes(), header.Miner.Bytes()) != 0 {
		return ErrInvalidBlockValidator
	}
	return d.updateConfirmedBlockHeader(chain, epochContext.DposContext.IsDpos())
}

func (d *Dpos) updateConfirmedBlockHeader(chain consensus.IChainReader, dpos bool) error {
	if d.confirmedBlockHeader == nil {
		header, err := d.loadConfirmedBlockHeader(chain)
		if err != nil {
			header = chain.GetBlockByHeight(0).BlockHeader()
			if header == nil {
				return err
			}
		}
		d.confirmedBlockHeader = header
	}

	if !dpos {
		if blk := chain.CurrentBlock(); blk != nil {
			d.confirmedBlockHeader = blk.BlockHeader()
			if err := d.storeConfirmedBlockHeader(chain.CurrentBlock()); err != nil {
				log.Errorf("dpos set confirmed block header success", "currentHeader", d.confirmedBlockHeader.Height, err)
				return err
			}
			log.Debugf("dpos set confirmed block header success", "currentHeader", d.confirmedBlockHeader.Height)
		}
		return nil
	}

	epoch := int64(-1)
	validatorMap := make(map[utils.Address]bool)
	curHeader := chain.CurrentBlock().BlockHeader()
	for d.confirmedBlockHeader.Hash() != curHeader.Hash() &&
		d.confirmedBlockHeader.Height.Uint64() < curHeader.Height.Uint64() {
		curEpoch := curHeader.TimeStamp.Int64() / Option.epochInterval()
		if curEpoch != epoch {
			epoch = curEpoch
			validatorMap = make(map[utils.Address]bool)
		}
		// fast return
		// if block number difference less Option.consensusSize()-witnessNum
		// there is no need to check block is confirmed
		if curHeader.Height.Int64()-d.confirmedBlockHeader.Height.Int64() < int64(Option.consensusSize()-int64(len(validatorMap))) {
			log.Debug("Dpos fast return", "current", curHeader.Height.String(), "confirmed", d.confirmedBlockHeader.Height.String(), "witnessCount", len(validatorMap))
			return nil
		}
		validatorMap[curHeader.Miner] = true
		if int64(len(validatorMap)) >= Option.consensusSize() {
			d.confirmedBlockHeader = curHeader
			if err := d.storeConfirmedBlockHeader(chain.CurrentBlock()); err != nil {
				log.Errorf("dpos set confirmed block header success", "currentHeader", d.confirmedBlockHeader.Height, err)
				return err
			}
			log.Debugf("dpos set confirmed block header success", "currentHeader", curHeader.Height)
			return nil
		}
		curHeader = chain.GetBlockByHash(curHeader.PreviousHash).BlockHeader()
		if curHeader == nil {
			return ErrNilBlockHeader
		}
	}
	return nil
}

func (d *Dpos) loadConfirmedBlockHeader(chain consensus.IChainReader) (*types.BlockHeader, error) {
	key, err := d.chainDb.Get(confirmedBlockHead)
	if err != nil {
		return nil, err
	}

	blk := chain.GetBlockByHash(utils.BytesToHash(key))
	if blk == nil {
		return nil, ErrNilBlockHeader
	}
	return blk.BlockHeader(), nil
}

// store inserts the snapshot into the database.
func (d *Dpos) storeConfirmedBlockHeader(lastBlock *types.Block) error {
	if statedb, err := state.New(lastBlock.StateRoot(), d.db); err == nil {
		if dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), lastBlock.BlockHeader().DposContext); err == nil {
			validators, _ := dposContext.GetValidators()
			for _, validator := range validators {
				if bytes.Compare(validator.Bytes(), d.coinbase.Bytes()) == 0 {
					confirmed := &types.Confirmed{
						BlockHash:   d.confirmedBlockHeader.Hash(),
						BlockHeight: d.confirmedBlockHeader.Height.Uint64(),
						Address:     d.coinbase,
					}
					if sighash, err := d.signFn(d.coinbase, confirmed.Hash().Bytes()); err == nil {
						confirmed.Signature = sighash
						d.eventMux.Post(feed.NewConfirmedEvent{Confirmed: confirmed})
						d.bftConfirmeds.Add(d.coinbase, confirmed.BlockHeight)
					} else {
						log.Errorf("confirmed sign err %v", err)
					}
				}
			}
		}
	}

	return d.chainDb.Put(confirmedBlockHead, d.confirmedBlockHeader.Hash().Bytes())
}

func (d *Dpos) GetConfirmedBlockNumber() (*big.Int, error) {
	header := d.confirmedBlockHeader
	if header == nil {
		return big.NewInt(0), nil
	}
	return header.Height, nil
}

func (dpos *Dpos) GetBFTConfirmedBlockNumber() (*big.Int, error) {
	irreversibles := UInt64Slice{}
	keys := dpos.bftConfirmeds.Keys()
	for _, key := range keys {
		if irreversible, ok := dpos.bftConfirmeds.Get(key); ok {
			irreversibles = append(irreversibles, irreversible.(uint64))
		}
	}

	if len(irreversibles) == 0 {
		return big.NewInt(0), nil
	}

	sort.Sort(irreversibles)

	/// 2/3 must be greater, so if I go 1/3 into the list sorted from low to high, then 2/3 are greater
	return big.NewInt(int64(irreversibles[(len(irreversibles)-1)/3])), nil
}

func (d *Dpos) CheckValidator(lastBlock *types.Block, coinbase utils.Address, now int64) error {
	prevSlot := prevSlot(now)
	nextSlot := nextSlot(now)
	if lastBlock.Time().Int64() >= nextSlot {
		return ErrMintFutureBlock
	}
	if lastBlock.Time().Int64() != prevSlot && nextSlot-now >= Option.BlockInterval/10 {
		return ErrWaitForPrevBlock
	}
	if now%Option.BlockInterval != 0 {
		return ErrInvalidMintBlockTime
	}

	statedb, err := state.New(lastBlock.StateRoot(), d.db)
	if err != nil {
		return err
	}

	dposContext, err := types.NewDposContextFromProto(statedb.Database().TrieDB(), lastBlock.BlockHeader().DposContext)
	if err != nil {
		return err
	}
	epochContext := &EpochContext{DposContext: dposContext}
	validator, err := epochContext.lookupValidator(now)
	if err != nil {
		return err
	}
	if (validator == utils.Address{}) || bytes.Compare(validator.Bytes(), coinbase.Bytes()) != 0 {
		return ErrInvalidBlockValidator
	}
	return nil
}

func (d *Dpos) Finalize(chain consensus.IChainReader, header *types.BlockHeader, state *state.StateDB, txs []*types.Transaction, actions []*types.Action, receipts []*types.Receipt, dposContext *types.DposContext) (*types.Block, error) {
	// Accumulate block rewards and commit the final state root
	state.AddBalance(header.Miner, params.BlockReward)
	header.StateRoot = state.IntermediateRoot(true)

	epochContext := &EpochContext{
		Statedb:     state,
		DposContext: dposContext,
		TimeStamp:   header.TimeStamp.Int64(),
	}
	parent := chain.GetBlockByHash(header.PreviousHash)
	if timeOfFirstBlock == 0 {
		if firstBlock := chain.GetBlockByHeight(1); firstBlock != nil {
			timeOfFirstBlock = firstBlock.BlockHeader().TimeStamp.Int64()
		}
	}

	//update mint count trie
	updateMintCnt(parent.BlockHeader().TimeStamp.Int64(), header.TimeStamp.Int64(), header.Miner, dposContext)

	genesis := chain.GetBlockByHeight(0)
	err := epochContext.tryElect(genesis.BlockHeader(), parent.BlockHeader())
	if err != nil {
		return nil, fmt.Errorf("got error when elect next epoch, err: %v", err)
	}
	header.DposContext = dposContext.ToProto()
	return types.NewBlock(header, txs, actions, receipts), nil
}

// UInt64Slice attaches the methods of sort.Interface to []uint64, sorting in increasing order.
type UInt64Slice []uint64

func (s UInt64Slice) Len() int           { return len(s) }
func (s UInt64Slice) Less(i, j int) bool { return s[i] < s[j] }
func (s UInt64Slice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
