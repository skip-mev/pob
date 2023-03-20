package abci_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/abci"
	"github.com/skip-mev/pob/mempool"
	"github.com/skip-mev/pob/x/auction/ante"
	"github.com/skip-mev/pob/x/auction/keeper"
	auctiontypes "github.com/skip-mev/pob/x/auction/types"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	ctx sdk.Context

	// mempool setup
	mempool         *mempool.AuctionMempool
	logger          log.Logger
	encodingConfig  encodingConfig
	proposalHandler *abci.ProposalHandler

	// auction bid setup
	auctionBidAmount sdk.Coins
	minBidIncrement  sdk.Coins

	// auction setup
	auctionKeeper    keeper.Keeper
	bankKeeper       *MockBankKeeper
	accountKeeper    *MockAccountKeeper
	distrKeeper      *MockDistributionKeeper
	stakingKeeper    *MockStakingKeeper
	auctionDecorator ante.AuctionDecorator
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress

	// account set up
	accounts []Account
	balances sdk.Coins
	random   *rand.Rand
	nonces   map[string]uint64
}

func TestPrepareProposalSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) SetupTest() {
	// General config
	suite.encodingConfig = createTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	suite.key = sdk.NewKVStoreKey(auctiontypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, sdk.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	// Mempool set up
	suite.mempool = mempool.NewAuctionMempool(suite.encodingConfig.TxConfig.TxDecoder(), 0)
	suite.auctionBidAmount = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000000)))
	suite.minBidIncrement = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000)))

	// Mock keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(auctiontypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = NewMockBankKeeper(ctrl)
	suite.distrKeeper = NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = NewMockStakingKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))

	// Auction keeper / decorator set up
	suite.auctionKeeper = keeper.NewKeeper(
		suite.encodingConfig.Codec,
		suite.key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		suite.authorityAccount.String(),
	)
	err := suite.auctionKeeper.SetParams(suite.ctx, auctiontypes.DefaultParams())
	suite.Require().NoError(err)
	suite.auctionDecorator = ante.NewAuctionDecorator(suite.auctionKeeper, suite.encodingConfig.TxConfig.TxDecoder(), suite.mempool)

	// Accounts set up
	suite.accounts = RandomAccounts(suite.random, 1)
	suite.balances = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000000000000000)))
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}

	// Proposal handler set up
	suite.logger = log.NewNopLogger()
	suite.proposalHandler = abci.NewProposalHandler(suite.mempool, suite.logger, suite, suite.encodingConfig.TxConfig.TxEncoder(), suite.encodingConfig.TxConfig.TxDecoder())
}

func (suite *IntegrationTestSuite) PrepareProposalVerifyTx(tx sdk.Tx) ([]byte, error) {
	_, err := suite.executeAnteHandler(tx)
	if err != nil {
		return nil, err
	}

	txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
	if err != nil {
		return nil, err
	}

	return txBz, nil
}

func (suite *IntegrationTestSuite) ProcessProposalVerifyTx(_ []byte) (sdk.Tx, error) {
	return nil, nil
}

func (suite *IntegrationTestSuite) executeAnteHandler(tx sdk.Tx) (sdk.Context, error) {
	signer := tx.GetMsgs()[0].GetSigners()[0]
	suite.bankKeeper.EXPECT().GetAllBalances(suite.ctx, signer).AnyTimes().Return(suite.balances)

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.auctionDecorator.AnteHandle(suite.ctx, tx, false, next)
}

func (suite *IntegrationTestSuite) createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs int, insertRefTxs bool) int {
	// Insert a bunch of normal transactions into the global mempool
	for i := 0; i < numNormalTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs
		randomMsgs := createRandomMsgs(acc.Address, 3)

		nonce := suite.nonces[acc.Address.String()]
		randomTx, err := createTx(suite.encodingConfig.TxConfig, acc, nonce, randomMsgs)
		suite.Require().NoError(err)

		suite.nonces[acc.Address.String()]++
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), randomTx))
	}

	suite.Require().Equal(numNormalTxs, suite.mempool.CountTx())
	suite.Require().Equal(0, suite.mempool.CountAuctionTx())

	// Insert a bunch of auction transactions into the global mempool and auction mempool
	for i := 0; i < numAuctionTxs; i++ {
		// randomly select a bidder to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a new auction bid msg with numBundledTxs bundled transactions
		nonce := suite.nonces[acc.Address.String()]
		bidMsg, err := createMsgAuctionBid(suite.encodingConfig.TxConfig, acc, suite.auctionBidAmount, nonce, numBundledTxs)
		suite.nonces[acc.Address.String()] += uint64(numBundledTxs)
		suite.Require().NoError(err)

		// create the auction tx
		nonce = suite.nonces[acc.Address.String()]
		auctionTx, err := createTx(suite.encodingConfig.TxConfig, acc, nonce, []sdk.Msg{bidMsg})
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), auctionTx))
		suite.nonces[acc.Address.String()]++

		if insertRefTxs {
			for _, refRawTx := range bidMsg.GetTransactions() {
				refTx, err := suite.encodingConfig.TxConfig.TxDecoder()(refRawTx)
				suite.Require().NoError(err)
				priority := suite.random.Int63n(100) + 1
				suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), refTx))
			}
		}

		// decrement the bid amount for the next auction tx
		suite.auctionBidAmount = suite.auctionBidAmount.Sub(suite.minBidIncrement...)
	}

	numSeenGlobalTxs := 0
	for iterator := suite.mempool.Select(suite.ctx, nil); iterator != nil; iterator = iterator.Next() {
		numSeenGlobalTxs++
	}

	numSeenAuctionTxs := 0
	for iterator := suite.mempool.AuctionBidSelect(suite.ctx); iterator != nil; iterator = iterator.Next() {
		numSeenAuctionTxs++
	}

	var totalNumTxs int
	suite.Require().Equal(numAuctionTxs, suite.mempool.CountAuctionTx())
	if insertRefTxs {
		totalNumTxs = numNormalTxs + numAuctionTxs*(numBundledTxs+1)
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
		suite.Require().Equal(totalNumTxs, numSeenGlobalTxs)
	} else {
		totalNumTxs = numNormalTxs + numAuctionTxs
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
		suite.Require().Equal(totalNumTxs, numSeenGlobalTxs)
	}

	suite.Require().Equal(numAuctionTxs, numSeenAuctionTxs)

	return totalNumTxs
}

type encodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func createTestEncodingConfig() encodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := types.NewInterfaceRegistry()

	banktypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	auctiontypes.RegisterInterfaces(interfaceRegistry)

	codec := codec.NewProtoCodec(interfaceRegistry)

	return encodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          tx.NewTxConfig(codec, tx.DefaultSignModes),
		Amino:             cdc,
	}
}

type Account struct {
	PrivKey cryptotypes.PrivKey
	PubKey  cryptotypes.PubKey
	Address sdk.AccAddress
	ConsKey cryptotypes.PrivKey
}

func (acc Account) Equals(acc2 Account) bool {
	return acc.Address.Equals(acc2.Address)
}

func RandomAccounts(r *rand.Rand, n int) []Account {
	accs := make([]Account, n)

	for i := 0; i < n; i++ {
		pkSeed := make([]byte, 15)
		r.Read(pkSeed)

		accs[i].PrivKey = secp256k1.GenPrivKeyFromSecret(pkSeed)
		accs[i].PubKey = accs[i].PrivKey.PubKey()
		accs[i].Address = sdk.AccAddress(accs[i].PubKey.Address())

		accs[i].ConsKey = ed25519.GenPrivKeyFromSecret(pkSeed)
	}

	return accs
}

func createTx(txCfg client.TxConfig, account Account, nonce uint64, msgs []sdk.Msg) (authsigning.Tx, error) {
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

func createRandomMsgs(acc sdk.AccAddress, numberMsgs int) []sdk.Msg {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < numberMsgs; i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: acc.String(),
			ToAddress:   acc.String(),
		}
	}

	return msgs
}

func createMsgAuctionBid(txCfg client.TxConfig, bidder Account, bid sdk.Coins, nonce uint64, numberMsgs int) (*auctiontypes.MsgAuctionBid, error) {
	bidMsg := &auctiontypes.MsgAuctionBid{
		Bidder:       bidder.Address.String(),
		Bid:          bid,
		Transactions: make([][]byte, numberMsgs),
	}

	for i := 0; i < numberMsgs; i++ {
		txBuilder := txCfg.NewTxBuilder()

		msgs := []sdk.Msg{
			&banktypes.MsgSend{
				FromAddress: bidder.Address.String(),
				ToAddress:   bidder.Address.String(),
			},
		}
		if err := txBuilder.SetMsgs(msgs...); err != nil {
			return nil, err
		}

		sigV2 := signing.SignatureV2{
			PubKey: bidder.PrivKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  txCfg.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: nonce + uint64(i),
		}
		if err := txBuilder.SetSignatures(sigV2); err != nil {
			return nil, err
		}

		bz, err := txCfg.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return nil, err
		}

		bidMsg.Transactions[i] = bz
	}

	return bidMsg, nil
}
