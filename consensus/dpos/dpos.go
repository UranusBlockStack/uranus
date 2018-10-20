package dpos

import (
	"bytes"
	"errors"
	"math/big"
	"time"

	"github.com/UranusBlockStack/uranus/common/crypto"
	"github.com/UranusBlockStack/uranus/common/crypto/sha3"
	"github.com/UranusBlockStack/uranus/common/rlp"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/consensus"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/params"
)

const (
	extraVanity        = 32   // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal          = 65   // Fixed number of extra-data suffix bytes reserved for signer seal
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

	blockInterval    = int64(10)
	epochInterval    = int64(86400)
	maxValidatorSize = 21
	safeSize         = maxValidatorSize*2/3 + 1
	consensusSize    = maxValidatorSize*2/3 + 1
)

var (
	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")
	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	errInvalidDifficulty = errors.New("invalid difficulty")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp           = errors.New("invalid timestamp")
	ErrWaitForPrevBlock           = errors.New("wait for last block arrived")
	ErrMintFutureBlock            = errors.New("mint the future block")
	ErrMismatchSignerAndValidator = errors.New("mismatch block signer and validator")
	ErrInvalidBlockValidator      = errors.New("invalid block validator")
	ErrInvalidMintBlockTime       = errors.New("invalid time to mint the block")
	ErrNilBlockHeader             = errors.New("nil block header returned")
)
var (
	big0  = big.NewInt(0)
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)

	timeOfFirstBlock = int64(0)

	confirmedBlockHead = []byte("confirmed-block-head")
)

// DposConfig is the consensus engine configs for delegated proof-of-stake based sealing.
type DposConfig struct {
	Validators []utils.Address `json:"validators"`
}

type SignerFn func(utils.Address, []byte) ([]byte, error)
type Dpos struct {
	signFn SignerFn
}

func NewDpos(signFn SignerFn) *Dpos {
	d := &Dpos{
		signFn: signFn,
	}
	return d
}

func prevSlot(now int64) int64 {
	return int64((now-1)/blockInterval) * blockInterval
}

func nextSlot(now int64) int64 {
	return int64((now+blockInterval-1)/blockInterval) * blockInterval
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
		//header.DposContext.Root(),
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
	header.ExtraData = append(header.ExtraData, make([]byte, extraSeal)...)
	block = block.WithSeal(header)
	// TODO
	return d.mine(block, stop)
}

func (d *Dpos) mine(block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.BlockHeader()
	if header == nil || header.Height == nil {
		return nil, consensus.ErrUnknownBlock
	}

	now := time.Now().Unix()
	delay := nextSlot(now) - now
	if delay > 0 {
		select {
		case <-stop:
			return nil, nil
		case <-time.After(time.Duration(delay) * time.Second):
		}
	}
	header.TimeStamp.SetInt64(time.Now().Unix())

	// time's up, sign the block
	sighash, err := d.signFn(header.Miner, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	header.ExtraData = append(header.ExtraData[len(header.ExtraData)-extraSeal:], sighash...)
	return block.WithSeal(header), nil
}

func (d *Dpos) Author(header *types.BlockHeader) (utils.Address, error) {
	return header.Miner, nil
}

func (d *Dpos) CalcDifficulty(config *params.ChainConfig, time uint64, parent *types.BlockHeader) *big.Int {
	return big.NewInt(1)
}

// func (d *Dpos) VerifyHeader(chain consensus.IChainReader, header *types.BlockHeader, seal bool) error {
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
// 	// Check that the extra-data contains both the vanity and signature
// 	if len(header.ExtraData) < extraVanity {
// 		return errMissingVanity
// 	}
// 	if len(header.ExtraData) < extraVanity+extraSeal {
// 		return errMissingSignature
// 	}
// 	// Difficulty always 1
// 	if header.Difficulty.Uint64() != 1 {
// 		return errInvalidDifficulty
// 	}
// 	if parentHeader.TimeStamp.Uint64()+uint64(blockInterval) > header.TimeStamp.Uint64() {
// 		return ErrInvalidTimestamp
// 	}
// 	return nil
// }

func (d *Dpos) VerifySeal(chain consensus.IChainReader, header *types.BlockHeader) error {
	if header == nil || header.Height == nil {
		return consensus.ErrUnknownBlock
	}

	number := header.Height.Uint64()
	parent := chain.GetBlockByHeight(number)
	if bytes.Compare(parent.Hash().Bytes(), header.PreviousHash.Bytes()) != 0 {
		return consensus.ErrUnknownAncestor
	}

	signer, err := ecrecover(header)
	if err != nil {
		return err
	}

	if bytes.Compare(signer.Bytes(), header.Miner.Bytes()) != 0 {
		return ErrMismatchSignerAndValidator
	}

	//TODO
	// if bytes.Compare(validator.Bytes(), header.Miner.Bytes()) != 0 {
	// 	return ErrInvalidBlockValidator
	// }
	return nil
}

func (d *Dpos) CheckValidator(lastBlock *types.Block, now int64) error {
	return nil
}
