package abci

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// TopAuctionTxDelimeter is the delimeter used to separate the auction tx
	// from the proposal.
	TopAuctionTxDelimeter = "top_auction_tx_delimeter"

	// VoteExtensionsDelimeter is the delimeter used to separate the vote extensions
	// from the proposal.
	VoteExtensionsDelimeter = "vote_extensions_delimeter"

	// MinProposalSize is the minimum size of a proposal. The proposal must contain
	// at least the top auction tx and the delimeters.
	MinProposalSize = 3
)

type (
	// UnwrappedProposal is a proposal that has been unwrapped from the proposal
	// format used by the comet ABCI. The proposal is structured as follows:
	// top_auction_tx
	// <top_auction_tx_delimeter>
	// vote_extension_1
	// vote_extension_2
	// ...
	// <vote_extensions_delimeter>
	// tx_1
	// tx_2
	// ...
	UnwrappedProposal struct {
		TopAuctionTx   []byte
		VoteExtensions [][]byte
		Txs            [][]byte
	}
)

// BuildTOBProposal inputs vote extensions that contain auction transactions and outputs a top of block
// proposal that contains the highest valid auction transaction and all other vote extensions. It also
// returns the context that the top auction tx was verified in.
func (h *ProposalHandler) BuildTOBProposal(ctx sdk.Context, voteExtensions [][]byte) (sdk.Context, [][]byte) {
	var (
		topBid   sdk.Coin
		topBidTx []byte
	)

	// Track the context that the top bid tx was verified in.
	topBidCtx := ctx
	// Cache is used to prevent duplicate vote extensions from being processed.
	cacheTx := make(map[string]struct{})

	// Iterate through all vote extensions and find the highest valid bid tx.
	txsToRemove := make(map[sdk.Tx]struct{})
	for _, voteExtension := range voteExtensions {
		hashBz := sha256.Sum256(voteExtension)
		hash := hex.EncodeToString(hashBz[:])

		// If the vote extension has already been processed, skip it.
		if _, ok := cacheTx[hash]; ok {
			continue
		}

		// Vote extension should be a valid transaction.
		bidTx, err := h.txDecoder(voteExtension)
		if err != nil {
			txsToRemove[bidTx] = struct{}{}
			continue
		}

		// Verify that the bid tx is valid.
		cacheCtx, err := h.verifyAuctionTx(ctx, bidTx)
		if err != nil {
			fmt.Println(err)
			txsToRemove[bidTx] = struct{}{}
			continue
		}

		bidInfo, err := h.mempool.GetAuctionBidInfo(bidTx)
		if err != nil {
			continue
		}

		// If the bid is higher than the current top bid, update the top bid.
		if topBid.IsNil() || topBid.IsLT(bidInfo.Bid) {
			topBid = bidInfo.Bid
			topBidTx = voteExtension
			topBidCtx = cacheCtx
		}

		// Cache the vote extension.
		cacheTx[hash] = struct{}{}
	}

	// Remove all invalid transactions from the vote extensions.
	for tx := range txsToRemove {
		h.RemoveTx(tx)
	}

	// Build the top of block portion of the proposal.
	proposal := make([][]byte, 0)
	proposal = append(proposal, topBidTx)
	proposal = append(proposal, []byte(TopAuctionTxDelimeter))
	proposal = append(proposal, voteExtensions...)
	proposal = append(proposal, []byte(VoteExtensionsDelimeter))

	return topBidCtx, proposal
}

// VerifyTOBProposal verifies that the proposal correctly contains the top auction tx based on the
// vote extensions. It also returns the context that the top auction tx was verified in.
func (h *ProposalHandler) VerifyTOBProposal(ctx sdk.Context, topBidTx []byte, voteExtensions [][]byte) (sdk.Context, error) {
	// Verify we can build the same proposal.
	cacheCtx, proposal := h.BuildTOBProposal(ctx, voteExtensions)
	if !bytes.Equal(proposal[0], topBidTx) {
		return ctx, fmt.Errorf("top vote extension is not the same as the top bid tx")
	}

	return cacheCtx, nil
}

// UnwrapProposal unwraps the proposal into its constituent parts.
func UnwrapProposal(proposal [][]byte) (*UnwrappedProposal, error) {
	if len(proposal) < MinProposalSize {
		return nil, fmt.Errorf("proposal is too small. expected at least %d slots, got %d slots", MinProposalSize, len(proposal))
	}

	// Get the top auction tx.
	topAuctionTx := proposal[0]

	// Verify that the proposal is structured correctly.
	if !bytes.Equal([]byte(TopAuctionTxDelimeter), proposal[1]) {
		return nil, fmt.Errorf("invalid proposal format. top auction tx delimeter not found")
	}

	var (
		voteExtensionEndIndex      int
		seenVoteExtensionDelimeter bool
	)

	for i, voteExtension := range proposal {
		if bytes.Equal([]byte(VoteExtensionsDelimeter), voteExtension) {
			voteExtensionEndIndex = i
			seenVoteExtensionDelimeter = true
			break
		}
	}

	if !seenVoteExtensionDelimeter {
		return nil, fmt.Errorf("invalid proposal format. vote extension delimeter not found")
	}

	return &UnwrappedProposal{
		TopAuctionTx:   topAuctionTx,
		VoteExtensions: proposal[2:voteExtensionEndIndex],
		Txs:            proposal[voteExtensionEndIndex+1:],
	}, nil
}

// verifyAuctionTx verifies that the bid tx is valid and returns the context that the bid tx was verified in.
func (h *ProposalHandler) verifyAuctionTx(ctx sdk.Context, bidTx sdk.Tx) (sdk.Context, error) {
	// Cache the context to prevent state changes.
	cache, _ := ctx.CacheContext()

	// Verify the bid tx.
	if err := h.verifyTx(cache, bidTx); err != nil {
		return ctx, err
	}

	bundledTxs, err := h.mempool.GetBundledTransactions(bidTx)
	if err != nil {
		return ctx, err
	}

	// Verify that the bundle is valid.
	for _, tx := range bundledTxs {
		wrappedTx, err := h.mempool.WrapBundleTransaction(tx)
		if err != nil {
			return ctx, err
		}

		if err := h.verifyTx(cache, wrappedTx); err != nil {
			return ctx, err
		}
	}

	return cache, nil
}
