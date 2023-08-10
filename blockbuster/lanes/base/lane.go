package base

import (
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/constructor"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

var _ blockbuster.Lane = (*DefaultLane)(nil)

// DefaultLane defines a default lane implementation. The default lane orders
// transactions by the sdk.Context priority. The default lane will accept any
// transaction that is not a part of the lane's IgnoreList. By default, the IgnoreList
// is empty and the default lane will accept any transaction. The default lane on its
// own implements the same functionality as the pre v0.47.0 tendermint mempool and proposal
// handlers.
type DefaultLane struct {
	*constructor.LaneConstructor[string]
}

// NewDefaultLane returns a new default lane.
func NewDefaultLane(cfg blockbuster.BaseLaneConfig) *DefaultLane {
	lane := constructor.NewLaneConstructor[string](
		cfg,
		LaneName,
		constructor.NewConstructorMempool[string](
			constructor.DefaultTxPriority(),
			cfg.TxEncoder,
			cfg.MaxTxs,
		),
		constructor.DefaultMatchHandler(),
	)

	return &DefaultLane{
		LaneConstructor: lane,
	}
}
