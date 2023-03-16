package keeper_test

import (
	"math/rand"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
			name: "invalid bidder address",
			msg: &types.MsgAuctionBid{
				Bidder: "foo",
			},
			malleate:  func() {},
			expectErr: true,
		},
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
		{
			name: "valid bundle with no proposer fee",
			msg: &types.MsgAuctionBid{
				Bidder:       bidder.Address.String(),
				Bid:          sdk.NewCoins(sdk.NewInt64Coin("foo", 1024)),
				Transactions: [][]byte{{0xFF}, {0xFF}},
			},
			malleate: func() {
				params := types.DefaultParams()
				params.ProposerFee = sdk.ZeroDec()
				suite.auctionKeeper.SetParams(suite.ctx, params)

				suite.bankKeeper.EXPECT().SendCoinsFromAccountToModule(
					suite.ctx,
					bidder.Address,
					types.ModuleName,
					sdk.NewCoins(sdk.NewInt64Coin("foo", 1024)),
				).Return(nil).AnyTimes()
			},
			expectErr: false,
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
