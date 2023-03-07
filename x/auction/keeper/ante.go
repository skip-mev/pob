package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/types"
)

type AuctionDecorator struct {
	AuctionKeeper Keeper
}

func NewAuctionDecorator(ak Keeper) AuctionDecorator {
	return AuctionDecorator{
		AuctionKeeper: ak,
	}
}

// AnteHandle is the ante handler for the auction module. It validates that the auction bid is valid if one exists.
func (ad AuctionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	maxBundleSize, err := ad.AuctionKeeper.GetMaxBundleSize(ctx)
	if err != nil {
		return ctx, err
	}

	if maxBundleSize == 0 {
		return next(ctx, tx, simulate)
	}

	// Extract the auction bid from the transaction if one exists.
	auctionMsg, err := mempool.GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return ctx, err
	}

	// If a bid exists, validate it and add it to the mempool.
	if auctionMsg != nil {
		if err := ad.ValidateAuctionTx(ctx, auctionMsg); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}

// ValidateAuctionTx validates that the MsgAuctionBid is valid. It checks that the bidder has sufficient funds to bid the
// amount specified in the message, that the bundle size is not greater than the max bundle size, and that the bundle
// transactions are valid.
func (ad AuctionDecorator) ValidateAuctionTx(ctx sdk.Context, msg *types.MsgAuctionBid) error {
	// Validate the bundle size.
	maxBundleSize, err := ad.AuctionKeeper.GetMaxBundleSize(ctx)
	if err != nil {
		return err
	}

	if uint32(len(msg.Transactions)) > maxBundleSize {
		return fmt.Errorf("bundle size (%d) exceeds max bundle size (%d)", len(msg.Transactions), maxBundleSize)
	}

	// Validate the bidder address.
	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return errors.Wrapf(err, "invalid bidder address (%s)", msg.Bidder)
	}

	// Validate the bid.
	balances := ad.AuctionKeeper.bankkeeper.GetAllBalances(ctx, bidder)
	if !balances.IsAllGTE(msg.Bid) {
		return fmt.Errorf("insufficient funds to bid %s", msg.Bid)
	}

	// Validate the bundle of transactions.
	if err := ad.ValidateBundle(ctx, msg.Transactions, bidder); err != nil {
		return err
	}

	return nil
}

// ValidateBundle validates the referenced transactions
func (ad AuctionDecorator) ValidateBundle(ctx sdk.Context, transactions [][]byte, bidder sdk.AccAddress) error {
	return nil
}
