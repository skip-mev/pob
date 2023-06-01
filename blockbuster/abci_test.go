package blockbuster_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

// TODO:
// - Test the case where partial proposals do not exceed the amount they should build
// - Test the case where the mempool is empty
// - Test the case where the maxTxBytes is small
func (suite *BlockBusterTestSuite) TestPrepareProposal() {
	var (
		// the modified transactions cannot exceed this size
		maxTxBytes int64 = 1000000000000000000

		// mempool configuration
		normalTxs        []sdk.Tx
		auctionTxs       []sdk.Tx
		winningBidTx     sdk.Tx
		insertBundledTxs = false

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name                        string
		malleate                    func()
		expectedNumberProposalTxs   int
		expectedMempoolDistribution map[string]int
	}{
		{
			"single valid tob transaction in the mempool",
			func() {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			2,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
			},
		},
		{
			"single invalid tob transaction in the mempool",
			func() {
				bidder := suite.accounts[0]
				bid := reserveFee.Sub(sdk.NewCoin("foo", sdk.NewInt(1))) // bid is less than the reserve fee
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = nil
				insertBundledTxs = false
			},
			0,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 0,
			},
		},
		{
			"normal transactions in the mempool",
			func() {
				account := suite.accounts[0]
				nonce := suite.nonces[account.Address.String()]
				timeout := uint64(100)
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{}
				winningBidTx = nil
				insertBundledTxs = false
			},
			1,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 0,
			},
		},
		{
			"normal transactions and tob transactions in the mempool",
			func() {
				// Create a valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid default transaction
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()] + 1
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			3,
			map[string]int{
				base.LaneName:    1,
				auction.LaneName: 1,
			},
		},
		{
			"multiple tob transactions where the first is invalid",
			func() {
				// Create an invalid tob transaction (frontrunning)
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder, bidder, suite.accounts[1]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				winningBidTx = bidTx2
				insertBundledTxs = false
			},
			2,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 1,
			},
		},
		{
			"multiple tob transactions where the first is valid",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				winningBidTx = bidTx
				insertBundledTxs = false
			},
			3,
			map[string]int{
				base.LaneName:    0,
				auction.LaneName: 2,
			},
		},
		{
			"multiple tob transactions where the first is valid and bundle is inserted into mempool",
			func() {
				frontRunningProtection = false

				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], suite.accounts[1], bidder, suite.accounts[3], suite.accounts[4]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				winningBidTx = bidTx
				insertBundledTxs = true
			},
			6,
			map[string]int{
				base.LaneName:    5,
				auction.LaneName: 1,
			},
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			suite.resetLanes()
			tc.malleate()

			// Insert all of the normal transactions into the default lane
			for _, tx := range normalTxs {
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
			}

			// Insert all of the auction transactions into the TOB lane
			for _, tx := range auctionTxs {
				suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
			}

			// Insert all of the bundled transactions into the TOB lane if desired
			if insertBundledTxs {
				for _, tx := range auctionTxs {
					bidInfo, err := suite.auctionFactory.GetAuctionBidInfo(tx)
					suite.Require().NoError(err)

					for _, txBz := range bidInfo.Transactions {
						tx, err := suite.encodingConfig.TxConfig.TxDecoder()(txBz)
						suite.Require().NoError(err)

						suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
					}
				}
			}

			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				FrontRunningProtection: frontRunningProtection,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

			suite.proposalHandler = blockbuster.NewProposalHandler(suite.logger, suite.mempool, suite.encodingConfig.TxConfig.TxEncoder())
			handler := suite.proposalHandler.PrepareProposalHandler()
			res := handler(suite.ctx, abcitypes.RequestPrepareProposal{
				MaxTxBytes: maxTxBytes,
			})

			// -------------------- Check Invariants -------------------- //
			// 1. total bytes must be less than or equal to maxTxBytes
			totalBytes := int64(0)
			for _, tx := range res.Txs {
				totalBytes += int64(len(tx))
			}
			suite.Require().LessOrEqual(totalBytes, maxTxBytes)

			// 2. the number of transactions in the response must be equal to the number of expected transactions
			suite.Require().Equal(tc.expectedNumberProposalTxs, len(res.Txs))

			// 3. if there are auction transactions, the first transaction must be the top bid
			// and the rest of the bundle must be in the response
			if winningBidTx != nil {
				auctionTx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[0])
				suite.Require().NoError(err)

				bidInfo, err := suite.auctionFactory.GetAuctionBidInfo(auctionTx)
				suite.Require().NoError(err)

				for index, tx := range bidInfo.Transactions {
					suite.Require().Equal(tx, res.Txs[index+1])
				}
			} else {
				if len(res.Txs) > 0 {
					tx, err := suite.encodingConfig.TxConfig.TxDecoder()(res.Txs[0])
					suite.Require().NoError(err)

					bidInfo, err := suite.auctionFactory.GetAuctionBidInfo(tx)
					suite.Require().NoError(err)
					suite.Require().Nil(bidInfo)
				}
			}

			// 4. All of the transactions must be unique
			uniqueTxs := make(map[string]bool)
			for _, tx := range res.Txs {
				suite.Require().False(uniqueTxs[string(tx)])
				uniqueTxs[string(tx)] = true
			}

			// 5. The number of transactions in the mempool must be correct
			suite.Require().Equal(tc.expectedMempoolDistribution, suite.mempool.GetTxDistribution())
		})
	}
}

func (suite *BlockBusterTestSuite) TestProcessProposal() {
	var (
		// mempool configuration
		normalTxs    []sdk.Tx
		auctionTxs   []sdk.Tx
		insertRefTxs = false

		// auction configuration
		maxBundleSize          uint32 = 10
		reserveFee                    = sdk.NewCoin("foo", sdk.NewInt(1000))
		frontRunningProtection        = true
	)

	cases := []struct {
		name     string
		malleate func()
		response abcitypes.ResponseProcessProposal_ProposalStatus
	}{
		{
			"no normal tx, no tob tx",
			func() {
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single default tx",
			func() {
				account := suite.accounts[0]
				nonce := suite.nonces[account.Address.String()]
				timeout := uint64(100)
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{normalTx}
				auctionTxs = []sdk.Tx{}
				insertRefTxs = false
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single tob tx without bundled txs in proposal",
			func() {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = false
			},
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"single tob tx with bundled txs in proposal",
			func() {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[1], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
		{
			"single invalid tob tx (front-running)",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], suite.accounts[1], bidder, suite.accounts[3], suite.accounts[4]}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"multiple tob txs in the proposal",
			func() {
				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a valid tob transaction
				bidder = suite.accounts[1]
				bid = sdk.NewCoin("foo", sdk.NewInt(1000))
				nonce = suite.nonces[bidder.Address.String()]
				timeout = uint64(100)
				signers = []testutils.Account{bidder}
				bidTx2, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{}
				auctionTxs = []sdk.Tx{bidTx, bidTx2}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_REJECT,
		},
		{
			"single tob tx with front-running disabled and multiple other txs",
			func() {
				frontRunningProtection = false

				// Create an valid tob transaction
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(10000000))
				nonce := suite.nonces[bidder.Address.String()]
				timeout := uint64(100)
				signers := []testutils.Account{suite.accounts[2], bidder}
				bidTx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, bidder, bid, nonce, timeout, signers)
				suite.Require().NoError(err)

				// Create a few other transactions
				account := suite.accounts[1]
				nonce = suite.nonces[account.Address.String()]
				timeout = uint64(100)
				numberMsgs := uint64(3)
				normalTx, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				account = suite.accounts[3]
				nonce = suite.nonces[account.Address.String()]
				timeout = uint64(100)
				numberMsgs = uint64(3)
				normalTx2, err := testutils.CreateRandomTx(suite.encodingConfig.TxConfig, account, nonce, numberMsgs, timeout)
				suite.Require().NoError(err)

				normalTxs = []sdk.Tx{normalTx, normalTx2}
				auctionTxs = []sdk.Tx{bidTx}
				insertRefTxs = true
			},
			abcitypes.ResponseProcessProposal_ACCEPT,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			tc.malleate()

			// Insert all of the transactions into the proposal
			txs := make([][]byte, 0)
			for _, tx := range auctionTxs {
				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				txs = append(txs, txBz)

				if insertRefTxs {
					bidInfo, err := suite.auctionFactory.GetAuctionBidInfo(tx)
					suite.Require().NoError(err)

					txs = append(txs, bidInfo.Transactions...)
				}
			}

			for _, tx := range normalTxs {
				txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
				suite.Require().NoError(err)

				txs = append(txs, txBz)
			}

			// create a new auction
			params := buildertypes.Params{
				MaxBundleSize:          maxBundleSize,
				ReserveFee:             reserveFee,
				FrontRunningProtection: frontRunningProtection,
			}
			suite.builderKeeper.SetParams(suite.ctx, params)
			suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

			handler := suite.proposalHandler.ProcessProposalHandler()
			res := handler(suite.ctx, abcitypes.RequestProcessProposal{
				Txs: txs,
			})

			// Check if the response is valid
			suite.Require().Equal(tc.response, res.Status)
		})
	}
}
