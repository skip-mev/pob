package priority

import (
	"fmt"

	"cosmossdk.io/api/tendermint/abci"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PrepareProposal returns a list of transactions that are ready to be included at the top of block. The transactions
// are selected from the mempool and are ordered by their bid price. The transactions are also validated and
// the winning bidder has their entire bundle of transactions included in the block proposal.
func (mempool *DefaultLane) PrepareProposal(ctx sdk.Context, req abci.RequestPrepareProposal) ([][]byte, int64) {
	var (
		selectedTxs  [][]byte
		totalTxBytes int64
	)

	iterator := mempool.Select(ctx, nil)
	txsToRemove := map[sdk.Tx]struct{}{}

	// Select remaining transactions for the block proposal until we've reached
	// size capacity.
selectTxLoop:
	for ; iterator != nil; iterator = iterator.Next() {
		memTx := iterator.Tx()

		txBz, err := mempool.PrepareProposalVerifyTx(ctx, memTx)
		if err != nil {
			txsToRemove[memTx] = struct{}{}
			continue selectTxLoop
		}

		txSize := int64(len(txBz))
		if totalTxBytes += txSize; totalTxBytes <= req.MaxTxBytes {
			selectedTxs = append(selectedTxs, txBz)
		} else {
			// We've reached capacity per req.MaxTxBytes so we cannot select any
			// more transactions.
			break selectTxLoop
		}
	}

	// Remove all invalid transactions from the mempool.
	for tx := range txsToRemove {
		mempool.Remove(tx)
	}

	return selectedTxs, totalTxBytes
}

func (mempool *DefaultLane) ProcessProposal(ctx sdk.Context, req abci.RequestProcessProposal) error {
	for _, txBz := range req.Txs {
		_, err := mempool.ProcessProposalVerifyTx(ctx, txBz)
		if err != nil {
			return fmt.Errorf("invalid transaction in block proposal: %w", err)
		}
	}

	return nil
}

// PrepareProposalVerifyTx encodes a transaction and verifies it.
func (mempool *DefaultLane) PrepareProposalVerifyTx(ctx sdk.Context, tx sdk.Tx) ([]byte, error) {
	txBz, err := mempool.txEncoder(tx)
	if err != nil {
		return nil, err
	}

	return txBz, mempool.verifyTx(ctx, tx)
}

// ProcessProposalVerifyTx decodes a transaction and verifies it.
func (mempool *DefaultLane) ProcessProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error) {
	tx, err := mempool.txDecoder(txBz)
	if err != nil {
		return nil, err
	}

	return tx, mempool.verifyTx(ctx, tx)
}

// verifyTx verifies a transaction.
func (mempool *DefaultLane) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	// We verify transaction by running them through a predeteremined set of antehandlers
	if _, err := mempool.cfg.AnteHandler(ctx, tx, false); err != nil {
		return err
	}

	if _, err := mempool.cfg.PostHandler(ctx, tx, false, true); err != nil {
		return err
	}

	return nil
}
