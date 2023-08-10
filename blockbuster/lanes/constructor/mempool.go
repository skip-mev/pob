package constructor

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	blockbuster "github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

type (
	ConstructorMempool[C comparable] struct {
		// index defines an index transactions.
		index sdkmempool.Mempool

		txPriority blockbuster.TxPriority[C]

		// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
		// to bytes.
		txEncoder sdk.TxEncoder

		// txCache is a map of all transactions in the mempool. It is used
		// to quickly check if a transaction is already in the mempool.
		txCache map[string]struct{}
	}
)

func NewConstructorMempool[C comparable](txPriority blockbuster.TxPriority[C], txEncoder sdk.TxEncoder, maxTx int) *ConstructorMempool[C] {
	return &ConstructorMempool[C]{
		index: blockbuster.NewPriorityMempool(
			blockbuster.PriorityNonceMempoolConfig[C]{
				TxPriority: txPriority,
				MaxTx:      maxTx,
			},
		),
		txEncoder: txEncoder,
		txCache:   make(map[string]struct{}),
	}
}

// Insert inserts a transaction into the mempool based on the transaction type (normal or auction).
func (cm *ConstructorMempool[C]) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := cm.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(cm.txEncoder, tx)
	if err != nil {
		cm.Remove(tx)
		return err
	}

	cm.txCache[txHashStr] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool based on the transaction type (normal or auction).
func (cm *ConstructorMempool[C]) Remove(tx sdk.Tx) error {
	if err := cm.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove transaction from the mempool: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(cm.txEncoder, tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	delete(cm.txCache, txHashStr)

	return nil
}

func (cm *ConstructorMempool[C]) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return cm.index.Select(ctx, txs)
}

func (cm *ConstructorMempool[C]) CountTx() int {
	return cm.index.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (cm *ConstructorMempool[C]) Contains(tx sdk.Tx) bool {
	_, txHashStr, err := utils.GetTxHashStr(cm.txEncoder, tx)
	if err != nil {
		return false
	}

	_, ok := cm.txCache[txHashStr]
	return ok
}

// Compare determines the relative priority of two transactions belonging in the same lane.
// In the default case, priority is determined by the fees of the transaction.
func (cm *ConstructorMempool[C]) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) int {
	firstPriority := cm.txPriority.GetTxPriority(ctx, this)
	secondPriority := cm.txPriority.GetTxPriority(ctx, other)
	return cm.txPriority.Compare(firstPriority, secondPriority)
}
