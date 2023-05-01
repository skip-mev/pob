package mempool_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutils "github.com/skip-mev/pob/testutils"
)

func (suite *IntegrationTestSuite) TestIsAuctionTx() {
	testCases := []struct {
		name          string
		createTx      func() sdk.Tx
		isAuctionTx   bool
		expectedError bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 2, 0)
				suite.Require().NoError(err)
				return tx
			},
			false,
			false,
		},
		{
			"malformed auction bid tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := testutils.CreateRandomMsgs(suite.accounts[0].Address, 2)
				msgs = append(msgs, msgAuctionBid)

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			false,
			true,
		},
		{
			"valid auction bid tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			true,
			false,
		},
		{
			"tx with multiple MsgAuctionBid messages",
			func() sdk.Tx {
				bid1, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				bid2, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 1, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{bid1, bid2}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			false,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := suite.config.GetAuctionBidInfo(tx)

			suite.Require().Equal(tc.isAuctionTx, bidInfo != nil)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetTransactionSigners() {
	testCases := []struct {
		name            string
		createTx        func() []byte
		expectedSigners []string
	}{
		{
			"normal sdk tx",
			func() []byte {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return bz
			},
			[]string{suite.accounts[0].Address.String()},
		},
		{
			"normal sdk tx with several messages",
			func() []byte {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 10, 0)
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return bz
			},
			[]string{suite.accounts[0].Address.String()},
		},
		{
			"multiple signers on tx",
			func() []byte {
				tx, err := testutils.CreateTxWithSigners(suite.encCfg.TxConfig, 0, 0, suite.accounts[0:3])
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return bz
			},
			[]string{suite.accounts[0].Address.String(), suite.accounts[1].Address.String(), suite.accounts[2].Address.String()},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			signers, err := suite.config.GetTransactionSigners(tx)
			suite.Require().NoError(err)
			suite.Require().Equal(len(tc.expectedSigners), len(signers))

			for _, signer := range tc.expectedSigners {
				suite.Require().Contains(signers, signer)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestWrapBundleTransaction() {
	testCases := []struct {
		name           string
		createBundleTx func() (sdk.Tx, []byte)
	}{
		{
			"normal sdk tx",
			func() (sdk.Tx, []byte) {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				bz, err := suite.encCfg.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				return tx, bz
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx, bz := tc.createBundleTx()

			wrappedTx, err := suite.config.WrapBundleTransaction(bz)
			suite.Require().NoError(err)

			txBytes, err := suite.encCfg.TxConfig.TxEncoder()(tx)
			suite.Require().NoError(err)

			wrappedTxBytes, err := suite.encCfg.TxConfig.TxEncoder()(wrappedTx)
			suite.Require().NoError(err)

			suite.Require().Equal(txBytes, wrappedTxBytes)
		})
	}
}

func (suite *IntegrationTestSuite) TestGetBidder() {
	testCases := []struct {
		name           string
		createTx       func() sdk.Tx
		expectedBidder string
		expectedError  bool
		isAuctionTx    bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				return tx
			},
			"",
			false,
			false,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			suite.accounts[0].Address.String(),
			false,
			true,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			"",
			true,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := suite.config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)

				if tc.isAuctionTx {
					suite.Require().Equal(tc.expectedBidder, bidInfo.Bidder.String())
				}
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetBid() {
	testCases := []struct {
		name          string
		createTx      func() sdk.Tx
		expectedBid   sdk.Coin
		expectedError bool
		isAuctionTx   bool
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				return tx
			},
			sdk.Coin{},
			false,
			false,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			sdk.NewInt64Coin("foo", 100),
			false,
			true,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx
			},
			sdk.Coin{},
			true,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := suite.config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)

				if tc.isAuctionTx {
					suite.Require().Equal(tc.expectedBid, bidInfo.Bid)
				}
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetBundledTransactions() {
	testCases := []struct {
		name          string
		createTx      func() (sdk.Tx, [][]byte)
		expectedError bool
		isAuctionTx   bool
	}{
		{
			"normal sdk tx",
			func() (sdk.Tx, [][]byte) {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 0)
				suite.Require().NoError(err)

				return tx, nil
			},
			false,
			false,
		},
		{
			"valid auction tx",
			func() (sdk.Tx, [][]byte) {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx, msgAuctionBid.Transactions
			},
			false,
			true,
		},
		{
			"invalid auction tx",
			func() (sdk.Tx, [][]byte) {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 0, msgs)
				suite.Require().NoError(err)
				return tx, nil
			},
			true,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx, expectedBundledTxs := tc.createTx()

			bidInfo, err := suite.config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)

				if tc.isAuctionTx {
					suite.Require().Equal(expectedBundledTxs, bidInfo.Transactions)
				}
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestGetTimeout() {
	testCases := []struct {
		name            string
		createTx        func() sdk.Tx
		expectedError   bool
		isAuctionTx     bool
		expectedTimeout uint64
	}{
		{
			"normal sdk tx",
			func() sdk.Tx {
				tx, err := testutils.CreateRandomTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 1, 1)
				suite.Require().NoError(err)

				return tx
			},
			false,
			false,
			1,
		},
		{
			"valid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 10, msgs)
				suite.Require().NoError(err)
				return tx
			},
			false,
			true,
			10,
		},
		{
			"invalid auction tx",
			func() sdk.Tx {
				msgAuctionBid, err := testutils.CreateMsgAuctionBid(suite.encCfg.TxConfig, suite.accounts[0], sdk.NewInt64Coin("foo", 100), 0, 2)
				suite.Require().NoError(err)

				randomMsg := testutils.CreateRandomMsgs(suite.accounts[0].Address, 1)[0]
				suite.Require().NoError(err)

				msgs := []sdk.Msg{msgAuctionBid, randomMsg}

				tx, err := testutils.CreateTx(suite.encCfg.TxConfig, suite.accounts[0], 0, 10, msgs)
				suite.Require().NoError(err)
				return tx
			},
			true,
			false,
			10,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tx := tc.createTx()

			bidInfo, err := suite.config.GetAuctionBidInfo(tx)
			if tc.expectedError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)

				if tc.isAuctionTx {
					suite.Require().Equal(tc.expectedTimeout, bidInfo.Timeout)
				}
			}
		})
	}
}
