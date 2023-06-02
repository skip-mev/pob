package blockbuster

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	// ProposalHandler is a wrapper around the ABCI++ PrepareProposal and ProcessProposal
	// handlers.
	ProposalHandler struct {
		logger              log.Logger
		prepareLanesHandler PrepareLanesHandler
		processLanesHandler ProcessLanesHandler
	}

	// Proposal defines a proposal.
	Proposal struct {
		// Txs is the list of transactions in the proposal.
		Txs [][]byte

		// SelectedTxs is a cache of the selected transactions in the proposal.
		Cache map[string]struct{}

		// TotalTxBytes is the total number of bytes currently included in the proposal.
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
func NewProposalHandler(logger log.Logger, mempool *Mempool, txEncoder sdk.TxEncoder) *ProposalHandler {
	return &ProposalHandler{
		logger:              logger,
		prepareLanesHandler: ChainPrepareLanes(mempool.registry...),
		processLanesHandler: ChainProcessLanes(mempool.registry...),
	}
}

// PrepareProposalHandler prepares the proposal by selecting transactions from each lane
// according to each lane's selection logic. We select transactions in a greedy fashion. Note that
// each lane has an boundary on the number of bytes that can be included in the proposal. By default,
// the default lane will not have a boundary on the number of bytes that can be included in the proposal and
// will include all valid transactions in the proposal (up to MaxTxBytes).
func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) (resp abci.ResponsePrepareProposal) {
		// In the case where there is a panic, we recover here and return an empty proposal.
		defer func() {
			if err := recover(); err != nil {
				resp = abci.ResponsePrepareProposal{Txs: make([][]byte, 0)}
				h.logger.Error("failed to prepare proposal", "err", err)
			}
		}()

		proposal := h.prepareLanesHandler(ctx, Proposal{
			Cache:      make(map[string]struct{}),
			Txs:        make([][]byte, 0),
			MaxTxBytes: req.MaxTxBytes,
		})

		resp = abci.ResponsePrepareProposal{
			Txs: proposal.Txs,
		}

		return
	}
}

// ProcessProposalHandler processes the proposal by verifying all transactions in the proposal
// according to each lane's verification logic. We verify proposals in a greedy fashion.
// If a lane's portion of the proposal is invalid, we reject the proposal. After a lane's portion
// of the proposal is verified, we pass the remaining transactions to the next lane in the chain.
func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) (resp abci.ResponseProcessProposal) {
		// In the case where any of the lanes panic, we recover here and return a reject status.
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("failed to process proposal", "err", err)
				resp = abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_REJECT}
			}
		}()

		// Verify the proposal using the verification logic from each lane.
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
//
// In the case where any of the lanes fail to prepare the proposal, the lane that failed
// will be skipped and the next lane in the chain will be called to prepare the proposal.
//
// TODO: Determine how expensive the caches are.
func ChainPrepareLanes(chain ...Lane) PrepareLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != Terminator{}) {
		chain = append(chain, Terminator{})
	}

	return func(ctx sdk.Context, partialProposal Proposal) (finalProposal Proposal) {
		// Cache the context in the case where any of the lanes fail to prepare the proposal.
		cacheCtx, write := ctx.CacheContext()

		defer func() {
			if err := recover(); err != nil {
				lanesRemaining := len(chain)
				switch {
				case lanesRemaining <= 2:
					// If there are only two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the proposal and the second lane in the
					// chain is the terminator lane. We return the proposal as is.
					finalProposal = partialProposal
				default:
					// If there are more than two lanes remaining, then the first lane in the chain
					// is the lane that failed to prepare the proposal but the second lane in the
					// chain is not the terminator lane so there could potentially be more transactions
					// added to the proposal
					finalProposal = chain[1].PrepareLane(ctx, partialProposal, ChainPrepareLanes(chain[2:]...))
				}

				return
			}

			// Write the cache to the context
			write()
		}()

		return chain[0].PrepareLane(cacheCtx, partialProposal, ChainPrepareLanes(chain[1:]...))
	}
}

// ChainProcessLanes chains together the proposal verification logic from each lane
// into a single function. The first lane in the chain is the first lane to be verified and
// the last lane in the chain is the last lane to be verified.
func ChainProcessLanes(chain ...Lane) ProcessLanesHandler {
	if len(chain) == 0 {
		return nil
	}

	// Handle non-terminated decorators chain
	if (chain[len(chain)-1] != Terminator{}) {
		chain = append(chain, Terminator{})
	}

	return func(ctx sdk.Context, proposalTxs [][]byte) (sdk.Context, error) {
		if len(proposalTxs) == 0 {
			return ctx, nil
		}

		if err := chain[0].ProcessLaneBasic(proposalTxs); err != nil {
			return ctx, err
		}

		return chain[0].ProcessLane(ctx, proposalTxs, ChainProcessLanes(chain[1:]...))
	}
}

// Terminator Lane will get added to the chain to simplify chaining code so that we
// don't need to check if next == nil further up the chain.
//
// sniped from the sdk
//
//	                      ______
//	                   <((((((\\\
//	                   /      . }\
//	                   ;--..--._|}
//	(\                 '--/\--'  )
//	 \\                | '-'  :'|
//	  \\               . -==- .-|
//	   \\               \.__.'   \--._
//	   [\\          __.--|       //  _/'--.
//	   \ \\       .'-._ ('-----'/ __/      \
//	    \ \\     /   __>|      | '--.       |
//	     \ \\   |   \   |     /    /       /
//	      \ '\ /     \  |     |  _/       /
//	       \  \       \ |     | /        /
//	 snd    \  \      \        /
type Terminator struct{}

var _ Lane = (*Terminator)(nil)

// PrepareLane is a no-op
func (t Terminator) PrepareLane(_ sdk.Context, proposal Proposal, _ PrepareLanesHandler) Proposal {
	return proposal
}

// ProcessLane is a no-op
func (t Terminator) ProcessLane(ctx sdk.Context, _ [][]byte, _ ProcessLanesHandler) (sdk.Context, error) {
	return ctx, nil
}

// Name returns the name of the lane
func (t Terminator) Name() string {
	return "Terminator"
}

// Match is a no-op
func (t Terminator) Match(sdk.Tx) bool {
	return false
}

// VerifyTx is a no-op
func (t Terminator) VerifyTx(sdk.Context, sdk.Tx) error {
	return fmt.Errorf("Terminator lane should not be called")
}

// Contains is a no-op
func (t Terminator) Contains(sdk.Tx) (bool, error) {
	return false, nil
}

// CountTx is a no-op
func (t Terminator) CountTx() int {
	return 0
}

// Insert is a no-op
func (t Terminator) Insert(context.Context, sdk.Tx) error {
	return nil
}

// Remove is a no-op
func (t Terminator) Remove(sdk.Tx) error {
	return nil
}

// Select is a no-op
func (t Terminator) Select(context.Context, [][]byte) sdkmempool.Iterator {
	return nil
}

// ValidateLaneBasic is a no-op
func (t Terminator) ProcessLaneBasic([][]byte) error {
	return nil
}
