package mempool

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PriorityTx defines a wrapper around an sd.Tx with a corresponding priority.
type PriorityTx struct {
	priority int64
	tx       sdk.Tx
}

func NewPriorityTx(tx sdk.Tx, priority int64) PriorityTx {
	return PriorityTx{
		priority: priority,
		tx:       tx,
	}
}

func (ptx PriorityTx) GetPriority() int64 { return ptx.priority }
func (ptx PriorityTx) GetTx() sdk.Tx      { return ptx.tx }
