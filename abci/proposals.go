package abci

import (
	"errors"
	"fmt"
	"reflect"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	types "github.com/skip-mev/pob/abci/types"
	mempool "github.com/skip-mev/pob/mempool"
)

const (
	// MinProposalSize is the minimum size of a proposal. Each proposal must contain
	// at least the auction info.
	MinProposalSize = 1

	// AuctionInfoIndex is the index of the auction info in the proposal.
	AuctionInfoIndex = 0
)

type (
	// ProposalMempool contains the methods required by the ProposalHandler
	// to interact with the local mempool.
	ProposalMempool interface {
		sdkmempool.Mempool
		GetBundledTransactions(tx sdk.Tx) ([][]byte, error)
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)
		IsAuctionTx(tx sdk.Tx) (bool, error)
		GetAuctionBidInfo(tx sdk.Tx) (mempool.AuctionBidInfo, error)
	}

	// ProposalHandler contains the functionality and handlers required to\
	// process, validate and build blocks.
	ProposalHandler struct {
		mempool     ProposalMempool
		logger      log.Logger
		anteHandler sdk.AnteHandler
		txEncoder   sdk.TxEncoder
		txDecoder   sdk.TxDecoder
	}
)

// NewProposalHandler returns a ProposalHandler that contains the functionality and handlers
// required to process, validate and build blocks.
func NewProposalHandler(
	mp ProposalMempool,
	logger log.Logger,
	anteHandler sdk.AnteHandler,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
) *ProposalHandler {
	return &ProposalHandler{
		mempool:     mp,
		logger:      logger,
		anteHandler: anteHandler,
		txEncoder:   txEncoder,
		txDecoder:   txDecoder,
	}
}

// PrepareProposalHandler returns the PrepareProposal ABCI handler that performs
// top-of-block auctioning and general block proposal construction.
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
		voteExtensions := make([][]byte, len(req.LocalLastCommit.Votes))
		for i, vote := range req.LocalLastCommit.Votes {
			voteExtensions[i] = vote.VoteExtension
		}

		// Extract vote extensions and bid transactions from the committed vote extensions.
		bidTxs := h.GetBidsFromVoteExtensions(voteExtensions)

		// Build the top of block portion of the proposal.
		txs, totalTxBytes := h.BuildTOB(ctx, bidTxs, req.MaxTxBytes)

		// Track info about how the auction was held for re-use in ProcessProposal.
		auctionInfo := types.AuctionInfo{
			VoteExtensions: voteExtensions,
			MaxTxBytes:     req.MaxTxBytes,
			NumTxs:         uint64(len(txs)),
		}

		// Track the transactions that need to be removed from the mempool.
		txsToRemove := make(map[sdk.Tx]struct{}, 0)

		// Select remaining transactions for the block proposal until we've reached
		// size capacity.
		iterator := h.mempool.Select(ctx, nil)
		for ; iterator != nil; iterator = iterator.Next() {
			memTx := iterator.Tx()

			txBz, err := h.PrepareProposalVerifyTx(ctx, memTx)
			if err != nil {
				txsToRemove[memTx] = struct{}{}
				continue
			}

			txSize := int64(len(txBz))
			if totalTxBytes += txSize; totalTxBytes <= req.MaxTxBytes {
				txs = append(txs, txBz)
			} else {
				// We've reached capacity per req.MaxTxBytes so we cannot select any
				// more transactions.
				break
			}
		}

		// Remove all invalid transactions from the mempool.
		for tx := range txsToRemove {
			h.RemoveTx(tx)
		}

		// Build the proposal. The proposal must include the auction info in the first slot.
		// The remaining transactions are the block's transactions.
		auctionInfoBz, _ := auctionInfo.Marshal()
		proposal := [][]byte{auctionInfoBz}
		proposal = append(proposal, txs...)

		return abci.ResponsePrepareProposal{Txs: proposal}
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		proposal := req.Txs
		if len(proposal) < MinProposalSize {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Extract the auction info from the proposal.
		auctionInfo := types.AuctionInfo{}
		if err := auctionInfo.Unmarshal(proposal[AuctionInfoIndex]); err != nil {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		if len(proposal) < int(auctionInfo.NumTxs)+MinProposalSize {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Build the top of block proposal from the auction info.
		bidTxs := h.GetBidsFromVoteExtensions(auctionInfo.VoteExtensions)
		tobProposal, _ := h.BuildTOB(ctx, bidTxs, auctionInfo.MaxTxBytes)

		// Verify that the top of block proposal matches the proposal.
		expectedTOBProposal := proposal[MinProposalSize : auctionInfo.NumTxs+MinProposalSize]
		if !reflect.DeepEqual(expectedTOBProposal, tobProposal) {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Verify that the remaining transactions in the proposal are valid.
		for _, txBz := range proposal[auctionInfo.NumTxs+MinProposalSize:] {
			tx, err := h.ProcessProposalVerifyTx(ctx, txBz)
			if err != nil {
				return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}

			// The only auction transactions that should be included in the block proposal
			// must be at the top of the block.
			if isAuctionTx, err := h.mempool.IsAuctionTx(tx); err != nil || isAuctionTx {
				return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}
		}

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}

// PrepareProposalVerifyTx encodes a transaction and verifies it.
func (h *ProposalHandler) PrepareProposalVerifyTx(ctx sdk.Context, tx sdk.Tx) ([]byte, error) {
	txBz, err := h.txEncoder(tx)
	if err != nil {
		return nil, err
	}

	return txBz, h.verifyTx(ctx, tx)
}

// ProcessProposalVerifyTx decodes a transaction and verifies it.
func (h *ProposalHandler) ProcessProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error) {
	tx, err := h.txDecoder(txBz)
	if err != nil {
		return nil, err
	}

	return tx, h.verifyTx(ctx, tx)
}

// RemoveTx removes a transaction from the application-side mempool.
func (h *ProposalHandler) RemoveTx(tx sdk.Tx) {
	if err := h.mempool.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}

// VerifyTx verifies a transaction against the application's state.
func (h *ProposalHandler) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if h.anteHandler != nil {
		_, err := h.anteHandler(ctx, tx, false)
		return err
	}

	return nil
}
