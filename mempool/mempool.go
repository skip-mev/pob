package mempool

import (
	"context"
	"crypto/sha256"
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
	globalIndex *sdkmempool.PriorityNonceMempool

	// auctionIndex defines an index of auction bids.
	auctionIndex *sdkmempool.PriorityNonceMempool

	// txDecoder defines the sdk.Tx decoder that allows us to decode transactions
	// from their hashes.
	txDecoder sdk.TxDecoder
}

func NewAuctionMempool(txDecoder sdk.TxDecoder, opts ...sdkmempool.PriorityNonceMempoolOption) *AuctionMempool {
	return &AuctionMempool{
		globalIndex:  sdkmempool.NewPriorityMempool(opts...),
		auctionIndex: sdkmempool.NewPriorityMempool(opts...),
		txDecoder:    txDecoder,
	}
}

func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := am.globalIndex.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into global index: %w", err)
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	if msg != nil {
		if err := am.auctionIndex.Insert(ctx, tx); err != nil {
			return fmt.Errorf("failed to insert tx into auction index: %w", err)
		}
	}

	return nil
}

func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	// 1. Remove the tx from the global index
	if err := am.globalIndex.Remove(tx); err != nil {
		return fmt.Errorf("failed to remove tx from global index: %w", err)
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	// 2. Remove the bid from the auction index (if applicable). In addition, we
	// remove all referenced transactions from the global and transaction indices.
	if msg != nil {
		if err := am.auctionIndex.Remove(tx); err != nil {
			return fmt.Errorf("failed to remove tx from auction index: %w", err)
		}

		// Decode the referenced transaction and remove them from the global index.
		for _, refTxRaw := range msg.GetTransactions() {
			refHash := sha256.Sum256(refTxRaw)
			refTx, err := am.txDecoder(refHash[:])
			if err != nil {
				return fmt.Errorf("failed to decode referenced tx: %w", err)
			}

			// Remove the referenced tx from the global index if it exists.
			if err := am.globalIndex.Remove(refTx); err != sdkmempool.ErrTxNotFound {
				return fmt.Errorf("failed to remove referenced tx from global index: %w", err)
			}
		}
	}

	return nil
}

// SelectTopAuctionBidTx returns the top auction bid tx in the mempool if one
// exists.
func (am *AuctionMempool) SelectTopAuctionBidTx() sdk.Tx {
	wBidTx := am.auctionIndex.Select(context.Background(), nil)
	if wBidTx == nil {
		return nil
	}

	return wBidTx.Tx()
}

func (am *AuctionMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.globalIndex.Select(ctx, txs)
}

func (am *AuctionMempool) CountTx() int {
	return am.globalIndex.CountTx()
}
