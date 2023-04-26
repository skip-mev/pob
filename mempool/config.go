package mempool

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type (
	// AuctionBidInfo defines the information about a bid to the auction house.
	AuctionBidInfo struct {
		Bidder       sdk.AccAddress
		Bid          sdk.Coin
		Transactions [][]byte
		Timeout      uint64
	}

	// Config defines the configuration for processing auction transactions. It is
	// a wrapper around all of the functionality that each application chain must implement
	// in order for auction processing to work.
	Config interface {
		// IsAuctionTx defines a function that returns true iff a transaction is an
		// auction bid transaction.
		IsAuctionTx(tx sdk.Tx) (bool, error)

		// GetTransactionSigners defines a function that returns the signers of a
		// bundle transaction i.e. transaction that was included in the auction transaction's bundle.
		GetTransactionSigners(tx []byte) (map[string]struct{}, error)

		// GetBundleSigners defines a function that returns the signers of every transaction in a bundle.
		GetBundleSigners(tx [][]byte) ([]map[string]struct{}, error)

		// WrapBundleTransaction defines a function that wraps a bundle transaction into a sdk.Tx.
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)

		// GetBidder defines a function that returns the bidder of an auction transaction transaction.
		GetBidder(tx sdk.Tx) (sdk.AccAddress, error)

		// GetBid defines a function that returns the bid of an auction transaction.
		GetBid(tx sdk.Tx) (sdk.Coin, error)

		// GetBundledTransactions defines a function that returns the bundled transactions
		// that the user wants to execute at the top of the block given an auction transaction.
		GetBundledTransactions(tx sdk.Tx) ([][]byte, error)

		// GetTimeout defines a function that returns the timeout of an auction transaction.
		GetTimeout(tx sdk.Tx) (uint64, error)

		// GetAuctionBidInfo defines a function that returns the bid info from an auction transaction.
		GetAuctionBidInfo(tx sdk.Tx) (AuctionBidInfo, error)
	}

	// DefaultConfig defines a default configuration for processing auction transactions.
	DefaultConfig struct {
		txDecoder sdk.TxDecoder
	}

	// TxWithTimeoutHeight is used to extract timeouts from sdk.Tx transactions. In the case where,
	// timeouts are explicitly set on the sdk.Tx, we can use this interface to extract the timeout.
	TxWithTimeoutHeight interface {
		sdk.Tx

		GetTimeoutHeight() uint64
	}
)

var _ Config = (*DefaultConfig)(nil)

// NewDefaultConfig returns a default transaction configuration.
func NewDefaultConfig(txDecoder sdk.TxDecoder) Config {
	return &DefaultConfig{
		txDecoder: txDecoder,
	}
}

// NewDefaultIsAuctionTx defines a default function that returns true iff a transaction
// is an auction bid transaction. In the default case, the transaction must contain a single
// MsgAuctionBid message.
func (config *DefaultConfig) IsAuctionTx(tx sdk.Tx) (bool, error) {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return false, err
	}

	return msg != nil, nil
}

// GetTransactionSigners defines a default function that returns the signers
// of a transaction. In the default case, each bundle transaction will be an sdk.Tx and the
// signers are the signers of each sdk.Msg in the transaction.
func (config *DefaultConfig) GetTransactionSigners(tx []byte) (map[string]struct{}, error) {
	sdkTx, err := config.txDecoder(tx)
	if err != nil {
		return nil, err
	}

	sigTx, ok := sdkTx.(signing.SigVerifiableTx)
	if !ok {
		return nil, fmt.Errorf("transaction is not valid")
	}

	signers := make(map[string]struct{})
	for _, signer := range sigTx.GetSigners() {
		signers[signer.String()] = struct{}{}
	}

	return signers, nil
}

// GetBundleSigners defines a default function that returns the signers of every transaction
// in a bundle. In the default case, each bundle transaction will be an sdk.Tx and the
// signers are the signers of each sdk.Msg in the transaction.
func (config *DefaultConfig) GetBundleSigners(txs [][]byte) ([]map[string]struct{}, error) {
	signers := make([]map[string]struct{}, len(txs))

	for index, tx := range txs {
		txSigners, err := config.GetTransactionSigners(tx)
		if err != nil {
			return nil, err
		}

		signers[index] = txSigners
	}

	return signers, nil
}

// WrapBundleTransaction defines a default function that wraps a transaction
// that is included in the bundle into a sdk.Tx. In the default case, the transaction
// that is included in the bundle will be the raw bytes of an sdk.Tx so we can just
// decode it.
func (config *DefaultConfig) WrapBundleTransaction(tx []byte) (sdk.Tx, error) {
	return config.txDecoder(tx)
}

// GetBidder defines a default function that returns the bidder of an auction transaction.
// In the default case, the bidder is the address defined in MsgAuctionBid.
func (config *DefaultConfig) GetBidder(tx sdk.Tx) (sdk.AccAddress, error) {
	isAuctionTx, err := config.IsAuctionTx(tx)
	if err != nil {
		return nil, err
	}

	if !isAuctionTx {
		return nil, fmt.Errorf("transaction is not an auction transaction")
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return nil, err
	}

	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return nil, fmt.Errorf("invalid bidder address (%s): %w", msg.Bidder, err)
	}

	return bidder, nil
}

// GetBid defines a default function that returns the bid of an auction transaction.
// In the default case, the bid is the amount defined in MsgAuctionBid.
func (config *DefaultConfig) GetBid(tx sdk.Tx) (sdk.Coin, error) {
	isAuctionTx, err := config.IsAuctionTx(tx)
	if err != nil {
		return sdk.Coin{}, err
	}

	if !isAuctionTx {
		return sdk.Coin{}, fmt.Errorf("transaction is not an auction transaction")
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return sdk.Coin{}, err
	}

	return msg.Bid, nil
}

// GetBundledTransactions defines a default function that returns the bundled
// transactions that the user wants to execute at the top of the block. In the default case,
// the bundled transactions will be the raw bytes of sdk.Tx's that are included in the
// MsgAuctionBid.
func (config *DefaultConfig) GetBundledTransactions(tx sdk.Tx) ([][]byte, error) {
	isAuctionTx, err := config.IsAuctionTx(tx)
	if err != nil {
		return nil, err
	}

	if !isAuctionTx {
		return nil, fmt.Errorf("transaction is not an auction transaction")
	}

	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return nil, err
	}

	return msg.Transactions, nil
}

// GetTimeout defines a default function that returns the timeout of an auction transaction.
func (config *DefaultConfig) GetTimeout(tx sdk.Tx) (uint64, error) {
	isAuctionTx, err := config.IsAuctionTx(tx)
	if err != nil {
		return 0, err
	}

	if !isAuctionTx {
		return 0, fmt.Errorf("transaction is not an auction transaction")
	}

	timeoutTx, ok := tx.(TxWithTimeoutHeight)
	if !ok {
		return 0, fmt.Errorf("transaction does not implement TxWithTimeoutHeight")
	}

	return timeoutTx.GetTimeoutHeight(), nil
}

// GetAuctionBidInfo returns the auction bid info from an auction transaction.
func (config *DefaultConfig) GetAuctionBidInfo(tx sdk.Tx) (AuctionBidInfo, error) {
	bid, err := config.GetBid(tx)
	if err != nil {
		return AuctionBidInfo{}, err
	}

	bidder, err := config.GetBidder(tx)
	if err != nil {
		return AuctionBidInfo{}, err
	}

	bundle, err := config.GetBundledTransactions(tx)
	if err != nil {
		return AuctionBidInfo{}, err
	}

	timeout, err := config.GetTimeout(tx)
	if err != nil {
		return AuctionBidInfo{}, err
	}

	return AuctionBidInfo{
		Bid:          bid,
		Bidder:       bidder,
		Transactions: bundle,
		Timeout:      timeout,
	}, nil
}
