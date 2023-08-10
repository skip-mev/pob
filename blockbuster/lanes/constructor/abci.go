package constructor

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

func (l *LaneConstructor[C]) DefaultPrepareLaneHandler() blockbuster.PrepareLaneHandler {
	return func(ctx sdk.Context, proposal blockbuster.BlockProposal, maxTxBytes int64) ([][]byte, []sdk.Tx, error) {
		var (
			totalSize   int64
			txs         [][]byte
			txsToRemove []sdk.Tx
		)

		// Select all transactions in the mempool that are valid and not already in the
		// partial proposal.
		for iterator := l.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()

			txBytes, hash, err := utils.GetTxHashStr(l.Cfg.TxEncoder, tx)
			if err != nil {
				l.Logger().Info("failed to get hash of tx", "err", err)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// if the transaction is already in the (partial) block proposal, we skip it.
			if proposal.Contains(txBytes) {
				l.Logger().Info(
					"failed to select tx for lane; tx is already in proposal",
					"tx_hash", hash,
					"lane", l.Name(),
				)

				continue
			}

			// If the transaction is too large, we break and do not attempt to include more txs.
			txSize := int64(len(txBytes))
			if updatedSize := totalSize + txSize; updatedSize > maxTxBytes {
				l.Logger().Info("maximum tx bytes reached", "lane", l.Name())
				break
			}

			// Verify the transaction.
			if err := l.VerifyTx(ctx, tx); err != nil {
				l.Logger().Info(
					"failed to verify tx",
					"tx_hash", hash,
					"err", err,
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			totalSize += txSize
			txs = append(txs, txBytes)
		}

		return txs, txsToRemove, nil
	}
}

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
	for index, tx := range txs {
		if l.Match(ctx, tx) {
			if err := l.VerifyTx(ctx, tx); err != nil {
				return ctx, fmt.Errorf("failed to verify tx: %w", err)
			}
		} else {
			return next(ctx, txs[index:])
		}
	}

	// This means we have processed all transactions in the proposal.
	return ctx, nil
}

// transactions that belong to this lane are not misplaced in the block proposal i.e.
// the proposal only contains contiguous transactions that belong to this lane - there
// can be no interleaving of transactions from other lanes.
func (l *LaneConstructor[C]) ProcessLaneBasic(ctx sdk.Context, txs []sdk.Tx) error {
	seenOtherLaneTx := false

	for index, tx := range txs {
		if l.Match(ctx, tx) {
			if seenOtherLaneTx {
				return fmt.Errorf("the %s lane contains a transaction that belongs to another lane", l.Name())
			}

			// If the transactions do not respect the priority defined by the mempool, we consider the proposal
			// to be invalid
			if index > 0 && l.Compare(ctx, txs[index-1], tx) == -1 {
				return fmt.Errorf("transaction at index %d has a higher priority than %d", index, index-1)
			}
		} else {
			seenOtherLaneTx = true
		}
	}

	return nil
}

// VerifyTx does basic verification of the transaction using the ante handler.
func (l *LaneConstructor[C]) VerifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if l.Cfg.AnteHandler != nil {
		_, err := l.Cfg.AnteHandler(ctx, tx, false)
		return err
	}

	return nil
}
