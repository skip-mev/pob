package base

import (
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the base lane.
	LaneName = "base"
)

var _ blockbuster.Lane = (*BaseLane)(nil)

// BaseLane defines a base lane implementation. It contains a priority-nonce
// index along with core lane functionality.
type BaseLane struct {
	// Mempool defines the mempool for the lane.
	Mempool

	// LaneConfig defines the base lane configuration.
	*blockbuster.LaneConfig
}

func NewBaseLane(logger log.Logger, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int, anteHandler sdk.AnteHandler, maxBlockSpace sdk.Dec) *BaseLane {
	return &BaseLane{
		Mempool:    NewBaseMempool(txDecoder, txEncoder, maxTx),
		LaneConfig: blockbuster.NewLaneConfig(logger, txEncoder, txDecoder, anteHandler, LaneName, maxBlockSpace),
	}
}

func (l *BaseLane) Match(sdk.Tx) bool {
	return true
}
