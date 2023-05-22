package abci

import (
	"context"
	"fmt"

	cometabci "github.com/cometbft/cometbft/abci/types"
	log "github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/skip-mev/pob/mempool"
)

type (
	// CheckTxHandler is a wrapper around baseapp's CheckTx method that allows us to
	// verify bid transactions against the latest committed state. All other transactions
	// are executed normally. This defines all of the dependencies that are required to
	// verify a bid transaction.
	CheckTxHandler struct {
		baseApp     BaseApp
		txDecoder   sdk.TxDecoder
		mempool     CheckTxMempool
		anteHandler sdk.AnteHandler
		chainID     string
	}

	// CheckTx is baseapp's CheckTx method that checks the validity of a
	// transaction.
	CheckTx func(cometabci.RequestCheckTx) cometabci.ResponseCheckTx

	// CheckTxMempool is an interface that allows us to check if a transaction
	// exists in the mempool and get the bid info of a transaction.
	CheckTxMempool interface {
		// GetAuctionBidInfo is utilized to retrieve the bid info of a transaction.
		GetAuctionBidInfo(tx sdk.Tx) (*mempool.AuctionBidInfo, error)

		// Insert is utilized to insert a transaction into the application-side mempool.
		Insert(ctx context.Context, tx sdk.Tx) error

		// WrapBundleTransaction is utilized to wrap a transaction included in a bid transaction
		// into an sdk.Tx.
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)
	}

	// BaseApp is an interface that allows us to call baseapp's CheckTx method
	// as well as retrieve the latest committed state.
	BaseApp interface {
		// CommitMultiStore is utilized to retrieve the latest committed state.
		CommitMultiStore() sdk.CommitMultiStore

		// CheckTx is baseapp's CheckTx method that checks the validity of a
		// transaction.
		CheckTx(cometabci.RequestCheckTx) cometabci.ResponseCheckTx

		// Logger is utilized to log errors.
		Logger() log.Logger

		// LastBlockHeight is utilized to retrieve the latest block height.
		LastBlockHeight() int64
	}
)

// NewCheckTxHandler is a constructor for CheckTxHandler.
func NewCheckTxHandler(baseApp BaseApp, txDecoder sdk.TxDecoder, mempool CheckTxMempool, anteHandler sdk.AnteHandler, chainID string) CheckTxHandler {
	return CheckTxHandler{
		baseApp:     baseApp,
		txDecoder:   txDecoder,
		mempool:     mempool,
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
	return func(req cometabci.RequestCheckTx) (resp cometabci.ResponseCheckTx) {
		defer func() {
			if err := recover(); err != nil {
				resp = sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("panic in check tx handler: %s", err), 0, 0, nil, false)
			}
		}()

		sdkTx, err := handler.txDecoder(req.Tx)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("failed to decode tx: %w", err), 0, 0, nil, false)
		}

		// Attempt to get the bid info of the transaction.
		bidInfo, err := handler.mempool.GetAuctionBidInfo(sdkTx)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("failed to get auction bid info: %w", err), 0, 0, nil, false)
		}

		// If this is not a bid transaction, we just execute it normally.
		if bidInfo == nil {
			return handler.baseApp.CheckTx(req)
		}

		// We attempt to get the latest committed state in order to verify transactions
		// as if they were to be executed at the top of the block. After verification, this
		// context will be discarded and will not apply any state changes.
		ctx := handler.GetContextForBidTx(req)

		// Verify the bid transaction.
		ctx, err = handler.anteHandler(ctx, sdkTx, false)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err), 0, 0, nil, false)
		}

		// Verify all of the bundled transactions.
		for _, tx := range bidInfo.Transactions {
			bundledTx, err := handler.mempool.WrapBundleTransaction(tx)
			if err != nil {
				return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err), 0, 0, nil, false)
			}

			bidInfo, err := handler.mempool.GetAuctionBidInfo(bundledTx)
			if err != nil {
				return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to get auction bid info: %w", err), 0, 0, nil, false)
			}

			// Bid txs cannot be included in bundled txs.
			if bidInfo != nil {
				return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; bundled tx cannot be a bid tx"), 0, 0, nil, false)
			}

			if ctx, err = handler.anteHandler(ctx, bundledTx, false); err != nil {
				return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err), 0, 0, nil, false)
			}
		}

		// If the bid transaction is valid, we know we can insert it into the mempool for consideration in the next block.
		if err := handler.mempool.Insert(ctx, sdkTx); err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to insert bid transaction into mempool: %w", err), 0, 0, nil, false)
		}

		return cometabci.ResponseCheckTx{Code: cometabci.CodeTypeOK}
	}
}

// GetContextForBidTx is a function that returns a context for a bid transaction.
// This context is used to verify the bid transaction against the latest committed state.
func (handler *CheckTxHandler) GetContextForBidTx(req cometabci.RequestCheckTx) sdk.Context {
	// Retrieve the commit multi-store which is used to retrieve the latest committed state.
	ms := handler.baseApp.CommitMultiStore().CacheMultiStore()

	// Create a new context based off of the latest committed state.
	ctx, _ := sdk.NewContext(ms, tmproto.Header{}, false, handler.baseApp.Logger()).CacheContext()

	// Set the context to the correct checking mode.
	switch req.Type {
	case cometabci.CheckTxType_New:
		ctx = ctx.WithIsCheckTx(true)
	case cometabci.CheckTxType_Recheck:
		ctx = ctx.WithIsReCheckTx(true)
	default:
		panic("unknown check tx type")
	}

	// Set the remaining important context values.
	ctx = ctx.
		WithBlockHeight(handler.baseApp.LastBlockHeight()).
		WithTxBytes(req.Tx).
		WithChainID(handler.chainID) // TODO: Replace with actual chain ID. This is currently not exposed by the app.

	return ctx
}
