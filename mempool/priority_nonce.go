package mempool

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/huandu/skiplist"
)

var (
	_ sdkmempool.Mempool  = (*PriorityNonceMempool)(nil)
	_ sdkmempool.Iterator = (*PriorityNonceIterator)(nil)
)

type TxPriority struct {
	// GetTxPriority returns the priority of the transaction. A priority must be
	// comparable via CompareTxPriority.
	GetTxPriority func(ctx context.Context, tx sdk.Tx) any
	// CompareTxPriority compares two transaction priorities. The result should be
	// 0 if a == b, -1 if a < b, and +1 if a > b.
	CompareTxPriority func(a, b any) int
}

// NewDefaultTxPriority returns a TxPriority comparator using ctx.Priority as
// the defining transaction priority.
func NewDefaultTxPriority() TxPriority {
	return TxPriority{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) any {
			return sdk.UnwrapSDKContext(goCtx).Priority()
		},
		CompareTxPriority: func(a, b any) int {
			switch {
			case a == nil && b == nil:
				return 0
			case a == nil:
				return -1
			case b == nil:
				return 1
			default:
				aPriority := a.(int64)
				bPriority := b.(int64)

				return skiplist.Int64.Compare(aPriority, bPriority)
			}
		},
	}
}

// PriorityNonceMempool is a mempool implementation that stores txs
// in a partially ordered set by 2 dimensions: priority, and sender-nonce
// (sequence number). Internally it uses one priority ordered skip list and one
// skip list per sender ordered by sender-nonce (sequence number). When there
// are multiple txs from the same sender, they are not always comparable by
// priority to other sender txs and must be partially ordered by both sender-nonce
// and priority.
//
// NOTE: This implementation is a fork from the Cosmos SDK. It contains the changes
// implemented in https://github.com/cosmos/cosmos-sdk/pull/15328. If and when
// those changes are merged and included in a tagged v0.47 release, this forked
// implementation can be removed.
type PriorityNonceMempool struct {
	priorityIndex  *skiplist.SkipList
	priorityCounts map[any]int
	senderIndices  map[string]*skiplist.SkipList
	scores         map[txMeta]txMeta
	onRead         func(tx sdk.Tx)
	txReplacement  func(op, np any, oTx, nTx sdk.Tx) bool
	maxTx          int
	txPriority     TxPriority
}

type PriorityNonceIterator struct {
	mempool       *PriorityNonceMempool
	priorityNode  *skiplist.Element
	senderCursors map[string]*skiplist.Element
	sender        string
	nextPriority  any
}

// txMeta stores transaction metadata used in indices
type txMeta struct {
	// nonce is the sender's sequence number
	nonce uint64
	// priority is the transaction's priority
	priority any
	// sender is the transaction's sender
	sender string
	// weight is the transaction's weight, used as a tiebreaker for transactions
	// with the same priority
	weight any
	// senderElement is a pointer to the transaction's element in the sender index
	senderElement *skiplist.Element
}

// skiplistComparable is a comparator for txKeys that first compares priority,
// then weight, then sender, then nonce, uniquely identifying a transaction.
//
// Note, skiplistComparable is used as the comparator in the priority index.
func skiplistComparable(txPriority TxPriority) skiplist.Comparable {
	return skiplist.LessThanFunc(func(a, b any) int {
		keyA := a.(txMeta)
		keyB := b.(txMeta)

		res := txPriority.CompareTxPriority(keyA.priority, keyB.priority)
		if res != 0 {
			return res
		}

		// Weight is used as a tiebreaker for transactions with the same priority.
		// Weight is calculated in a single pass in .Select(...) and so will be 0
		// on .Insert(...).
		res = txPriority.CompareTxPriority(keyA.weight, keyB.weight)
		if res != 0 {
			return res
		}

		// Because weight will be 0 on .Insert(...), we must also compare sender and
		// nonce to resolve priority collisions. If we didn't then transactions with
		// the same priority would overwrite each other in the priority index.
		res = skiplist.String.Compare(keyA.sender, keyB.sender)
		if res != 0 {
			return res
		}

		return skiplist.Uint64.Compare(keyA.nonce, keyB.nonce)
	})
}

type PriorityNonceMempoolOption func(*PriorityNonceMempool)

// PriorityNonceWithOnRead sets a callback to be called when a tx is read from
// the mempool.
func PriorityNonceWithOnRead(onRead func(tx sdk.Tx)) PriorityNonceMempoolOption {
	return func(mp *PriorityNonceMempool) {
		mp.onRead = onRead
	}
}

// PriorityNonceWithTxReplacement sets a callback to be called when duplicated
// transaction nonce detected during mempool insert. An application can define a
// transaction replacement rule based on tx priority or certain transaction fields.
func PriorityNonceWithTxReplacement(txReplacementRule func(op, np any, oTx, nTx sdk.Tx) bool) PriorityNonceMempoolOption {
	return func(mp *PriorityNonceMempool) {
		mp.txReplacement = txReplacementRule
	}
}

// PriorityNonceWithMaxTx sets the maximum number of transactions allowed in the
// mempool with the semantics:
//
// <0: disabled, `Insert` is a no-op
// 0: unlimited
// >0: maximum number of transactions allowed
func PriorityNonceWithMaxTx(maxTx int) PriorityNonceMempoolOption {
	return func(mp *PriorityNonceMempool) {
		mp.maxTx = maxTx
	}
}

// DefaultPriorityMempool returns a priorityNonceMempool with no options.
func DefaultPriorityMempool(txPriority TxPriority) sdkmempool.Mempool {
	return NewPriorityMempool(txPriority)
}

// NewPriorityMempool returns the SDK's default mempool implementation which
// returns txs in a partial order by 2 dimensions; priority, and sender-nonce.
func NewPriorityMempool(txPriority TxPriority, opts ...PriorityNonceMempoolOption) *PriorityNonceMempool {
	mp := &PriorityNonceMempool{
		priorityIndex:  skiplist.New(skiplistComparable(txPriority)),
		priorityCounts: make(map[any]int),
		senderIndices:  make(map[string]*skiplist.SkipList),
		scores:         make(map[txMeta]txMeta),
		txPriority:     txPriority,
	}

	for _, opt := range opts {
		opt(mp)
	}

	return mp
}

// NextSenderTx returns the next transaction for a given sender by nonce order,
// i.e. the next valid transaction for the sender. If no such transaction exists,
// nil will be returned.
func (mp *PriorityNonceMempool) NextSenderTx(sender string) sdk.Tx {
	senderIndex, ok := mp.senderIndices[sender]
	if !ok {
		return nil
	}

	cursor := senderIndex.Front()
	return cursor.Value.(sdk.Tx)
}

// Insert attempts to insert a Tx into the app-side mempool in O(log n) time,
// returning an error if unsuccessful. Sender and nonce are derived from the
// transaction's first signature.
//
// Transactions are unique by sender and nonce. Inserting a duplicate tx is an
// O(log n) no-op.
//
// Inserting a duplicate tx with a different priority overwrites the existing tx,
// changing the total order of the mempool.
func (mp *PriorityNonceMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if mp.maxTx > 0 && mp.CountTx() >= mp.maxTx {
		return sdkmempool.ErrMempoolTxMaxCapacity
	} else if mp.maxTx < 0 {
		return nil
	}

	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("tx must have at least one signer")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	priority := mp.txPriority.GetTxPriority(ctx, tx)
	nonce := sig.Sequence
	key := txMeta{nonce: nonce, priority: priority, sender: sender}

	senderIndex, ok := mp.senderIndices[sender]
	if !ok {
		senderIndex = skiplist.New(skiplist.LessThanFunc(func(a, b any) int {
			return skiplist.Uint64.Compare(b.(txMeta).nonce, a.(txMeta).nonce)
		}))

		// initialize sender index if not found
		mp.senderIndices[sender] = senderIndex
	}

	// Since mp.priorityIndex is scored by priority, then sender, then nonce, a
	// changed priority will create a new key, so we must remove the old key and
	// re-insert it to avoid having the same tx with different priorityIndex indexed
	// twice in the mempool.
	//
	// This O(log n) remove operation is rare and only happens when a tx's priority
	// changes.
	sk := txMeta{nonce: nonce, sender: sender}
	if oldScore, txExists := mp.scores[sk]; txExists {
		if mp.txReplacement != nil && !mp.txReplacement(oldScore.priority, priority, senderIndex.Get(key).Value.(sdk.Tx), tx) {
			return fmt.Errorf(
				"tx doesn't fit the replacement rule, oldPriority: %v, newPriority: %v, oldTx: %v, newTx: %v",
				oldScore.priority,
				priority,
				senderIndex.Get(key).Value.(sdk.Tx),
				tx,
			)
		}

		mp.priorityIndex.Remove(txMeta{
			nonce:    nonce,
			sender:   sender,
			priority: oldScore.priority,
			weight:   oldScore.weight,
		})
		mp.priorityCounts[oldScore.priority]--
	}

	mp.priorityCounts[priority]++

	// Since senderIndex is scored by nonce, a changed priority will overwrite the
	// existing key.
	key.senderElement = senderIndex.Set(key, tx)

	mp.scores[sk] = txMeta{priority: priority}
	mp.priorityIndex.Set(key, tx)

	return nil
}

func (i *PriorityNonceIterator) iteratePriority() sdkmempool.Iterator {
	// beginning of priority iteration
	if i.priorityNode == nil {
		i.priorityNode = i.mempool.priorityIndex.Front()
	} else {
		i.priorityNode = i.priorityNode.Next()
	}

	// end of priority iteration
	if i.priorityNode == nil {
		return nil
	}

	i.sender = i.priorityNode.Key().(txMeta).sender

	nextPriorityNode := i.priorityNode.Next()
	if nextPriorityNode != nil {
		i.nextPriority = nextPriorityNode.Key().(txMeta).priority
	} else {
		i.nextPriority = nil
	}

	return i.Next()
}

func (i *PriorityNonceIterator) Next() sdkmempool.Iterator {
	if i.priorityNode == nil {
		return nil
	}

	cursor, ok := i.senderCursors[i.sender]
	if !ok {
		// beginning of sender iteration
		cursor = i.mempool.senderIndices[i.sender].Front()
	} else {
		// middle of sender iteration
		cursor = cursor.Next()
	}

	// end of sender iteration
	if cursor == nil {
		return i.iteratePriority()
	}

	key := cursor.Key().(txMeta)

	// We've reached a transaction with a priority lower than the next highest
	// priority in the pool.
	if i.mempool.txPriority.CompareTxPriority(key.priority, i.nextPriority) < 0 {
		return i.iteratePriority()
	} else if i.mempool.txPriority.CompareTxPriority(key.priority, i.nextPriority) == 0 {
		// Weight is incorporated into the priority index key only (not sender index)
		// so we must fetch it here from the scores map.
		weight := i.mempool.scores[txMeta{nonce: key.nonce, sender: key.sender}].weight
		if i.mempool.txPriority.CompareTxPriority(weight, i.priorityNode.Next().Key().(txMeta).weight) < 0 {
			return i.iteratePriority()
		}
	}

	i.senderCursors[i.sender] = cursor
	return i
}

func (i *PriorityNonceIterator) Tx() sdk.Tx {
	return i.senderCursors[i.sender].Value.(sdk.Tx)
}

// Select returns a set of transactions from the mempool, ordered by priority
// and sender-nonce in O(n) time. The passed in list of transactions are ignored.
// This is a readonly operation, the mempool is not modified.
//
// The maxBytes parameter defines the maximum number of bytes of transactions to
// return.
func (mp *PriorityNonceMempool) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	if mp.priorityIndex.Len() == 0 {
		return nil
	}

	mp.reorderPriorityTies()

	iterator := &PriorityNonceIterator{
		mempool:       mp,
		senderCursors: make(map[string]*skiplist.Element),
	}

	return iterator.iteratePriority()
}

type reorderKey struct {
	deleteKey txMeta
	insertKey txMeta
	tx        sdk.Tx
}

func (mp *PriorityNonceMempool) reorderPriorityTies() {
	node := mp.priorityIndex.Front()

	var reordering []reorderKey
	for node != nil {
		key := node.Key().(txMeta)
		if mp.priorityCounts[key.priority] > 1 {
			newKey := key
			newKey.weight = senderWeight(mp.txPriority, key.senderElement)
			reordering = append(reordering, reorderKey{deleteKey: key, insertKey: newKey, tx: node.Value.(sdk.Tx)})
		}

		node = node.Next()
	}

	for _, k := range reordering {
		mp.priorityIndex.Remove(k.deleteKey)
		delete(mp.scores, txMeta{nonce: k.deleteKey.nonce, sender: k.deleteKey.sender})
		mp.priorityIndex.Set(k.insertKey, k.tx)
		mp.scores[txMeta{nonce: k.insertKey.nonce, sender: k.insertKey.sender}] = k.insertKey
	}
}

// senderWeight returns the weight of a given tx (t) at senderCursor. Weight is
// defined as the first (nonce-wise) same sender tx with a priority not equal to
// t. It is used to resolve priority collisions, that is when 2 or more txs from
// different senders have the same priority.
func senderWeight(txPriority TxPriority, senderCursor *skiplist.Element) any {
	if senderCursor == nil {
		return 0
	}

	weight := senderCursor.Key().(txMeta).priority
	senderCursor = senderCursor.Next()
	for senderCursor != nil {
		p := senderCursor.Key().(txMeta).priority
		if txPriority.CompareTxPriority(p, weight) != 0 {
			weight = p
		}

		senderCursor = senderCursor.Next()
	}

	return weight
}

// CountTx returns the number of transactions in the mempool.
func (mp *PriorityNonceMempool) CountTx() int {
	return mp.priorityIndex.Len()
}

// Remove removes a transaction from the mempool in O(log n) time, returning an
// error if unsuccessful.
func (mp *PriorityNonceMempool) Remove(tx sdk.Tx) error {
	sigs, err := tx.(signing.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return fmt.Errorf("attempted to remove a tx with no signatures")
	}

	sig := sigs[0]
	sender := sdk.AccAddress(sig.PubKey.Address()).String()
	nonce := sig.Sequence

	scoreKey := txMeta{nonce: nonce, sender: sender}
	score, ok := mp.scores[scoreKey]
	if !ok {
		return sdkmempool.ErrTxNotFound
	}
	tk := txMeta{nonce: nonce, priority: score.priority, sender: sender, weight: score.weight}

	senderTxs, ok := mp.senderIndices[sender]
	if !ok {
		return fmt.Errorf("sender %s not found", sender)
	}

	mp.priorityIndex.Remove(tk)
	senderTxs.Remove(tk)
	delete(mp.scores, scoreKey)
	mp.priorityCounts[score.priority]--

	return nil
}

func IsEmpty(mempool sdkmempool.Mempool) error {
	mp := mempool.(*PriorityNonceMempool)
	if mp.priorityIndex.Len() != 0 {
		return fmt.Errorf("priorityIndex not empty")
	}

	var countKeys []any
	for k := range mp.priorityCounts {
		countKeys = append(countKeys, k)
	}

	for _, k := range countKeys {
		if mp.priorityCounts[k] != 0 {
			return fmt.Errorf("priorityCounts not zero at %v, got %v", k, mp.priorityCounts[k])
		}
	}

	var senderKeys []string
	for k := range mp.senderIndices {
		senderKeys = append(senderKeys, k)
	}

	for _, k := range senderKeys {
		if mp.senderIndices[k].Len() != 0 {
			return fmt.Errorf("senderIndex not empty for sender %v", k)
		}
	}

	return nil
}
