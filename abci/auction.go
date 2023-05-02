package abci

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/skip-mev/pob/abci/types"
)

const (
	// VoteExtensionAuctionKey is the key used to extract the auction transaction from the vote extension.
	VoteExtensionAuctionKey = "auction_tx"

	// AuctionInfoIndex is the index of the auction info in the top-of-block proposal.
	AuctionInfoIndex = 0

	// TopBidIndex is the index of the top bid transaction in the top-of-block proposal.
	TopBidIndex = 1

	// TopOfBlockSize is the size of the top-of-block proposal. This includes the auction info and the top auction tx.
	TopOfBlockSize = 2
)

func (h *ProposalHandler) BuildTOB(ctx sdk.Context, voteExtensions [][]byte, maxBytes int64) ([][]byte, int64) {
	// Host the auction to determine which auction transaction will be included in at the very top of the block.
	bidTxBz := h.VoteExtensionAuction(ctx, voteExtensions, maxBytes)

	// Build the auction info that will be used to verify the block proposal in ProcessProposal.
	auctionInfo := types.AuctionInfo{
		TopAuctionTx:   bidTxBz,
		VoteExtensions: voteExtensions,
		MaxTxBytes:     maxBytes,
	}

	// Build the top-of-block proposal.
	topOfBlockTxs := make([][]byte, 0)

	// If there is a valid auction transaction, add the auction transaction along with
	// the transactions included in the bundle.
	if bidTxBz != nil {
		topOfBlockTxs = append(topOfBlockTxs, bidTxBz)

		bidTx, err := h.txDecoder(bidTxBz)
		if err != nil {
			panic(err)
		}

		bidInfo, err := h.mempool.GetAuctionBidInfo(bidTx)
		if err != nil {
			panic(err)
		}

		topOfBlockTxs = append(topOfBlockTxs, bidInfo.Transactions...)
		auctionInfo.NumTxs = int64(len(bidInfo.Transactions)) + 1 // +1 for the auction tx
	}

	// Marshal the auction info to be included in the proposal.
	auctionInfoBz, err := auctionInfo.Marshal()
	if err != nil {
		panic(err)
	}

	proposal := append([][]byte{auctionInfoBz}, topOfBlockTxs...)

	return proposal, int64(len(bidTxBz))
}

// VerifyTOBProposal verifies that the proposal correctly contains the top auction tx based on the
// vote extensions. It also returns the context that the top auction tx was verified in.
func (h *ProposalHandler) VerifyTOB(ctx sdk.Context, expectedProposal [][]byte, auctionInfo types.AuctionInfo) error {
	// Verify we can build the same proposal.
	if proposal, _ := h.BuildTOB(ctx, auctionInfo.VoteExtensions, auctionInfo.MaxTxBytes); !reflect.DeepEqual(proposal, expectedProposal) {
		return fmt.Errorf("proposal does not match the expected proposal")
	}

	return nil
}

// VoteExtensionAuction inputs vote extensions that contain auction transactions and outputs the top bid transaction.
func (h *ProposalHandler) VoteExtensionAuction(ctx sdk.Context, voteExtensions [][]byte, maxBytes int64) []byte {
	var (
		// Track the highest bid.
		topBid sdk.Coin
		// Track the highest bid transaction.
		topBidTxBytes []byte
		// Track the state changes of the highest bid transaction.
		write func()
	)

	// Cache is used to prevent duplicate vote extensions from being processed.
	cache := make(map[string]struct{})
	// txsToRemove is used to track invalid transactions.
	txsToRemove := make(map[sdk.Tx]struct{})

	// Iterate through all vote extensions and find the highest valid bid tx.
	for _, voteExtension := range voteExtensions {
		// Extract the auction transaction from the vote extension.
		auctionTxBz := h.GetAuctionTxFromVoteExtension(voteExtension)

		// If the vote extension is empty or too large, skip it.
		size := int64(len(auctionTxBz))
		if size == 0 || size > maxBytes {
			continue
		}

		hashBz := sha256.Sum256(auctionTxBz)
		hash := hex.EncodeToString(hashBz[:])

		// If the vote extension has already been processed, skip it.
		if _, ok := cache[hash]; ok {
			continue
		}

		// Vote extension should be a valid transaction.
		bidTx, err := h.txDecoder(auctionTxBz)
		if err != nil {
			txsToRemove[bidTx] = struct{}{}
			continue
		}

		// Verify that the bid tx is a valid auction tx and that the
		// bid is higher than the current top bid.
		bidInfo, err := h.mempool.GetAuctionBidInfo(bidTx)
		if err != nil || bidInfo.Bid.IsLT(topBid) {
			continue
		}

		// Verify that the bid tx is valid.
		if write, err = h.verifyAuctionTx(ctx, bidTx); err != nil {
			txsToRemove[bidTx] = struct{}{}
			continue
		}

		topBidTxBytes = auctionTxBz
		topBid = bidInfo.Bid

		// Cache the vote extension.
		cache[hash] = struct{}{}
	}

	// Apply the state changes of the top bid transaction.
	if topBidTxBytes != nil {
		write()
	}

	// Remove all invalid transactions from the vote extensions.
	for tx := range txsToRemove {
		h.RemoveTx(tx)
	}

	return topBidTxBytes
}

// GetAuctionTxFromVoteExtension extracts the auction transaction from the vote extension.
func (h *ProposalHandler) GetAuctionTxFromVoteExtension(voteExtension []byte) []byte {
	voteExtensionInfo := types.VoteExtensionInfo{}
	if err := voteExtensionInfo.Unmarshal(voteExtension); err != nil {
		return nil
	}

	if auctionTx, ok := voteExtensionInfo.Registry[VoteExtensionAuctionKey]; ok {
		return auctionTx
	}

	return nil
}

// verifyAuctionTx verifies that the bid tx is valid and returns the context that the bid tx was verified in.
func (h *ProposalHandler) verifyAuctionTx(ctx sdk.Context, bidTx sdk.Tx) (func(), error) {
	// Cache the context to prevent state changes.
	cache, write := ctx.CacheContext()

	// Verify the bid tx.
	if err := h.verifyTx(cache, bidTx); err != nil {
		return write, err
	}

	bundledTxs, err := h.mempool.GetBundledTransactions(bidTx)
	if err != nil {
		return write, err
	}

	// Verify that the bundle is valid.
	for _, tx := range bundledTxs {
		wrappedTx, err := h.mempool.WrapBundleTransaction(tx)
		if err != nil {
			return write, err
		}

		if err := h.verifyTx(cache, wrappedTx); err != nil {
			return write, err
		}
	}

	return write, nil
}
