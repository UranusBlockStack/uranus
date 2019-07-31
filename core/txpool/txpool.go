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

package txpool

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/UranusBlockStack/uranus/common/log"
	"github.com/UranusBlockStack/uranus/common/utils"
	"github.com/UranusBlockStack/uranus/core/state"
	"github.com/UranusBlockStack/uranus/core/types"
	"github.com/UranusBlockStack/uranus/feed"
	"github.com/UranusBlockStack/uranus/params"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

// TxPool contains all currently known transactions.
type TxPool struct {
	config        *Config
	chainconfig   *params.ChainConfig
	chain         blockChainHelper
	gasPrice      *big.Int
	txFeed        feed.Feed
	txScription   feed.Subscription
	chainBlockCh  chan feed.BlockAndLogsEvent
	chainBlockSub feed.Subscription
	signer        types.Signer

	currentState *state.StateDB      // Current state in the blockchain head
	tmpState     *state.ManagedState // Pending state tracking virtual nonces
	curMaxGas    uint64              // Current gas limit for transaction caps

	pending map[utils.Address]*txList   // All currently processable transactions
	queue   map[utils.Address]*txList   // Queued but non-processable transactions
	beats   map[utils.Address]time.Time // Last heartbeat from each known account

	dList *deferredList

	txs       *allTxs    // All transactions cache
	priceList *priceList // All transactions sorted by price

	addTxsChan chan types.Transactions

	mu sync.RWMutex
	wg sync.WaitGroup // for shutdown sync

}

// New creates a new transaction pool to gather, sort and filter inbound transactions from the network.
func New(config *Config, chainconfig *params.ChainConfig, chain blockChainHelper) *TxPool {
	// check priceLimit and priceBump are set
	if config.PriceLimit < 1 {
		log.Warnf("Sanitizing invalid txpool price limit provided: %v updated: %v", config.PriceLimit, DefaultTxPoolConfig.PriceLimit)
		config.PriceLimit = DefaultTxPoolConfig.PriceLimit
	}
	if config.PriceBump < 1 {
		log.Warnf("Sanitizing invalid txpool price bump provided: %v updated: %v", config.PriceBump, DefaultTxPoolConfig.PriceBump)
		config.PriceBump = DefaultTxPoolConfig.PriceBump
	}

	tp := &TxPool{}
	tp.config = config
	tp.chainconfig = chainconfig
	tp.chain = chain
	tp.signer = types.Signer{}
	tp.pending = make(map[utils.Address]*txList)
	tp.queue = make(map[utils.Address]*txList)
	tp.beats = make(map[utils.Address]time.Time)
	tp.txs = newallTxs()
	tp.dList = newDeferredList()
	tp.chainBlockCh = make(chan feed.BlockAndLogsEvent, 10)
	tp.gasPrice = new(big.Int).SetUint64(config.PriceLimit)
	tp.priceList = newpriceList(tp.txs)
	tp.resetTxpoolState(nil, chain.CurrentBlock())

	tp.chainBlockSub = tp.chain.SubscribeChainBlockEvent(tp.chainBlockCh)

	tp.addTxsChan = make(chan types.Transactions, tp.config.GlobalQueue)

	tp.wg.Add(1)
	go tp.loop()
	go tp.txloop()

	return tp
}

var timeoutInterval = time.Minute

// loop  waiting for  reacting to outside blockchain events and check timeout trnasaction
func (tp *TxPool) loop() {
	defer tp.wg.Done()
	timeout := time.NewTicker(timeoutInterval)
	defer timeout.Stop()

	block := tp.chain.CurrentBlock()

	// Keep waiting for and reacting to the various events
	for {
		select {
		// remove time out transactions
		case <-timeout.C:
			tp.mu.Lock()
			for addr := range tp.queue {
				// Any non-locals old enough should be removed
				if time.Since(tp.beats[addr]) > tp.config.TimeoutDuration {
					for _, tx := range tp.queue[addr].Flatten() {
						tp.removeTx(tx.Hash(), true)
					}
				}
			}
			tp.mu.Unlock()
		// Be unsubscribed due to system stopped
		case <-tp.chainBlockSub.Err():
			return
		// Handle chainBlockEvent
		case ev := <-tp.chainBlockCh:
			if ev.Block != nil {
				tp.mu.Lock()

				tp.dList.Remove(ev.Block.Actions())
				tp.resetTxpoolState(block, ev.Block)
				block = ev.Block
				tp.mu.Unlock()
			}

		}
	}
}

func (tp *TxPool) txloop() {
	for {
		select {
		case txs := <-tp.addTxsChan:
			tp.addTxs(txs)
		}
	}
}

// Stop stop the transaction pool.
func (tp *TxPool) Stop() {
	if tp.txScription != nil {
		tp.txScription.Unsubscribe()
	}

	tp.chainBlockSub.Unsubscribe()
	tp.wg.Wait()
	log.Info("Transaction pool service stopped")
}

// reinjectTxs reorging an old state, reinject all dropped transactions
func (tp *TxPool) reinjectTxs(old, new *types.Block) types.Transactions {
	var reinject types.Transactions
	if old != nil && old.Hash() != new.PreviousHash() {
		oldHeight := old.Height().Uint64()
		newHeight := new.Height().Uint64()
		if depth := uint64(math.Abs(float64(oldHeight) - float64(newHeight))); depth > 64 {
			log.Debugf("Skipping deep transaction reorg depth: %v ", depth)
		} else {
			// Reorg seems shallow enough to pull in all transactions into memory
			var discarded, included types.Transactions

			var (
				rem = tp.chain.GetBlock(old.Hash())
				add = tp.chain.GetBlock(new.Hash())
			)
			for rem.Height().Uint64() > add.Height().Uint64() {
				discarded = append(discarded, rem.Transactions()...)
				if rem = tp.chain.GetBlock(rem.PreviousHash()); rem == nil {
					log.Errorf("Unrooted old chain seen by tx pool block height: %v ,hash: %v", old.Height(), old.Hash())
					return nil
				}
			}
			for add.Height().Uint64() > rem.Height().Uint64() {
				included = append(included, add.Transactions()...)
				if add = tp.chain.GetBlock(add.PreviousHash()); add == nil {
					log.Errorf("Unrooted new chain seen by tx pool block height: %v ,hash: %v", new.Height(), new.Hash())
					return nil
				}
			}
			for rem.Hash() != add.Hash() {
				discarded = append(discarded, rem.Transactions()...)
				if rem = tp.chain.GetBlock(rem.PreviousHash()); rem == nil {
					log.Errorf("Unrooted old chain seen by tx pool block height: %v ,hash: %v", old.Height(), old.Hash())
					return nil
				}
				included = append(included, add.Transactions()...)
				if add = tp.chain.GetBlock(add.PreviousHash()); add == nil {
					log.Errorf("Unrooted new chain seen by tx pool block height: %v ,hash: %v", new.Height(), new.Hash())
					return nil
				}
			}
			reinject = TxDifference(discarded, included)
		}
	}
	return reinject
}

func (tp *TxPool) processTxslist() {
	tp.demotePending()
	// Update all accounts to the latest known usable nonce
	for addr, list := range tp.pending {
		txs := list.Flatten()
		tp.tmpState.SetNonce(addr, txs[len(txs)-1].Nonce()+1)
	}
	tp.promoteQueue(nil)
}

func (tp *TxPool) resetTxpoolState(old, new *types.Block) {
	txs := tp.reinjectTxs(old, new)
	// Initialize the internal state to the current head
	if new == nil {
		new = tp.chain.CurrentBlock()
	}
	statedb, err := tp.chain.StateAt(new.StateRoot())
	if err != nil {
		log.Errorf("Failed to reset txpool state err %v", err)
		return
	}
	tp.currentState = statedb
	tp.tmpState = state.ManageState(statedb)
	tp.curMaxGas = new.GasLimit()

	// Inject any transactions discarded due to reorgs
	log.Debugf("Reinjecting stale transactions count %v", len(txs))
	tp.addTxsLocked(txs)

	tp.processTxslist()
}

// SubscribeNewTxsEvent registers a subscription of NewTxsEvent and starts sending event to the given channel.
func (tp *TxPool) SubscribeNewTxsEvent(ch chan<- feed.NewTxsEvent) feed.Subscription {
	tp.txScription = tp.txFeed.Subscribe(ch)
	return tp.txScription
}

// GasPrice returns the current gas price enforced by the transaction tp.
func (tp *TxPool) GasPrice() *big.Int {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return new(big.Int).Set(tp.gasPrice)
}

// SetGasPrice updates the minimum price required by the transaction pool for a
// new transaction, and drops all transactions below this threshold.
func (tp *TxPool) SetGasPrice(price *big.Int) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.gasPrice = price
	for _, pn := range tp.priceList.Cap(price) {
		tp.removeTx(pn.hash, false)
	}
	log.Debugf("Transaction pool price threshold updated price %v", price)
}

// State returns the virtual managed state of the transaction tp.
func (tp *TxPool) State() *state.ManagedState {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	return tp.tmpState
}

// Stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (tp *TxPool) Stats() (int, int) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.stats()
}

// stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (tp *TxPool) stats() (int, int) {
	pending := 0
	for _, list := range tp.pending {
		pending += list.Len()
	}
	queued := 0
	for _, list := range tp.queue {
		queued += list.Len()
	}
	return pending, queued
}

// Content retrieves the data content of the transaction pool, returning all the
// pending as well as queued transactions, grouped by account and sorted by nonce.
func (tp *TxPool) Content() (map[utils.Address]types.Transactions, map[utils.Address]types.Transactions) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	pending := make(map[utils.Address]types.Transactions)
	for addr, list := range tp.pending {
		pending[addr] = list.Flatten()
	}
	queued := make(map[utils.Address]types.Transactions)
	for addr, list := range tp.queue {
		queued[addr] = list.Flatten()
	}
	return pending, queued
}

// Pending retrieves all currently processable transactions, groupped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (tp *TxPool) Pending() (map[utils.Address]types.Transactions, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	pending := make(map[utils.Address]types.Transactions)
	for addr, list := range tp.pending {
		pending[addr] = list.Flatten()
	}
	return pending, nil
}

// Actions get actions
func (tp *TxPool) Actions() []*types.Action {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	now := big.NewInt(time.Now().Unix())
	return tp.dList.Cap(new(big.Int).Sub(now, big.NewInt(tp.chainconfig.DelayDuration)))
}

// validateTx checks whether a transaction is valid according to the consensus
// rules and adheres to some heuristic limits of the local node (price and size).
func (tp *TxPool) validateTx(tx *types.Transaction) error {
	// Heuristic limit, reject transactions over 32KB to prevent DOS attacks
	if err := tx.Validate(tp.chainconfig); err != nil {
		return err
	}
	if tx.Size() > 32*1024 {
		return ErrOversizedData
	}
	// Transactions can't be negative.
	if tx.Value().Sign() < 0 {
		return ErrNegativeValue
	}
	// Ensure the transaction doesn't exceed the current block limit gas.
	if tp.curMaxGas < tx.Gas() {
		return ErrGasLimit
	}
	// Make sure the transaction is signed properly
	from, err := tx.Sender(tp.signer)
	if err != nil {
		return ErrInvalidSender
	}
	if tx.Type() == types.LogoutCandidate && bytes.Compare(from.Bytes(), utils.HexToAddress(tp.chainconfig.GenesisCandidate).Bytes()) == 0 {
		return fmt.Errorf("genesis candidate not allow logout")
	}
	// Drop transactions under our own minimal accepted gas price
	if tp.gasPrice.Cmp(tx.GasPrice()) > 0 {
		return ErrUnderPriced
	}

	// Ensure the transaction adheres to nonce ordering
	if tp.currentState.GetNonce(from) > tx.Nonce() {
		return ErrNonceTooLow
	}
	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if tp.currentState.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}
	intrGas, err := IntrinsicGas(tx.Payload(), tx.Type(), len(tx.Tos()) == 0 && tx.Type() == types.Binary)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return ErrIntrinsicGas
	}
	return nil
}

// add validates a transaction and inserts it into the non-executable queue for later pending promotion and execution.
func (tp *TxPool) add(tx *types.Transaction) (bool, error) {
	// If the transaction is already known, discard it
	hash := tx.Hash()
	if tp.txs.Get(hash) != nil {
		log.Warnf("Discarding already known transaction hash: %v, nonce: %v", hash, tx.Nonce())
		return false, fmt.Errorf("known transaction: %x", hash)
	}
	// If the transaction fails basic validation, discard it
	if err := tp.validateTx(tx); err != nil {
		log.Warnf("Discarding invalid transaction hash: %v, nonce: %v,err: %v", hash, tx.Nonce(), err)
		return false, err
	}
	// If the transaction pool is full, discard underpriceList transactions
	if uint64(tp.txs.Count()) >= tp.config.GlobalSlots+tp.config.GlobalQueue {
		// If the new transaction is underpriceList, don't accept it
		if tp.priceList.Underpriced(tx) {
			log.Warnf("Discarding underpriceList transaction hash: %v ,price: %v", hash, tx.GasPrice())
			return false, ErrUnderPriced
		}
		// New transaction is better than our worse ones, make room for it
		drop := tp.priceList.Discard(tp.txs.Count() - int(tp.config.GlobalSlots+tp.config.GlobalQueue-1))
		for _, pn := range drop {
			log.Warnf("Discarding freshly underpriceList transaction hash: %v ,price: %v", tx.Hash(), tx.GasPrice())
			tp.removeTx(pn.hash, false)
		}
	}
	// If the transaction is replacing an already pending one, do directly
	from, _ := tx.Sender(tp.signer)
	if list := tp.pending[from]; list != nil && list.Overlaps(tx) {
		// Nonce already pending, check if required price bump is met
		inserted, old := list.Add(tx, tp.config.PriceBump)
		if !inserted {
			return false, ErrReplaceUnderpriced
		}
		// New transaction is better, replace old one
		if old != nil {
			tp.txs.Remove(old.Hash())
			tp.priceList.Removed()
		}
		tp.txs.Add(tx)
		tp.priceList.Put(tx)

		log.Warnf("Pooled new executable transaction hash: %v, from: %v, to: %v", hash, from, tx.Tos())

		go tp.txFeed.Send(feed.NewTxsEvent{Txs: types.Transactions{tx}})

		return old != nil, nil
	}
	// New transaction isn't replacing a pending one, push into queue
	replace, err := tp.enqueueTx(hash, tx)
	if err != nil {
		return false, err
	}

	log.Warnf("Pooled new future transaction hash: %v, from: %v ,to: %v", hash, from, tx.Tos())
	return replace, nil
}

// enqueueTx inserts a new transaction into the non-executable transaction queue.
//
// Note, this method assumes the pool lock is held!
func (tp *TxPool) enqueueTx(hash utils.Hash, tx *types.Transaction) (bool, error) {
	// Try to insert the transaction into the future queue
	from, _ := tx.Sender(tp.signer)
	if tp.queue[from] == nil {
		tp.queue[from] = newTxList(false)
	}
	inserted, old := tp.queue[from].Add(tx, tp.config.PriceBump)
	if !inserted {
		return false, ErrReplaceUnderpriced
	}
	// Discard any previous transaction and mark this
	if old != nil {
		tp.txs.Remove(old.Hash())
		tp.priceList.Removed()
	}
	if tp.txs.Get(hash) == nil {
		tp.txs.Add(tx)
		tp.priceList.Put(tx)
	}
	return old != nil, nil
}

// promoteTx adds a transaction to the pending (processable) list of transactions
// and returns whether it was inserted or an older was better.
//
// Note, this method assumes the pool lock is held!
func (tp *TxPool) promoteTx(addr utils.Address, hash utils.Hash, tx *types.Transaction) bool {
	// Try to insert the transaction into the pending queue
	if tp.pending[addr] == nil {
		tp.pending[addr] = newTxList(true)
	}
	list := tp.pending[addr]

	inserted, old := list.Add(tx, tp.config.PriceBump)
	if !inserted {
		// An older transaction was better, discard this
		tp.txs.Remove(hash)
		tp.priceList.Removed()
		return false
	}
	// Otherwise discard any previous transaction and mark this
	if old != nil {
		tp.txs.Remove(old.Hash())
		tp.priceList.Removed()
	}
	// Failsafe to work around direct pending inserts (tests)
	if tp.txs.Get(hash) == nil {
		tp.txs.Add(tx)
		tp.priceList.Put(tx)
	}
	// Set the potentially new pending nonce and notify any subsystems of the new tx
	tp.beats[addr] = time.Now()
	tp.tmpState.SetNonce(addr, tx.Nonce()+1)

	return true
}

func (tp *TxPool) AddAction(a *types.Action) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	tp.dList.Put(a)
}

func (tp *TxPool) AddTx(tx *types.Transaction) error {
	return tp.addTx(tx)
}

func (tp *TxPool) AddTxsChan(txs types.Transactions) bool {
	if len(tp.addTxsChan) == int(tp.config.GlobalQueue) {
		for _, tx := range txs {
			from, _ := tx.Sender(tp.signer)
			log.Warningf("Removed cap-exceeding handle transaction hash: %v, addr: %v, nonce: %v", tx.Hash(), from.String(), tx.Nonce())
		}
		return false
	}
	tp.addTxsChan <- txs
	return true
}

func (tp *TxPool) AddTxs(txs []*types.Transaction) []error {
	return tp.addTxs(txs)
}

// addTx enqueues a single transaction into the pool if it is valid.
func (tp *TxPool) addTx(tx *types.Transaction) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Try to inject the transaction and update any state
	replace, err := tp.add(tx)
	if err != nil {
		return err
	}
	// If we added a new transaction, run promotion checks and return
	if !replace {
		from, _ := tx.Sender(tp.signer)
		tp.promoteQueue([]utils.Address{from})
	}
	return nil
}

// addTxs attempts to queue a batch of transactions if they are valid.
func (tp *TxPool) addTxs(txs []*types.Transaction) []error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	return tp.addTxsLocked(txs)
}

// addTxsLocked attempts to queue a batch of transactions if they are valid,
// whilst assuming the transaction pool lock is already held.
func (tp *TxPool) addTxsLocked(txs []*types.Transaction) []error {
	// Add the batch of transaction, tracking the accepted ones
	dirty := make(map[utils.Address]struct{})
	errs := make([]error, len(txs))
	for i, tx := range txs {
		var replace bool
		if replace, errs[i] = tp.add(tx); errs[i] == nil {
			if !replace {
				from, _ := tx.Sender(tp.signer) // already validated
				dirty[from] = struct{}{}
			}
		}
	}
	// Only reprocess the internal state if something was actually added
	if len(dirty) > 0 {
		addrs := make([]utils.Address, 0, len(dirty))
		for addr := range dirty {
			addrs = append(addrs, addr)
		}
		tp.promoteQueue(addrs)
	}
	return errs
}

// Status returns the status (unknown/pending/queued) of a batch of transactions
// identified by their hashes.
func (tp *TxPool) Status(hashes []utils.Hash) []TxStatus {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	status := make([]TxStatus, len(hashes))
	for i, hash := range hashes {
		if tx := tp.txs.Get(hash); tx != nil {
			from, _ := tx.Sender(tp.signer)
			if tp.pending[from] != nil && tp.pending[from].txs.items[tx.Nonce()] != nil {
				status[i] = TxStatusPending
			} else {
				status[i] = TxStatusQueued
			}
		}
	}
	return status
}

// Get returns a transaction if it is contained in the pool
// and nil otherwise.
func (tp *TxPool) Get(hash utils.Hash) *types.Transaction {
	return tp.txs.Get(hash)
}

// removeTx removes a single transaction from the queue, moving all subsequent
// transactions back to the future queue.
func (tp *TxPool) removeTx(hash utils.Hash, outofbound bool) {
	// Fetch the transaction we wish to delete
	tx := tp.txs.Get(hash)
	if tx == nil {
		return
	}
	addr, _ := tx.Sender(tp.signer)

	// Remove it from the list of known transactions
	tp.txs.Remove(hash)
	if outofbound {
		tp.priceList.Removed()
	}
	// Remove the transaction from the pending lists and reset the account nonce
	if pending := tp.pending[addr]; pending != nil {
		if removed, invalids := pending.Remove(tx); removed {
			// If no more pending transactions are left, remove the list
			if pending.Empty() {
				delete(tp.pending, addr)
				delete(tp.beats, addr)
			}
			// Postpone any invalidated transactions
			for _, tx := range invalids {
				tp.enqueueTx(tx.Hash(), tx)
			}
			// Update the account nonce if needed
			if nonce := tx.Nonce(); tp.tmpState.GetNonce(addr) > nonce {
				tp.tmpState.SetNonce(addr, nonce)
			}
			return
		}
	}
	// Transaction is in the future queue
	if future := tp.queue[addr]; future != nil {
		future.Remove(tx)
		if future.Empty() {
			delete(tp.queue, addr)
		}
	}
}

// promoteQueue moves transactions that have become processable from the future queue to the set of pending transactions.
func (tp *TxPool) promoteQueue(accounts []utils.Address) {
	// Track the promoted transactions to broadcast them at once
	var promoted []*types.Transaction

	// Gather all the accounts potentially needing updates
	if accounts == nil {
		accounts = make([]utils.Address, 0, len(tp.queue))
		for addr := range tp.queue {
			accounts = append(accounts, addr)
		}
	}
	// Iterate over all accounts and promote any executable transactions
	for _, addr := range accounts {
		list := tp.queue[addr]
		if list == nil {
			continue // Just in case someone calls with a non existing account
		}
		// Drop all transactions that are deemed too old (low nonce)
		for _, tx := range list.Forward(tp.currentState.GetNonce(addr)) {
			hash := tx.Hash()
			log.Warnf("Removed old queued transaction hash: %v, addr: %v, nonce: %v", hash, addr.String(), tx.Nonce())
			tp.txs.Remove(hash)
			tp.priceList.Removed()
		}
		// Drop all transactions that are too costly (low balance or out of gas)
		drops, _ := list.Filter(tp.currentState.GetBalance(addr), tp.curMaxGas)
		for _, tx := range drops {
			hash := tx.Hash()
			log.Warnf("Removed unpayable queued transaction hash: %v, addr: %v, nonce: %v", hash, addr.String(), tx.Nonce())
			tp.txs.Remove(hash)
			tp.priceList.Removed()
		}
		// Gather all executable transactions and promote them
		for _, tx := range list.Ready(tp.tmpState.GetNonce(addr)) {
			hash := tx.Hash()
			if tp.promoteTx(addr, hash, tx) {
				log.Warnf("Promoting queued transaction hash: %v, addr: %v, nonce: %v", hash, addr.String(), tx.Nonce())
				promoted = append(promoted, tx)
			}
		}
		// Drop all transactions over the allowed limit
		for _, tx := range list.Cap(int(tp.config.AccountQueue)) {
			hash := tx.Hash()
			tp.txs.Remove(hash)
			tp.priceList.Removed()
			log.Warningf("Removed cap-exceeding queued transaction hash: %v, addr: %v, nonce: %v", hash, addr.String(), tx.Nonce())
		}

		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(tp.queue, addr)
		}
	}
	// Notify subsystem for new promoted transactions.
	if len(promoted) > 0 {
		tp.txFeed.Send(feed.NewTxsEvent{Txs: promoted})
	}
	// If the pending limit is overflown, start equalizing allowances
	pending := uint64(0)
	for _, list := range tp.pending {
		pending += uint64(list.Len())
	}
	if pending > tp.config.GlobalSlots {
		// Assemble a spam order to penalize large transactors first
		spammers := prque.New()
		for addr, list := range tp.pending {
			// Only evict transactions from high rollers
			if uint64(list.Len()) > tp.config.AccountSlots {
				spammers.Push(addr, float32(list.Len()))
			}
		}
		// Gradually drop transactions from offenders
		offenders := []utils.Address{}
		for pending > tp.config.GlobalSlots && !spammers.Empty() {
			// Retrieve the next offender if not local address
			offender, _ := spammers.Pop()
			offenders = append(offenders, offender.(utils.Address))

			// Equalize balances until all the same or below threshold
			if len(offenders) > 1 {
				// Calculate the equalization threshold for all current offenders
				threshold := tp.pending[offender.(utils.Address)].Len()

				// Iteratively reduce all offenders until below limit or threshold reached
				for pending > tp.config.GlobalSlots && tp.pending[offenders[len(offenders)-2]].Len() > threshold {
					for i := 0; i < len(offenders)-1; i++ {
						list := tp.pending[offenders[i]]
						for _, tx := range list.Cap(list.Len() - 1) {
							// Drop the transaction from the global pools too
							hash := tx.Hash()
							tp.txs.Remove(hash)
							tp.priceList.Removed()

							// Update the account nonce to the dropped transaction
							if nonce := tx.Nonce(); tp.tmpState.GetNonce(offenders[i]) > nonce {
								tp.tmpState.SetNonce(offenders[i], nonce)
							}
							log.Warnf("Removed fairness-exceeding pending transaction hash: %v ", hash)
						}
						pending--
					}
				}
			}
		}
		// If still above threshold, reduce to limit or min allowance
		if pending > tp.config.GlobalSlots && len(offenders) > 0 {
			for pending > tp.config.GlobalSlots && uint64(tp.pending[offenders[len(offenders)-1]].Len()) > tp.config.AccountSlots {
				for _, addr := range offenders {
					list := tp.pending[addr]
					for _, tx := range list.Cap(list.Len() - 1) {
						// Drop the transaction from the global pools too
						hash := tx.Hash()
						tp.txs.Remove(hash)
						tp.priceList.Removed()

						// Update the account nonce to the dropped transaction
						if nonce := tx.Nonce(); tp.tmpState.GetNonce(addr) > nonce {
							tp.tmpState.SetNonce(addr, nonce)
						}
						log.Warnf("Removed fairness-exceeding pending transaction hash: %v", hash)
					}
					pending--
				}
			}
		}
	}
	// If we've queued more transactions than the hard limit, drop oldest ones
	queued := uint64(0)
	for _, list := range tp.queue {
		queued += uint64(list.Len())
	}
	if queued > tp.config.GlobalQueue {
		// Sort all accounts with queued transactions by heartbeat
		addresses := make(addresssByHeartbeat, 0, len(tp.queue))
		for addr := range tp.queue {
			addresses = append(addresses, addressByHeartbeat{addr, tp.beats[addr]})
		}
		sort.Sort(addresses)
		// Drop transactions until the total is below the limit or only locals remain
		for drop := queued - tp.config.GlobalQueue; drop > 0 && len(addresses) > 0; {

			addr := addresses[len(addresses)-1]
			list := tp.queue[addr.address]

			addresses = addresses[:len(addresses)-1]

			// Drop all transactions if they are less than the overflow
			if size := uint64(list.Len()); size <= drop {
				for _, tx := range list.Flatten() {
					tp.removeTx(tx.Hash(), true)
				}
				drop -= size
				continue
			}
			// Otherwise drop only last few transactions
			txs := list.Flatten()
			for i := len(txs) - 1; i >= 0 && drop > 0; i-- {
				tp.removeTx(txs[i].Hash(), true)
				drop--
			}
		}
	}
}

// demotePending removes invalid and processed transactions from the pools
// executable/pending queue and any subsequent transactions that become unexecutable
// are moved back into the future queue.
func (tp *TxPool) demotePending() {
	// Iterate over all accounts and demote any non-executable transactions
	for addr, list := range tp.pending {
		nonce := tp.currentState.GetNonce(addr)

		// Drop all transactions that are deemed too old (low nonce)
		for _, tx := range list.Forward(nonce) {
			hash := tx.Hash()
			log.Warnf("Removed old pending transaction hash: %v", hash)
			tp.txs.Remove(hash)
			tp.priceList.Removed()
		}
		// Drop all transactions that are too costly (low balance or out of gas), and queue any invalids back for later
		drops, invalids := list.Filter(tp.currentState.GetBalance(addr), tp.curMaxGas)
		for _, tx := range drops {
			hash := tx.Hash()
			log.Warnf("Removed unpayable pending transaction hash: %v", hash)
			tp.txs.Remove(hash)
			tp.priceList.Removed()
		}
		for _, tx := range invalids {
			hash := tx.Hash()
			log.Warnf("Demoting pending transaction hash: %v", hash)
			tp.enqueueTx(hash, tx)
		}
		// If there's a gap in front, warn (should never happen) and postpone all transactions
		if list.Len() > 0 && list.txs.Get(nonce) == nil {
			for _, tx := range list.Cap(0) {
				hash := tx.Hash()
				log.Errorf("Demoting invalidated transaction hash: %v", hash)
				tp.enqueueTx(hash, tx)
			}
		}
		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(tp.pending, addr)
			delete(tp.beats, addr)
		}
	}
}
