package abci

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
		mempool       MempoolVoteExtensionI
		txDecoder     sdk.TxDecoder
		txEncoder     sdk.TxEncoder
		anteHandler   sdk.AnteHandler
		cache         map[string]error
		currentHeight int64
	}
)

// NewVoteExtensionHandler returns an VoteExtensionHandler that contains the functionality and handlers
// required to inject, process, and validate vote extensions.
func NewVoteExtensionHandler(mp MempoolVoteExtensionI, txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder, ah sdk.AnteHandler) *VoteExtensionHandler {
	return &VoteExtensionHandler{
		mempool:       mp,
		txDecoder:     txDecoder,
		txEncoder:     txEncoder,
		anteHandler:   ah,
		cache:         make(map[string]error),
		currentHeight: 0,
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

		// Reset the cache if necessary
		h.checkStaleCache(ctx)

		// Iterate through auction bids until we find a valid one
		auctionIterator := h.mempool.AuctionBidSelect(ctx)
		txsToRemove := make(map[sdk.Tx]struct{})

		for auctionIterator != nil {
			bidTx := auctionIterator.Tx()

			// Verify bid tx can be encoded and included in vote extension
			bidBz, err := h.txEncoder(bidTx)
			if err != nil {
				txsToRemove[bidTx] = struct{}{}
			}

			hashBz := sha256.Sum256(bidBz)
			hash := hex.EncodeToString(hashBz[:])

			// Validate the auction transaction and cache result
			if err := h.verifyTx(ctx, bidTx); err != nil {
				h.cache[hash] = err
				txsToRemove[bidTx] = struct{}{}
			} else {
				h.cache[hash] = nil
				voteExtension = bidBz
				break
			}

			auctionIterator = auctionIterator.Next()
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
		txBz := req.VoteExtension
		hashBz := sha256.Sum256(txBz)
		hash := hex.EncodeToString(hashBz[:])

		// Short circuit if we have already verified this vote extension
		if err, ok := h.cache[hash]; ok {
			if err != nil {
				return &ResponseVerifyVoteExtension{Status: ResponseVerifyVoteExtension_REJECT}, err
			}

			return &ResponseVerifyVoteExtension{Status: ResponseVerifyVoteExtension_ACCEPT}, nil
		}

		// Decode the vote extension which should be a valid auction transaction
		bidTx, err := h.txDecoder(txBz)
		if err != nil {
			return &ResponseVerifyVoteExtension{
				Status: ResponseVerifyVoteExtension_REJECT,
			}, err
		}

		// Verify the auction transaction and cache the result
		if err := h.verifyTx(ctx, bidTx); err != nil {
			h.cache[hash] = err

			return &ResponseVerifyVoteExtension{Status: ResponseVerifyVoteExtension_REJECT}, err
		}

		h.cache[hash] = nil

		return &ResponseVerifyVoteExtension{Status: ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

// RemoveTx removes a transaction from the application-side mempool.
func (h *VoteExtensionHandler) RemoveTx(tx sdk.Tx) {
	if err := h.mempool.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}

// checkStaleCache checks if the current height is greater than the previous height at which
// the vote extensions were verified in. If so, it resets the cache to allow transactions to be
// reverified.
func (h *VoteExtensionHandler) checkStaleCache(ctx sdk.Context) {
	if blockHeight := ctx.BlockHeight(); h.currentHeight != blockHeight {
		h.cache = make(map[string]error)
		h.currentHeight = blockHeight
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
