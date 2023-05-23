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
	LaneNameTOB = "tob"
)

var _ Lane = (*TOBLane)(nil)

type TOBLane struct {
	index sdkmempool.Mempool
	af    mempool.AuctionFactory

	// txIndex is a map of all transactions in the mempool. It is used
	// to quickly check if a transaction is already in the mempool.
	txIndex map[string]struct{}

	// txEncoder defines the sdk.Tx encoder that allows us to encode transactions
	// to bytes.
	txEncoder sdk.TxEncoder
}

func (l *TOBLane) Name() string {
	return LaneNameTOB
}

// Match determines if a transaction belongs to this lane.
func (l *TOBLane) Match(tx sdk.Tx) bool {
	_, err := l.af.GetAuctionBidInfo(tx)
	if err != nil {
		return false
	}

	return true
}

// Contains returns true if the mempool contains the given transaction.
func (l *TOBLane) Contains(tx sdk.Tx) (bool, error) {
	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return false, fmt.Errorf("failed to get tx hash string: %w", err)
	}

	_, ok := l.txIndex[txHashStr]
	return ok, nil
}

// VerifyTx verifies the transaction belonging to this lane.
func (l *TOBLane) VerifyTx(ctx sdk.Context, tx sdk.Tx) error {
	panic("not implemented")
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
	if err := l.index.Insert(goCtx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return err
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
	err := l.index.Remove(tx)
	if err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err)
	}

	txHashStr, err := l.getTxHashStr(tx)
	if err != nil {
		return fmt.Errorf("failed to get tx hash string: %w", err)
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
