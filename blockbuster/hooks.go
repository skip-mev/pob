package blockbuster

import sdk "github.com/cosmos/cosmos-sdk/types"

type UnitMempoolHooks interface {
	// Called when a transaction is removed from this mempool.
	AfterRemoveHook(tx sdk.Tx)

	// Called when a transaction is added to this mempool.
	AfterInsertHook(tx sdk.Tx)

	// AfterPrepareUnitHook is called after a unit has finished the process of ordering its transactions for PrepareProposal.
	AfterPrepareUnitHook(ctx sdk.Context, txs []sdk.Tx)

	// AfterProcessUnitHook is called after a unit has finished verifying the ordering of transactions in ProcessProposal.
	AfterProcessUnitHook(ctx sdk.Context, txs []sdk.Tx)
}

var _ UnitMempoolHooks = MultiUnitMempoolHooks{}

// MultiUnitMempoolHooks is a collection of UnitMempoolHooks that are called sequentially.
type MultiUnitMempoolHooks []UnitMempoolHooks

// Create hooks for the unit mempool.
func NewUnitMempoolHooks(hooks ...UnitMempoolHooks) MultiUnitMempoolHooks {
	return hooks
}

// AfterRemoveHook implements UnitMempoolHooks.
func (h MultiUnitMempoolHooks) AfterRemoveHook(tx sdk.Tx) {
	for _, hook := range h {
		hook.AfterRemoveHook(tx)
	}
}

// AfterInsertHook implements UnitMempoolHooks.
func (h MultiUnitMempoolHooks) AfterInsertHook(tx sdk.Tx) {
	for _, hook := range h {
		hook.AfterInsertHook(tx)
	}
}

// AfterPrepareUnitHook implements UnitMempoolHooks.
func (h MultiUnitMempoolHooks) AfterPrepareUnitHook(ctx sdk.Context, txs []sdk.Tx) {
	for _, hook := range h {
		hook.AfterPrepareUnitHook(ctx, txs)
	}
}

// AfterProcessUnitHook implements UnitMempoolHooks.
func (h MultiUnitMempoolHooks) AfterProcessUnitHook(ctx sdk.Context, txs []sdk.Tx) {
	for _, hook := range h {
		hook.AfterProcessUnitHook(ctx, txs)
	}
}
