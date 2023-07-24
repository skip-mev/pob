package abci

import (
	"context"
	"fmt"

	log "cosmossdk.io/log"
	cometabci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/skip-mev/pob/x/builder/types"
)

type (
	// CheckTxHandler is a wrapper around baseapp's CheckTx method that allows us to
	// verify bid transactions against the latest committed state. All other transactions
	// are executed normally using base app's CheckTx. This defines all of the
	// dependencies that are required to verify a bid transaction.
	CheckTxHandler struct {
		// baseApp is utilized to retrieve the latest committed state and to call
		// baseapp's CheckTx method.
		baseApp BaseApp

		// txDecoder is utilized to decode transactions to determine if they are
		// bid transactions.
		txDecoder sdk.TxDecoder

		// TOBLane is utilized to retrieve the bid info of a transaction and to
		// insert a bid transaction into the application-side mempool.
		tobLane TOBLane

		// anteHandler is utilized to verify the bid transaction against the latest
		// committed state.
		anteHandler sdk.AnteHandler

		// chainID is the chain ID of the blockchain.
		chainID string
	}

	// CheckTx is baseapp's CheckTx method that checks the validity of a
	// transaction.
	CheckTx func(req *cometabci.RequestCheckTx) (*cometabci.ResponseCheckTx, error)

	// TOBLane is the interface that defines all of the dependencies that
	// are required to interact with the top of block lane.
	TOBLane interface {
		// GetAuctionBidInfo is utilized to retrieve the bid info of a transaction.
		GetAuctionBidInfo(tx sdk.Tx) (*types.BidInfo, error)

		// Insert is utilized to insert a transaction into the application-side mempool.
		Insert(ctx context.Context, tx sdk.Tx) error

		// WrapBundleTransaction is utilized to wrap a transaction included in a bid transaction
		// into an sdk.Tx.
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)
	}

	// BaseApp is an interface that allows us to call baseapp's CheckTx method
	// as well as retrieve the latest committed state.
	BaseApp interface {
		GetFinalizeBlockStateCtx() sdk.Context

		// CheckTx is baseapp's CheckTx method that checks the validity of a
		// transaction.
		CheckTx(req *cometabci.RequestCheckTx) (*cometabci.ResponseCheckTx, error)

		// Logger is utilized to log errors.
		Logger() log.Logger
	}
)

// NewCheckTxHandler is a constructor for CheckTxHandler.
func NewCheckTxHandler(
	baseApp BaseApp,
	txDecoder sdk.TxDecoder,
	tobLane TOBLane,
	anteHandler sdk.AnteHandler,
	chainID string,
) *CheckTxHandler {
	return &CheckTxHandler{
		baseApp:     baseApp,
		txDecoder:   txDecoder,
		tobLane:     tobLane,
		anteHandler: anteHandler,
		chainID:     chainID,
	}
}

// CheckTxHandler is a wrapper around baseapp's CheckTx method that allows us to
// verify bid transactions against the latest committed state. All other transactions
// are executed normally. We must verify each bid tx and all of its bundled transactions
// before we can insert it into the mempool against the latest commit state because
// otherwise the auction can be griefed. No state changes are applied to the state
// during this process.
func (handler *CheckTxHandler) CheckTx() CheckTx {
	return func(req *cometabci.RequestCheckTx) (resp *cometabci.ResponseCheckTx, err error) {
		defer func() {
			if err := recover(); err != nil {
				resp = sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("panic in check tx handler: %s", err), 0, 0, nil, false)
			}
		}()

		tx, err := handler.txDecoder(req.Tx)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("failed to decode tx: %w", err), 0, 0, nil, false), err
		}

		// Attempt to get the bid info of the transaction.
		bidInfo, err := handler.tobLane.GetAuctionBidInfo(tx)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("failed to get auction bid info: %w", err), 0, 0, nil, false), err
		}

		// If this is not a bid transaction, we just execute it normally.
		if bidInfo == nil {
			return handler.baseApp.CheckTx(req)
		}

		// We attempt to get the latest committed state in order to verify transactions
		// as if they were to be executed at the top of the block. After verification, this
		// context will be discarded and will not apply any state changes.
		ctx := handler.baseApp.GetFinalizeBlockStateCtx()

		// Verify the bid transaction.
		gasInfo, err := handler.ValidateBidTx(ctx, tx, bidInfo)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx: %w", err), gasInfo.GasWanted, gasInfo.GasUsed, nil, false), err
		}

		// If the bid transaction is valid, we know we can insert it into the mempool for consideration in the next block.
		if err := handler.tobLane.Insert(ctx, tx); err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to insert bid transaction into mempool: %w", err), gasInfo.GasWanted, gasInfo.GasUsed, nil, false), err
		}

		return &cometabci.ResponseCheckTx{
			Code:      cometabci.CodeTypeOK,
			GasWanted: int64(gasInfo.GasWanted),
			GasUsed:   int64(gasInfo.GasUsed),
		}, nil
	}
}

// ValidateBidTx is utilized to verify the bid transaction against the latest committed state.
func (handler *CheckTxHandler) ValidateBidTx(ctx sdk.Context, bidTx sdk.Tx, bidInfo *types.BidInfo) (sdk.GasInfo, error) {
	// Verify the bid transaction.
	ctx, err := handler.anteHandler(ctx, bidTx, false)
	if err != nil {
		return sdk.GasInfo{}, fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err)
	}

	// Store the gas info and priority of the bid transaction before applying changes with other transactions.
	gasInfo := sdk.GasInfo{
		GasWanted: ctx.GasMeter().Limit(),
		GasUsed:   ctx.GasMeter().GasConsumed(),
	}

	// Verify all of the bundled transactions.
	for _, tx := range bidInfo.Transactions {
		bundledTx, err := handler.tobLane.WrapBundleTransaction(tx)
		if err != nil {
			return gasInfo, fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err)
		}

		// bid txs cannot be included in bundled txs
		bidInfo, _ := handler.tobLane.GetAuctionBidInfo(bundledTx)
		if bidInfo != nil {
			return gasInfo, fmt.Errorf("invalid bid tx; bundled tx cannot be a bid tx")
		}

		if ctx, err = handler.anteHandler(ctx, bundledTx, false); err != nil {
			return gasInfo, fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err)
		}
	}

	return gasInfo, nil
}
