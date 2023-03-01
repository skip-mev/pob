package mempool_test

import (
	"math/rand"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/stretchr/testify/require"
)

var emptyHash = [32]byte{}

func TestAuctionBidList_Insert(t *testing.T) {
	abl := mempool.NewAuctionBidList()

	require.Nil(t, abl.TopBid())

	// insert a bid which should be the head and tail
	bid1 := sdk.NewCoins(sdk.NewInt64Coin("foo", 100))
	abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid1))
	require.Equal(t, bid1, abl.TopBid().GetBid())

	// insert a bid which should be the new head, where tail is the highest bid
	bid2 := sdk.NewCoins(sdk.NewInt64Coin("foo", 50))
	abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid2))
	require.Equal(t, bid1, abl.TopBid().GetBid())

	// insert a bid which should be the new tail, thus the highest bid
	bid3 := sdk.NewCoins(sdk.NewInt64Coin("foo", 200))
	abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid3))
	require.Equal(t, bid3, abl.TopBid().GetBid())

	// insert 500 random bids between [1, 1000)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 500; i++ {
		randomBid := rng.Int63n(1000-1) + 1

		bid := sdk.NewCoins(sdk.NewInt64Coin("foo", randomBid))
		abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid))
	}

	// insert a bid which should be the new tail, thus the highest bid
	bid4 := sdk.NewCoins(sdk.NewInt64Coin("foo", 1000))
	abl.Insert(mempool.NewWrappedBidTx(nil, emptyHash, bid4))
	require.Equal(t, bid4, abl.TopBid().GetBid())
}
