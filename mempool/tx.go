package mempool

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IsAuctionTx returns true if the transaction is a transaction that is attempting to
// bid to the auction.
func (am *AuctionMempool) IsAuctionTx(tx sdk.Tx) (bool, error) {
	return am.config.isAuctionTx(tx)
}

// GetTransactionSigners returns the signers of the bundle transaction.
func (am *AuctionMempool) GetTransactionSigners(tx []byte) (map[string]bool, error) {
	return am.config.getTxSigners(tx)
}

// WrapBundleTransaction wraps a bundle transaction into sdk.Tx transaction.
func (am *AuctionMempool) WrapBundleTransaction(tx []byte) (sdk.Tx, error) {
	return am.config.wrapBundleTx(tx)
}

// GetBidInfo returns the bid info from an auction transaction.
func (am *AuctionMempool) GetBidInfo(tx sdk.Tx) (BidInfo, error) {
	bidder, err := am.GetBidder(tx)
	if err != nil {
		return BidInfo{}, err
	}

	bid, err := am.GetBid(tx)
	if err != nil {
		return BidInfo{}, err
	}

	transactions, err := am.GetBundledTransactions(tx)
	if err != nil {
		return BidInfo{}, err
	}

	return BidInfo{
		Bidder:       bidder,
		Bid:          bid,
		Transactions: transactions,
	}, nil
}

// GetBidder returns the bidder from an auction transaction.
func (am *AuctionMempool) GetBidder(tx sdk.Tx) (sdk.AccAddress, error) {
	if isAuctionTx, err := am.IsAuctionTx(tx); err != nil || !isAuctionTx {
		return nil, fmt.Errorf("transaction is not an auction transaction")
	}

	return am.config.getBidder(tx)
}

// GetBid returns the bid from an auction transaction.
func (am *AuctionMempool) GetBid(tx sdk.Tx) (sdk.Coin, error) {
	if isAuctionTx, err := am.IsAuctionTx(tx); err != nil || !isAuctionTx {
		return sdk.Coin{}, fmt.Errorf("transaction is not an auction transaction")
	}

	return am.config.getBid(tx)
}

// GetBundledTransactions returns the transactions that are bundled in an auction transaction.
func (am *AuctionMempool) GetBundledTransactions(tx sdk.Tx) ([][]byte, error) {
	if isAuctionTx, err := am.IsAuctionTx(tx); err != nil || !isAuctionTx {
		return nil, fmt.Errorf("transaction is not an auction transaction")
	}

	return am.config.getBundledTxs(tx)
}

// GetBundleSigners returns all of the signers for each transaction in the bundle.
func (am *AuctionMempool) GetBundleSigners(txs [][]byte) ([]map[string]bool, error) {
	signers := make([]map[string]bool, len(txs))

	for index, tx := range txs {
		txSigners, err := am.GetTransactionSigners(tx)
		if err != nil {
			return nil, err
		}

		signers[index] = txSigners
	}

	return signers, nil
}
