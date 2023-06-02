package auction

import (
	"bytes"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
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
	cfg blockbuster.BaseLaneConfig

	// Factory defines the API/functionality which is responsible for determining
	// if a transaction is a bid transaction and how to extract relevant
	// information from the transaction (bid, timeout, bidder, etc.).
	Factory
}

// NewTOBLane returns a new TOB lane.
func NewTOBLane(
	logger log.Logger,
	txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder,
	maxTx int,
	anteHandler sdk.AnteHandler,
	af Factory,
	maxBlockSpace sdk.Dec,
) *TOBLane {
	return &TOBLane{
		Mempool: NewMempool(txEncoder, maxTx, af),
		cfg:     blockbuster.NewBaseLaneConfig(logger, txEncoder, txDecoder, anteHandler, maxBlockSpace),
		Factory: af,
	}
}

// Match returns true if the transaction is a bid transaction. This is determined
// by the AuctionFactory.
func (l *TOBLane) Match(tx sdk.Tx) bool {
	bidInfo, err := l.GetAuctionBidInfo(tx)
	return bidInfo != nil && err == nil
}

// Name returns the name of the lane.
func (l *TOBLane) Name() string {
	return LaneName
}

// ProcessLaneBasic does basic validation on the block proposal to ensure that
// transactions that belong to this lane are not misplaced in the block proposal.
// In this case, we ensure that the bid transaction is the first transaction in the
// block proposal, if present, and that all of the bundled transactions are included
// after the bid transaction. We enforce that at most one auction bid transaction
// is included in the block proposal.
func (l *TOBLane) ProcessLaneBasic(txs [][]byte) error {
	tx, err := l.cfg.TxDecoder(txs[0])
	if err != nil {
		return fmt.Errorf("failed to decode tx in lane %s: %w", l.Name(), err)
	}

	// If there is a bid transaction, it must be the first transaction in the block proposal.
	if !l.Match(tx) {
		for _, txBz := range txs[1:] {
			tx, err := l.cfg.TxDecoder(txBz)
			if err != nil {
				return fmt.Errorf("failed to decode tx in lane %s: %w", l.Name(), err)
			}

			if l.Match(tx) {
				return fmt.Errorf("multiple bid transactions in lane %s", l.Name())
			}
		}

		return nil
	}

	bidInfo, err := l.GetAuctionBidInfo(tx)
	if err != nil {
		return fmt.Errorf("failed to get bid info for lane %s: %w", l.Name(), err)
	}

	if len(txs) < len(bidInfo.Transactions)+1 {
		return fmt.Errorf("invalid number of transactions in lane %s; expected at least %d, got %d", l.Name(), len(bidInfo.Transactions)+1, len(txs))
	}

	// Ensure that the order of transactions in the bundle is preserved.
	for i, bundleTxBz := range txs[1 : len(bidInfo.Transactions)+1] {
		tx, err := l.WrapBundleTransaction(bundleTxBz)
		if err != nil {
			return fmt.Errorf("failed to decode bundled tx in lane %s: %w", l.Name(), err)
		}

		if l.Match(tx) {
			return fmt.Errorf("multiple bid transactions in lane %s", l.Name())
		}

		txBz, err := l.cfg.TxEncoder(tx)
		if err != nil {
			return fmt.Errorf("failed to encode bundled tx in lane %s: %w", l.Name(), err)
		}

		if !bytes.Equal(txBz, bidInfo.Transactions[i]) {
			return fmt.Errorf("invalid order of transactions in lane %s", l.Name())
		}
	}

	// Ensure that there are no more bid transactions in the block proposal.
	for _, txBz := range txs[len(bidInfo.Transactions)+1:] {
		tx, err := l.cfg.TxDecoder(txBz)
		if err != nil {
			return fmt.Errorf("failed to decode tx in lane %s: %w", l.Name(), err)
		}

		if l.Match(tx) {
			return fmt.Errorf("multiple bid transactions in lane %s", l.Name())
		}
	}

	return nil
}
