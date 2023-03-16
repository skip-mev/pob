package keeper_test

import (
	"math/rand"
	"time"

	"github.com/skip-mev/pob/x/auction/types"
)

func (suite *KeeperTestSuite) TestMsgAuctionBid() {
	rng := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := RandomAccounts(rng, 1)[0]

	testCases := []struct {
		name      string
		msg       *types.MsgAuctionBid
		malleate  func()
		expectErr bool
	}{
		{
			name: "too many bundled transactions",
			msg: &types.MsgAuctionBid{
				Bidder:       bidder.Address.String(),
				Transactions: [][]byte{{0xFF}, {0xFF}, {0xFF}},
			},
			malleate: func() {
				params := types.DefaultParams()
				params.MaxBundleSize = 2
				suite.auctionKeeper.SetParams(suite.ctx, params)
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			_, err := suite.msgServer.AuctionBid(suite.ctx, tc.msg)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
