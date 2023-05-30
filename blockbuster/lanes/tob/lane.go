package tob

import (
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneNameTOB defines the name of the top-of-block auction lane.
	LaneNameTOB = "tob"
)

var _ blockbuster.Lane = (*TOBLane)(nil)

// TOBLane defines a top-of-block auction lane. The top of block auction lane
// hosts transactions that want to bid for inclusion at the top of the next block.
// The top of block auction lane stores bid transactions that are sorted by
// their bid price. The highest valid bid transaction is selected for inclusion in the
// next block. The bundled transactions of the selected bid transaction are also
// included in the next block.
type TOBLane struct {
	// Mempool defines the mempool for the lane.
	Mempool

	// LaneConfig defines the configuration for the lane.
	*blockbuster.LaneConfig

	// AuctionFactory defines the API/functionality which is responsible for determining
	// if a transaction is a bid transaction and how to extract relevant
	// information from the transaction (bid, timeout, bidder, etc.).
	AuctionFactory
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
		Mempool:        NewAuctionMempool(txDecoder, txEncoder, maxTx, af),
		LaneConfig:     blockbuster.NewLaneConfig(logger, txEncoder, txDecoder, anteHandler, LaneNameTOB),
		AuctionFactory: af,
	}
}

func (l *TOBLane) Match(tx sdk.Tx) bool {
	bidInfo, err := l.GetAuctionBidInfo(tx)
	return bidInfo != nil && err == nil
}
