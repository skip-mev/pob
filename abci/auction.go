package abci

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/skip-mev/pob/abci/types"
)

const (
	// VoteExtensionAuctionKey is the key used to extract the auction transaction from the vote extension.
	VoteExtensionAuctionKey = "auction_tx"
)

// TopOfBlockProposal contains the top of block proposal that is returned by the
// BuildTOB method.
type TopOfBlockProposal struct {
	// Proposal is the top of block proposal.
	Proposal [][]byte

	// ProposalSize is the total size of the top of block proposal.
	Size int64

	// Cache is the cache of transactions that were seen in order to ignore them
	// when building the rest of the block.
	Cache map[string]struct{}
}

// NewTopOfBlockProposal returns a new TopOfBlockProposal.
func NewTopOfBlockProposal() TopOfBlockProposal {
	return TopOfBlockProposal{
		Cache: make(map[string]struct{}),
	}
}

// BuildTOB inputs all of the bid transactions and outputs a top of block proposal that includes
// the highest bidding valid transaction along with all the bundled transactions.
func (h *ProposalHandler) BuildTOB(ctx sdk.Context, bidTxs []sdk.Tx, maxBytes int64) TopOfBlockProposal {
	var topOfBlockProposal TopOfBlockProposal

	// Sort the auction transactions by their bid amount in descending order.
	sort.Slice(bidTxs, func(i, j int) bool {
		bidInfoI, err := h.mempool.GetAuctionBidInfo(bidTxs[i])

		// In the case of an error, we want to sort the transaction to the end of the list.
		if err != nil {
			return false
		}

		bidInfoJ, err := h.mempool.GetAuctionBidInfo(bidTxs[j])
		// In the case of an error, we want to sort the transaction to the end of the list.
		if err != nil {
			return true

		}

		return bidInfoI.Bid.IsGTE(bidInfoJ.Bid)
	})

	// Track the transactions we can remove from the block
	txsToRemove := make(map[sdk.Tx]struct{})

	// Attempt to select the highest bid transaction that is valid and whose
	// bundled transactions are valid.
	for _, bidTx := range bidTxs {
		// Cache the context so that we can write it back to the original context
		// when we know we have a valid top of block bundle.
		cacheCtx, write := ctx.CacheContext()

		topOfBlockProposal, err := h.buildTOB(cacheCtx, bidTx)
		if err != nil {
			h.logger.Info(
				"vote extension auction failed to verify auction tx",
				"err", err,
			)
			txsToRemove[bidTx] = struct{}{}
			continue
		}

		if topOfBlockProposal.Size <= maxBytes {
			// At this point, both the bid transaction itself and all the bundled
			// transactions are valid. So we select the bid transaction along with
			// all the bundled transactions and apply the state changes to the cache
			// context.
			write()
			break
		}

		h.logger.Info(
			"failed to select auction bid tx; auction tx size is too large",
			"tx_size", topOfBlockProposal.Size,
			"max_size", maxBytes,
		)
	}

	// Remove all of the transactions that were not valid.
	for tx := range txsToRemove {
		h.RemoveTx(tx)
	}

	return topOfBlockProposal
}

// getVoteExtensionInfo returns all of the vote extensions supplied in the request alongside
// a sorted - potentially subset - of the vote extensions that contain auction transactions.
func (h *ProposalHandler) GetBidsFromVoteExtensions(voteExtensions [][]byte) []sdk.Tx {
	bidTxs := make([]sdk.Tx, 0)

	// Iterate through all vote extensions and extract the auction transactions.
	for i, voteExtension := range voteExtensions {
		// Check if the vote extension contains an auction transaction.
		if bidTx, err := h.getAuctionTxFromVoteExtension(voteExtension); err == nil {
			bidTxs = append(bidTxs, bidTx)
		}

		voteExtensions[i] = voteExtension
	}

	return bidTxs
}

// getAuctionTxFromVoteExtension extracts the auction transaction from the vote extension.
func (h *ProposalHandler) getAuctionTxFromVoteExtension(voteExtension []byte) (sdk.Tx, error) {
	voteExtensionInfo := types.VoteExtensionInfo{}
	if err := voteExtensionInfo.Unmarshal(voteExtension); err != nil {
		return nil, err
	}

	// Check if the vote extension's registry contains an auction transaction.
	bidTxBz, ok := voteExtensionInfo.Registry[VoteExtensionAuctionKey]
	if !ok {
		return nil, fmt.Errorf("vote extension does not contain auction transaction in its registry")
	}

	// Attempt to unmarshal the auction transaction.
	bidTx, err := h.txDecoder(bidTxBz)
	if err != nil {
		return nil, err
	}

	// Verify the auction transaction has bid information.
	if isAuctionTx, err := h.mempool.IsAuctionTx(bidTx); err != nil || !isAuctionTx {
		return nil, err
	}

	return bidTx, nil
}

// buildTOB verifies that the auction transaction is valid and returns the bytes of the
// auction transaction and all of its bundled transactions.
func (h *ProposalHandler) buildTOB(ctx sdk.Context, bidTx sdk.Tx) (TopOfBlockProposal, error) {
	proposal := NewTopOfBlockProposal()

	bidTxBz, err := h.PrepareProposalVerifyTx(ctx, bidTx)
	if err != nil {
		return proposal, err
	}

	bundledTransactions, err := h.mempool.GetBundledTransactions(bidTx)
	if err != nil {
		return proposal, err
	}

	// store the bytes of each ref tx as sdk.Tx bytes in order to build a valid proposal
	sdkTxBytes := make([][]byte, len(bundledTransactions))

	// Ensure that the bundled transactions are valid
	for index, rawRefTx := range bundledTransactions {
		refTx, err := h.mempool.WrapBundleTransaction(rawRefTx)
		if err != nil {
			return TopOfBlockProposal{}, err
		}

		txBz, err := h.PrepareProposalVerifyTx(ctx, refTx)
		if err != nil {
			return TopOfBlockProposal{}, err
		}

		hashBz := sha256.Sum256(refTx)
		hash := hex.EncodeToString(hashBz[:])

		sdkTxBytes[index] = txBz
	}

	proposal := [][]byte{bidTxBz}
	proposal = append(proposal, sdkTxBytes...)

	return proposal, int64(len(bidTxBz)), nil
}
