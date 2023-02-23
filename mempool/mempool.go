package mempool

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

var _ sdkmempool.Mempool = (*AuctionMempool)(nil)

type AuctionMempool struct {
}

func (am *AuctionMempool) Insert(context.Context, sdk.Tx) error {
	panic("not implemented")
}

func (am *AuctionMempool) Select(context.Context, [][]byte) sdkmempool.Iterator {
	panic("not implemented")
}

func (am *AuctionMempool) CountTx() int {
	panic("not implemented")
}

func (am *AuctionMempool) Remove(sdk.Tx) error {
	panic("not implemented")
}
