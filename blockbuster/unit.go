package mempool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// Unit defines the interface that a unit mempool must implement. Unit mempools are
// mempools that are used to build blocks. All transactions in a unit mempool share the
// same ordering, validation and prioritization rules.
type Unit interface {
	// Define the mempool interface that is required
	sdkmempool.Mempool

	// Contains returns true if the transaction is in the mempool. Otherwise, false is returned.
	Contains(tx sdk.Tx) (bool, error)

	// Define the block building functionailty that is required
	BlockBuilder
}

// BlockBuilder defines the interface that a unit mempool must implement
// in order to be included in the block building process.
type BlockBuilder interface {
	// PrepareProposal returns a list of transactions that are ready to be included
	// in a block proposal.
	PrepareProposal(ctx sdk.Context) []sdk.Tx

	// ProcessProposal processes a block proposal and returns an error if the
	// proposal is invalid.
	ProcessProposal(ctx sdk.Context, txs []sdk.Tx) error

	// PrepareProposalVerifyTx encodes a transaction and verifies it.
	PrepareProposalVerifyTx(ctx sdk.Context, tx sdk.Tx) ([]byte, error)

	// ProcessProposalVerifyTx decodes a transaction and verifies it.
	ProcessProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error)
}
