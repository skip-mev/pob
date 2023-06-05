package free

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
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
}

func NewFreeLane(cfg blockbuster.BaseLaneConfig) *FreeLane {
	return &FreeLane{
		DefaultLane: base.NewDefaultLane(cfg),
	}
}

// Match returns true if the transaction belongs to this lane. Since
// this is the default lane, it always returns true. This means that
// any transaction can be included in this lane.
func (l *FreeLane) Match(tx sdk.Tx) bool {
	for _, msg := range tx.GetMsgs() {
		switch msg.(type) {
		case *types.MsgDelegate:
			return true
		case *types.MsgBeginRedelegate:
			return true
		}
	}

	return false
}

// Name returns the name of the lane.
func (l *FreeLane) Name() string {
	return LaneName
}
