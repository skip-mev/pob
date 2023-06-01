package base

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/mempool"
)

var _ sdkmempool.Mempool = (*BaseMempool)(nil)

type (
	Mempool interface {
		sdkmempool.Mempool

		// Contains returns true if the transaction is contained in the mempool.
		Contains(tx sdk.Tx) (bool, error)
	}

	// BaseMempool defines the most basic mempool. It can be seen as an extension of
	// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
	// two-dimensional priority ordering, with the additional support of prioritizing
	// and indexing auction bids.
	BaseMempool struct {
		// index defines an index transactions.
		index sdkmempool.Mempool

		// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
		// to bytes.
		txEncoder sdk.TxEncoder

		// txIndex is a map of all transactions in the mempool. It is used
		// to quickly check if a transaction is already in the mempool.
		txIndex map[string]struct{}
	}
)

func NewDefaultMempool(txEncoder sdk.TxEncoder, maxTx int) *BaseMempool {
	return &BaseMempool{
		index: mempool.NewPriorityMempool(
			mempool.DefaultPriorityNonceMempoolConfig(),
		),
		txEncoder: txEncoder,
		txIndex:   make(map[string]struct{}),
	}
}

// Insert inserts a transaction into the mempool based on the transaction type (normal or auction).
func (am *BaseMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := am.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	txHashStr, err := blockbuster.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return err
	}

	am.txIndex[txHashStr] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool based on the transaction type (normal or auction).
func (am *BaseMempool) Remove(tx sdk.Tx) error {
	am.removeTx(am.index, tx)
	return nil
}

func (am *BaseMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.index.Select(ctx, txs)
}

func (am *BaseMempool) CountTx() int {
	return am.index.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (am *BaseMempool) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := blockbuster.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := am.txIndex[txHashStr]
	return ok, nil
}

func (am *BaseMempool) removeTx(mp sdkmempool.Mempool, tx sdk.Tx) {
	err := mp.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}

	txHashStr, err := blockbuster.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		panic(fmt.Errorf("failed to get tx hash string: %w", err))
	}

	delete(am.txIndex, txHashStr)
}
