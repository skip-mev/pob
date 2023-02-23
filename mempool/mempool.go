package mempool

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	"github.com/skip-mev/pob/types/heap"
)

var _ sdkmempool.Mempool = (*AuctionMempool)(nil)

type AuctionMempool struct {
	globalIndex  sdkmempool.PriorityNonceMempool
	auctionIndex *heap.Heap[PriorityTx]
	txIndex      map[string]sdk.Tx
	txEncoder    sdk.TxEncoder
}

func NewAuctionMempool(txEncoder sdk.TxEncoder, opts ...sdkmempool.PriorityNonceMempoolOption) *AuctionMempool {
	return &AuctionMempool{
		globalIndex:  *sdkmempool.NewPriorityMempool(opts...),
		auctionIndex: heap.New[PriorityTx](func(a, b PriorityTx) bool { return a.GetPriority() > b.GetPriority() }),
		txIndex:      make(map[string]sdk.Tx),
		txEncoder:    txEncoder,
	}
}

func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	sdkContext := sdk.UnwrapSDKContext(ctx)

	if err := am.globalIndex.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into global index: %w", err)
	}

	am.auctionIndex.Push(NewPriorityTx(tx, sdkContext.Priority()))

	bz, err := am.txEncoder(tx)
	if err != nil {
		return fmt.Errorf("failed to encode tx: %w", err)
	}

	hash := sha256.Sum256(bz)
	am.txIndex[base64.StdEncoding.EncodeToString(hash[:])] = tx

	return nil
}

func (am *AuctionMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.globalIndex.Select(ctx, txs)
}

func (am *AuctionMempool) CountTx() int {
	return am.globalIndex.CountTx()
}

func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	// if err := am.globalIndex.Remove(tx); err != nil {
	// 	return fmt.Errorf("failed to remove tx from global index: %w", err)
	// }

	// if err := am.auctionIndex.Remove(tx); err != nil {
	// 	return fmt.Errorf("failed to remove tx from auction index: %w", err)
	// }
}
