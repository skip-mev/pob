package abci

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	MempoolVoteExtensionI interface {
		Remove(tx sdk.Tx) error
		AuctionBidSelect(ctx context.Context) sdkmempool.Iterator
		IsAuctionTx(tx sdk.Tx) (bool, error)
	}

	VoteExtensionHandler struct {
		mempool     MempoolVoteExtensionI
		txDecoder   sdk.TxDecoder
		txEncoder   sdk.TxEncoder
		anteHandler sdk.AnteHandler
	}
)

// NewVoteExtensionHandler returns an VoteExtensionHandler that contains the functionality and handlers
// required to inject, process, and validate vote extensions.
func NewVoteExtensionHandler(mp MempoolVoteExtensionI, txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder, ah sdk.AnteHandler) *VoteExtensionHandler {
	return &VoteExtensionHandler{
		mempool:     mp,
		txDecoder:   txDecoder,
		txEncoder:   txEncoder,
		anteHandler: ah,
	}
}

// ExtendVoteHandler returns the ExtendVoteHandler ABCI handler that extracts
// the top bidding valid auction transaction from a validator's local mempool and
// returns it in its vote extension.
func (h *VoteExtensionHandler) ExtendVoteHandler() ExtendVoteHandler {
	return func(ctx sdk.Context, req *RequestExtendVote) (*ResponseExtendVote, error) {
		var (
			voteExtension []byte
		)

		auctionIterator := h.mempool.AuctionBidSelect(ctx)
		txsToRemove := make(map[sdk.Tx]struct{})

		// need to add caching here

		// Iterate through auction bids until we find a valid one
		for auctionIterator != nil {
			bidTx := auctionIterator.Tx()

			// Validate the auction transaction
			if err := h.verifyTx(ctx, bidTx); err != nil {
				txsToRemove[bidTx] = struct{}{}
				continue
			}

			// Encode the auction transaction to be included in the vote extension
			txBz, err := h.txEncoder(bidTx)
			if err != nil {
				txsToRemove[bidTx] = struct{}{}
				continue
			}

			voteExtension = txBz
			break
		}

		// Remove all invalid auction bids from the mempool
		for tx := range txsToRemove {
			h.RemoveTx(tx)
		}

		return &ResponseExtendVote{
			VoteExtension: voteExtension,
		}, nil
	}
}

// VerifyVoteExtensionHandler returns the VerifyVoteExtensionHandler ABCI handler
// that verifies the vote extension included in RequestVerifyVoteExtension.
// In particular, it verifies that the vote extension is a valid auction transaction.
func (h *VoteExtensionHandler) VerifyVoteExtensionHandler() VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *RequestVerifyVoteExtension) (*ResponseVerifyVoteExtension, error) {
		panic("implement me")
	}
}
