package keeper

import (
	"fmt"
	"sort"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/types"
)

type AuctionDecorator struct {
	AuctionKeeper Keeper
	sdk.TxDecoder
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

	// Validate the bid.
	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return errors.Wrapf(err, "invalid bidder address (%s)", msg.Bidder)
	}

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

// ValidateBundle validates the ordering of the referenced transactions. Bundles are valid if
// 1. all of the transactions are signed by the same bidder.
// 2. the first transactions are signed by the same party (not sender), and the subsequent txs are signed by the bidder.
func (ad AuctionDecorator) ValidateBundle(ctx sdk.Context, transactions [][]byte, bidder sdk.AccAddress) error {
	// Get the signers for each transaction.
	signers := make([][]sdk.AccAddress, len(transactions))
	for i, txBytes := range transactions {
		tx, err := ad.DecodeTx(txBytes)
		if err != nil {
			return err
		}

		signers[i] = getTxSigners(tx)
	}

	return nil
}

// getTxSigners returns the signers of a transaction sorted by address.
func getTxSigners(tx sdk.Tx) []sdk.AccAddress {
	signers := make([]sdk.AccAddress, 0)
	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		for _, signer := range msg.GetSigners() {
			signers = append(signers, signer)
		}
	}

	sort.SliceStable(signers, func(i, j int) bool {
		return signers[i].String() < signers[j].String()
	})

	return signers
}

// Checks if there is any overlap between the transaction and bundle signers. Used to check whether a
// transaction was also signed by the bundle sender.
func checkTransactionHasBundleSigner(transactionSigners, bundleSigners map[string][]byte) bool {
	for signer := range bundleSigners {
		if _, isTxFromSender := transactionSigners[signer]; isTxFromSender {
			return true
		}
	}

	return false
}

func (ad AuctionDecorator) DecodeTx(txBytes []byte) (sdk.Tx, error) {
	return ad.TxDecoder(txBytes)
}
