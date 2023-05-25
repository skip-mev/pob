package blockbuster

import (
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/mempool"
)

const (
	// LaneNameFreeTx defines the name of the free transaction lane.
	LaneNameFreeTx = "free-tx"
)

var _ Lane = (*FreeTxLane)(nil)

type FreeTxLane struct {
	logger      log.Logger
	index       sdkmempool.Mempool
	af          mempool.AuctionFactory
	txEncoder   sdk.TxEncoder
	txDecoder   sdk.TxDecoder
	anteHandler sdk.AnteHandler

	// txIndex is a map of all transactions in the mempool. It is used
	// to quickly check if a transaction is already in the mempool.
	txIndex map[string]struct{}
}

func NewFreeTxLane(logger log.Logger, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int, af mempool.AuctionFactory, anteHandler sdk.AnteHandler) *FreeTxLane {
}
