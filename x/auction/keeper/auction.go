package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/auction/types"
)

// ValidateAuctionMsg validates that the MsgAuctionBid is valid. It checks that the bidder has sufficient funds to bid the
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

	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return errors.Wrapf(err, "invalid bidder address (%s)", msg.Bidder)
	}

	// Validate the bid amount.
	if err := k.ValidateAuctionBid(ctx, msg.Bid, bidder); err != nil {
		return err
	}

	// Validate the bundle of transactions.
	if err := k.ValidateAuctionBundle(ctx, msg.Transactions, bidder); err != nil {
		return err
	}

	return nil
}

// ValidateAuctionBid validates that the bidder has sufficient funds to participate in the auction
// and distributes the entrance fee to the escrow account.
func (k Keeper) ValidateAuctionBid(ctx sdk.Context, bid sdk.Coins, bidder sdk.AccAddress) error {
	// Auction bid floor.
	minBuyInFee, err := k.GetMinBuyInFee(ctx)
	if err != nil {
		return err
	}

	if !bid.IsAllGTE(minBuyInFee) {
		return fmt.Errorf("bid amount (%s) is less than the minimum bid amount (%s)", bid, minBuyInFee)
	}

	// Entrance fee.
	reserveFee, err := k.GetReserveFee(ctx)
	if err != nil {
		return err
	}

	// Minimum balance required to participate in the auction.
	minBalance := minBuyInFee.Add(reserveFee...)
	balances := k.bankkeeper.GetAllBalances(ctx, bidder)
	if !balances.IsAllGTE(minBalance) {
		return fmt.Errorf("insufficient funds to bid %s (reserve fee + min buy in) with balance %s", minBalance, balances)
	}

	return nil
}

// ValidateAuctionBundle validates the ordering of the referenced transactions. Bundles are valid if
//  1. all of the transactions are signed by the signer.
//  2. some subset of contiguous transactions starting from the first tx are signed by the same signer, and all other tranasctions
//     are signed by the bidder.
//
// example:
//  1. valid: [tx1, tx2, tx3] where tx1 is signed by the signer 1 and tx2 and tx3 are signed by the bidder.
//  2. valid: [tx1, tx2, tx3, tx4] where tx1 - tx4 are signed by the bidder.
//  3. invalid: [tx1, tx2, tx3] where tx1 and tx3 are signed by the bidder and tx2 is signed by some other signer. (possible sandwich attack)
//  4. invalid: [tx1, tx2, tx3] where tx1 is signed by the bidder, and tx2 - tx3 are signed by some other signer. (possible front-running attack)
func (k Keeper) ValidateAuctionBundle(ctx sdk.Context, transactions [][]byte, bidder sdk.AccAddress) error {
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

		// If there are no overlapping signers from the previous tx and the bidder address has not been seen, then the bundle can still be valid
		// as long as all subsequent transactions are signed by the bidder.
		if len(prevSigners) == 0 {
			if seenBidder {
				return fmt.Errorf("bundle contains transactions signed by multiple parties. possible front-running or sandwich attack.")
			} else {
				seenBidder = true
				prevSigners = map[string]bool{bidder.String(): true}
				filterSigners(prevSigners, txSigners)

				if len(prevSigners) == 0 {
					return fmt.Errorf("bundle contains transactions signed by multiple parties. possible front-running or sandwich attack.")
				}
			}
		}
	}

	return nil
}

// SendReserveFee deducts the reserve fee from the bidder's account and sends it to the escrow account.
func (k Keeper) SendReserveFee(ctx sdk.Context, bidder string) error {
	bidderAccount, err := sdk.AccAddressFromBech32(bidder)
	if err != nil {
		return err
	}

	// Deduct the entrance fee from the bidder's account and send to the escrow account.
	escrowAccount, err := k.GetEscrowAccount(ctx)
	if err != nil {
		return err
	}

	reserveFee, err := k.GetReserveFee(ctx)
	if err != nil {
		return err
	}

	if err := k.bankkeeper.SendCoins(ctx, bidderAccount, escrowAccount, reserveFee); err != nil {
		return err
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

// filterSigners removes any signers from the currentSigners map that are not in the txSigners map.
func filterSigners(currentSigners, txSigners map[string]bool) {
	for signer := range currentSigners {
		if _, ok := txSigners[signer]; !ok {
			delete(currentSigners, signer)
		}
	}
}
