package abci_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/abci"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/types"
)

func (suite *ABCITestSuite) TestBuildTOBProposal() {
	params := types.Params{
		ReserveFee:             sdk.NewCoin("foo", sdk.NewInt(100)),
		MaxBundleSize:          5,
		MinBidIncrement:        sdk.NewCoin("foo", sdk.NewInt(100)),
		MinBuyInFee:            sdk.NewCoin("foo", sdk.NewInt(100)),
		FrontRunningProtection: true,
	}

	err := suite.builderKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	testCases := []struct {
		name             string
		expectedProposal func() ([][]byte, [][]byte) // returns (proposal, vote extensions)
	}{
		{
			"no vote extensions",
			func() ([][]byte, [][]byte) {
				proposal := [][]byte{
					nil,
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte(abci.VoteExtensionsDelimeter),
				}

				return proposal, nil
			},
		},
		{
			"single vote extension",
			func() ([][]byte, [][]byte) {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(100))
				bidBz, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					bidder,
					bid,
					0,
					1,
					[]testutils.Account{bidder},
				)
				suite.Require().NoError(err)

				proposal := [][]byte{
					bidBz,
					[]byte(abci.TopAuctionTxDelimeter),
					bidBz,
					[]byte(abci.VoteExtensionsDelimeter),
				}

				return proposal, [][]byte{
					bidBz,
				}
			},
		},
		{
			/// TODO: ABCI testing has to be bumpde to have several accounts testing
			"single vote extension that is front-running",
			func() ([][]byte, [][]byte) {
				bidder := suite.accounts[0]
				bid := sdk.NewCoin("foo", sdk.NewInt(100))
				bidBz, err := testutils.CreateAuctionTxWithSignerBz(
					suite.encodingConfig.TxConfig,
					bidder,
					bid,
					0,
					1,
					[]testutils.Account{bidder, suite.accounts[1]}, //front-running here
				)
				suite.Require().NoError(err)

				proposal := [][]byte{
					nil,
					[]byte(abci.TopAuctionTxDelimeter),
					bidBz,
					[]byte(abci.VoteExtensionsDelimeter),
				}

				return proposal, [][]byte{
					bidBz,
				}
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			expectedProposal, voteExtensions := tc.expectedProposal()

			// Build the proposal
			_, proposal := suite.proposalHandler.BuildTOBProposal(suite.ctx, voteExtensions)

			// Check invarients
			suite.Require().Equal(expectedProposal, proposal)
		})
	}
}

func (suite *ABCITestSuite) TestVerifyAuction() {
	panic("TODO")
}

func (suite *ABCITestSuite) TestUnwrapProposal() {
	testCases := []struct {
		name          string
		proposalSetup func() (abci.UnwrappedProposal, [][]byte)
		expectedErr   bool
	}{
		{
			"valid proposal",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					[]byte("top auction tx"),
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte("vote extension 1"),
					[]byte(abci.VoteExtensionsDelimeter),
					[]byte("tx 1"),
					[]byte("tx 2"),
				}

				return abci.UnwrappedProposal{
					TopAuctionTx:   []byte("top auction tx"),
					VoteExtensions: [][]byte{[]byte("vote extension 1")},
					Txs:            [][]byte{[]byte("tx 1"), []byte("tx 2")},
				}, proposal
			},
			false,
		},
		{
			"empty proposal",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					nil,
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte(abci.VoteExtensionsDelimeter),
				}

				return abci.UnwrappedProposal{
					TopAuctionTx:   nil,
					VoteExtensions: [][]byte{},
					Txs:            [][]byte{},
				}, proposal
			},
			false,
		},
		{
			"invalid proposal format",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					[]byte("top auction tx"),
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte("tx 1"),
					[]byte("tx 2"),
				}

				return abci.UnwrappedProposal{}, proposal
			},
			true,
		},
		{
			"invalid proposal format. top auction tx slot is missing",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte("vote extension 1"),
					[]byte(abci.VoteExtensionsDelimeter),
					[]byte("tx 1"),
					[]byte("tx 2"),
				}

				return abci.UnwrappedProposal{}, proposal
			},
			true,
		},
		{
			"invalid proposal format. missing auction and vote extension slots",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					[]byte("tx 1"),
					[]byte("tx 2"),
				}

				return abci.UnwrappedProposal{}, proposal
			},
			true,
		},
		{
			"completely empty proposal",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{}

				return abci.UnwrappedProposal{}, proposal
			},
			true,
		},
		{
			"valid proposal with several vote extensions",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					[]byte("top auction tx"),
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte("vote extension 1"),
					[]byte("vote extension 2"),
					[]byte(abci.VoteExtensionsDelimeter),
				}

				return abci.UnwrappedProposal{
					TopAuctionTx:   []byte("top auction tx"),
					VoteExtensions: [][]byte{[]byte("vote extension 1"), []byte("vote extension 2")},
					Txs:            [][]byte{},
				}, proposal
			},
			false,
		},
		{
			"invalid proposal with several auction transaction before the delimeter",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					[]byte("top auction tx 1"),
					[]byte("top auction tx 2"),
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte("vote extension 1"),
					[]byte(abci.VoteExtensionsDelimeter),
				}

				return abci.UnwrappedProposal{}, proposal
			},
			true,
		},
		{
			"no vote extensions",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					[]byte("top auction tx"),
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte(abci.VoteExtensionsDelimeter),
				}

				return abci.UnwrappedProposal{
					TopAuctionTx:   []byte("top auction tx"),
					VoteExtensions: [][]byte{},
					Txs:            [][]byte{},
				}, proposal
			},
			false,
		},
		{
			"no vote extensions and no auction transaction",
			func() (abci.UnwrappedProposal, [][]byte) {
				proposal := [][]byte{
					nil,
					[]byte(abci.TopAuctionTxDelimeter),
					[]byte(abci.VoteExtensionsDelimeter),
					[]byte("tx 1"),
					[]byte("tx 2"),
					[]byte("tx 3"),
					[]byte("tx 4"),
				}

				return abci.UnwrappedProposal{
					TopAuctionTx:   nil,
					VoteExtensions: [][]byte{},
					Txs:            [][]byte{[]byte("tx 1"), []byte("tx 2"), []byte("tx 3"), []byte("tx 4")},
				}, proposal
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			expectedProposal, proposal := tc.proposalSetup()

			unwrappedProposal, err := abci.UnwrapProposal(proposal)
			if tc.expectedErr {
				suite.Require().Error(err)
				return
			}

			suite.Require().NoError(err)
			suite.Require().Equal(expectedProposal.TopAuctionTx, unwrappedProposal.TopAuctionTx)
			suite.Require().Equal(expectedProposal.VoteExtensions, unwrappedProposal.VoteExtensions)
			suite.Require().Equal(expectedProposal.Txs, unwrappedProposal.Txs)
		})
	}
}
