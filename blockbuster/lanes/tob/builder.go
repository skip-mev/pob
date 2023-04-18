package tob

import (
	"bytes"
	"fmt"

	"cosmossdk.io/api/tendermint/abci"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PrepareProposal returns a list of transactions that are ready to be included at the top of block. The transactions
// are selected from the mempool and are ordered by their bid price. The transactions are also validated and
// the winning bidder has their entire bundle of transactions included in the block proposal.
func (mempool *AuctionLane) PrepareProposal(ctx sdk.Context, req abci.RequestPrepareProposal) ([][]byte, int64) {
	var (
		selectedTxs  [][]byte
		totalTxBytes int64
	)

	iterator := mempool.Select(ctx, nil)
	txsToRemove := make(map[sdk.Tx]struct{}, 0)

selectBidTxLoop:
	for ; iterator != nil; iterator = iterator.Next() {
		cacheCtx, write := ctx.CacheContext()
		tx := iterator.Tx()

		txBz, err := mempool.PrepareProposalVerifyTx(cacheCtx, tx)
		if err != nil {
			txsToRemove[tx] = struct{}{}
			continue selectBidTxLoop
		}

		txSize := int64(len(txBz))
		if txSize <= req.MaxTxBytes {
			bidMsg, err := GetMsgAuctionBidFromTx(tx)
			if err != nil {
				// This should never happen, as CheckTx will ensure only valid bids
				// enter the mempool, but in case it does, we need to remove the
				// transaction from the mempool.
				txsToRemove[tx] = struct{}{}
				continue selectBidTxLoop
			}

			for _, refTxRaw := range bidMsg.Transactions {
				refTx, err := mempool.txDecoder(refTxRaw)
				if err != nil {
					// Malformed bundled transaction, so we remove the bid transaction
					// and try the next top bid.
					txsToRemove[tx] = struct{}{}
					continue selectBidTxLoop
				}

				if _, err := mempool.PrepareProposalVerifyTx(cacheCtx, refTx); err != nil {
					// Invalid bundled transaction, so we remove the bid transaction
					// and try the next top bid.
					txsToRemove[tx] = struct{}{}
					continue selectBidTxLoop
				}
			}

			// At this point, both the bid transaction itself and all the bundled
			// transactions are valid. So we select the bid transaction along with
			// all the bundled transactions. We also mark these transactions as seen and
			// update the total size selected thus far.
			totalTxBytes += txSize
			selectedTxs = append(selectedTxs, txBz)
			selectedTxs = append(selectedTxs, bidMsg.Transactions...)

			// Write the cache context to the original context when we know we have a
			// valid top of block bundle.
			write()

			break selectBidTxLoop
		}

		txsToRemove[tx] = struct{}{}
	}

	// Remove all invalid transactions from the mempool.
	for tx := range txsToRemove {
		mempool.RemoveWithoutRefTx(tx)
	}

	return selectedTxs, totalTxBytes
}

func (mempool *AuctionLane) ProcessProposal(ctx sdk.Context, req abci.RequestProcessProposal) error {
	for index, txBz := range req.Txs {
		tx, err := mempool.ProcessProposalVerifyTx(ctx, txBz)
		if err != nil {
			return fmt.Errorf("invalid transaction in block proposal: %w", err)
		}

		msgAuctionBid, err := GetMsgAuctionBidFromTx(tx)
		if err != nil {
			return nil
		}

		if msgAuctionBid != nil {
			// Only the first transaction can be an auction bid tx
			if index != 0 {
				return fmt.Errorf("the first transaction in the block proposal must be an auction bid transaction")
			}

			// The order of transactions in the block proposal must follow the order of transactions in the bid.
			if len(req.Txs) < len(msgAuctionBid.Transactions)+1 {
				return fmt.Errorf("the transactions in the auction bid must be included in the block proposal")
			}

			for i, refTxRaw := range msgAuctionBid.Transactions {
				if !bytes.Equal(refTxRaw, req.Txs[i+1]) {
					return fmt.Errorf("the transactions in the auction bid must be included in the block proposal")
				}
			}
		}

	}

	return nil
}

// PrepareProposalVerifyTx encodes a transaction and verifies it.
func (mempool *AuctionLane) PrepareProposalVerifyTx(ctx sdk.Context, tx sdk.Tx) ([]byte, error) {
	txBz, err := mempool.txEncoder(tx)
	if err != nil {
		return nil, err
	}

	return txBz, mempool.verifyTx(ctx, tx)
}

// ProcessProposalVerifyTx decodes a transaction and verifies it.
func (mempool *AuctionLane) ProcessProposalVerifyTx(ctx sdk.Context, txBz []byte) (sdk.Tx, error) {
	tx, err := mempool.txDecoder(txBz)
	if err != nil {
		return nil, err
	}

	return tx, mempool.verifyTx(ctx, tx)
}

// verifyTx verifies a transaction.
func (mempool *AuctionLane) verifyTx(ctx sdk.Context, tx sdk.Tx) error {
	// We verify transaction by running them through a predeteremined set of antehandlers
	if _, err := mempool.cfg.AnteHandler(ctx, tx, false); err != nil {
		return err
	}

	if _, err := mempool.cfg.PostHandler(ctx, tx, false, true); err != nil {
		return err
	}

	return nil
}
