package tob

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Prepare
func (mempool *AuctionMempool) PrepareProposal(ctx sdk.Context) []sdk.Tx {
	return []sdk.Tx{}
}

func (mempool *AuctionMempool) ProcessProposal(ctx sdk.Context, txs []sdk.Tx) error {
	return nil
}

// PrepareProposalVerifyTx encodes a transaction and verifies it.
func (mempool *AuctionMempool) PrepareProposalVerifyTx(ctx sdk.Context, tx sdk.Tx) ([]byte, error) {
	txBz, err := mempool.txEncoder(tx)
	if err != nil {
		return nil, err
	}

	return txBz, mempool.verifyTx(ctx, tx)
}

// ProcessProposalVerifyTx decodes a transaction and verifies it.
func (mempool *AuctionMempool) ProcessProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error) {
	tx, err := mempool.txDecoder(txBz)
	if err != nil {
		return nil, err
	}

	return tx, mempool.verifyTx(ctx, tx)
}

// verifyTx verifies a transaction.
func (mempool *AuctionMempool) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	// We verify transaction by running them through a predeteremined set of antehandlers
	if _, err := mempool.anteHandler(ctx, tx, false); err != nil {
		return err
	}

	if _, err := mempool.postHandler(ctx, tx, false, true); err != nil {
		return err
	}

	return nil
}
