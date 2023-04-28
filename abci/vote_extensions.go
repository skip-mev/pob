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
	// VoteExtensionMempool contains the methods required by the VoteExtensionHandler
	// to interact with the local mempool.
	VoteExtensionMempool interface {
		Remove(tx sdk.Tx) error
		AuctionBidSelect(ctx context.Context) sdkmempool.Iterator
		IsAuctionTx(tx sdk.Tx) (bool, error)
		GetBundledTransactions(tx sdk.Tx) ([][]byte, error)
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)
		CountAuctionTx() int
		CountTx() int
	}

	BuilderKeeper interface {
		SetIsCheckVoteExtension(ctx sdk.Context, on bool) error
	}

	VoteExtensionHandler struct {
		mempool       VoteExtensionMempool
		builderKeeper BuilderKeeper
		txDecoder     sdk.TxDecoder
		txEncoder     sdk.TxEncoder
		anteHandler   sdk.AnteHandler
		cache         map[string]error
		currentHeight int64
	}
)

// NewVoteExtensionHandler returns an VoteExtensionHandler that contains the functionality and handlers
// required to inject, process, and validate vote extensions.
func NewVoteExtensionHandler(mp VoteExtensionMempool, bk BuilderKeeper, txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder, ah sdk.AnteHandler,
) *VoteExtensionHandler {
	return &VoteExtensionHandler{
		mempool:       mp,
		builderKeeper: bk,
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
		var voteExtension []byte

		// Reset the cache if necessary
		h.checkStaleCache(ctx.BlockHeight())

		txsToRemove := make(map[sdk.Tx]struct{})

		// Iterate through auction bids until we find a valid one
		for auctionIterator := h.mempool.AuctionBidSelect(ctx); auctionIterator != nil; auctionIterator = auctionIterator.Next() {
			bidTx := auctionIterator.Tx()

			// Verify bid tx can be encoded and included in vote extension
			bidBz, err := h.txEncoder(bidTx)
			if err != nil {
				txsToRemove[bidTx] = struct{}{}
				continue
			}

			hashBz := sha256.Sum256(bidBz)
			hash := hex.EncodeToString(hashBz[:])

			// Validate the auction transaction and cache result
			if err := h.verifyAuctionTx(ctx, bidTx); err != nil {
				h.cache[hash] = err
				txsToRemove[bidTx] = struct{}{}
			} else {
				h.cache[hash] = nil
				voteExtension = bidBz
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
		txBz := req.VoteExtension
		if len(txBz) == 0 {
			return &ResponseVerifyVoteExtension{Status: ResponseVerifyVoteExtension_ACCEPT}, nil
		}

		// Reset the cache if necessary
		h.checkStaleCache(ctx.BlockHeight())

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
			return &ResponseVerifyVoteExtension{Status: ResponseVerifyVoteExtension_REJECT}, err
		}

		// Verify the auction transaction and cache the result
		err = h.verifyAuctionTx(ctx, bidTx)
		h.cache[hash] = err
		if err != nil {
			return &ResponseVerifyVoteExtension{Status: ResponseVerifyVoteExtension_REJECT}, err
		}

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
func (h *VoteExtensionHandler) checkStaleCache(blockHeight int64) {
	if h.currentHeight != blockHeight {
		h.cache = make(map[string]error)
		h.currentHeight = blockHeight
	}
}

// verifyAuctionTx verifies a transaction against the application's state.
func (h *VoteExtensionHandler) verifyAuctionTx(ctx sdk.Context, bidTx sdk.Tx) error {
	// Verify the vote extension is a auction transaction
	isAuctionTx, err := h.mempool.IsAuctionTx(bidTx)
	if err != nil {
		return err
	}

	if !isAuctionTx {
		return fmt.Errorf("vote extension is not a valid auction transaction")
	}

	if h.anteHandler == nil {
		return nil
	}

	// Cache context is used to avoid state changes
	cache, _ := ctx.CacheContext()

	// Set the isCheckVoteExtension flag to avoid checks with the validator's local mempool
	if err := h.builderKeeper.SetIsCheckVoteExtension(ctx, true); err != nil {
		return fmt.Errorf("failed to set isCheckVoteExtension: %s", err)
	}

	if _, err := h.anteHandler(cache, bidTx, false); err != nil {
		return err
	}

	bundledTxs, err := h.mempool.GetBundledTransactions(bidTx)
	if err != nil {
		return err
	}

	// Verify all bundled transactions
	for _, tx := range bundledTxs {
		wrappedTx, err := h.mempool.WrapBundleTransaction(tx)
		if err != nil {
			return err
		}

		if _, err := h.anteHandler(cache, wrappedTx, false); err != nil {
			return err
		}
	}

	return nil
}
