package tob

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

var _ blockbuster.LaneInterface = (*AuctionLane)(nil)

// AuctionLane defines an auction mempool. It can be seen as an extension of
// an SDK PriorityNonceMempool, i.e. a mempool that supports <sender, nonce>
// two-dimensional priority ordering, with the additional support of prioritizing
// and indexing auction bids.
type AuctionLane struct {
	cfg blockbuster.LaneConfig[string]

	// txDecoder defines the sdk.Tx decoder that allows us to decode transactions
	// and construct sdk.Txs from the bundled transactions.
	txDecoder sdk.TxDecoder

	// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
	// to bytes.
	txEncoder sdk.TxEncoder

	// txIndex is a map of all transactions in the mempool. It is used
	// to quickly check if a transaction is already in the mempool.
	txIndex map[string]struct{}
}

// AuctionTxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func AuctionTxPriority() blockbuster.TxPriority[string] {
	return blockbuster.TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			msgAuctionBid, err := GetMsgAuctionBidFromTx(tx)
			if err != nil {
				panic(err)
			}

			return msgAuctionBid.Bid.String()
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

func NewAuctionLane(txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int) *AuctionLane {
	return &AuctionLane{
		cfg: blockbuster.LaneConfig[string]{
			GlobalIndex: blockbuster.NewPriorityMempool(
				blockbuster.PriorityNonceMempoolConfig[string]{
					TxPriority: AuctionTxPriority(),
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
func (am *AuctionLane) Insert(ctx context.Context, tx sdk.Tx) error {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	// Insert the transactions into the appropriate index.
	switch {
	case msg != nil:
		if err := am.cfg.GlobalIndex.Insert(ctx, tx); err != nil {
			return fmt.Errorf("failed to insert tx into auction index: %w", err)
		}
	}

	txHashStr, err := am.getTxHashStr(tx)
	if err != nil {
		return err
	}

	am.txIndex[txHashStr] = struct{}{}

	// Insert the transaction into any other lanes that are subscribed to this
	// lane.
	am.cfg.Hooks.AfterInsertHook(tx)

	return nil
}

// Remove removes a transaction from the mempool. If the transaction is a special
// auction tx (tx that contains a single MsgAuctionBid), it will also remove all
// referenced transactions from the global mempool.
func (am *AuctionLane) Remove(tx sdk.Tx) error {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	// Remove the transactions from the appropriate index.
	switch {
	case msg != nil:
		// Remove the transaction from the auction index.
		am.removeTx(tx)

		// Run the after remove hook to let other mempools know that the transaction
		// has been removed from the auction index.
		am.cfg.Hooks.AfterRemoveHook(tx)
	}

	return nil
}

// RemoveWithoutRefTx removes a transaction from the mempool without removing
// any referenced transactions. Referenced transactions only exist in special
// auction transactions (txs that only include a single MsgAuctionBid). This
// API is used to ensure that searchers are unable to remove valid transactions
// from the global mempool.
func (am *AuctionLane) RemoveWithoutRefTx(tx sdk.Tx) error {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return err
	}

	if msg != nil {
		am.removeTx(tx)
	}

	return nil
}

// GetTopAuctionTx returns the highest bidding transaction in the auction mempool.
func (am *AuctionLane) GetTopAuctionTx(ctx context.Context) sdk.Tx {
	iterator := am.cfg.GlobalIndex.Select(ctx, nil)
	if iterator == nil {
		return nil
	}

	return iterator.Tx()
}

// Select iterates through all of the transactions in the mempool
func (am *AuctionLane) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return am.cfg.GlobalIndex.Select(ctx, nil)
}

// CountTx returns the number of transactions that are attempting to bid for top of block execution.
func (am *AuctionLane) CountTx() int {
	return am.cfg.GlobalIndex.CountTx()
}

// Contains returns true if the transaction is contained in the mempool.
func (am *AuctionLane) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := am.getTxHashStr(tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := am.txIndex[txHashStr]
	return ok, nil
}

// Match returns true if the transaction is a valid auction transaction.
func (am *AuctionLane) Match(tx sdk.Tx) bool {
	msg, err := GetMsgAuctionBidFromTx(tx)
	return msg != nil && err == nil
}

// getTxHashStr returns the transaction hash string for a given transaction.
func (am *AuctionLane) getTxHashStr(tx sdk.Tx) (string, error) {
	txBz, err := am.txEncoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txHashStr, nil
}

// removeTx will remove a transaction from the auction mempool and remove it from the
// tx index.
func (am *AuctionLane) removeTx(tx sdk.Tx) {
	// Remove the transaction from the mempool. If the transaction is not found,
	// we ignore the error.
	err := am.cfg.GlobalIndex.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}

	// Remove the transaction from the tx index.
	txHashStr, err := am.getTxHashStr(tx)
	if err != nil {
		panic(fmt.Errorf("failed to get tx hash string: %w", err))
	}
	delete(am.txIndex, txHashStr)
}
