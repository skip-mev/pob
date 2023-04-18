package mempool

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// IsAuctionTx defines a function that returns true iff a transaction is an
	// auction bid transaction.
	IsAuctionTx func(tx sdk.Tx) (bool, error)

	// GetTransactionSigners defines a function that returns the signers of a
	// bundle transaction i.e. transaction that was included in the auction transaction's bundle.
	GetTransactionSigners func(tx []byte) (map[string]bool, error)

	// WrapBundleTransaction defines a function that wraps a bundle transaction into a sdk.Tx.
	WrapBundleTransaction func(tx []byte) (sdk.Tx, error)

	// GetBidder defines a function that returns the bidder of an auction transaction transaction.
	GetBidder func(tx sdk.Tx) (sdk.AccAddress, error)

	// GetBid defines a function that returns the bid of an auction transaction.
	GetBid func(tx sdk.Tx) (sdk.Coin, error)

	// GetBundledTransactions defines a function that returns the bundled transactions
	// that the user wants to execute at the top of the block given an auction transaction.
	GetBundledTransactions func(tx sdk.Tx) ([][]byte, error)

	// BidInfo defines the information about a bid.
	BidInfo struct {
		Bidder       sdk.AccAddress
		Bid          sdk.Coin
		Transactions [][]byte
	}

	// Config defines the configuration for processing auction transactions. It is
	// a wrapper around all of the functionality that each application chain must implement
	// in order for auction processing to work.
	Config struct {
		isAuctionTx   IsAuctionTx
		getTxSigners  GetTransactionSigners
		wrapBundleTx  WrapBundleTransaction
		getBidder     GetBidder
		getBid        GetBid
		getBundledTxs GetBundledTransactions
	}
)

// NewConfig returns a new transaction configuration.
func NewConfig(isAuctionTx IsAuctionTx, getTxSigners GetTransactionSigners, wrapBundleTx WrapBundleTransaction, getBidder GetBidder, getBid GetBid, getBundledTxs GetBundledTransactions) Config {
	return Config{
		isAuctionTx:   isAuctionTx,
		getTxSigners:  getTxSigners,
		wrapBundleTx:  wrapBundleTx,
		getBidder:     getBidder,
		getBid:        getBid,
		getBundledTxs: getBundledTxs,
	}
}

// NewDefaultConfig returns a default transaction configuration.
func NewDefaultConfig(txDecoder sdk.TxDecoder) Config {
	return Config{
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
// of a transaction. In the default case, each bundle transaction will be an sdk.Tx and the
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
// that is included in the bundle will be the raw bytes of an sdk.Tx so we can just
// decode it.
func NewDefaultWrapBundleTransaction(txDecoder sdk.TxDecoder) WrapBundleTransaction {
	return func(tx []byte) (sdk.Tx, error) {
		return txDecoder(tx)
	}
}

// NewDefaultGetBidder defines a default function that returns the bidder of an auction transaction.
// In the default case, the bidder is the address defined in MsgAuctionBid.
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

// NewDefaultGetBid defines a default function that returns the bid of an auction transaction.
// In the default case, the bid is the amount defined in MsgAuctionBid.
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
// the bundled transactions will be the raw bytes of sdk.Tx's that are included in the
// MsgAuctionBid.
func NewDefaultGetBundledTransactions() GetBundledTransactions {
	return func(tx sdk.Tx) ([][]byte, error) {
		msg, err := GetMsgAuctionBidFromTx(tx)
		if err != nil {
			return nil, err
		}

		return msg.Transactions, nil
	}
}
