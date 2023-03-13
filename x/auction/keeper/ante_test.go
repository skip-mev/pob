package keeper_test

import (
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/pob/x/auction/keeper"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
)

func (suite *IntegrationTestSuite) TestValidateAuctionMsg() {
	var (
		// Tx building variables
		accounts []Account = []Account{} // tracks the order of signers in the bundle
		balance  sdk.Coins = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(10000)))
		bid      sdk.Coins = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(2000)))

		// Auction params
		maxBundleSize uint32         = 10
		reserveFee    sdk.Coins      = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		minBuyInFee   sdk.Coins      = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))
		escrowAddress sdk.AccAddress = sdk.AccAddress([]byte("escrow"))
	)

	rnd := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := RandomAccounts(rnd, 1)[0]

	cases := []struct {
		name     string
		malleate func()
		pass     bool
	}{
		{
			"insufficient bid amount",
			func() {
				bid = sdk.NewCoins()
			},
			false,
		},
		{
			"insufficient balance",
			func() {
				balance = sdk.NewCoins()
			},
			false,
		},
		{
			"just enough to enter the auction",
			func() {
				balance = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(2000)))
				bid = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(2000)))
			},
			true,
		},
		{
			"too many transactions in the bundle",
			func() {
				accounts = RandomAccounts(rnd, int(maxBundleSize+1))
			},
			false,
		},
		{
			"frontrunning bundle",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{bidder, randomAccount}
			},
			false,
		},
		{
			"sandwiching bundle",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{bidder, randomAccount, bidder}
			},
			false,
		},
		{
			"valid bundle",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{randomAccount, randomAccount, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid bundle with only bidder txs",
			func() {
				accounts = []Account{bidder, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid bundle with only random txs from single same user",
			func() {
				randomAccount := RandomAccounts(rnd, 1)[0]
				accounts = []Account{randomAccount, randomAccount, randomAccount, randomAccount}
			},
			true,
		},
		{
			"invalid bundle with random accounts",
			func() {
				accounts = RandomAccounts(rnd, 2)
			},
			false,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			tc.malleate()

			// Set up the new auction keeper with mocks customized for this test case
			suite.bankKeeper.EXPECT().GetAllBalances(suite.ctx, bidder.Address).Return(balance).AnyTimes()
			suite.bankKeeper.EXPECT().SendCoins(suite.ctx, bidder.Address, escrowAddress, reserveFee).Return(nil).AnyTimes()

			suite.auctionKeeper = keeper.NewKeeper(
				suite.encCfg.Codec,
				suite.key,
				suite.accountKeeper,
				suite.bankKeeper,
				suite.authorityAccount.String(),
				suite.encCfg.TxConfig.TxDecoder(),
			)
			params := auctiontypes.Params{
				MaxBundleSize:        maxBundleSize,
				ReserveFee:           reserveFee,
				MinBuyInFee:          minBuyInFee,
				EscrowAccountAddress: escrowAddress.String(),
			}
			suite.auctionKeeper.SetParams(suite.ctx, params)

			// Create the bundle of transactions ordered by accounts
			bundle := make([]sdk.Tx, 0)
			for _, acc := range accounts {
				tx, err := createRandomTx(suite.encCfg.TxConfig, acc, 0, 1)
				suite.Require().NoError(err)
				bundle = append(bundle, tx)
			}

			// Create the auction msg
			auctionMsg, err := createMsgAuctionBid(suite.encCfg.TxConfig, bidder, bid, bundle)
			suite.Require().NoError(err)

			err = suite.auctionKeeper.ValidateAuctionMsg(suite.ctx, auctionMsg)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *IntegrationTestSuite) TestValidateBundle() {
	var (
		// TODO: Update this to be multi-dimensional to test multi-sig
		// https://github.com/skip-mev/pob/issues/14
		accounts []Account // tracks the order of signers in the bundle
	)

	rng := rand.New(rand.NewSource(time.Now().Unix()))
	bidder := RandomAccounts(rng, 1)[0]

	cases := []struct {
		name     string
		malleate func()
		pass     bool
	}{
		{
			"valid empty bundle",
			func() {
				accounts = make([]Account, 0)
			},
			true,
		},
		{
			"valid single tx bundle",
			func() {
				accounts = []Account{bidder}
			},
			true,
		},
		{
			"valid multi-tx bundle by same account",
			func() {
				accounts = []Account{bidder, bidder, bidder, bidder}
			},
			true,
		},
		{
			"valid single-tx bundle by a different account",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{randomAccount}
			},
			true,
		},
		{
			"valid multi-tx bundle by a different accounts",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{randomAccount, bidder}
			},
			true,
		},
		{
			"invalid frontrunning bundle",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{bidder, randomAccount}
			},
			false,
		},
		{
			"invalid sandwiching bundle",
			func() {
				randomAccount := RandomAccounts(rng, 1)[0]
				accounts = []Account{bidder, randomAccount, bidder}
			},
			false,
		},
		{
			"invalid multi account bundle",
			func() {
				accounts = RandomAccounts(rng, 3)
			},
			false,
		},
		{
			"invalid multi account bundle without bidder",
			func() {
				randomAccount1 := RandomAccounts(rng, 1)[0]
				randomAccount2 := RandomAccounts(rng, 1)[0]
				accounts = []Account{randomAccount1, randomAccount2}
			},
			false,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Malleate the test case
			tc.malleate()

			// Track the bundle that will be create for this test case as well as the nonces
			bundle := make([]sdk.Tx, 0)
			for _, acc := range accounts {
				// Create a random tx
				tx, err := createRandomTx(suite.encCfg.TxConfig, acc, 0, 1)
				suite.Require().NoError(err)
				bundle = append(bundle, tx)
			}

			// Create the bundle
			bid := sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(100)))
			auctionMsg, err := createMsgAuctionBid(suite.encCfg.TxConfig, bidder, bid, bundle)
			suite.Require().NoError(err)

			// Validate the bundle
			err = suite.auctionKeeper.ValidateAuctionBundle(suite.ctx, auctionMsg.Transactions, bidder.Address)
			if tc.pass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// createRandomTx creates a random transaction with the given number of messages and signer.
func createRandomTx(txCfg client.TxConfig, account Account, nonce, numberMsgs uint64) (authsigning.Tx, error) {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < int(numberMsgs); i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: account.Address.String(),
			ToAddress:   account.Address.String(),
		}
	}

	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txCfg.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}

// createMsgAuctionBid wraps around createMsgAuctionBid to create a MsgAuctionBid and a valid transaction.
func createMsgAuctionBidTx(txConfig client.TxConfig, account Account, bid sdk.Coins, nonce uint64, txs []sdk.Tx) (authsigning.Tx, error) {
	bidMsg, err := createMsgAuctionBid(txConfig, account, bid, txs)
	if err != nil {
		return nil, err
	}

	txBuilder := txConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(bidMsg); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	return txBuilder.GetTx(), nil
}

// createMsgAuctionBid is a helper function to create a new MsgAuctionBid given the required params.
func createMsgAuctionBid(txCfg client.TxConfig, bidder Account, bid sdk.Coins, txs []sdk.Tx) (*auctiontypes.MsgAuctionBid, error) {
	bidMsg := &auctiontypes.MsgAuctionBid{
		Bidder:       bidder.Address.String(),
		Bid:          bid,
		Transactions: make([][]byte, len(txs)),
	}

	for i, tx := range txs {
		bz, err := txCfg.TxEncoder()(tx)
		if err != nil {
			return nil, err
		}

		bidMsg.Transactions[i] = bz
	}

	return bidMsg, nil
}
