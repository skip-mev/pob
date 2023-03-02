package mempool

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	// the SDK's builtin PriorityNonceMempool. Once a bid if selected for top-of-block,
	// all subsequent transactions in the mempool will be selected from this index.
	globalIndex sdkmempool.PriorityNonceMempool

	// auctionIndex defines an index of auction bids.
	auctionIndex *AuctionBidList

	// txIndex defines an index of all transactions in the mempool by hash.
	txIndex map[string]struct{}

	// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
	// and construct their hashes.
	txEncoder sdk.TxEncoder
}

func NewAuctionMempool(txEncoder sdk.TxEncoder, opts ...sdkmempool.PriorityNonceMempoolOption) *AuctionMempool {
	return &AuctionMempool{
		globalIndex:  *sdkmempool.NewPriorityMempool(opts...),
		auctionIndex: NewAuctionBidList(),
		txIndex:      make(map[string]struct{}),
		txEncoder:    txEncoder,
	}
}

func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	bz, err := am.txEncoder(tx)
	if err != nil {
		return fmt.Errorf("failed to encode tx: %w", err)
	}

	hash := sha256.Sum256(bz)
	hashStr := base64.StdEncoding.EncodeToString(hash[:])
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
		am.auctionIndex.Insert(NewWrappedBidTx(tx, hash, msg.GetBid()))
	}

	am.txIndex[hashStr] = struct{}{}

	return nil
}

func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	panic("not implemented")
}

func (am *AuctionMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.globalIndex.Select(ctx, txs)
}

func (am *AuctionMempool) CountTx() int {
	return am.globalIndex.CountTx()
}
