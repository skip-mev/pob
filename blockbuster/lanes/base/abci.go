package base

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

// PrepareLane will prepare a partial proposal for the base lane.
func (l *DefaultLane) PrepareLane(
	ctx sdk.Context,
	proposal blockbuster.Proposal,
	maxTxBytes int64,
	next blockbuster.PrepareLanesHandler,
) blockbuster.Proposal {
	// Define all of the info we need to select transactions for the partial proposal.
	txs := make([][]byte, 0)
	txsToRemove := make(map[sdk.Tx]struct{}, 0)
	totalSize := int64(0)

	// Select all transactions in the mempool that are valid and not already in the
	// partial proposal.
	for iterator := l.Mempool.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
		tx := iterator.Tx()

		txBytes, hash, err := utils.GetTxHashStr(l.cfg.TxEncoder, tx)
		if err != nil {
			txsToRemove[tx] = struct{}{}
			continue
		}

		// if the transaction is already in the (partial) block proposal, we skip it.
		if _, ok := proposal.Cache[hash]; ok {
			continue
		}

		// if the transaction is too big, we break and do not attempt to include more txs.
		txSize := int64(len(txBytes))
		if updatedSize := totalSize + txSize; updatedSize > maxTxBytes {
			break
		}

		// Verify the transaction.
		if err := l.VerifyTx(ctx, tx); err != nil {
			txsToRemove[tx] = struct{}{}
			continue
		}

		totalSize += txSize
		txs = append(txs, txBytes)
	}

	// Remove all transactions that were invalid during the creation of the partial proposal.
	if err := utils.RemoveTxsFromLane(txsToRemove, l.Mempool); err != nil {
		l.cfg.Logger.Error("failed to remove txs from mempool", "lane", l.Name(), "err", err)
		return proposal
	}

	proposal = blockbuster.UpdateProposal(proposal, txs, totalSize)

	return next(ctx, proposal)
}

// ProcessLane verifies the default lane's portion of a block proposal.
func (l *DefaultLane) ProcessLane(ctx sdk.Context, proposalTxs [][]byte, next blockbuster.ProcessLanesHandler) (sdk.Context, error) {
	for index, tx := range proposalTxs {
		tx, err := l.cfg.TxDecoder(tx)
		if err != nil {
			return ctx, fmt.Errorf("failed to decode tx: %w", err)
		}

		if l.Match(tx) {
			if err := l.VerifyTx(ctx, tx); err != nil {
				return ctx, fmt.Errorf("failed to verify tx: %w", err)
			}
		} else {
			return next(ctx, proposalTxs[index:])
		}
	}

	// This means we have processed all transactions in the proposal.
	return ctx, nil
}

// ProcessLaneBasic does basic validation on the block proposal to ensure that
// transactions that belong to this lane are not misplaced in the block proposal.
func (l *DefaultLane) ProcessLaneBasic(txs [][]byte) error {
	seenOtherLaneTx := false
	lastSeenIndex := 0

	for _, txBz := range txs {
		tx, err := l.cfg.TxDecoder(txBz)
		if err != nil {
			return fmt.Errorf("failed to decode tx in lane %s: %w", l.Name(), err)
		}

		if l.Match(tx) {
			if seenOtherLaneTx {
				return fmt.Errorf("the %s lane contains a transaction that belongs to another lane", l.Name())
			}

			lastSeenIndex++
			continue
		}

		seenOtherLaneTx = true
	}

	return nil
}

// VerifyTx does basic verification of the transaction using the ante handler.
func (l *DefaultLane) VerifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if l.cfg.AnteHandler != nil {
		_, err := l.cfg.AnteHandler(ctx, tx, false)
		return err
	}

	return nil
}
