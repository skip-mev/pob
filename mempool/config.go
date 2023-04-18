package mempool

import (
	"cosmossdk.io/errors"
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

	// GetBidder defines a function that returns the bidder of a transaction.
	GetBidder func(tx sdk.Tx) (sdk.AccAddress, error)

	// GetBid defines a function that returns the bid of a transaction.
	GetBid func(tx sdk.Tx) (sdk.Coin, error)

	// GetBundledTransactions defines a function that returns the bundled transactions
	// that the user wants to execute at the top of the block.
	GetBundledTransactions func(tx sdk.Tx) ([][]byte, error)

	// BidInfo defines the information about a bid.
	BidInfo struct {
		Bidder       sdk.AccAddress
		Bid          sdk.Coin
		Transactions [][]byte
	}

	// TransactionConfig defines the configuration for processing auction transactions. It is
	// a wrapper around all of the functionality that each application chain must implement
	// in order for auction processing to work.
	TransactionConfig struct {
		isAuctionTx   IsAuctionTx
		getTxSigners  GetTransactionSigners
		wrapBundleTx  WrapBundleTransaction
		getBidder     GetBidder
		getBid        GetBid
		getBundledTxs GetBundledTransactions
	}
)

func NewTransactionConfig(isAuctionTx IsAuctionTx, getTxSigners GetTransactionSigners, wrapBundleTx WrapBundleTransaction, getBidder GetBidder, getBid GetBid, getBundledTxs GetBundledTransactions) TransactionConfig {
	return TransactionConfig{
		isAuctionTx:   isAuctionTx,
		getTxSigners:  getTxSigners,
		wrapBundleTx:  wrapBundleTx,
		getBidder:     getBidder,
		getBid:        getBid,
		getBundledTxs: getBundledTxs,
	}
}

// NewDefaultTransactionConfig returns a default transaction configuration.
func NewDefaultTransactionConfig(txDecoder sdk.TxDecoder) TransactionConfig {
	return TransactionConfig{
		isAuctionTx:   NewDefaultIsAuctionTx(),
		getTxSigners:  NewDefaultGetTransactionSigners(txDecoder),
		wrapBundleTx:  NewDefaultWrapBundleTransaction(txDecoder),
		getBidder:     NewDefaultGetBidder(),
		getBid:        NewDefaultGetBid(),
		getBundledTxs: NewDefaultGetBundledTransactions(),
	}
}

// NewDefaultIsAuctionTx defines a default function that returns true iff a transaction
// is an auction bid transaction. In the default case, the transaction must contain a single
// MsgAuctionBid message.
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
// of a transaction. In the default case, the transaction will be an sdk.Tx and the
// signers are the signers of each sdk.Msg in the transaction.
func NewDefaultGetTransactionSigners(txDecoder sdk.TxDecoder) GetTransactionSigners {
	return func(tx []byte) (map[string]bool, error) {
		sdkTx, err := txDecoder(tx)
		if err != nil {
			return nil, err
		}

		signers := make(map[string]bool)
		for _, msg := range sdkTx.GetMsgs() {
			for _, signer := range msg.GetSigners() {
				signers[signer.String()] = true
			}
		}

		return signers, nil
	}
}

// NewDefaultWrapBundleTransaction defines a default function that wraps a transaction
// that is included in the bundle into a sdk.Tx. In the default case, the transaction
// that is included in the bundle will be the raw bytes of an sdk.Tx.
func NewDefaultWrapBundleTransaction(txDecoder sdk.TxDecoder) WrapBundleTransaction {
	return func(tx []byte) (sdk.Tx, error) {
		return txDecoder(tx)
	}
}

// NewDefaultGetBidder defines a default function that returns the bidder of a transaction.
func NewDefaultGetBidder() GetBidder {
	return func(tx sdk.Tx) (sdk.AccAddress, error) {
		msg, err := GetMsgAuctionBidFromTx(tx)
		if err != nil {
			return nil, err
		}

		bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
		if err != nil {
			return nil, errors.Wrap(err, "invalid bidder address")
		}

		return bidder, nil
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
// transactions that the user wants to execute at the top of the block. In the default case,
// the bundled transactions will be the raw bytes of sdk.Tx's.
func NewDefaultGetBundledTransactions() GetBundledTransactions {
	return func(tx sdk.Tx) ([][]byte, error) {
		msg, err := GetMsgAuctionBidFromTx(tx)
		if err != nil {
			return nil, err
		}

		return msg.Transactions, nil
	}
}