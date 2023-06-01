package blockbuster

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

func GetTxHashStr(txEncoder sdk.TxEncoder, tx sdk.Tx) (string, error) {
	txBz, err := txEncoder(tx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}

	txHash := sha256.Sum256(txBz)
	txHashStr := hex.EncodeToString(txHash[:])

	return txHashStr, nil
}

func RemoveTxsFromLane(txs map[sdk.Tx]struct{}, mempool sdkmempool.Mempool) error {
	for tx := range txs {
		if err := mempool.Remove(tx); err != nil {
			return err
		}
	}

	return nil
}

func GetMaxTxBytesForLane(proposal Proposal, ratio sdk.Dec) int64 {
	// In the case where the ratio is zero, we return the max tx bytes. Note, the only
	// lane that should have a ratio of zero is the base lane. This means the base lane
	// will have no limit on the number of transactions it can include in a block and is only
	// limited by the maxTxBytes included in the PrepareProposalRequest.
	if ratio.IsZero() {
		remainder := proposal.MaxTxBytes - proposal.TotalTxBytes
		if remainder < 0 {
			return 0
		}

		return remainder
	}

	return int64(ratio.MulInt64(proposal.MaxTxBytes).TruncateInt().Int64())
}

func UpdateProposal(proposal Proposal, txs [][]byte, txSize int64) Proposal {
	proposal.Txs = append(proposal.Txs, txs...)
	proposal.TotalTxBytes += txSize

	for _, tx := range txs {
		txHash := sha256.Sum256(tx)
		txHashStr := hex.EncodeToString(txHash[:])
		proposal.SelectedTxs[txHashStr] = struct{}{}
	}

	return proposal
}
