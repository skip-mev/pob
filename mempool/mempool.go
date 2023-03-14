package mempool

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

	// txDecoder defines the sdk.Tx decoder that allows us to decode transactions
	// and construct sdk.Txs from the bundled transactions.
	txDecoder sdk.TxDecoder
}

// AuctionTxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func AuctionTxPriority() TxPriority {
	return TxPriority{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) any {
			return hashablePriorityForTx(tx)
		},
		CompareTxPriority: func(a, b any) int {
			switch {
			case a == nil && b == nil:
				return 0
			case a == nil:
				return -1
			case b == nil:
				return 1
			default:
				aPriority := getPriorityFromHash(a)
				bPriority := getPriorityFromHash(b)

				switch {
				case aPriority.IsAllGT(bPriority):
					return 1
				case aPriority.IsAllLT(bPriority):
					return -1
				default:
					return 0
				}
			}
		},
	}
}

func NewAuctionMempool(txDecoder sdk.TxDecoder, opts ...PriorityNonceMempoolOption) *AuctionMempool {
	return &AuctionMempool{
		globalIndex: NewPriorityMempool(
			NewDefaultTxPriority(),
			opts...,
		),
		auctionIndex: NewPriorityMempool(
			AuctionTxPriority(),
			opts...,
		),
		txDecoder: txDecoder,
	}
}

// Insert inserts a transaction into the mempool. If the transaction is a special auction tx (tx
// that contains a single MsgAuctionBid), it will also insert the transaction into the auction
// index.
func (am *AuctionMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := am.globalIndex.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into global index: %w", err)
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	if msg != nil {
		if err := am.auctionIndex.Insert(ctx, NewWrappedBidTx(tx, msg.GetBid())); err != nil {
			removeTx(am.globalIndex, tx)
			return fmt.Errorf("failed to insert tx into auction index: %w", err)
		}
	}

	return nil
}

// Remove removes a transaction from the mempool. If the transaction is a special auction tx (tx
// that contains a single MsgAuctionBid), it will also remove all referenced transactions from the global mempool.
func (am *AuctionMempool) Remove(tx sdk.Tx) error {
	// 1. Remove the tx from the global index
	removeTx(am.globalIndex, tx)

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	// 2. Remove the bid from the auction index (if applicable). In addition, we
	// remove all referenced transactions from the global mempool.
	if msg != nil {
		removeTx(am.auctionIndex, NewWrappedBidTx(tx, msg.GetBid()))

		for _, refRx := range msg.GetTransactions() {
			tx, err := am.txDecoder(refRx)
			if err != nil {
				return fmt.Errorf("failed to decode referenced tx: %w", err)
			}

			removeTx(am.globalIndex, tx)
		}
	}

	return nil
}

// RemoveWithoutRefTx removes a transaction from the mempool without removing any referenced
// transactions. Referenced transactions only exist in special auction transactions (txs that only include
// a single MsgAuctionBid). This API is used to ensure that searchers are unable to remove valid
// transactions from the global mempool.
func (am *AuctionMempool) RemoveWithoutRefTx(tx sdk.Tx) error {
	// 1. Remove the tx from the global index
	removeTx(am.globalIndex, tx)

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	if msg != nil {
		removeTx(am.auctionIndex, NewWrappedBidTx(tx, msg.GetBid()))
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

func (am *AuctionMempool) CountAuctionTx() int {
	return am.auctionIndex.CountTx()
}

func (am *AuctionMempool) CountTx() int {
	return am.globalIndex.CountTx()
}

func removeTx(mp sdkmempool.Mempool, tx sdk.Tx) {
	err := mp.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}

// hashablePriorityForTx returns a string representation of the bid coins i.e.
// the priority. This is used to index auction bids in the auction index. This is
// necessary because sdk.Coins are not hashable.
func hashablePriorityForTx(tx sdk.Tx) string {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		panic(err)
	}

	prioriy := make([]string, len(msg.GetBid()))
	for i, coin := range msg.GetBid() {
		prioriy[i] = fmt.Sprintf("%s:%s", coin.Denom, coin.Amount.String())
	}

	return strings.Join(prioriy, ",")
}

// getPriorityFromHash returns an sdk.Coins from a string representation of the
// bid coins i.e. the priority.
func getPriorityFromHash(priority any) sdk.Coins {
	priorityStr := priority.(string)
	formattedCoins := strings.Split(priorityStr, ",")

	coins := make(sdk.Coins, len(formattedCoins))
	for i, part := range formattedCoins {
		metaData := strings.Split(part, ":")

		denom := metaData[0]
		amount, ok := sdk.NewIntFromString(metaData[1])
		if !ok {
			panic(fmt.Errorf("failed to parse amount %s", metaData[1]))
		}

		coins[i] = sdk.NewCoin(denom, amount)
	}

	return coins
}
