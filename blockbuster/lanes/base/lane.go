package base

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

var _ blockbuster.Lane = (*DefaultLane)(nil)

// DefaultLane defines a default lane implementation. It contains a priority-nonce
// index along with core lane functionality.
type DefaultLane struct {
	// Mempool defines the mempool for the lane.
	Mempool

	// LaneConfig defines the base lane configuration.
	cfg blockbuster.BaseLaneConfig
}

func NewDefaultLane(logger log.Logger, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, anteHandler sdk.AnteHandler, maxBlockSpace sdk.Dec) *DefaultLane {
	return &DefaultLane{
		Mempool: NewDefaultMempool(txEncoder),
		cfg:     blockbuster.NewBaseLaneConfig(logger, txEncoder, txDecoder, anteHandler, maxBlockSpace),
	}
}

// Match returns true if the transaction belongs to this lane. Since
// this is the default lane, it always returns true. This means that
// any transaction can be included in this lane.
func (l *DefaultLane) Match(sdk.Tx) bool {
	return true
}

// Name returns the name of the lane.
func (l *DefaultLane) Name() string {
	return LaneName
}

// ValidateLaneBasic does basic validation on the block proposal to ensure that
// transactions that belong to this lane are not misplaced in the block proposal.
func (l *DefaultLane) ProcessLaneBasic(txs [][]byte) error {
	seenOtherLaneTx := false
	lastSeenIndex := 0

	for _, txBz := range txs {
		tx, err := l.cfg.TxDecoder(txBz)
		if err != nil {
			return fmt.Errorf("failed to decode tx in lane %s: %w", l.Name(), err)
		}

		if l.Match(tx) {
			if seenOtherLaneTx {
				return fmt.Errorf("the %s lane contains a transaction that belongs to another lane", l.Name())
			}

			lastSeenIndex++
			continue
		}

		seenOtherLaneTx = true
	}

	return nil
}
