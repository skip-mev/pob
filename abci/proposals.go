package abci

import (
	"context"
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	types "github.com/skip-mev/pob/abci/types"
	"github.com/skip-mev/pob/mempool"
)

type (
	// ProposalMempool contains the methods required by the ProposalHandler
	// to interact with the local mempool.
	ProposalMempool interface {
		sdkmempool.Mempool
		AuctionBidSelect(ctx context.Context) sdkmempool.Iterator
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
		// Utilize vote extensions to determine the top bidding valid auction transaction
		// across the network.
		voteExtensions := make([][]byte, len(req.LocalLastCommit.Votes))
		for i, vote := range req.LocalLastCommit.Votes {
			voteExtensions[i] = vote.VoteExtension
		}

		// Build the top of block portion of the proposal and apply state changes relevant to the top valid auction
		// transaction.
		proposal, totalTxBytes := h.BuildTOB(ctx, voteExtensions, req.MaxTxBytes)

		// Select remaining transactions for the block proposal until we've reached
		// size capacity.
		txsToRemove := map[sdk.Tx]struct{}{}
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
				proposal = append(proposal, txBz)
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

		return abci.ResponsePrepareProposal{Txs: proposal}
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		proposal := req.Txs

		// Ensure that the proposal is not empty. Empty proposals still must include the auction info.
		if len(proposal) <= TopOfBlockSize {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		auctionInfo, err := h.getAuctionInfoFromProposal(proposal[AuctionInfoIndex])
		if err != nil {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// The proposal should include the auction info and the top auction tx along
		// with the transactions included in the bundle.
		minProposalSize := auctionInfo.NumTxs + 1

		// Verify that the proposal is the correct size.
		if len(proposal) < int(minProposalSize) {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Verify the top of the block transactions.
		if err := h.VerifyTOB(ctx, proposal[:minProposalSize], auctionInfo); err != nil {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Verify the remaining transactions.
		for _, txBz := range proposal[minProposalSize:] {
			tx, err := h.ProcessProposalVerifyTx(ctx, txBz)
			if err != nil {
				return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}

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

func (h *ProposalHandler) getAuctionInfoFromProposal(infoBz []byte) (types.AuctionInfo, error) {
	auctionInfo := types.AuctionInfo{}
	if err := auctionInfo.Unmarshal(infoBz); err != nil {
		return types.AuctionInfo{}, err
	}

	return auctionInfo, nil
}

// VerifyTx verifies a transaction against the application's state.
func (h *ProposalHandler) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if h.anteHandler != nil {
		_, err := h.anteHandler(ctx, tx, false)
		return err
	}

	return nil
}
