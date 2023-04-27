package abci

import (
	"context"
	"errors"
	"fmt"

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

		// Iterate through auction bids until we find a valid one
		for auctionIterator != nil {
			bidTx := auctionIterator.Tx()

			// Validate the auction transaction
			if err := h.verifyTx(ctx, bidTx); err != nil {
				txsToRemove[bidTx] = struct{}{}
				continue
			}

			// Encode the auction transaction to be included in the vote extension
			if txBz, err := h.txEncoder(bidTx); err != nil {
				txsToRemove[bidTx] = struct{}{}
			} else {
				voteExtension = txBz
				break
			}
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
		// Decode the vote extension which should be a valid auction transaction
		txBz := req.VoteExtension
		bidTx, err := h.txDecoder(txBz)
		if err != nil {
			return &ResponseVerifyVoteExtension{
				Status: ResponseVerifyVoteExtension_REJECT,
			}, err
		}

		// Verify the transaction
		if err := h.verifyTx(ctx, bidTx); err != nil {
			return &ResponseVerifyVoteExtension{
				Status: ResponseVerifyVoteExtension_REJECT,
			}, err
		}

		return &ResponseVerifyVoteExtension{
			Status: ResponseVerifyVoteExtension_ACCEPT,
		}, nil
	}
}

// RemoveTx removes a transaction from the application-side mempool.
func (h *VoteExtensionHandler) RemoveTx(tx sdk.Tx) {
	if err := h.mempool.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}

// verifyTx verifies a transaction against the application's state.
func (h *VoteExtensionHandler) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if h.anteHandler != nil {
		_, err := h.anteHandler(ctx, tx, false)
		return err
	}

	return nil
}
