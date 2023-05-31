package tob

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/mempool"
)

var _ sdkmempool.Mempool = (*AuctionMempool)(nil)

type (
	Mempool interface {
		sdkmempool.Mempool

		// GetTopAuctionTx returns the highest bidding transaction in the auction mempool.
		GetTopAuctionTx(ctx context.Context) sdk.Tx

		// Contains returns true if the transaction is contained in the mempool.
		Contains(tx sdk.Tx) (bool, error)
	}

	// AuctionMempool defines an auction mempool. It can be seen as an extension of
	// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
	// two-dimensional priority ordering, with the additional support of prioritizing
	// and indexing auction bids.
	AuctionMempool struct {
		// auctionIndex defines an index of auction bids.
		index sdkmempool.Mempool

		// txDecoder defines the sdk.Tx decoder that allows us to decode transactions
		// and construct sdk.Txs from the bundled transactions.
		txDecoder sdk.TxDecoder

		// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
		// to bytes.
		txEncoder sdk.TxEncoder

		// txIndex is a map of all transactions in the mempool. It is used
		// to quickly check if a transaction is already in the mempool.
		txIndex map[string]struct{}

		// AuctionFactory implements the functionality required to process auction transactions.
		AuctionFactory
	}
)

// AuctionTxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func AuctionTxPriority(config AuctionFactory) mempool.TxPriority[string] {
	return mempool.TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			bidInfo, err := config.GetAuctionBidInfo(tx)
			if err != nil {
				panic(err)
			}

			return bidInfo.Bid.String()
		},
		Compare: func(a, b string) int {
			aCoins, _ := sdk.ParseCoinsNormalized(a)
			bCoins, _ := sdk.ParseCoinsNormalized(b)

			switch {
			case aCoins == nil && bCoins == nil:
				return 0

			case aCoins == nil:
				return -1

			case bCoins == nil:
				return 1

			default:
				switch {
				case aCoins.IsAllGT(bCoins):
					return 1

				case aCoins.IsAllLT(bCoins):
					return -1

				default:
					return 0
				}
			}
		},
		MinValue: "",
	}
}

func NewAuctionMempool(txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int, config AuctionFactory) *AuctionMempool {
	return &AuctionMempool{
		index: mempool.NewPriorityMempool(
			mempool.PriorityNonceMempoolConfig[string]{
				TxPriority: AuctionTxPriority(config),
				MaxTx:      maxTx,
			},
		),
		txDecoder:      txDecoder,
		txEncoder:      txEncoder,
		txIndex:        make(map[string]struct{}),
		AuctionFactory: config,
	}
}

// Insert inserts a transaction into the mempool based on the transaction type (normal or auction).
func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	bidInfo, err := am.GetAuctionBidInfo(tx)
	if err != nil {
		return err
	}

	// Insert the transactions into the appropriate index.
	if bidInfo != nil {
		if err := am.index.Insert(ctx, tx); err != nil {
			return fmt.Errorf("failed to insert tx into auction index: %w", err)
		}
	} else {
		return errors.New("invalid transaction type")
	}

	txHashStr, err := blockbuster.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return err
	}

	am.txIndex[txHashStr] = struct{}{}

	return nil
}

// Remove removes a transaction from the mempool based on the transaction type (normal or auction).
func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	bidInfo, err := am.GetAuctionBidInfo(tx)
	if err != nil {
		return err
	}

	// Remove the transactions from the appropriate index.
	if bidInfo != nil {
		am.removeTx(am.index, tx)
	} else {
		return errors.New("invalid transaction type")
	}

	return nil
}

// GetTopAuctionTx returns the highest bidding transaction in the auction mempool.
func (am *AuctionMempool) GetTopAuctionTx(ctx context.Context) sdk.Tx {
	iterator := am.index.Select(ctx, nil)
	if iterator == nil {
		return nil
	}

	return iterator.Tx()
}

func (am *AuctionMempool) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.index.Select(ctx, txs)
}

func (am *AuctionMempool) CountTx() int {
	return am.index.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (am *AuctionMempool) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := blockbuster.GetTxHashStr(am.txEncoder, tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := am.txIndex[txHashStr]
	return ok, nil
}

func (am *AuctionMempool) removeTx(mp sdkmempool.Mempool, tx sdk.Tx) {
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
