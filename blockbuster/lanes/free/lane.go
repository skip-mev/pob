package free

import (
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/constructor"
)

const (
	// LaneName defines the name of the free lane.
	LaneName = "free"
)

var _ blockbuster.Lane = (*FreeLane)(nil)

// FreeLane defines the lane that is responsible for processing free transactions.
type FreeLane struct {
	*constructor.LaneConstructor[string]
}

// NewFreeLane returns a new free lane.
func NewFreeLane(cfg blockbuster.BaseLaneConfig) *FreeLane {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	factory := NewDefaultFreeFactory(cfg.TxDecoder)

	lane := constructor.NewLaneConstructor[string](
		cfg,
		LaneName,
		constructor.NewConstructorMempool[string](
			constructor.DefaultTxPriority(),
			cfg.TxEncoder,
			cfg.MaxTxs,
		),
		factory.MatchHandler(),
	)

	if err := lane.ValidateBasic(); err != nil {
		panic(err)
	}

	return &FreeLane{
		LaneConstructor: lane,
	}
}
