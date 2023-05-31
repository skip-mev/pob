package base

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

// PrepareLane will prepare a partial proposal for the base lane. It will return
// an error if there are any unexpected errors.
func (l *BaseLane) PrepareLane(ctx sdk.Context, proposal blockbuster.Proposal, next blockbuster.PrepareLanesHandler) blockbuster.Proposal {
	// Define all of the info we need to select transactions for the partial proposal.
	txs := make([][]byte, 0)
	txsToRemove := make(map[sdk.Tx]struct{}, 0)

	// Calculate the max tx bytes for the lane and track the total size of the
	// transactions we have selected so far.
	maxTxBytes := blockbuster.GetMaxTxBytesForLane(proposal, l.MaxBlockSpace)
	totalSize := int64(0)

	// Select all transactions in the mempool that are valid and not already in the
	// partial proposal.
	for iterator := l.Mempool.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
		tx := iterator.Tx()

		txBytes, err := l.TxEncoder(tx)
		if err != nil {
			txsToRemove[tx] = struct{}{}
			continue
		}

		// if the transaction is already in the (partial) block proposal, we skip it.
		hash, err := blockbuster.GetTxHashStr(l.TxEncoder, tx)
		if err != nil {
			txsToRemove[tx] = struct{}{}
			continue
		}
		if _, ok := proposal.SelectedTxs[hash]; ok {
			continue
		}

		// if the transaction is too big, we break and do not attempt to include more txs.
		size := int64(len(txBytes))
		if updatedSize := totalSize + size; updatedSize > maxTxBytes {
			break
		}

		// Verify the transaction.
		if err := l.VerifyTx(ctx, tx); err != nil {
			txsToRemove[tx] = struct{}{}
			continue
		}

		totalSize += size
		txs = append(txs, txBytes)
	}

	// Remove all transactions that were invalid during the creation of the partial proposal.
	if err := blockbuster.RemoveTxsFromLane(txsToRemove, l.Mempool); err != nil {
		l.Logger.Error("failed to remove txs from mempool", "lane", l.Name(), "err", err)
		return proposal
	}

	proposal = blockbuster.UpdateProposal(proposal, txs, totalSize)

	return next(ctx, proposal)
}

// ProcessLane will process the base lane. It will verify all transactions in the
// lane and return an error if any of the transactions are invalid. If there are
// transactions from other lanes in the lane, it will return an error.
func (l *BaseLane) ProcessLane(ctx sdk.Context, proposalTxs [][]byte, next blockbuster.ProcessLanesHandler) (sdk.Context, error) {
	seenOtherLaneTxs := false
	endIndex := 0

	// Verify all transactions in the lane.
	for _, tx := range proposalTxs {
		tx, err := l.TxDecoder(tx)
		if err != nil {
			return ctx, fmt.Errorf("failed to decode tx: %w", err)
		}

		// If this lane intersects with another lane, we return an error.
		if l.Match(tx) {
			if seenOtherLaneTxs {
				return ctx, fmt.Errorf("lane %s contains txs from other lanes", l.Name())
			}

			if err := l.VerifyTx(ctx, tx); err != nil {
				return ctx, fmt.Errorf("failed to verify tx: %w", err)
			}

			endIndex++
		} else {
			seenOtherLaneTxs = true
		}
	}

	return next(ctx, proposalTxs[endIndex:])
}

func (l *BaseLane) VerifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if l.AnteHandler != nil {
		_, err := l.AnteHandler(ctx, tx, false)
		return err
	}

	return nil
}
