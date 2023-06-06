package free

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
)

const (
	// LaneName defines the name of the free lane.
	LaneName = "free"
)

var _ blockbuster.Lane = (*FreeLane)(nil)

// FreeLane defines the lane that is responsible for processing free transactions.
type FreeLane struct {
	*base.DefaultLane
	Factory
}

// NewFreeLane returns a new free lane.
func NewFreeLane(cfg blockbuster.BaseLaneConfig, factory Factory) *FreeLane {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	return &FreeLane{
		DefaultLane: base.NewDefaultLane(cfg),
		Factory:     factory,
	}
}

// Match returns true if the transaction is a free transaction.
func (l *FreeLane) Match(tx sdk.Tx) bool {
	return l.IsFreeTx(tx)
}

// Name returns the name of the free lane.
func (l *FreeLane) Name() string {
	return LaneName
}
