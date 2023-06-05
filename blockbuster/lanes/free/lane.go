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

// DefaultLane defines a default lane implementation. It contains a priority-nonce
// index along with core lane functionality.
type FreeLane struct {
	*base.DefaultLane
	Factory
}

func NewFreeLane(cfg blockbuster.BaseLaneConfig, factory Factory) *FreeLane {
	return &FreeLane{
		DefaultLane: base.NewDefaultLane(cfg),
		Factory:     factory,
	}
}

// Match returns true if the transaction belongs to this lane. Since
// this is the default lane, it always returns true. This means that
// any transaction can be included in this lane.
func (l *FreeLane) Match(tx sdk.Tx) bool {
	return l.IsFreeTx(tx)
}

// Name returns the name of the lane.
func (l *FreeLane) Name() string {
	return LaneName
}
