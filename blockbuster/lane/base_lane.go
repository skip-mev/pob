package lane

import (
	"context"
	"errors"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/mempool"
)

const (
	// LaneNameBase defines the name of the base lane, which other lanes can extend.
	LaneNameBase = "base"
)

var _ Lane = (*BaseLane)(nil)

// BaseLane defines a base lane implementation. It contains a priority-nonce
// index along with core lane functionality. The base lane is meant to be extended
// by other lanes as some methods are no-ops.
type BaseLane struct {
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

func NewBaseLane(logger log.Logger, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int, af mempool.AuctionFactory, anteHandler sdk.AnteHandler) *BaseLane {
	return &BaseLane{
		logger: logger,
		index: mempool.NewPriorityMempool(
			mempool.PriorityNonceMempoolConfig[int64]{
				TxPriority: mempool.NewDefaultTxPriority(),
				MaxTx:      maxTx,
			},
		),
		af:          af,
		txEncoder:   txEncoder,
		txDecoder:   txDecoder,
		anteHandler: anteHandler,
		txIndex:     make(map[string]struct{}),
	}
}

func (l *BaseLane) Name() string {
	return LaneNameBase
}

func (l *BaseLane) Match(sdk.Tx) bool {
	return false
}

func (l *BaseLane) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := getTxHashStr(l.txEncoder, tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := l.txIndex[txHashStr]
	return ok, nil
}

func (l *BaseLane) VerifyTx(sdk.Context, sdk.Tx) error {
	return nil
}

func (l *BaseLane) PrepareLane(sdk.Context, int64, map[string][]byte) ([][]byte, error) {
	return nil, nil
}

func (l *BaseLane) ProcessLane(sdk.Context, [][]byte) error {
	return nil
}

func (l *BaseLane) Insert(goCtx context.Context, tx sdk.Tx) error {
	txHashStr, err := getTxHashStr(l.txEncoder, tx)
	if err != nil {
		return err
	}

	if err := l.index.Insert(goCtx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	l.txIndex[txHashStr] = struct{}{}
	return nil
}

func (l *BaseLane) Select(goCtx context.Context, txs [][]byte) sdkmempool.Iterator {
	return l.index.Select(goCtx, txs)
}

func (l *BaseLane) CountTx() int {
	return l.index.CountTx()
}

func (l *BaseLane) Remove(tx sdk.Tx) error {
	txHashStr, err := getTxHashStr(l.txEncoder, tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	if err := l.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err)
	}

	delete(l.txIndex, txHashStr)
	return nil
}
