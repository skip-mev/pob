package abci_test

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/ante"
	"github.com/skip-mev/pob/x/auction/types"
)

func (suite *IntegrationTestSuite) TestPrepareProposal() {
	var (
		// the modified transactions cannot exceed this size
		maxTxBytes int64 = 100000000000000000

		// mempool configuration
		numNormalTxs  = 100
		numAuctionTxs = 100
		numBundledTxs = 3
		insertRefTxs  = false

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		minBuyInFee                   = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		frontRunningProtection        = true
	)

	cases := []struct {
		name          string
		malleate      func()
		txs           int
		isTopBidValid bool
	}{
		{
			"single bundle in the mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true
			},
			4,
			true,
		},
		{
			"single bundle in the mempool, no ref txs in mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = false
			},
			4,
			true,
		},
		{
			"single bundle in the mempool, not valid",
			func() {
				suite.maxBidAmount = 1
				numAuctionTxs = 1
				numBundledTxs = 3
			},
			0,
			false,
		},
		{
			"multiple bundles in the mempool, no normal txs",
			func() {
				suite.maxBidAmount = 10000000
				numNormalTxs = 0
				numAuctionTxs = 100
				numBundledTxs = 3
				insertRefTxs = false
			},
			103,
			true,
		},
		{
			"normal txs only",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 0
				numBundledTxs = 0
			},
			1,
			false,
		},
		{
			"many normal txs only",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 0
				numBundledTxs = 0
			},
			100,
			false,
		},
		{
			"single normal tx, single auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			2,
			true,
		},
		{
			"single normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 3
			},
			5,
			true,
		},
		{
			"many normal tx, single auction tx with no ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			101,
			true,
		},
		{
			"many normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 3
			},
			104,
			true,
		},
		{
			"many normal tx, many auction tx with ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 100
				numBundledTxs = 1
			},
			201,
			true,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()

			suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

			// create a new auction
			// TODO: add the min bid increment here
			params := types.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBuyInFee:            minBuyInFee,
				FrontRunningProtection: frontRunningProtection,
			}
			suite.auctionKeeper.SetParams(suite.ctx, params)
			suite.auctionDecorator = ante.NewAuctionDecorator(suite.auctionKeeper, suite.encodingConfig.TxConfig.TxDecoder())

			handler := suite.proposalHandler.PrepareProposalHandler()
			res := handler(suite.ctx, abci.RequestPrepareProposal{
				MaxTxBytes: maxTxBytes,
			})

			// -------------------- Check Invariants -------------------- //
			// 1. The auction tx must fail if we know it is invalid
			suite.Require().Equal(tc.isTopBidValid, suite.isTopBidValid())

			// 2. total bytes must be less than or equal to maxTxBytes
			totalBytes := int64(0)
			if suite.isTopBidValid() {
				totalBytes += int64(len(res.Txs[0]))

				for _, tx := range res.Txs[1+numBundledTxs:] {
					totalBytes += int64(len(tx))
				}
			} else {
				for _, tx := range res.Txs {
					totalBytes += int64(len(tx))
				}
			}
			suite.Require().LessOrEqual(totalBytes, maxTxBytes)

			// 3. the number of transactions in the response must be equal to the number of transactions
			suite.Require().Equal(tc.txs, len(res.Txs))

			// 4. if there are auction transactions, the first transaction must be the top bid
			// and the rest of the bundle must be in the response
			if suite.isTopBidValid() {
				auctionTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[0])
				suite.Require().NoError(err)

				msgAuctionBid, err := mempool.GetMsgAuctionBidFromTx(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range msgAuctionBid.GetTransactions() {
					suite.Require().Equal(tx, res.Txs[index+1])
				}
			}
		})
	}
}

// isTopBidValid returns true if the top bid is valid. We purposefully insert invalid
// auction transactions into the mempool to test the handlers.
func (suite *IntegrationTestSuite) isTopBidValid() bool {
	iterator := suite.mempool.AuctionBidSelect(suite.ctx)
	if iterator == nil {
		return false
	}

	// check if the top bid is valid
	_, err := suite.executeAnteHandler(iterator.Tx().(*mempool.WrappedBidTx).Tx)
	return err == nil
}
