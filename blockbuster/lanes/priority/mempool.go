package priority

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
)

var _ blockbuster.LaneInterface = (*DefaultLane)(nil)

// DefaultLane defines an auction mempool. It can be seen as an extension of
// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
// two-dimensional priority ordering, with the additional support of prioritizing
// and indexing auction bids.
type DefaultLane struct {
	cfg blockbuster.LaneConfig[int64]

	// txDecoder defines the sdk.Tx decoder that allows us to decode transactions
	// and construct sdk.Txs from the bundled transactions.
	txDecoder sdk.TxDecoder

	// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
	// to bytes.
	txEncoder sdk.TxEncoder

	// txIndex is a map of all transactions in the mempool. It is used
	// to quickly check if a transaction is already in the mempool.
	txIndex map[string]struct{}

	// anteHandler defines the ante handler used to validate transactions.
	anteHandler sdk.AnteHandler

	// postHandler defines the post handler used to validate transactions.
	postHandler sdk.PostHandler
}

func NewDefaultLane(txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int) *DefaultLane {
	return &DefaultLane{
		cfg: blockbuster.LaneConfig[int64]{
			GlobalIndex: blockbuster.NewPriorityMempool(
				blockbuster.PriorityNonceMempoolConfig[int64]{
					TxPriority: blockbuster.DefaultPriorityNonceMempoolConfig().TxPriority,
					MaxTx:      maxTx,
				},
			),
		},
		txDecoder: txDecoder,
		txEncoder: txEncoder,
		txIndex:   make(map[string]struct{}),
	}
}

// Insert inserts a transaction into the mempool. If the transaction is a special
// auction tx (tx that contains a single MsgAuctionBid), it will also insert the
// transaction into the auction index.
func (mempool *DefaultLane) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := mempool.cfg.GlobalIndex.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	txHashStr, err := mempool.getTxHashStr(tx)
	if err != nil {
		return err
	}

	mempool.txIndex[txHashStr] = struct{}{}

	// Insert the transaction into any other mempools that are registered with
	// this mempool.
	mempool.cfg.Hooks.AfterInsertHook(tx)

	return nil
}

// Remove removes a transaction from the mempool. If the transaction is a special
// auction tx (tx that contains a single MsgAuctionBid), it will also remove all
// referenced transactions from the global mempool.
func (mempool *DefaultLane) Remove(tx sdk.Tx) error {
	// Remove the transaction from the global mempool
	mempool.removeTx(tx)

	// Remove the transaction from any other mempools that are registered with
	// this mempool.
	mempool.cfg.Hooks.AfterRemoveHook(tx)

	return nil
}

// GetTopTx returns the top transaction in the mempool.
func (mempool *DefaultLane) GetTopTx(ctx context.Context) sdk.Tx {
	iterator := mempool.cfg.GlobalIndex.Select(ctx, nil)
	if iterator == nil {
		return nil
	}

	return iterator.Tx()
}

// Select iterates through all of the transactions in the mempool
func (mempool *DefaultLane) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return mempool.cfg.GlobalIndex.Select(ctx, nil)
}

// CountTx returns the number of transactions in the mempool.
func (mempool *DefaultLane) CountTx() int {
	return mempool.cfg.GlobalIndex.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (mempool *DefaultLane) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := mempool.getTxHashStr(tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := mempool.txIndex[txHashStr]
	return ok, nil
}

func (mempool *DefaultLane) Match(_ sdk.Tx) bool {
	return true
}

// getTxHashStr returns the transaction hash string for a given transaction.
func (mempool *DefaultLane) getTxHashStr(tx sdk.Tx) (string, error) {
	txBz, err := mempool.txEncoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txHashStr, nil
}

// removeTx will remove a transaction from the auction mempool and remove it from the
// tx index.
func (mempool *DefaultLane) removeTx(tx sdk.Tx) {
	// Remove the transaction from the mempool. If the transaction is not found, we ignore the error.
	err := mempool.cfg.GlobalIndex.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}

	// Remove the transaction from the tx index.
	txHashStr, err := mempool.getTxHashStr(tx)
	if err != nil {
		panic(fmt.Errorf("failed to get tx hash string: %w", err))
	}
	delete(mempool.txIndex, txHashStr)
}
