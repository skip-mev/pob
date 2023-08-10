package constructor

import "github.com/skip-mev/pob/blockbuster"

const (
	// LaneName defines the name of the constructor lane.
	LaneName = "constructor"
)

type LaneConstructor[C comparable] struct {
	txPriority blockbuster.TxPriority[C]
	laneName string
	Cfg blockbuster.BaseLaneConfig
