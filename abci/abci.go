package abci

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
)

type Handler struct {
	logger    log.Logger
	mempool   *mempool.AuctionMempool
	txEncoder sdk.TxEncoder
}

func NewHandler(mp *mempool.AuctionMempool, txEncoder sdk.TxEncoder, logger log.Logger) *Handler {
	return &Handler{
		mempool:   mp,
		txEncoder: txEncoder,
	}
}

// PrepareProposalHandler returns the PrepareProposal ABCI handler that performs
// top-of-block auctioning.
func (h *Handler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req abci.RequestPrepareProposal) abci.ResponsePrepareProposal {
		var (
			selectedTxs  [][]byte
			totalTxBytes int64
		)

		txsMap := make(map[string]struct{})

		// attempt to select a bid for top-of-block auction
		bidTx := h.mempool.SelectTopAuctionBidTx()
		if bidTx != nil {
			bidTxBz, err := h.txEncoder(bidTx)
			if err != nil {
				panic(fmt.Errorf("failed to encode bid tx: %w", err))
			}

			txSize := int64(len(bidTxBz))
			if txSize <= req.MaxTxBytes {
				bidTxHash := sha256.Sum256(bidTxBz)
				bidTxHashStr := base64.StdEncoding.EncodeToString(bidTxHash[:])

				bidMsg, ok := bidTx.GetMsgs()[0].(*auctiontypes.MsgAuctionBid)
				if !ok {
					panic(fmt.Errorf("unexpected message type; expected %T, got %T", &auctiontypes.MsgAuctionBid{}, bidTx.GetMsgs()[0]))
				}

				// Mark transaction as selected, update total selected size and add to
				// proposal's selected transactions.
				totalTxBytes += txSize
				txsMap[bidTxHashStr] = struct{}{}
				selectedTxs = append(selectedTxs, bidTxBz)

				h.logger.Debug("selected auction bid tx", "tx", bidTxHashStr, "bid", bidMsg.Bid)

				for _, refTxRaw := range bidMsg.Transactions {
					refTxHash := sha256.Sum256(refTxRaw)
					refTxHashStr := base64.StdEncoding.EncodeToString(refTxHash[:])

					// auto-select referenced transactions and mark them as selected
					//
					// Note: We do not update total selected size as the bid tx is already
					// accounted for.
					txsMap[refTxHashStr] = struct{}{}
					selectedTxs = append(selectedTxs, refTxRaw)
				}
			} else {
				h.logger.Info("failed to select auction bid tx; tx size is too large", "tx_size", txSize, "max_size", req.MaxTxBytes)
			}

			fmt.Println(totalTxBytes)
		}

		// select remaining transactions for the block proposal

		return abci.ResponsePrepareProposal{Txs: selectedTxs}
	}
}

// ProcessProposalHandler returns the ProcessProposal ABCI handler that performs
// block proposal verification.
func (h *Handler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req abci.RequestProcessProposal) abci.ResponseProcessProposal {
		panic("not implemented")
	}
}

// func (h *Handler) getTxHash(tx sdk.Tx) ([32]byte, string, error) {
// 	bz, err := h.txEncoder(tx)
// 	if err != nil {
// 		return [32]byte{}, "", fmt.Errorf("failed to encode tx: %w", err)
// 	}

// 	hash := sha256.Sum256(bz)
// 	hashStr := base64.StdEncoding.EncodeToString(hash[:])

// 	return hash, hashStr, nil
// }
