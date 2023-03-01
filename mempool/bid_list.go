package mempool

import (
	"container/list"
)

type AuctionBidList struct {
	list *list.List
}

func NewAuctionBidList() *AuctionBidList {
	return &AuctionBidList{
		list: list.New(),
	}
}

// TopBid returns the WrappedBidTx with the highest bid.
func (a *AuctionBidList) TopBid() *WrappedBidTx {
	n := a.list.Back()
	if n == nil {
		return nil
	}

	return n.Value.(*WrappedBidTx)
}

func (a *AuctionBidList) Add(wBidTx *WrappedBidTx) {
	panic("not implemented")
}

func (a *AuctionBidList) Remove(wBidTx *WrappedBidTx) {
	panic("not implemented")
}
