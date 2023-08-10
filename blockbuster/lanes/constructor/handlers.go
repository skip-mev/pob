package constructor

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
)

func (l *LaneConstructor[C]) DefaultPrepareLaneHandler() blockbuster.PrepareLaneHandler {
	return func(ctx sdk.Context, proposal blockbuster.BlockProposal, maxTxBytes int64) ([][]byte, []sdk.Tx, error) {
		var (
			totalSize   int64
			txs         [][]byte
			txsToRemove []sdk.Tx
		)

		// Select all transactions in the mempool that are valid and not already in the
		// partial proposal.
		for iterator := l.Select(ctx, nil); iterator != nil; iterator = iterator.Next() {
			tx := iterator.Tx()

			txBytes, hash, err := utils.GetTxHashStr(l.Cfg.TxEncoder, tx)
			if err != nil {
				l.Logger().Info("failed to get hash of tx", "err", err)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			// if the transaction is already in the (partial) block proposal, we skip it.
			if proposal.Contains(txBytes) {
				l.Logger().Info(
					"failed to select tx for lane; tx is already in proposal",
					"tx_hash", hash,
					"lane", l.Name(),
				)

				continue
			}

			// If the transaction is too large, we break and do not attempt to include more txs.
			txSize := int64(len(txBytes))
			if updatedSize := totalSize + txSize; updatedSize > maxTxBytes {
				l.Logger().Info("maximum tx bytes reached", "lane", l.Name())
				break
			}

			// Verify the transaction.
			if err := l.VerifyTx(ctx, tx); err != nil {
				l.Logger().Info(
					"failed to verify tx",
					"tx_hash", hash,
					"err", err,
				)

				txsToRemove = append(txsToRemove, tx)
				continue
			}

			totalSize += txSize
			txs = append(txs, txBytes)
		}

		return txs, txsToRemove, nil
	}
}

func (l *LaneConstructor[C]) DefaultProcessLaneHandler() blockbuster.ProcessLaneHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) ([]sdk.Tx, error) {
		for index, tx := range txs {
			if l.Match(ctx, tx) {
				if err := l.VerifyTx(ctx, tx); err != nil {
					return nil, fmt.Errorf("failed to verify tx: %w", err)
				}
			} else {
				return txs[index:], nil
			}
		}

		// This means we have processed all transactions in the proposal.
		return nil, nil
	}
}

func (l *LaneConstructor[C]) DefaultProcessLaneBasicHandler() blockbuster.ProcessLaneBasicHandler {
	return func(ctx sdk.Context, txs []sdk.Tx) error {
		seenOtherLaneTx := false

		for index, tx := range txs {
			if l.Match(ctx, tx) {
				if seenOtherLaneTx {
					return fmt.Errorf("the %s lane contains a transaction that belongs to another lane", l.Name())
				}

				// If the transactions do not respect the priority defined by the mempool, we consider the proposal
				// to be invalid
				if index > 0 && l.Compare(ctx, txs[index-1], tx) == -1 {
					return fmt.Errorf("transaction at index %d has a higher priority than %d", index, index-1)
				}
			} else {
				seenOtherLaneTx = true
			}
		}

		return nil
	}
}

func DefaultMatchHandler() blockbuster.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		return true
	}
}

// TxPriority returns a TxPriority over the base lane transactions. It prioritizes
// transactions by their fee.
func DefaultTxPriority() blockbuster.TxPriority[string] {
	return blockbuster.TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			feeTx, ok := tx.(sdk.FeeTx)
			if !ok {
				return ""
			}

			return feeTx.GetFee().String()
		},
		Compare: func(a, b string) int {
			aCoins, _ := sdk.ParseCoinsNormalized(a)
			bCoins, _ := sdk.ParseCoinsNormalized(b)

			switch {
			case aCoins == nil && bCoins == nil:
				return 0

			case aCoins == nil:
				return -1

			case bCoins == nil:
				return 1

			default:
				switch {
				case aCoins.IsAllGT(bCoins):
					return 1

				case aCoins.IsAllLT(bCoins):
					return -1

				default:
					return 0
				}
			}
		},
		MinValue: "",
	}
}
