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
func (abl *AuctionBidList) TopBid() *WrappedBidTx {
	n := abl.list.Back()
	if n == nil {
		return nil
	}

	return n.Value.(*WrappedBidTx)
}

func (abl *AuctionBidList) Insert(wBidTx *WrappedBidTx) {
	// if the list is empty, insert at the front and return
	if abl.list.Len() == 0 {
		abl.list.PushFront(wBidTx)
		return
	}

	// check if the bid should be the head of the list
	head := abl.list.Front().Value.(*WrappedBidTx)
	if head.bid.IsAllGT(wBidTx.bid) {
		abl.list.PushFront(wBidTx)
		return
	}

	// check if the bid should be the tail of the list
	tail := abl.list.Back().Value.(*WrappedBidTx)
	if wBidTx.bid.IsAllGT(tail.bid) {
		abl.list.PushBack(wBidTx)
		return
	}

	// otherwise, insert into the middle of the list in the appropriate spot
	for e := abl.list.Front(); e != nil; e = e.Next() {
		curr := e.Value.(*WrappedBidTx)
		if wBidTx.bid.IsAllLT(curr.bid) {
			abl.list.InsertBefore(wBidTx, e)
			return
		}
	}
}

func (abl *AuctionBidList) Remove(wBidTx *WrappedBidTx) {
	panic("not implemented")
}
