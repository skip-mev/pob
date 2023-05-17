package ante

import (
	"bytes"
	"context"
	"fmt"

	"cosmossdk.io/errors"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/builder/keeper"
)

var _ sdk.AnteDecorator = BuilderDecorator{}

type (
	// Mempool is an interface that defines the methods required to interact with the application-side mempool.
	Mempool interface {
		Contains(tx sdk.Tx) (bool, error)
		GetAuctionBidInfo(tx sdk.Tx) (*mempool.AuctionBidInfo, error)
		GetTopAuctionTx(ctx context.Context) sdk.Tx
	}

	// BuilderDecorator is an AnteDecorator that validates the auction bid and bundled transactions.
	BuilderDecorator struct {
		builderKeeper keeper.Keeper
		txDecoder     sdk.TxDecoder
		txEncoder     sdk.TxEncoder
		mempool       Mempool
	}

	// GasTx is an interface that defines the methods required to extract gas information from a transaction.
	GasTx interface {
		sdk.Tx
		GetGas() uint64
	}
)

func NewBuilderDecorator(ak keeper.Keeper, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, mempool Mempool) BuilderDecorator {
	return BuilderDecorator{
		builderKeeper: ak,
		txDecoder:     txDecoder,
		txEncoder:     txEncoder,
		mempool:       mempool,
	}
}

// AnteHandle validates that the auction bid is valid if one exists. If valid it will deduct the entrance fee from the
// bidder's account.
func (bd BuilderDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// If comet is re-checking a transaction, we only need to check if the transaction is in the application-side mempool.
	if ctx.IsReCheckTx() {
		contains, err := bd.mempool.Contains(tx)
		if err != nil {
			return ctx, err
		}

		if !contains {
			return ctx, fmt.Errorf("transaction not found in application mempool")
		}
	}

	bidInfo, err := bd.mempool.GetAuctionBidInfo(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if bidInfo != nil {
		// Auction transactions must have a timeout set to a valid block height.
		if int64(bidInfo.Timeout) < ctx.BlockHeight() {
			return ctx, fmt.Errorf("timeout height cannot be less than the current block height")
		}

		// We only need to verify the auction bid relative to the local validator's mempool if the mode
		// is checkTx or recheckTx. Otherwise, the ABCI handlers (VerifyVoteExtension, ExtendVoteExtension, etc.)
		// will always compare the auction bid to the highest bidding transaction in the mempool leading to
		// poor liveness guarantees.
		topBid := sdk.Coin{}
		if ctx.IsCheckTx() || ctx.IsReCheckTx() {
			if topBidTx := bd.mempool.GetTopAuctionTx(ctx); topBidTx != nil {
				topBidBz, err := bd.txEncoder(topBidTx)
				if err != nil {
					return ctx, err
				}

				currentTxBz, err := bd.txEncoder(tx)
				if err != nil {
					return ctx, err
				}

				// Compare the bytes to see if the current transaction is the highest bidding transaction.
				if !bytes.Equal(topBidBz, currentTxBz) {
					topBidInfo, err := bd.mempool.GetAuctionBidInfo(topBidTx)
					if err != nil {
						return ctx, err
					}

					topBid = topBidInfo.Bid
				}
			}
		}

		if err := bd.builderKeeper.ValidateBidInfo(ctx, topBid, bidInfo); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}

		// Validate that the auction bid tx is valid based on the rest of the ante handler chain.
		if ctx, err = next(ctx, tx, simulate); err != nil {
			return ctx, err
		}

		// Validate the bundled transactions.
		if err = bd.ValidateBundleTxs(ctx, bidInfo.Transactions, simulate, next); err != nil {
			return ctx, errors.Wrap(err, "failed to validate bundled transactions")
		}

		// Short circuit the rest of the ante handler checks since we have already run them on the
		// bid tx and all bundled txs.
		return ctx, nil
	}

	return next(ctx, tx, simulate)
}

// ValidateBundleTxs validates the bundled transactions using the cache context. We do not directly write all of the changes
// to the state since we do not know if the bid tx will be accepted into the mempool. Additionally, we want
// to ensure that transactions that are broadcasted normally (not in a bundle) are able to be processed
// normally.
func (bd BuilderDecorator) ValidateBundleTxs(ctx sdk.Context, txs [][]byte, simulate bool, next sdk.AnteHandler) error {
	cacheCtx, _ := ctx.CacheContext()

	// Validate each transaction in the bundle and apply state changes to the cache context in
	// the order that they are received.
	for _, txBz := range txs {
		tx, err := bd.txDecoder(txBz)
		if err != nil {
			return errors.Wrap(err, "failed to decode bundled transaction")
		}

		// If the transaction is not in the mempool, we need to validate it.
		//
		// Is this a safe assumption? What if the transaction is in the mempool but the mempool
		// is out of sync with the application state?
		contains, err := bd.mempool.Contains(tx)
		if err != nil {
			return errors.Wrap(err, "failed to check if transaction is in mempool")
		}

		if !contains {
			// Set the gas meter to the gas limit of the transaction.
			gasTx, ok := tx.(GasTx)
			if !ok {
				return fmt.Errorf("transaction must implement GasTx interface")
			}

			// Set the gas meter to the gas limit of the transaction.
			cacheCtx = SetGasMeter(simulate, cacheCtx, gasTx.GetGas())

			// Validate the transaction by running the ante handler chain.
			if cacheCtx, err = next(cacheCtx, tx, simulate); err != nil {
				return errors.Wrap(err, "failed to validate bundled transaction")
			}
		}
	}

	return nil
}

// SetGasMeter returns a new context with a gas meter set from a given context.
func SetGasMeter(simulate bool, ctx sdk.Context, gasLimit uint64) sdk.Context {
	// In various cases such as simulation and during the genesis block, we do not
	// meter any gas utilization.
	if simulate || ctx.BlockHeight() == 0 {
		return ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	return ctx.WithGasMeter(storetypes.NewGasMeter(gasLimit))
}
