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

// GetAuctionBidInfo defines a default function that returns the auction bid info from
// an auction transaction. In the default case, the auction bid info is stored in the
// MsgAuctionBid message.
func (config *DefaultConfig) GetAuctionBidInfo(tx sdk.Tx) (*AuctionBidInfo, error) {
	msg, err := GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, nil
	}

	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return nil, fmt.Errorf("invalid bidder address (%s): %w", msg.Bidder, err)
	}

	timeoutTx, ok := tx.(TxWithTimeoutHeight)
	if !ok {
		return nil, fmt.Errorf("cannot extract timeout; transaction does not implement TxWithTimeoutHeight")
	}

	return &AuctionBidInfo{
		Bid:          msg.Bid,
		Bidder:       bidder,
		Transactions: msg.Transactions,
		Timeout:      timeoutTx.GetTimeoutHeight(),
	}, nil
}
