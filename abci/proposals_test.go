package abci_test

import (
	"bytes"

	comettypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/abci"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

func (suite *ABCITestSuite) TestPrepareProposal() {
	var (
		// the modified transactions cannot exceed this size
		maxTxBytes int64 = 1000000000000000000

		// mempool configuration
		numNormalTxs                = 100
		numAuctionTxs               = 100
		numBundledTxs               = 3
		insertRefTxs                = false
		expectedTopAuctionTx sdk.Tx = nil

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		minBuyInFee                   = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name                              string
		malleate                          func()
		expectedNumberProposalTxs         int
		expectedNumberTxsInMempool        int
		expectedNumberTxsInAuctionMempool int
	}{
		{
			"single bundle in the mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			5,
			3,
			1,
		},
		{
			"single bundle in the mempool, no ref txs in mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			5,
			0,
			1,
		},
		{
			"single bundle in the mempool, not valid",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(100000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(10000)) // this will fail the ante handler
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			1,
			0,
			0,
		},
		{
			"single bundle in the mempool, not valid with ref txs in mempool",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(100000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(10000)) // this will fail the ante handler
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			4,
			3,
			0,
		},
		{
			"multiple bundles in the mempool, no normal txs + no ref txs in mempool",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(10000000))
				numNormalTxs = 0
				numAuctionTxs = 10
				numBundledTxs = 3
				insertRefTxs = false

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			5,
			0,
			10,
		},
		{
			"multiple bundles in the mempool, normal txs + ref txs in mempool",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 10
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			32,
			30,
			10,
		},
		{
			"normal txs only",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 0
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			2,
			1,
			0,
		},
		{
			"many normal txs only",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 0
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			101,
			100,
			0,
		},
		{
			"single normal tx, single auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			3,
			1,
			1,
		},
		{
			"single normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			6,
			4,
			1,
		},
		{
			"single normal tx, single failing auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 3
				insertRefTxs = true
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(2000)) // this will fail the ante handler
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000000000))

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			5,
			4,
			0,
		},
		{
			"many normal tx, single auction tx with no ref txs",
			func() {
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(2000000))
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 0

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = nil
			},
			102,
			100,
			1,
		},
		{
			"many normal tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 100
				numBundledTxs = 3
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)
			},
			402,
			400,
			100,
		},
		{
			"many normal tx, many auction tx with ref txs but top bid is invalid",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 100
				numBundledTxs = 1
				insertRefTxs = true

				suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

				expectedTopAuctionTx = suite.mempool.GetTopAuctionTx(suite.ctx)

				// create a new bid that is greater than the current top bid
				bid := sdk.NewCoin("foo", sdk.NewInt(200000000000000000))
				bidTx, err := testutils.CreateAuctionTxWithSigners(
					suite.encodingConfig.TxConfig,
					suite.accounts[0],
					bid,
					0,
					0,
					[]testutils.Account{suite.accounts[0], suite.accounts[1]},
				)
				suite.Require().NoError(err)

				// add the new bid to the mempool
				err = suite.mempool.Insert(suite.ctx, bidTx)
				suite.Require().NoError(err)

				suite.Require().Equal(suite.mempool.CountAuctionTx(), 101)
			},
			202,
			200,
			100,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			tc.malleate()

			// Create a new auction.
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBuyInFee:            minBuyInFee,
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        suite.minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)

			// Reset the proposal handler with the new mempool.
			suite.proposalHandler = abci.NewProposalHandler(suite.mempool, suite.logger, suite.anteHandler, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())

			// Create a prepare proposal request based on the current state of the mempool.
			handler := suite.proposalHandler.PrepareProposalHandler()
			req := suite.createPrepareProposalRequest(maxTxBytes)
			res := handler(suite.ctx, req)

			// -------------------- Check Invariants -------------------- //
			// The first slot in the proposal must be the auction info
			auctionInfo := abci.AuctionInfo{}
			err := auctionInfo.Unmarshal(res.Txs[abci.AuctionInfoIndex])
			suite.Require().NoError(err)

			// Total bytes must be less than or equal to maxTxBytes
			totalBytes := int64(0)
			for _, tx := range res.Txs[abci.MinProposalSize:] {
				totalBytes += int64(len(tx))
			}
			suite.Require().LessOrEqual(totalBytes, maxTxBytes)

			// The number of transactions in the response must be equal to the number of expected transactions
			suite.Require().Equal(tc.expectedNumberProposalTxs, len(res.Txs))

			// If there are auction transactions, the first transaction must be the top bid
			// and the rest of the bundle must be in the response
			if expectedTopAuctionTx != nil {
				auctionTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[1])
				suite.Require().NoError(err)

				bidInfo, err := suite.mempool.GetAuctionBidInfo(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range bidInfo.Transactions {
					suite.Require().Equal(tx, res.Txs[abci.MinProposalSize+index+1])
				}
			}

			// 5. All of the transactions must be unique
			uniqueTxs := make(map[string]bool)
			for _, tx := range res.Txs[abci.MinProposalSize:] {
				suite.Require().False(uniqueTxs[string(tx)])
				uniqueTxs[string(tx)] = true
			}

			// 6. The number of transactions in the mempool must be correct
			suite.Require().Equal(tc.expectedNumberTxsInMempool, suite.mempool.CountTx())
			suite.Require().Equal(tc.expectedNumberTxsInAuctionMempool, suite.mempool.CountAuctionTx())
		})
	}
}

func (suite *ABCITestSuite) TestProcessProposal() {
	var (
		// mempool set up
		numNormalTxs   = 100
		numAuctionTxs  = 1
		numBundledTxs  = 3
		insertRefTxs   = true
		exportRefTxs   = true
		frontRunningTx sdk.Tx

		// auction set up
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		minBuyInFee                   = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name          string
		malleate      func()
		isTopBidValid bool
		response      comettypes.ResponseProcessProposal_ProposalStatus
	}{
		{
			"single normal tx, no auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 0
				numBundledTxs = 0
			},
			false,
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, no normal txs",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			true,
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, single auction tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 0
			},
			true,
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, single auction tx with ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 4
			},
			true,
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single auction tx, single auction tx with no ref txs",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 1
				numBundledTxs = 4
				insertRefTxs = false
				exportRefTxs = false
			},
			true,
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple auction txs, single normal tx",
			func() {
				numNormalTxs = 1
				numAuctionTxs = 2
				numBundledTxs = 4
				insertRefTxs = true
				exportRefTxs = true
			},
			true,
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single auction txs, multiple normal tx",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
			},
			true,
			comettypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single invalid auction tx, multiple normal tx",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(100000000000000000))
				insertRefTxs = true
			},
			false,
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single valid auction txs but missing ref txs",
			func() {
				numNormalTxs = 0
				numAuctionTxs = 1
				numBundledTxs = 4
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				insertRefTxs = false
				exportRefTxs = false
			},
			true,
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"single valid auction txs but missing ref txs, with many normal txs",
			func() {
				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
				reserveFee = sdk.NewCoin("foo", sdk.NewInt(1000))
				insertRefTxs = false
				exportRefTxs = false
			},
			true,
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"auction tx with frontrunning",
			func() {
				randomAccount := testutils.RandomAccounts(suite.random, 1)[0]
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(696969696969))
				nonce := suite.nonces[bidder.Address.String()]
				frontRunningTx, _ = testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, suite.accounts[0], bid, nonce+1, 1000, []testutils.Account{bidder, randomAccount})
				suite.Require().NotNil(frontRunningTx)

				numNormalTxs = 100
				numAuctionTxs = 1
				numBundledTxs = 4
				insertRefTxs = true
				exportRefTxs = true
			},
			false,
			comettypes.ResponseProcessProposal_REJECT,
		},
		{
			"auction tx with frontrunning, but frontrunning protection disabled",
			func() {
				randomAccount := testutils.RandomAccounts(suite.random, 1)[0]
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(696969696969))
				nonce := suite.nonces[bidder.Address.String()]
				frontRunningTx, _ = testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, suite.accounts[0], bid, nonce+1, 1000, []testutils.Account{bidder, randomAccount})
				suite.Require().NotNil(frontRunningTx)

				numAuctionTxs = 0
				frontRunningProtection = false
			},
			true,
			comettypes.ResponseProcessProposal_ACCEPT,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()

			suite.createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs, insertRefTxs)

			// reset the proposal handler with the new mempool
			suite.proposalHandler = abci.NewProposalHandler(suite.mempool, suite.logger, suite.anteHandler, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())

			if frontRunningTx != nil {
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, frontRunningTx))
			}

			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				MinBuyInFee:            minBuyInFee,
				FrontRunningProtection: frontRunningProtection,
				MinBidIncrement:        suite.minBidIncrement,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.mempool)
			suite.Require().Equal(tc.isTopBidValid, suite.isTopBidValid())

			txs := suite.exportMempool(exportRefTxs)

			if frontRunningTx != nil {
				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(frontRunningTx)
				suite.Require().NoError(err)

				suite.Require().True(bytes.Equal(txs[0], txBz))
			}

			handler := suite.proposalHandler.ProcessProposalHandler()
			res := handler(suite.ctx, comettypes.RequestProcessProposal{
				Txs: txs,
			})

			// Check if the response is valid
			suite.Require().Equal(tc.response, res.Status)
		})
	}
}

// isTopBidValid returns true if the top bid is valid. We purposefully insert invalid
// auction transactions into the mempool to test the handlers.
func (suite *ABCITestSuite) isTopBidValid() bool {
	iterator := suite.mempool.AuctionBidSelect(suite.ctx)
	if iterator == nil {
		return false
	}

	// check if the top bid is valid
	_, err := suite.anteHandler(suite.ctx, iterator.Tx(), true)
	return err == nil
}

func (suite *ABCITestSuite) createPrepareProposalRequest(maxBytes int64) comettypes.RequestPrepareProposal {
	voteExtensions := make([]comettypes.ExtendedVoteInfo, 0)

	auctionIterator := suite.mempool.AuctionBidSelect(suite.ctx)
	for ; auctionIterator != nil; auctionIterator = auctionIterator.Next() {
		tx := auctionIterator.Tx()

		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
		suite.Require().NoError(err)

		suite.createVoteExtension(txBz)

		voteExtensions = append(voteExtensions, comettypes.ExtendedVoteInfo{
			VoteExtension: suite.createVoteExtension(txBz),
		})
	}

	return comettypes.RequestPrepareProposal{
		MaxTxBytes: maxBytes,
		LocalLastCommit: comettypes.ExtendedCommitInfo{
			Votes: voteExtensions,
		},
	}
}

func (suite *ABCITestSuite) createAuctionInfo(bidTxs [][]byte) abci.AuctionInfo {
	size := int(0)
	for _, tx := range bidTxs {
		size += len(tx)
	}

	return abci.AuctionInfo{
		NumTxs:         uint64(len(bidTxs)),
		Size:           uint64(size),
		VoteExtensions: bidTxs,
	}
}
