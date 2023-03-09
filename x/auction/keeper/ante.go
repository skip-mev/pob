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

// AnteHandle validates that the auction bid is valid if one exists.
func (ad AuctionDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Extract the auction bid from the transaction if one exists.
	auctionMsg, err := mempool.GetMsgAuctionBidFromTx(tx)
	if err != nil {
		return ctx, err
	}

	// Validate the auction bid if one exists.
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
//  1. all of the transactions are signed by the signer.
//  2. some subset of contiguous transactions are signed by the signer (starting from the front), and all other tranasctions
//     are signed by the bidder.
//
// example:
//  1. valid: [tx1, tx2, tx3] where tx1 is signed by the signer 1 and tx2 and tx3 are signed by the bidder.
//  2. valid: [tx1, tx2, tx3, tx4] where tx1 - tx4 are signed by the bidder.
//  3. invalid: [tx1, tx2, tx3] where tx1 is signed by the signer 1 and tx2 is signed by the bidder, and tx3 is signed by the signer 2. (possible sandwich attack)
//  4. invalid: [tx1, tx2, tx3] where tx1 is signed by the bidder, and tx2 - tx3 is signed by the signer 1. (possible front-running attack)
func (k Keeper) ValidateBundle(ctx sdk.Context, transactions [][]byte, bidder sdk.AccAddress) error {
	if len(transactions) <= 1 {
		return nil
	}

	// prevSigners is used to track whether the signers of the current transaction overlap.
	prevSigners, err := k.getTxSigners(transactions[0])
	if err != nil {
		return err
	}
	seenBidder := prevSigners[bidder.String()]

	// Check that all subsequent transactions are signed by either
	// 1. the same party as the first transaction
	// 2. the same party for some arbitrary number of txs and then are all remaining txs are signed by the bidder.
	for _, txbytes := range transactions[1:] {
		txSigners, err := k.getTxSigners(txbytes)
		if err != nil {
			return err
		}

		// Filter the signers to only those that signed the current transaction.
		filterSigners(prevSigners, txSigners)

		// If there are no current signers and the bidder address has not been seen, then the bundle can still be valid
		// as long as all subsequent transactions are signed by the bidder.
		if len(prevSigners) == 0 {
			if seenBidder {
				return fmt.Errorf("bundle contains transactions signed by multiple parties. bundle must be signed by the same party.")
			} else {
				seenBidder = true
				prevSigners = map[string]bool{bidder.String(): true}
				filterSigners(prevSigners, txSigners)

				if len(prevSigners) == 0 {
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
			// TODO: check for multi-sig accounts
			// https://github.com/skip-mev/pob/issues/14
			signers[signer.String()] = true
		}
	}

	return signers, nil
}

// filterSigners removes any signers from the authorizedSigners map that are not in the txSigners map.
// This is used to check that all transactions in a bundle are signed by the correct party.
func filterSigners(authorizedSigners, txSigners map[string]bool) {
	for signer := range authorizedSigners {
		if _, ok := txSigners[signer]; !ok {
			delete(authorizedSigners, signer)
		}
	}
}
