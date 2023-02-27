package mempool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// WrappedTx defines a wrapper around an sdk.Tx with additional metadata.
	WrappedTx struct {
		sdk.Tx

		hash [32]byte
	}

	// WrappedBidTx defines a wrapper around an sdk.Tx that contains a single
	// MsgAuctionBid message with additional metadata.
	WrappedBidTx struct {
		sdk.Tx

		hash [32]byte
		bid  sdk.Coins
	}
)
