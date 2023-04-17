package mempool

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// isAuctionTx defines a function that returns true iff a transaction is an
	// auction bid transaction.
	IsAuctionTx func(tx sdk.Tx) (bool, error)

	// getTransactionSigners defines a function that returns the signers of a
	// transaction.
	GetTransactionSigners func(tx []byte) (map[string]bool, error)

	// wrapBundleTransaction defines a function that wraps a transaction that is included
	// in the bundle into a sdk.Tx.
	WrapBundleTransaction func(tx []byte) (sdk.Tx, error)

	// GetBid defines a function that returns the bid of a transaction.
	GetBid func(tx sdk.Tx) (sdk.Coin, error)

	// GetBundledTransactions defines a function that returns the bundled transactions
	// that the user wants to execute at the top of the block.
	GetBundledTransactions func(tx sdk.Tx) ([]sdk.Tx, error)
)

// NewDefaultIsAuctionTx defines a default function that returns true if a transaction
// is an auction bid transaction.
func NewDefaultIsAuctionTx() IsAuctionTx {
	return func(tx sdk.Tx) (bool, error) {
		msg, err := GetMsgAuctionBidFromTx(tx)
		if err != nil {
			return false, err
		}

		return msg != nil, nil
	}
}

// NewDefaultGetTransactionSigners defines a default function that returns the signers
// of a transaction.
func NewDefaultGetTransactionSigners(txDecoder sdk.TxDecoder) GetTransactionSigners {
	return func(tx []byte) (map[string]bool, error) {
		sdkTx, err := txDecoder(tx)
		if err != nil {
			return nil, err
		}

		signers := make(map[string]bool, 0)
		for _, msg := range sdkTx.GetMsgs() {
			for _, signer := range msg.GetSigners() {
				signers[signer.String()] = true
			}
		}

		return signers, nil
	}
}

// NewDefaultWrapBundleTransaction defines a default function that wraps a transaction
// that is included in the bundle into a sdk.Tx.
func NewDefaultWrapBundleTransaction(txDecoder sdk.TxDecoder) WrapBundleTransaction {
	return func(tx []byte) (sdk.Tx, error) {
		return txDecoder(tx)
	}
}

// NewDefaultGetBid defines a default function that returns the bid of a transaction.
func NewDefaultGetBid() GetBid {
	return func(tx sdk.Tx) (sdk.Coin, error) {
		msg, err := GetMsgAuctionBidFromTx(tx)
		if err != nil {
			return sdk.Coin{}, err
		}

		return msg.Bid, nil
	}
}

// NewDefaultGetBundledTransactions defines a default function that returns the bundled
// transactions that the user wants to execute at the top of the block.
func NewDefaultGetBundledTransactions(txDecoder sdk.TxDecoder) GetBundledTransactions {
	return func(tx sdk.Tx) ([]sdk.Tx, error) {
		msg, err := GetMsgAuctionBidFromTx(tx)
		if err != nil {
			return nil, err
		}

		wrap := NewDefaultWrapBundleTransaction(txDecoder)
		wrappedTxs := make([]sdk.Tx, len(msg.Transactions))
		for i, txBz := range msg.Transactions {
			tx, err := wrap(txBz)
			if err != nil {
				return nil, err
			}

			wrappedTxs[i] = tx
		}

		return wrappedTxs, nil
	}
}

// AuctionTxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func AuctionTxPriority() TxPriority[string] {
	return TxPriority[string]{
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
