package base

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

var _ sdkmempool.Mempool = (*DefaultMempool)(nil)

type (
	// Mempool defines the interface of the default mempool.
	Mempool interface {
		sdkmempool.Mempool

		// Contains returns true if the transaction is contained in the mempool.
		Contains(tx sdk.Tx) bool
	}

	// DefaultMempool defines the most basic mempool. It can be seen as an extension of
	// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
	// two-dimensional priority ordering, with the additional support of prioritizing
	// and indexing auction bids.
	DefaultMempool struct {
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

// TxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func TxPriority(gasToken string) blockbuster.TxPriority[math.Int] {
	return blockbuster.TxPriority[math.Int]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) math.Int {
			feeTx, ok := tx.(sdk.FeeTx)
			if !ok {
				panic(fmt.Errorf("tx is not a FeeTx: %T", tx))
			}

			fee := feeTx.GetFee()

			found, coin := fee.Find(gasToken)
			if !found {
				return math.ZeroInt()
			}

			return coin.Amount
		},
		Compare: func(a, b math.Int) int {
			switch {
			case a.IsNil() && b.IsNil():
				return 0

			case a.IsNil():
				return -1

			case b.IsNil():
				return 1

			default:
				switch {
				case a.GT(b):
					return 1

				case b.GT(a):
					return -1

				default:
					return 0
				}
			}
		},
		MinValue: math.ZeroInt(),
	}
}

// NewDefaultMempool returns a new default mempool instance. The default mempool
// orders transactions by the sdk.Context priority.
func NewDefaultMempool(txEncoder sdk.TxEncoder, maxTx int, gasToken string) *DefaultMempool {
	return &DefaultMempool{
		index: blockbuster.NewPriorityMempool(
			blockbuster.PriorityNonceMempoolConfig[math.Int]{
				TxPriority: TxPriority(gasToken),
				MaxTx:      maxTx,
			},
		),
		txEncoder: txEncoder,
		txIndex:   make(map[string]struct{}),
	}
}

// Insert inserts a transaction into the mempool based on the transaction type (normal or auction).
func (am *DefaultMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := am.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		am.Remove(tx)
		return err
	}

	am.txIndex[txHashStr] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool based on the transaction type (normal or auction).
func (am *DefaultMempool) Remove(tx sdk.Tx) error {
	if err := am.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove transaction from the mempool: %w", err)
	}

	_, txHashStr, err := utils.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	delete(am.txIndex, txHashStr)

	return nil
}

func (am *DefaultMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.index.Select(ctx, txs)
}

func (am *DefaultMempool) CountTx() int {
	return am.index.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (am *DefaultMempool) Contains(tx sdk.Tx) bool {
	_, txHashStr, err := utils.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return false
	}

	_, ok := am.txIndex[txHashStr]
	return ok
}
