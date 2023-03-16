package abci_test

import (
	"math/rand"
	"reflect"
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
	maxBidAmount    int64

	// auction setup
	auctionKeeper    keeper.Keeper
	bankKeeper       *MockBankKeeper
	accountKeeper    *MockAccountKeeper
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

	// Mock keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(auctiontypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = NewMockBankKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))

	// Auction keeper / decorator set up
	suite.auctionKeeper = keeper.NewKeeper(suite.encodingConfig.Codec, suite.key, suite.accountKeeper, suite.bankKeeper, suite.authorityAccount.String())
	err := suite.auctionKeeper.SetParams(suite.ctx, auctiontypes.DefaultParams())
	suite.Require().NoError(err)
	suite.auctionDecorator = ante.NewAuctionDecorator(suite.auctionKeeper, suite.encodingConfig.TxConfig.TxDecoder())

	// Accounts set up
	suite.accounts = RandomAccounts(suite.random, 1)
	suite.balances = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000000000)))
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}
	suite.fundAccounds()

	// Mempool set up
	suite.mempool = mempool.NewAuctionMempool(suite.encodingConfig.TxConfig.TxDecoder(), 0)
	suite.maxBidAmount = 40000

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

func (suite *IntegrationTestSuite) ProcessProposalVerifyTx(txBz []byte) (sdk.Tx, error) {
	return nil, nil
}

func (suite *IntegrationTestSuite) executeAnteHandler(tx sdk.Tx) (sdk.Context, error) {
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.auctionDecorator.AnteHandle(suite.ctx, tx, false, next)
}

// CreateFilledMempool creates a pre-filled mempool with numNormalTxs normal transactions, numAuctionTxs auction transactions, and numBundledTxs bundled
// transactions per auction transaction. If insertRefTxs is true, it will also insert a the referenced transactions into the mempool. This returns
// the total number of transactions inserted into the mempool.
func (suite *IntegrationTestSuite) createFilledMempool(numNormalTxs, numAuctionTxs, numBundledTxs int, insertRefTxs bool) int {
	// Insert a bunch of normal transactions into the global mempool
	for i := 0; i < numNormalTxs; i++ {
		// create a few random msgs
		randomMsgs := createRandomMsgs(3)

		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]
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
		bid := sdk.NewCoins(sdk.NewInt64Coin("foo", suite.maxBidAmount))
		nonce := suite.nonces[acc.Address.String()]
		bidMsg, err := createMsgAuctionBid(suite.encodingConfig.TxConfig, acc, bid, nonce, numBundledTxs)
		suite.nonces[acc.Address.String()] += uint64(numBundledTxs)
		suite.Require().NoError(err)

		// create the auction tx
		nonce = suite.nonces[acc.Address.String()]
		auctionTx, err := createTx(suite.encodingConfig.TxConfig, acc, nonce, []sdk.Msg{bidMsg})
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(suite.maxBidAmount), auctionTx))
		suite.nonces[acc.Address.String()]++

		if insertRefTxs {
			for _, refRawTx := range bidMsg.GetTransactions() {
				refTx, err := suite.encodingConfig.TxConfig.TxDecoder()(refRawTx)
				suite.Require().NoError(err)
				suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(suite.maxBidAmount), refTx))
			}
		}
	}

	var totalNumTxs int
	suite.Require().Equal(numAuctionTxs, suite.mempool.CountAuctionTx())
	if insertRefTxs {
		totalNumTxs = numNormalTxs + numAuctionTxs*(numBundledTxs+1)
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
	} else {
		totalNumTxs = numNormalTxs + numAuctionTxs
		suite.Require().Equal(totalNumTxs, suite.mempool.CountTx())
	}

	return totalNumTxs
}

func (suite *IntegrationTestSuite) fundAccounds() {
	for _, acc := range suite.accounts {
		suite.bankKeeper.EXPECT().GetAllBalances(suite.ctx, acc.Address).AnyTimes().Return(suite.balances)
	}
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

// RandomAccounts returns a slice of n random accounts.
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

func createRandomMsgs(numberMsgs int) []sdk.Msg {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < numberMsgs; i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: sdk.AccAddress([]byte("addr1_______________")).String(),
			ToAddress:   sdk.AccAddress([]byte("addr2_______________")).String(),
		}
	}

	return msgs
}

// createMsgAuctionBid creates a new MsgAuctionBid with numberMsgs of referenced transactions embedded into the message and the inputted bid/bidder.
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

// MockAccountKeeper is a mock of AccountKeeper interface.
type MockAccountKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockAccountKeeperMockRecorder
}

// MockAccountKeeperMockRecorder is the mock recorder for MockAccountKeeper.
type MockAccountKeeperMockRecorder struct {
	mock *MockAccountKeeper
}

// NewMockAccountKeeper creates a new mock instance.
func NewMockAccountKeeper(ctrl *gomock.Controller) *MockAccountKeeper {
	mock := &MockAccountKeeper{ctrl: ctrl}
	mock.recorder = &MockAccountKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAccountKeeper) EXPECT() *MockAccountKeeperMockRecorder {
	return m.recorder
}

// GetModuleAddress mocks base method.
func (m *MockAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAddress", name)
	ret0, _ := ret[0].(sdk.AccAddress)
	return ret0
}

// GetModuleAddress indicates an expected call of GetModuleAddress.
func (mr *MockAccountKeeperMockRecorder) GetModuleAddress(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAddress", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAddress), name)
}

// MockBankKeeper is a mock of BankKeeper interface.
type MockBankKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockBankKeeperMockRecorder
}

// MockBankKeeperMockRecorder is the mock recorder for MockBankKeeper.
type MockBankKeeperMockRecorder struct {
	mock *MockBankKeeper
}

// NewMockBankKeeper creates a new mock instance.
func NewMockBankKeeper(ctrl *gomock.Controller) *MockBankKeeper {
	mock := &MockBankKeeper{ctrl: ctrl}
	mock.recorder = &MockBankKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBankKeeper) EXPECT() *MockBankKeeperMockRecorder {
	return m.recorder
}

// GetAllBalances mocks base method.
func (m *MockBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllBalances", ctx, addr)
	ret0 := ret[0].(sdk.Coins)
	return ret0
}

// GetAllBalances indicates an expected call of GetAllBalances.
func (mr *MockBankKeeperMockRecorder) GetAllBalances(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllBalances", reflect.TypeOf((*MockBankKeeper)(nil).GetAllBalances), ctx, addr)
}

// SendCoins mocks base method.
func (m *MockBankKeeper) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendCoins", ctx, fromAddr, toAddr, amt)
	return nil
}

// SendCoins indicates an expected call of SendCoins.
func (mr *MockBankKeeperMockRecorder) SendCoins(ctx, fromAddr, toAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoins", reflect.TypeOf((*MockBankKeeper)(nil).SendCoins), ctx, fromAddr, toAddr, amt)
}

// SendCoinsFromAccountToModule mocks base method.
func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromAccountToModule", ctx, senderAddr, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromAccountToModule indicates an expected call of SendCoinsFromAccountToModule.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromAccountToModule), ctx, senderAddr, recipientModule, amt)
}
