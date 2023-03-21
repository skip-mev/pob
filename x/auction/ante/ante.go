package ante

import (
	"bytes"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/keeper"
)

var _ sdk.AnteDecorator = AuctionDecorator{}

type AuctionDecorator struct {
	auctionKeeper keeper.Keeper
	txDecoder     sdk.TxDecoder
	txEncoder     sdk.TxEncoder
	mempool       *mempool.AuctionMempool
}

func NewAuctionDecorator(ak keeper.Keeper, txDecoder sdk.TxDecoder, txEncoder sdk.TxEncoder, mempool *mempool.AuctionMempool) AuctionDecorator {
	return AuctionDecorator{
		auctionKeeper: ak,
		txDecoder:     txDecoder,
		txEncoder:     txEncoder,
		mempool:       mempool,
	}
}

// AnteHandle validates that the auction bid is valid if one exists. If valid it will deduct the entrance fee from the
// bidder's account.
func (ad AuctionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	auctionMsg, err := mempool.GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
	if auctionMsg != nil {
		bidder, err := sdk.AccAddressFromBech32(auctionMsg.Bidder)
		if err != nil {
			return ctx, errors.Wrapf(err, "invalid bidder address (%s)", auctionMsg.Bidder)
		}

		transactions := make([]sdk.Tx, len(auctionMsg.Transactions))
		for i, tx := range auctionMsg.Transactions {
			decodedTx, err := ad.txDecoder(tx)
			if err != nil {
				return ctx, errors.Wrapf(err, "failed to decode transaction (%s)", tx)
			}

			transactions[i] = decodedTx
		}

		highestBid, err := ad.GetTopAuctionBid(ctx, tx)
		if err != nil {
			return ctx, errors.Wrap(err, "failed to get highest auction bid")
		}

		if err := ad.auctionKeeper.ValidateAuctionMsg(ctx, bidder, auctionMsg.Bid, highestBid, transactions); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}

// GetTopAuctionBid returns the highest auction bid if one exists. If the current transaction is the highest
// bidding transaction, then an empty coin set is returned.
func (ad AuctionDecorator) GetTopAuctionBid(ctx sdk.Context, currTx sdk.Tx) (sdk.Coins, error) {
	auctionTx := ad.mempool.GetTopAuctionTx(ctx)
	if auctionTx == nil {
		return sdk.NewCoins(), nil
	}

	wrappedTx := auctionTx.(*mempool.WrappedBidTx)

	// Check if the current transaction is the highest bidding transaction.
	auctionBz, err := ad.txEncoder(wrappedTx.Tx)
	if err != nil {
		return sdk.NewCoins(), errors.Wrap(err, "failed to encode auction transaction")
	}

	currBz, err := ad.txEncoder(currTx)
	if err != nil {
		return sdk.NewCoins(), errors.Wrap(err, "failed to encode current transaction")
	}

	if bytes.Equal(auctionBz, currBz) {
		return sdk.NewCoins(), nil
	}

	return wrappedTx.GetBid(), nil
}
