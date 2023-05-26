package lane

import (
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
)

const (
	// LaneNameFreeTx defines the name of the free transaction lane.
	LaneNameFreeTx = "free-tx"
)

var _ Lane = (*FreeTxLane)(nil)

type FreeTxLane struct {
	*BaseLane
}

func NewFreeTxLane(
	logger log.Logger,
	txDecoder sdk.TxDecoder,
	txEncoder sdk.TxEncoder,
	maxTx int,
	af mempool.AuctionFactory,
	anteHandler sdk.AnteHandler,
) *FreeTxLane {
	logger = logger.With("lane", LaneNameTOB)
	baseLane := NewBaseLane(logger, txDecoder, txEncoder, maxTx, af, anteHandler)

	return &FreeTxLane{
		BaseLane: baseLane,
	}
}

func (l *FreeTxLane) Name() string {
	return LaneNameFreeTx
}

func (l *FreeTxLane) Match(tx sdk.Tx) bool {
	panic("not implemented")
}

func (l *FreeTxLane) VerifyTx(ctx sdk.Context, tx sdk.Tx) error {
	panic("not implemented")
}

func (l *FreeTxLane) PrepareLane(ctx sdk.Context, maxTxBytes int64, selectedTxs map[string][]byte) ([][]byte, error) {
	panic("not implemented")
}

func (l *FreeTxLane) ProcessLane(ctx sdk.Context, proposalTxs [][]byte) error {
	panic("not implemented")
}
