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

		// WrapBundleTransaction defines a function that wraps a bundle transaction into a sdk.Tx.
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)

		// GetAuctionBidInfo defines a function that returns the bid info from an auction transaction.
		GetAuctionBidInfo(tx sdk.Tx) (*AuctionBidInfo, error)
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

// WrapBundleTransaction defines a default function that wraps a transaction
// that is included in the bundle into a sdk.Tx. In the default case, the transaction
// that is included in the bundle will be the raw bytes of an sdk.Tx so we can just
// decode it.
func (config *DefaultConfig) WrapBundleTransaction(tx []byte) (sdk.Tx, error) {
	return config.txDecoder(tx)
}

// GetAuctionBidInfo returns the auction bid info from an auction transaction.
func (config *DefaultConfig) GetAuctionBidInfo(tx sdk.Tx) (*AuctionBidInfo, error) {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, fmt.Errorf("transaction is not a valid auction bid transaction")
	}

	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return nil, fmt.Errorf("invalid bidder address (%s): %w", msg.Bidder, err)
	}

	timeout, err := config.getTimeout(tx)
	if err != nil {
		return nil, err
	}

	return &AuctionBidInfo{
		Bid:          msg.Bid,
		Bidder:       bidder,
		Transactions: msg.Transactions,
		Timeout:      timeout,
	}, nil
}

// getTimeout defines a default function that returns the timeout of an auction transaction.
func (config *DefaultConfig) getTimeout(tx sdk.Tx) (uint64, error) {
	timeoutTx, ok := tx.(TxWithTimeoutHeight)
	if !ok {
		return 0, fmt.Errorf("transaction does not implement TxWithTimeoutHeight")
	}

	return timeoutTx.GetTimeoutHeight(), nil
}
