package blockbuster

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/skip-mev/pob/mempool"
)

const (
	// LaneNameTOB defines the name of the top-of-block auction lane.
	LaneNameTOB = "tob"
)

var _ Lane = (*TOBLane)(nil)

type TOBLane struct {
	index       sdkmempool.Mempool
	af          mempool.AuctionFactory
	txEncoder   sdk.TxEncoder
	txDecoder   sdk.TxDecoder
	anteHandler sdk.AnteHandler

	// txIndex is a map of all transactions in the mempool. It is used
	// to quickly check if a transaction is already in the mempool.
	txIndex map[string]struct{}
}

func NewTOBLane(txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, maxTx int, af mempool.AuctionFactory, anteHandler sdk.AnteHandler) *TOBLane {
	return &TOBLane{
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

func (l *TOBLane) Name() string {
	return LaneNameTOB
}

func (l *TOBLane) Match(tx sdk.Tx) bool {
	bidInfo, err := l.af.GetAuctionBidInfo(tx)
	return bidInfo != nil && err == nil
}

func (l *TOBLane) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := l.txIndex[txHashStr]
	return ok, nil
}

func (l *TOBLane) VerifyTx(ctx sdk.Context, bidTx sdk.Tx) error {
	bidInfo, err := l.af.GetAuctionBidInfo(bidTx)
	if err != nil {
		return fmt.Errorf("failed to get auction bid info: %w", err)
	}

	// verify the top-level bid transaction
	ctx, err = l.anteHandler(ctx, bidTx, false)
	if err != nil {
		return fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
	}

	// verify all of the bundled transactions
	for _, tx := range bidInfo.Transactions {
		bundledTx, err := l.af.WrapBundleTransaction(tx)
		if err != nil {
			return fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		// bid txs cannot be included in bundled txs
		bidInfo, _ := l.af.GetAuctionBidInfo(bundledTx)
		if bidInfo != nil {
			return fmt.Errorf("invalid bid tx; bundled tx cannot be a bid tx")
		}

		if ctx, err = l.anteHandler(ctx, bundledTx, false); err != nil {
			return fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return nil
}

// PrepareLane which builds a portion of the block. Inputs a cache of transactions
// that have already been included by a previous lane.
func (l *TOBLane) PrepareLane(ctx sdk.Context, cache map[string]struct{}) [][]byte {
	panic("not implemented")
}

// ProcessLane which verifies the lane's portion of a proposed block.
func (l *TOBLane) ProcessLane(ctx sdk.Context, txs [][]byte) error {
	panic("not implemented")
}

func (l *TOBLane) Insert(goCtx context.Context, tx sdk.Tx) error {
	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return err
	}

	if err := l.index.Insert(goCtx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	l.txIndex[txHashStr] = struct{}{}
	return nil
}

func (l *TOBLane) Select(goCtx context.Context, txs [][]byte) sdkmempool.Iterator {
	return l.index.Select(goCtx, txs)
}

func (l *TOBLane) CountTx() int {
	return l.index.CountTx()
}

func (l *TOBLane) Remove(tx sdk.Tx) error {
	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
	}

	if err := l.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err)
	}

	delete(l.txIndex, txHashStr)
	return nil
}

// getTxHashStr returns the transaction hash string for a given transaction.
func (l *TOBLane) getTxHashStr(tx sdk.Tx) (string, error) {
	txBz, err := l.txEncoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txHashStr, nil
}
