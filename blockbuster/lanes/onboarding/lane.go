package onboarding

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
)

const (
	// LaneName defines the name of the onboarding lane.
	LaneName = "onboarding"
)

type Lane struct {
	*base.DefaultLane
	Factory
}

// NewOnboardingLane returns a new onboarding lane.
func NewOnboardingLane(cfg blockbuster.BaseLaneConfig, factory Factory) *Lane {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	return &Lane{
		DefaultLane: base.NewDefaultLane(cfg).WithName(LaneName),
		Factory:     factory,
	}
}

// Match returns true if the transaction is considered to be an onboarding transaction.
func (l *Lane) Match(ctx sdk.Context, tx sdk.Tx) bool {
	if l.MatchIgnoreList(ctx, tx) {
		return false
	}

	return l.IsOnboardingTx(ctx, tx)
}
