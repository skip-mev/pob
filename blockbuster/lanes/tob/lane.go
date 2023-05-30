package tob

import (
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster/lanes"
)

const (
	// LaneNameTOB defines the name of the top-of-block auction lane.
	LaneNameTOB = "tob"
)

var _ lanes.Lane = (*TOBLane)(nil)

// TOBLane defines a top-of-block auction lane, which extends a base lane.
type TOBLane struct {
	Mempool
	*lanes.LaneConfig
	af AuctionFactory
}

func NewTOBLane(
	logger log.Logger,
	txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder,
	maxTx int,
	anteHandler sdk.AnteHandler,
	af AuctionFactory,
) *TOBLane {
	logger = logger.With("lane", LaneNameTOB)

	return &TOBLane{
		Mempool:    NewAuctionMempool(txDecoder, txEncoder, maxTx, af),
		LaneConfig: lanes.NewLaneConfig(logger, txEncoder, txDecoder, anteHandler, LaneNameTOB),
		af:         af,
	}
}

func (l *TOBLane) Name() string {
	return LaneNameTOB
}

func (l *TOBLane) Match(tx sdk.Tx) bool {
	bidInfo, err := l.af.GetAuctionBidInfo(tx)
	return bidInfo != nil && err == nil
}
