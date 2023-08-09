package auction

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
)

const (
	// LaneName defines the name of the top-of-block auction lane.
	LaneName = "top-of-block"
)

var (
	_ blockbuster.Lane = (*TOBLane)(nil)
	_ Factory          = (*TOBLane)(nil)
)

// TOBLane defines a top-of-block auction lane. The top of block auction lane
// hosts transactions that want to bid for inclusion at the top of the next block.
// The top of block auction lane stores bid transactions that are sorted by
// their bid price. The highest valid bid transaction is selected for inclusion in the
// next block. The bundled transactions of the selected bid transaction are also
// included in the next block.
type TOBLane struct {
	// Mempool defines the mempool for the lane.
	Mempool

	// LaneConfig defines the base lane configuration.
	*base.DefaultLane

	// Factory defines the API/functionality which is responsible for determining
	// if a transaction is a bid transaction and how to extract relevant
	// information from the transaction (bid, timeout, bidder, etc.).
	Factory

	// txPriority maintains how the mempool determines relative ordering
	// of transactions
	txPriority blockbuster.TxPriority[string]
}

// NewTOBLane returns a new TOB lane.
func NewTOBLane(
	cfg blockbuster.BaseLaneConfig,
	maxTx int,
	factory Factory,
) *TOBLane {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	return &TOBLane{
		Mempool:     NewMempool(cfg.TxEncoder, maxTx, factory),
		DefaultLane: base.NewDefaultLane(cfg, "").WithName(LaneName),
		txPriority:  TxPriority(factory),
		Factory:     factory,
	}
}

// Compare determines the relative priority of two transactions belonging in the same lane.
// In the default case, priority is determined by the priority of the context passed down
// to the API.
func (l *TOBLane) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) int {
	firstPriority := l.txPriority.GetTxPriority(ctx, this)
	secondPriority := l.txPriority.GetTxPriority(ctx, other)
	return l.txPriority.Compare(firstPriority, secondPriority)
}

// Match returns true if the transaction is a bid transaction. This is determined
// by the AuctionFactory.
func (l *TOBLane) Match(ctx sdk.Context, tx sdk.Tx) bool {
	if l.MatchIgnoreList(ctx, tx) {
		return false
	}

	bidInfo, err := l.GetAuctionBidInfo(tx)
	return bidInfo != nil && err == nil
}
