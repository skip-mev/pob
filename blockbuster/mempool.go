package blockbuster

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ sdkmempool.Mempool = (*Mempool)(nil)

// Mempool defines the Blockbuster mempool implement. It contains a registry
// of lanes, which allows for customizable block proposal construction.
type Mempool struct {
	registry []Lane
}

func (m *Mempool) CountTx() int {
	var total int
	for _, lane := range m.registry {
		// TODO: If a global lane exists, we assume that lane has all transactions
		// and we return the total.
		//
		// if lane.Name() == LaneNameGlobal {
		// 	return lane.CountTx()
		// }

		total += lane.CountTx()
	}

	return total
}

func (m *Mempool) Insert(ctx context.Context, tx sdk.Tx) error {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			if err := lane.Insert(ctx, tx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Mempool) Select(context.Context, [][]byte) sdkmempool.Iterator {
	panic("not implemented")
}

func (m *Mempool) Remove(sdk.Tx) error {
	panic("not implemented")
}
