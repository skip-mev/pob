package mempool_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/pob/mempool"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	encCfg   encodingConfig
	mempool  *mempool.AuctionMempool
	ctx      sdk.Context
	random   *rand.Rand
	accounts []Account
	nonces   map[string]uint64
}

func TestMempoolTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) SetupTest() {
	// Mempool setup
	suite.encCfg = createTestEncodingConfig()
	suite.mempool = mempool.NewAuctionMempool(suite.encCfg.TxConfig.TxDecoder())
	suite.ctx = sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger())

	// Init accounts
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.accounts = RandomAccounts(suite.random, 5)
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}
}

func (suite *IntegrationTestSuite) CreateFilledMempool(numNormalTxs, numAuctionTxs int) {
	for i := 0; i < numNormalTxs; i++ {
		// randomly select an account to create the tx
		p := suite.random.Int63n(500-1) + 1
		j := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[j]
		txBuilder := suite.encCfg.TxConfig.NewTxBuilder()

		msgs := []sdk.Msg{
			&banktypes.MsgSend{
				FromAddress: acc.Address.String(),
				ToAddress:   acc.Address.String(),
			},
		}
		err := txBuilder.SetMsgs(msgs...)
		suite.Require().NoError(err)

		sigV2 := signing.SignatureV2{
			PubKey: acc.PrivKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  suite.encCfg.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: suite.nonces[acc.Address.String()],
		}
		err = txBuilder.SetSignatures(sigV2)
		suite.Require().NoError(err)

		suite.nonces[acc.Address.String()]++
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(p), txBuilder.GetTx()))
	}
}

func (suite *IntegrationTestSuite) TestAuctionMempool() {

	suite.Require().Nil(suite.mempool.AuctionBidSelect(suite.ctx, nil))

	// insert bid transactions
	var highestBid sdk.Coins
	biddingAccs := RandomAccounts(suite.random, 100)

	for _, acc := range biddingAccs {
		p := suite.random.Int63n(500-1) + 1
		txBuilder := suite.encCfg.TxConfig.NewTxBuilder()

		// keep track of highest bid
		bid := sdk.NewCoins(sdk.NewInt64Coin("foo", p))
		if bid.IsAllGT(highestBid) {
			highestBid = bid
		}

		bidMsg, err := createMsgAuctionBid(suite.encCfg.TxConfig, acc, bid)
		suite.Require().NoError(err)

		msgs := []sdk.Msg{bidMsg}
		err = txBuilder.SetMsgs(msgs...)
		suite.Require().NoError(err)

		sigV2 := signing.SignatureV2{
			PubKey: acc.PrivKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  suite.encCfg.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: 0,
		}
		err = txBuilder.SetSignatures(sigV2)
		suite.Require().NoError(err)
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(p), txBuilder.GetTx()))

		// Insert the referenced txs just to ensure that they are removed from the
		// mempool in cases where they exist.
		for _, refRawTx := range bidMsg.GetTransactions() {
			refTx, err := suite.encCfg.TxConfig.TxDecoder()(refRawTx)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(0), refTx))
		}
	}

	expectedCount := 1000 + 100 + 200
	suite.Require().Equal(expectedCount, suite.mempool.CountTx())

	// select the top bid and misc txs
	bidTx := suite.mempool.AuctionBidSelect(suite.ctx).Tx()
	suite.Require().Len(bidTx.GetMsgs(), 1)
	suite.Require().Equal(highestBid, bidTx.GetMsgs()[0].(*auctiontypes.MsgAuctionBid).Bid)

	// remove the top bid tx (without removing the referenced txs)
	prevAuctionCount := suite.mempool.CountAuctionTx()
	suite.Require().NoError(suite.mempool.RemoveWithoutRefTx(bidTx))
	suite.Require().Equal(expectedCount-1, suite.mempool.CountTx())
	suite.Require().Equal(prevAuctionCount-1, suite.mempool.CountAuctionTx())

	// the next bid tx should be less than or equal to the previous highest bid
	nextBidTx := suite.mempool.AuctionBidSelect(suite.ctx).Tx()
	suite.Require().Len(nextBidTx.GetMsgs(), 1)
	msgAuctionBid := nextBidTx.GetMsgs()[0].(*auctiontypes.MsgAuctionBid)
	suite.Require().True(msgAuctionBid.Bid.IsAllLTE(highestBid))

	// remove the top bid tx (including the ref txs)
	prevGlobalCount := suite.mempool.CountTx()
	suite.Require().NoError(suite.mempool.Remove(nextBidTx))
	suite.Require().Equal(prevGlobalCount-1-2, suite.mempool.CountTx())
}

func TestAuctionMempoolInsertion(t *testing.T) {
}

func TestAuctionMempoolRemoval(t *testing.T) {

}

func TestAuctionMempoolSelect(t *testing.T) {

}
