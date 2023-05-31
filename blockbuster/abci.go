package blockbuster

import (
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// ProposalHandler is a wrapper around baseapp's PrepareProposal and ProcessProposal.
	ProposalHandler struct {
		logger              log.Logger
		mempool             Mempool
		txEncoder           sdk.TxEncoder
		prepareLanesHandler PrepareLanesHandler
		processLanesHandler ProcessLanesHandler
	}

	// Proposal defines a proposal.
	Proposal struct {
		// Txs is the list of transactions in the proposal.
		Txs [][]byte

		// SelectedTxs is the hash of the selected transactions in the proposal.
		SelectedTxs map[string]struct{}

		// TotalTxBytes is the total number of bytes in the proposal.
		TotalTxBytes int64

		// MaxTxBytes is the maximum number of bytes that can be included in the proposal.
		MaxTxBytes int64
	}

	// PrepareLanesHandler wraps all of the lanes Prepare function into a single function.
	PrepareLanesHandler func(ctx sdk.Context, proposal Proposal) Proposal

	// ProcessLanesHandler wraps all of the lanes Process function into a single function.
	ProcessLanesHandler func(ctx sdk.Context, proposalTxs [][]byte) (sdk.Context, error)
)

// NewProposalHandler returns a new proposal handler.
func NewProposalHandler(logger log.Logger, mempool Mempool, txEncoder sdk.TxEncoder) *ProposalHandler {
	return &ProposalHandler{
		logger:              logger,
		mempool:             mempool,
		txEncoder:           txEncoder,
		prepareLanesHandler: ChainPrepareLanes(mempool.registry...),
		processLanesHandler: ChainProcessLanes(mempool.registry...),
	}
}

// PrepareProposalHandler prepares the proposal by selecting transactions from each lane
// according to each lane's selection logic. We select transactions in a greedy fashion. Note that
// each lane has an boundary on the number of bytes that can be included in the proposal. By default,
// the base lane will not have a boundary on the number of bytes that can be included in the proposal and
// will include all valid transactions in the proposal (up to MaxTxBytes).
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
		proposal := h.prepareLanesHandler(ctx, Proposal{
			SelectedTxs: make(map[string]struct{}),
			Txs:         make([][]byte, 0),
			MaxTxBytes:  req.MaxTxBytes,
		})

		return abci.ResponsePrepareProposal{Txs: proposal.Txs}
	}
}

// ProcessProposalHandler processes the proposal by verifying all transactions in the
// according to each lane's verification logic. We verify proposals in a greedy fashion.
// If a lane's portion of the proposal is invalid, we reject the proposal. After a lane's portion
// of the proposal is verified, we pass the remaining transactions to the next lane in the chain.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		if _, err := h.processLanesHandler(ctx, req.Txs); err != nil {
			h.logger.Error("failed to validate the proposal", "err", err)
			return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
		}

		return abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}
	}
}

// ChainPrepareLanes chains together the proposal preparation logic from each lane
// into a single function. The first lane in the chain is the first lane to be prepared and
// the last lane in the chain is the last lane to be prepared.
func ChainPrepareLanes(chain ...Lane) PrepareLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	return func(ctx sdk.Context, proposal Proposal) Proposal {
		return chain[0].PrepareLane(ctx, proposal, ChainPrepareLanes(chain[1:]...))
	}
}

// ChainProcessLanes chains together the proposal verification logic from each lane
// into a single function. The first lane in the chain is the first lane to be verified and
// the last lane in the chain is the last lane to be verified.
func ChainProcessLanes(chain ...Lane) ProcessLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	return func(ctx sdk.Context, proposalTxs [][]byte) (sdk.Context, error) {
		return chain[0].ProcessLane(ctx, proposalTxs, ChainProcessLanes(chain[1:]...))
	}
}
