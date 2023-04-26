package abci

import (
	"context"
	"errors"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

type (
	Mempool interface {
		sdkmempool.Mempool
		AuctionBidSelect(ctx context.Context) sdkmempool.Iterator
		GetBundledTransactions(tx sdk.Tx) ([][]byte, error)
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)
		IsAuctionTx(tx sdk.Tx) (bool, error)
	}

	//nolint:all
	ABCIHandler struct {
		mempool     Mempool
		logger      log.Logger
		anteHandler sdk.AnteHandler
		txEncoder   sdk.TxEncoder
		txDecoder   sdk.TxDecoder
	}
)

// NewABCIHandler returns an ABCIHandler that contains the functionality and handlers
// required to process, validate and build blocks.
func NewABCIHandler(
	mp Mempool,
	logger log.Logger,
	anteHandler sdk.AnteHandler,
	txEncoder sdk.TxEncoder,
	txDecoder sdk.TxDecoder,
) *ABCIHandler {
	return &ABCIHandler{
		mempool:     mp,
		logger:      logger,
		anteHandler: anteHandler,
		txEncoder:   txEncoder,
		txDecoder:   txDecoder,
	}
}

// RemoveTx removes a transaction from the application-side mempool.
func (h *ABCIHandler) RemoveTx(tx sdk.Tx) {
	if err := h.mempool.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		panic(fmt.Errorf("failed to remove invalid transaction from the mempool: %w", err))
	}
}

// VerifyTx verifies a transaction against the application's state.
func (h *ABCIHandler) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	if h.anteHandler != nil {
		_, err := h.anteHandler(ctx, tx, false)
		return err
	}

	return nil
}
