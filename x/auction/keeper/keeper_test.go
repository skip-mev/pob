package keeper_test

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/x/auction/keeper"
	"github.com/skip-mev/pob/x/auction/types"

	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite

	auctionKeeper    keeper.Keeper
	encCfg           encodingConfig
	AuctionDecorator sdk.AnteDecorator
	ctx              sdk.Context
	msgServer        types.MsgServer
}

type encodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) SetupTest() {
	suite.encCfg = createTestEncodingConfig()
	key := sdk.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, sdk.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	// Auction Keeper setup
	// TODO: Implement the mock bank keeper.
	// https://github.com/skip-mev/pob/issues/15
	ctrl := gomock.NewController(suite.T())
	accountKeeper := NewMockAccountKeeper(ctrl)
	accountKeeper.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{})

	authorityAccount := sdk.AccAddress([]byte("authority"))
	suite.auctionKeeper = keeper.NewKeeper(suite.encCfg.Codec, key, accountKeeper, nil, authorityAccount.String(), suite.encCfg.TxConfig.TxDecoder())
	err := suite.auctionKeeper.SetParams(suite.ctx, types.DefaultParams())
	suite.Require().NoError(err)

	suite.AuctionDecorator = keeper.NewAuctionDecorator(suite.auctionKeeper)
	suite.msgServer = keeper.NewMsgServerImpl(suite.auctionKeeper)
}

func createTestEncodingConfig() encodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()

	banktypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)

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

// GetModuleAccount mocks base method.
func (m *MockAccountKeeper) GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleAccount", ctx, moduleName)
	ret0, _ := ret[0].(authtypes.ModuleAccountI)
	return ret0
}

// GetModuleAccount indicates an expected call of GetModuleAccount.
func (mr *MockAccountKeeperMockRecorder) GetModuleAccount(ctx, moduleName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleAccount", reflect.TypeOf((*MockAccountKeeper)(nil).GetModuleAccount), ctx, moduleName)
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

// SetModuleAccount mocks base method.
func (m *MockAccountKeeper) SetModuleAccount(arg0 sdk.Context, arg1 authtypes.ModuleAccountI) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetModuleAccount", arg0, arg1)
}

// SetModuleAccount indicates an expected call of SetModuleAccount.
func (mr *MockAccountKeeperMockRecorder) SetModuleAccount(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetModuleAccount", reflect.TypeOf((*MockAccountKeeper)(nil).SetModuleAccount), arg0, arg1)
}
