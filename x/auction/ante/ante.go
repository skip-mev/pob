package ante

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/keeper"
)

var _ sdk.AnteDecorator = AuctionDecorator{}

type AuctionDecorator struct {
	auctionKeeper keeper.Keeper
}

func NewAuctionDecorator(ak keeper.Keeper) AuctionDecorator {
	return AuctionDecorator{
		auctionKeeper: ak,
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
		if err := ad.auctionKeeper.ValidateAuctionMsg(ctx, auctionMsg); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}

		// Deduct the entrance fee from the bidder's account and send to the escrow account.
		if err := ad.auctionKeeper.SendReserveFee(ctx, auctionMsg.Bidder); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}
