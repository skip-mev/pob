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

type AuctionMempool struct {
	globalIndex sdkmempool.PriorityNonceMempool
	// auctionIndex *heap.Heap[PriorityTx]
	txIndex   map[string]*WrappedTx
	txEncoder sdk.TxEncoder
}

func NewAuctionMempool(txEncoder sdk.TxEncoder, opts ...sdkmempool.PriorityNonceMempoolOption) *AuctionMempool {
	return &AuctionMempool{
		globalIndex: *sdkmempool.NewPriorityMempool(opts...),
		txIndex:     make(map[string]*WrappedTx),
		txEncoder:   txEncoder,
	}
}

func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	bz, err := am.txEncoder(tx)
	if err != nil {
		return fmt.Errorf("failed to encode tx: %w", err)
	}

	hash := sha256.Sum256(bz)
	hashStr := base64.StdEncoding.EncodeToString(hash[:])
	if _, ok := am.txIndex[hashStr]; ok {
		return fmt.Errorf("tx already exists: %s", hashStr)
	}

	wrappedTx := &WrappedTx{
		Tx:   tx,
		hash: hash,
	}

	am.txIndex[hashStr] = wrappedTx

	if err := am.globalIndex.Insert(ctx, wrappedTx); err != nil {
		// TODO: Remove from txIndex and auctionIndex.
		return fmt.Errorf("failed to insert tx into global index: %w", err)
	}

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
