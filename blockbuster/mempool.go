package blockbuster

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ Mempool = (*BBMempool)(nil)

type (
	// Mempool defines the Blockbuster mempool interface.
	Mempool interface {
		sdkmempool.Mempool

		// Registry returns the mempool's lane registry.
		Registry() []Lane

		// Contains returns true if the transaction is contained in the mempool.
		Contains(tx sdk.Tx) (bool, error)

		// GetTxDistribution returns the number of transactions in each lane.
		GetTxDistribution() map[string]int
	}

	// Mempool defines the Blockbuster mempool implement. It contains a registry
	// of lanes, which allows for customizable block proposal construction.
	BBMempool struct {
		registry []Lane
	}
)

// NewMempool returns a new Blockbuster mempool. The blockbuster mempool is
// comprised of a registry of lanes. Each lane is responsible for selecting
// transactions according to its own selection logic. The lanes are ordered
// according to their priority. The first lane in the registry has the highest
// priority. Proposals are verified according to the order of the lanes in the
// registry. Basic mempool API, such as insertion, removal, and contains, are
// delegated to the first lane that matches the transaction. Each transaction
// should only belong in one lane.
func NewMempool(lanes ...Lane) *BBMempool {
	mempool := &BBMempool{
		registry: lanes,
	}

	if err := mempool.ValidateBasic(); err != nil {
		panic(err)
	}

	return mempool
}

// CountTx returns the total number of transactions in the mempool. This will
// be the sum of the number of transactions in each lane.
func (m *BBMempool) CountTx() int {
	var total int
	for _, lane := range m.registry {
		total += lane.CountTx()
	}

	return total
}

// GetTxDistribution returns the number of transactions in each lane.
func (m *BBMempool) GetTxDistribution() map[string]int {
	counts := make(map[string]int, len(m.registry))

	for _, lane := range m.registry {
		counts[lane.Name()] = lane.CountTx()
	}

	return counts
}

// Insert inserts a transaction into the mempool. It inserts the transaction
// into the first lane that it matches.
func (m *BBMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			return lane.Insert(ctx, tx)
		}
	}

	return nil
}

// Insert returns a nil iterator.
//
// TODO:
// - Determine if it even makes sense to return an iterator. What does that even
// mean in the context where you have multiple lanes?
// - Perhaps consider implementing and returning a no-op iterator?
func (m *BBMempool) Select(_ context.Context, _ [][]byte) sdkmempool.Iterator {
	return nil
}

// Remove removes a transaction from the mempool. It removes the transaction
// from the first lane that it matches.
func (m *BBMempool) Remove(tx sdk.Tx) error {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			return lane.Remove(tx)
		}
	}

	return nil
}

// Contains returns true if the transaction is contained in the mempool.
func (m *BBMempool) Contains(tx sdk.Tx) (bool, error) {
	for _, lane := range m.registry {
		if lane.Match(tx) {
			return lane.Contains(tx)
		}
	}

	return false, nil
}

// Registry returns the mempool's lane registry.
func (m *BBMempool) Registry() []Lane {
	return m.registry
}

// ValidateBasic validates the mempools configuration.
func (m *BBMempool) ValidateBasic() error {
	sum := sdk.ZeroDec()
	seenZeroMaxBlockSpace := false

	for _, lane := range m.registry {
		maxBlockSpace := lane.GetMaxBlockSpace()
		if maxBlockSpace.IsZero() {
			seenZeroMaxBlockSpace = true
		}

		sum = sum.Add(lane.GetMaxBlockSpace())
	}

	switch {
	// Ensure that the sum of the lane max block space percentages is less than
	// or equal to 1.
	case sum.GT(sdk.OneDec()):
		return fmt.Errorf("sum of lane max block space percentages must be less than or equal to 1, got %s", sum)
	// Ensure that there is no unused block space.
	case sum.LT(sdk.OneDec()) && !seenZeroMaxBlockSpace:
		return fmt.Errorf("sum of total block space percentages will be less than 1")
	}

	return nil
}
