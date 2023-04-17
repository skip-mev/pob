package blockbuster

import (
	"context"

	"cosmossdk.io/api/tendermint/abci"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ sdkmempool.Mempool = (*BlockBuster)(nil)

type BlockBuster struct {
	lanes []LaneInterface
}

// NewBlockBuster creates a new instance of BlockBuster.
func NewBlockBuster(lanes ...LaneInterface) *BlockBuster {
	return &BlockBuster{
		lanes: lanes,
	}
}

func NewDefaultBlockBuster() *BlockBuster {
	return &BlockBuster{}
}

func (block *BlockBuster) Insert(ctx context.Context, tx sdk.Tx) error {
	for _, lane := range block.lanes {
		if lane.Match(tx) {
			if err := lane.Insert(ctx, tx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (block *BlockBuster) Remove(tx sdk.Tx) error {
	for _, lane := range block.lanes {
		contains, err := lane.Contains(tx)
		if err != nil {
			return err
		}

		if contains {
			return lane.Remove(tx)
		}
	}

	return nil
}

func (block *BlockBuster) Select(context.Context, [][]byte) sdkmempool.Iterator {
	return nil
}

func (block *BlockBuster) CountTx() int {
	totalTx := 0
	for _, lane := range block.lanes {
		totalTx += lane.CountTx()
	}

	return totalTx
}

func (block *BlockBuster) PrepareProposal(ctx sdk.Context, req abci.RequestPrepareProposal) [][]byte {
	blockProposal := make([][]byte, 0)
	for _, lane := range block.lanes {
		txs, numBytes := lane.PrepareProposal(ctx, req)

		req.MaxTxBytes -= numBytes
		blockProposal = append(blockProposal, txs...)
	}
	return blockProposal
}
