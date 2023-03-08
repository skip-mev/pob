package keeper_test

import (
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
)

func (suite *IntegrationTestSuite) TestValidateBundle() {
	var (
		accounts []Account
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
			err = suite.auctionKeeper.ValidateBundle(suite.ctx, auctionMsg.Transactions, bidder.Address)
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
