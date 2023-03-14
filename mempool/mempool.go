package mempool

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ sdkmempool.Mempool = (*AuctionMempool)(nil)

// AuctionMempool defines an auction mempool. It can be seen as an extension of
// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
// two-dimensional priority ordering, with the additional support of prioritizing
// and indexing auction bids.
type AuctionMempool struct {
	// globalIndex defines the index of all transactions in the mempool. It uses
	// the SDK's builtin PriorityNonceMempool. Once a bid is selected for top-of-block,
	// all subsequent transactions in the mempool will be selected from this index.
	globalIndex *PriorityNonceMempool

	// auctionIndex defines an index of auction bids.
	auctionIndex *PriorityNonceMempool

	// txIndex defines an index of all transactions in the mempool by hash.
	txIndex map[string]sdk.Tx

	// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
	// and construct their hashes.
	txEncoder sdk.TxEncoder
}

// AuctionTxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func AuctionTxPriority() TxPriority {
	return TxPriority{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) any {
			panic("TODO: IMPLEMENT ME!")
		},
		CompareTxPriority: func(a, b any) int {
			panic("TODO: IMPLEMENT ME!")
			// switch {
			// case a == nil && b == nil:
			// 	return 0
			// case a == nil:
			// 	return -1
			// case b == nil:
			// 	return 1
			// default:
			// 	aPriority := a.(int64)
			// 	bPriority := b.(int64)

			// 	return skiplist.Int64.Compare(aPriority, bPriority)
			// }
		},
	}
}

func NewAuctionMempool(txEncoder sdk.TxEncoder, opts ...PriorityNonceMempoolOption) *AuctionMempool {
	return &AuctionMempool{
		globalIndex: NewPriorityMempool(
			NewDefaultTxPriority(),
			opts...,
		),
		auctionIndex: NewPriorityMempool(
			AuctionTxPriority(),
			opts...,
		),
		txIndex:   make(map[string]sdk.Tx),
		txEncoder: txEncoder,
	}
}

func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	hash, hashStr, err := am.getTxHash(tx)
	if err != nil {
		return err
	}

	if _, ok := am.txIndex[hashStr]; ok {
		return fmt.Errorf("tx already exists: %s", hashStr)
	}

	if err := am.globalIndex.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into global index: %w", err)
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	if msg != nil {
		if err := am.auctionIndex.Insert(ctx, NewWrappedBidTx(tx, hash, msg.GetBid())); err != nil {
			removeTx(am.globalIndex, tx)
			return fmt.Errorf("failed to insert tx into auction index: %w", err)
		}
	}

	am.txIndex[hashStr] = tx

	return nil
}

func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	hash, hashStr, err := am.getTxHash(tx)
	if err != nil {
		return err
	}

	// 1. Remove the tx from the global index
	removeTx(am.globalIndex, tx)

	// 2. Remove from the transaction index
	delete(am.txIndex, hashStr)

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	// 3. Remove the bid from the auction index (if applicable). In addition, we
	// remove all referenced transactions from the global and transaction indices.
	if msg != nil {
		am.auctionIndex.Remove(NewWrappedBidTx(tx, hash, msg.GetBid()))

		for _, refTxRaw := range msg.GetTransactions() {
			refHash := sha256.Sum256(refTxRaw)
			refHashStr := base64.StdEncoding.EncodeToString(refHash[:])

			// check if we have the referenced transaction first
			if refTx, ok := am.txIndex[refHashStr]; ok {
				removeTx(am.globalIndex, refTx)
			}

			delete(am.txIndex, refHashStr)
		}
	}

	return nil
}

// AuctionBidSelect returns an iterator over auction bids transactions only.
func (am *AuctionMempool) AuctionBidSelect(ctx context.Context, _ [][]byte) sdkmempool.Iterator {
	return am.auctionIndex.Select(ctx, nil)
}

func (am *AuctionMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.globalIndex.Select(ctx, txs)
}

func (am *AuctionMempool) AuctionBidSelect(ctx context.Context) sdkmempool.Iterator {
	// TODO: return am.auctionIndex.Select(ctx, nil)
	//
	// Ref: ENG-547
	panic("not implemented")
}

func (am *AuctionMempool) CountTx() int {
	return am.globalIndex.CountTx()
}

func (am *AuctionMempool) getTxHash(tx sdk.Tx) ([32]byte, string, error) {
	bz, err := am.txEncoder(tx)
	if err != nil {
		return [32]byte{}, "", fmt.Errorf("failed to encode tx: %w", err)
	}

	hash := sha256.Sum256(bz)
	hashStr := base64.StdEncoding.EncodeToString(hash[:])

	return hash, hashStr, nil
}

func removeTx(mp sdkmempool.Mempool, tx sdk.Tx) {
	err := mp.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}
