package constructor

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

// PrepareLane will prepare a partial proposal for the default lane. It will select and include
// all valid transactions in the mempool that are not already in the partial proposal.
// The default lane orders transactions by the sdk.Context priority.
func (l *LaneConstructor[C]) PrepareLane(
	ctx sdk.Context,
	proposal blockbuster.BlockProposal,
	maxTxBytes int64,
	next blockbuster.PrepareLanesHandler,
) (blockbuster.BlockProposal, error) {
	txs, txsToRemove, err := l.prepareLaneHandler(ctx, proposal, maxTxBytes)
	if err != nil {
		return proposal, err
	}

	// Remove all transactions that were invalid during the creation of the partial proposal.
	if err := utils.RemoveTxsFromLane(txsToRemove, l.LaneMempool); err != nil {
		l.Logger().Error(
			"failed to remove transactions from lane",
			"err", err,
		)

		return proposal, err
	}

	// Update the partial proposal with the selected transactions. If the proposal is unable to
	// be updated, we return an error. The proposal will only be modified if it passes all
	// of the invarient checks.
	if err := proposal.UpdateProposal(l, txs); err != nil {
		return proposal, err
	}

	return next(ctx, proposal)
}

// ProcessLane verifies the default lane's portion of a block proposal. Since the default lane's
// ProcessLaneBasic function ensures that all of the default transactions are in the correct order,
// we only need to verify the contiguous set of transactions that match to the default lane.
func (l *LaneConstructor[C]) ProcessLane(ctx sdk.Context, txs []sdk.Tx, next blockbuster.ProcessLanesHandler) (sdk.Context, error) {
	remainingTxs, err := l.processLaneHandler(ctx, txs)
	if err != nil {
		return ctx, err
	}

	return next(ctx, remainingTxs)
}

// transactions that belong to this lane are not misplaced in the block proposal i.e.
// the proposal only contains contiguous transactions that belong to this lane - there
// can be no interleaving of transactions from other lanes.
func (l *LaneConstructor[C]) ProcessLaneBasic(ctx sdk.Context, txs []sdk.Tx) error {
	return l.processLaneBasicHandler(ctx, txs)
}

// VerifyTx does basic verification of the transaction using the ante handler.
func (l *LaneConstructor[C]) VerifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if l.Cfg.AnteHandler != nil {
		_, err := l.Cfg.AnteHandler(ctx, tx, false)
		return err
	}

	return nil
}
