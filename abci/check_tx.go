package abci

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/skip-mev/pob/mempool"
)

type (
	// CheckTx is baseapp's CheckTx method that checks the validity of a
	// transaction.
	CheckTx func(abci.RequestCheckTx) abci.ResponseCheckTx

	// GetContextForBidTx is an interface that allows us to get a context for
	// a bid transaction.
	GetContextForBidTx func(req abci.RequestCheckTx) sdk.Context

	// CheckTxMempool is an interface that allows us to check if a transaction
	// exists in the mempool and get the bid info of a transaction.
	CheckTxMempool interface {
		GetAuctionBidInfo(tx sdk.Tx) (*mempool.AuctionBidInfo, error)
		Insert(ctx context.Context, tx sdk.Tx) error
		WrapBundleTransaction(tx []byte) (sdk.Tx, error)
	}

	// BaseApp is an interface that allows us to call baseapp's CheckTx method.
	BaseApp interface {
		CheckTx(abci.RequestCheckTx) abci.ResponseCheckTx
	}
)

// CheckTxHandler is a wrapper around baseapp's CheckTx method that allows us to
// verify bid transactions against the latest committed state. All other transactions
// are executed normally. We must verify each bid tx and all of its bundled transactions
// before we can insert it into the mempool against the latest commit state because
// otherwise the auction can be griefed. No state changes are applied to the state
// during this process.
func CheckTxHandler(baseApp BaseApp, getContextForBidTx GetContextForBidTx, txDecoder sdk.TxDecoder, mempool CheckTxMempool, anteHandler sdk.AnteHandler) CheckTx {
	return func(req abci.RequestCheckTx) abci.ResponseCheckTx {
		sdkTx, err := txDecoder(req.Tx)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("failed to decode tx: %w", err), 0, 0, nil, false)
		}

		// Attempt to get the bid info of the transaction.
		bidInfo, err := mempool.GetAuctionBidInfo(sdkTx)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("failed to get auction bid info: %w", err), 0, 0, nil, false)
		}

		// If this is not a bid transaction, we just execute it normally.
		if bidInfo == nil {
			return baseApp.CheckTx(req)
		}

		// We attempt to get the latest committed state in order to verify transactions
		// as if they were to be executed at the top of the block. After verification, this
		// context will be discarded and will not apply any state changes.
		ctx := getContextForBidTx(req)

		// Verify the bid transaction.
		ctx, err = anteHandler(ctx, sdkTx, false)
		if err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to execute ante handler: %w", err), 0, 0, nil, false)
		}

		// Get the gas used and logs from the context.
		gasUsed := ctx.GasMeter().GasConsumed()
		logs := ctx.EventManager().ABCIEvents()
		sender := bidInfo.Bidder.String()

		// Verify all of the bundled transactions.
		for _, tx := range bidInfo.Transactions {
			bundledTx, err := mempool.WrapBundleTransaction(tx)
			if err != nil {
				return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to decode bundled tx: %w", err), 0, 0, nil, false)
			}

			if ctx, err = anteHandler(ctx, bundledTx, false); err != nil {
				return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to execute bundled transaction: %w", err), 0, 0, nil, false)
			}
		}

		// If the bid transaction is valid, we know we can insert it into the mempool for consideration in the next block.
		if err := mempool.Insert(ctx, sdkTx); err != nil {
			return sdkerrors.ResponseCheckTxWithEvents(fmt.Errorf("invalid bid tx; failed to insert bid transaction into mempool: %w", err), 0, 0, nil, false)
		}

		return abci.ResponseCheckTx{
			Code:    abci.CodeTypeOK,
			GasUsed: int64(gasUsed),
			Events:  logs,
			Info:    "valid bid transaction inserted into mempool",
			Sender:  sender,
		}
	}
}
