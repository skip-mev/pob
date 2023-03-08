package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/types"
)

var _ sdk.AnteDecorator = AuctionDecorator{}

type AuctionDecorator struct {
	auctionKeeper Keeper
}

func NewAuctionDecorator(ak Keeper) AuctionDecorator {
	return AuctionDecorator{
		auctionKeeper: ak,
	}
}

// AnteHandle is the ante handler for the auction module. It validates that the auction bid is valid if one exists.
func (ad AuctionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	maxBundleSize, err := ad.auctionKeeper.GetMaxBundleSize(ctx)
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
		if err := ad.auctionKeeper.ValidateAuctionMsg(ctx, auctionMsg); err != nil {
			return ctx, errors.Wrap(err, "failed to validate auction bid")
		}
	}

	return next(ctx, tx, simulate)
}

// ValidateAuctionTx validates that the MsgAuctionBid is valid. It checks that the bidder has sufficient funds to bid the
// amount specified in the message, that the bundle size is not greater than the max bundle size, and that the bundle
// transactions are valid.
func (k Keeper) ValidateAuctionMsg(ctx sdk.Context, msg *types.MsgAuctionBid) error {
	// Validate the bundle size.
	maxBundleSize, err := k.GetMaxBundleSize(ctx)
	if err != nil {
		return err
	}

	if uint32(len(msg.Transactions)) > maxBundleSize {
		return fmt.Errorf("bundle size (%d) exceeds max bundle size (%d)", len(msg.Transactions), maxBundleSize)
	}

	// Validate the bid.
	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return errors.Wrapf(err, "invalid bidder address (%s)", msg.Bidder)
	}

	balances := k.bankkeeper.GetAllBalances(ctx, bidder)
	if !balances.IsAllGTE(msg.Bid) {
		return fmt.Errorf("insufficient funds to bid %s", msg.Bid)
	}

	// Validate the bundle of transactions.
	if err := k.ValidateBundle(ctx, msg.Transactions, bidder); err != nil {
		return err
	}

	return nil
}

// ValidateBundle validates the ordering of the referenced transactions. Bundles are valid if
// 1. all of the transactions are signed by the signer.
// 2. the first transactions are signed by the same party (not sender), and the subsequent txs are signed by the bidder.
func (k Keeper) ValidateBundle(ctx sdk.Context, transactions [][]byte, bidder sdk.AccAddress) error {
	if len(transactions) <= 1 {
		return nil
	}

	// Get the signers of the first transaction and check if the bidder is one of them.
	authorizedSigners, err := k.getTxSigners(transactions[0])
	if err != nil {
		return err
	}
	seenBidder := authorizedSigners[bidder.String()]

	// Check that all subsequent transactions are signed by either
	// 1. the same party as the first transaction, or
	// 2. the bidder.
	for _, txbytes := range transactions[1:] {

		txSigners, err := k.getTxSigners(txbytes)
		if err != nil {
			return err
		}

		// Check that all transactions are signed by the same bidder.
		filterSigners(authorizedSigners, txSigners)

		// If there are no authorized signers and the bidder address has not been seen, then the bundle can still be valid
		// as long as all subsequent transactions are signed by the bidder.
		if len(authorizedSigners) == 0 {
			if seenBidder {
				return fmt.Errorf("bundle contains transactions signed by multiple parties. bundle must be signed by the same party.")
			} else {
				seenBidder = true
				authorizedSigners = map[string]bool{bidder.String(): true}
				filterSigners(authorizedSigners, txSigners)

				if len(authorizedSigners) == 0 {
					return fmt.Errorf("bundle contains transactions signed by multiple parties. bundle must be signed by the same party.")
				}
			}
		}
	}

	return nil
}

// getTxSigners returns the signers of a transaction.
func (k Keeper) getTxSigners(txBytes []byte) (map[string]bool, error) {
	tx, err := k.DecodeTx(txBytes)
	if err != nil {
		return nil, err
	}

	signers := make(map[string]bool, 0)
	for _, msg := range tx.GetMsgs() {
		for _, signer := range msg.GetSigners() {
			signers[signer.String()] = true
		}
	}

	return signers, nil
}

// filterSigners removes any signers fromt he authorizedSigners map that are not in the txSigners map.
// This is used to check that all transactions in a bundle are signed by the correct party.
func filterSigners(authorizedSigners, txSigners map[string]bool) {
	for signer := range authorizedSigners {
		if _, ok := txSigners[signer]; !ok {
			delete(authorizedSigners, signer)
		}
	}
}
