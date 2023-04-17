package blockbuster

import (
	"cosmossdk.io/api/tendermint/abci"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// Lane defines the interface that a Lane must implement. Unit mempools are
// mempools that are used to build blocks. All transactions in a unit mempool share the
// same ordering, validation and prioritization rules.
type (
	LaneInterface interface {
		// Define the mempool interface that is required
		sdkmempool.Mempool

		// Define the block building functionailty that is required
		BlockBuilder

		// Contains returns true if the transaction is in the mempool. Otherwise, false is returned.
		Contains(tx sdk.Tx) (bool, error)

		// Match returns true if the transaction should be inserted into the mempool. Otherwise, false is returned.
		Match(tx sdk.Tx) bool
	}

	// BlockBuilder defines the interface that a unit mempool must implement
	// in order to be included in the block building process.
	BlockBuilder interface {
		// PrepareProposal returns a list of transactions that are ready to be included
		// in a block proposal.
		PrepareProposal(ctx sdk.Context, req abci.RequestPrepareProposal) ([][]byte, int64)

		// ProcessProposal processes a block proposal and returns an error if the
		// proposal is invalid.
		ProcessProposal(ctx sdk.Context, req abci.RequestProcessProposal) error

		// PrepareProposalVerifyTx encodes a transaction and verifies it.
		PrepareProposalVerifyTx(ctx sdk.Context, tx sdk.Tx) ([]byte, error)

		// ProcessProposalVerifyTx decodes a transaction and verifies it.
		ProcessProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error)
	}

	// Lane defines a unit mempool that is used to build blocks. All transactions in a
	// unit mempool share the same ordering, validation and prioritization rules.
	LaneConfig[c comparable] struct {
		// Can maybe store the
		// globalIndex stores the global index of all transactions in this mempool.
		GlobalIndex *PriorityNonceMempool[c]

		// hooks
		Hooks MultiUnitMempoolHooks

		// anteHandler defines the ante handler used to validate transactions.
		AnteHandler sdk.AnteHandler

		// postHandler defines the post handler used to validate transactions.
		PostHandler sdk.PostHandler

		// config
		// logger
	}
)

func NewLaneConfig[c comparable](mempool *PriorityNonceMempool[c], hooks MultiUnitMempoolHooks) *LaneConfig[c] {
	return &LaneConfig[c]{
		GlobalIndex: mempool,
		Hooks:       hooks,
	}
}
