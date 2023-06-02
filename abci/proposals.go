package abci

import (
	"errors"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
)

const (
	// NumInjectedTxs is the minimum number of transactions that were injected into
	// the proposal but are not actual transactions. In this case, the auction
	// info is injected into the proposal but should be ignored by the application.ß
	NumInjectedTxs = 1

	// AuctionInfoIndex is the index of the auction info in the proposal.
	AuctionInfoIndex = 0
)

type (
	TOBLaneProposal interface {
		auction.Factory
		sdkmempool.Mempool
		VerifyTx(ctx sdk.Context, tx sdk.Tx) error
	}

	// ProposalHandler contains the functionality and handlers required to\
	// process, validate and build blocks.
	ProposalHandler struct {
		prepareLanesHandler blockbuster.PrepareLanesHandler
		processLanesHandler blockbuster.ProcessLanesHandler
		lane                TOBLaneProposal
		logger              log.Logger
		txEncoder           sdk.TxEncoder
		txDecoder           sdk.TxDecoder
	}
)

// NewProposalHandler returns a ProposalHandler that contains the functionality and handlers
// required to process, validate and build blocks.
func NewProposalHandler(
	lanes []blockbuster.Lane,
	lane TOBLaneProposal,
	logger log.Logger,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
) *ProposalHandler {
	return &ProposalHandler{
		prepareLanesHandler: blockbuster.ChainPrepareLanes(lanes...),
		processLanesHandler: blockbuster.ChainProcessLanes(lanes...),
		lane:                lane,
		logger:              logger,
		txEncoder:           txEncoder,
		txDecoder:           txDecoder,
	}
}

// PrepareProposalHandler returns the PrepareProposal ABCI handler that performs
// top-of-block auctioning and general block proposal construction.
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
		// Proposal includes all of the transactions that will be included in the
		// block along with the vote extensions from the previous block included at
		// the beginning of the proposal. Vote extensions must be included in the
		// first slot of the proposal because they are inaccessible in ProcessProposal.
		txs := make([][]byte, 0)

		// Build the top of block portion of the proposal given the vote extensions
		// from the previous block.
		topOfBlock := h.BuildTOB(ctx, req.LocalLastCommit, req.MaxTxBytes)

		// If information is unable to be marshaled, we return an empty proposal. This will
		// cause another proposal to be generated after it is rejected in ProcessProposal.
		lastCommitInfo, err := req.LocalLastCommit.Marshal()
		if err != nil {
			return abci.ResponsePrepareProposal{Txs: txs}
		}

		auctionInfo := &AuctionInfo{
			ExtendedCommitInfo: lastCommitInfo,
			MaxTxBytes:         req.MaxTxBytes,
			NumTxs:             uint64(len(topOfBlock.Txs)),
		}

		// Add the auction info and top of block transactions into the proposal.
		auctionInfoBz, err := auctionInfo.Marshal()
		if err != nil {
			return abci.ResponsePrepareProposal{Txs: txs}
		}

		txs = append(txs, auctionInfoBz)
		txs = append(txs, topOfBlock.Txs...)

		proposal := blockbuster.Proposal{
			Txs:          txs,
			Cache:        topOfBlock.Cache,
			TotalTxBytes: topOfBlock.Size,
			MaxTxBytes:   req.MaxTxBytes,
		}

		// Prepare the proposal by selecting transactions from each lane according to
		// each lane's selection logic.
		proposal = h.prepareLanesHandler(ctx, proposal)

		return abci.ResponsePrepareProposal{Txs: proposal.Txs}
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		proposal := req.Txs

		// Verify that the same top of block transactions can be built from the vote
		// extensions included in the proposal.
		auctionInfo, err := h.VerifyTOB(ctx, proposal)
		if err != nil {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		// Verify that the rest of the proposal is valid according to each lane's verification logic.
		if _, err = h.processLanesHandler(ctx, proposal[auctionInfo.NumTxs:]); err != nil {
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}

// RemoveTx removes a transaction from the application-side mempool.
func (h *ProposalHandler) RemoveTx(tx sdk.Tx) {
	if err := h.lane.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}
