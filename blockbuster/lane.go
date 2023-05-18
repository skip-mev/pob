package blockbuster

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// Lane defines an interface used for block construction
type Lane interface {
	sdkmempool.Mempool

	// Name returns the name of the lane.
	Name() string

	// Match determines if a transaction belongs to this lane.
	Match(tx sdk.Tx) bool

	// VerifyTx verifies the transaction belonging to this lane.
	VerifyTx(ctx sdk.Context, tx sdk.Tx) error

	// Contains returns true if the mempool contains the given transaction.
	Contains(tx sdk.Tx) (bool, error)

	// PrepareLane which builds a portion of the block. Inputs a cache of transactions
	// that have already been included by a previous lane.
	PrepareLane(ctx sdk.Context, cache map[string]struct{}) [][]byte

	// ProcessLane which verifies the lane's portion of a proposed block.
	ProcessLane(ctx sdk.Context, txs [][]byte) error
}
