package blockbuster_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/auction"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/skip-mev/pob/x/builder/ante"
	"github.com/skip-mev/pob/x/builder/keeper"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"github.com/stretchr/testify/suite"
)

type BlockBusterTestSuite struct {
	suite.Suite
	logger log.Logger
	ctx    sdk.Context

	// Define basic tx configuration
	encodingConfig testutils.EncodingConfig
	auctionFactory auction.Factory

	// Define all of the lanes utilized in the test suite
	tobBlockSpace sdk.Dec
	tobLane       *auction.TOBLane

	baseBlockSpace sdk.Dec
	baseLane       *base.DefaultLane

	lanes   []blockbuster.Lane
	mempool *blockbuster.Mempool
	txCache map[sdk.Tx]struct{}

	// Proposal handler set up
	proposalHandler *blockbuster.ProposalHandler

	// account set up
	accounts []testutils.Account
	random   *rand.Rand
	nonces   map[string]uint64

	// Keeper set up
	builderKeeper    keeper.Keeper
	bankKeeper       *testutils.MockBankKeeper
	accountKeeper    *testutils.MockAccountKeeper
	distrKeeper      *testutils.MockDistributionKeeper
	stakingKeeper    *testutils.MockStakingKeeper
	builderDecorator ante.BuilderDecorator
}

func TestBlockBusterTestSuite(t *testing.T) {
	suite.Run(t, new(BlockBusterTestSuite))
}

func (suite *BlockBusterTestSuite) SetupTest() {
	// General config for transactions and randomness for the test suite
	suite.encodingConfig = testutils.CreateTestEncodingConfig()
	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
	key := sdk.NewKVStoreKey(buildertypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx.WithBlockHeight(1)

	// Lanes configuration
	//
	// TOB lane set up
	suite.auctionFactory = auction.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder())
	suite.tobBlockSpace = sdk.NewDecFromBigIntWithPrec(big.NewInt(1), 1) // 10% of the block space
	suite.tobLane = auction.NewTOBLane(
		suite.logger,
		suite.encodingConfig.TxConfig.TxDecoder(),
		suite.encodingConfig.TxConfig.TxEncoder(),
		0, // No bound on the number of transactions in the lane
		suite.anteHandler,
		suite.auctionFactory,
		sdk.NewDecFromBigIntWithPrec(big.NewInt(1), 1), // 10% of the block space
	)

	// Base lane set up
	suite.baseBlockSpace = sdk.ZeroDec()
	suite.baseLane = base.NewDefaultLane(
		suite.logger,
		suite.encodingConfig.TxConfig.TxDecoder(),
		suite.encodingConfig.TxConfig.TxEncoder(),
		suite.anteHandler,
		sdk.ZeroDec(),
	)

	// Mempool set up
	suite.lanes = []blockbuster.Lane{suite.tobLane, suite.baseLane}
	suite.mempool = blockbuster.NewMempool(suite.lanes...)
	suite.txCache = make(map[sdk.Tx]struct{})

	// Accounts set up
	suite.accounts = testutils.RandomAccounts(suite.random, 10)
	suite.nonces = make(map[string]uint64)
	for _, acc := range suite.accounts {
		suite.nonces[acc.Address.String()] = 0
	}

	// Set up the keepers and decorators
	// Mock keepers set up
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = testutils.NewMockAccountKeeper(ctrl)
	suite.accountKeeper.EXPECT().GetModuleAddress(buildertypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
	suite.bankKeeper = testutils.NewMockBankKeeper(ctrl)
	suite.distrKeeper = testutils.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = testutils.NewMockStakingKeeper(ctrl)

	// Builder keeper / decorator set up
	suite.builderKeeper = keeper.NewKeeper(
		suite.encodingConfig.Codec,
		key,
		suite.accountKeeper,
		suite.bankKeeper,
		suite.distrKeeper,
		suite.stakingKeeper,
		sdk.AccAddress([]byte("authority")).String(),
	)

	// Set the default params for the builder keeper
	err := suite.builderKeeper.SetParams(suite.ctx, buildertypes.DefaultParams())
	suite.Require().NoError(err)

	// Set up the ante handler
	suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane)

	// Proposal handler set up
	suite.proposalHandler = blockbuster.NewProposalHandler(log.NewNopLogger(), suite.mempool, suite.encodingConfig.TxConfig.TxEncoder())
}

// fillBaseLane fills the base lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillBaseLane(numTxs int) {
	for i := 0; i < numTxs; i++ {
		// randomly select an account to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a few random msgs and construct the tx
		nonce := suite.nonces[acc.Address.String()]
		randomMsgs := testutils.CreateRandomMsgs(acc.Address, 3)
		tx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, randomMsgs)
		suite.Require().NoError(err)

		// insert the tx into the lane and update the account
		suite.nonces[acc.Address.String()]++
		priority := suite.random.Int63n(100) + 1
		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), tx))
	}
}

// fillTOBLane fills the TOB lane with numTxs transactions that are randomly created.
func (suite *BlockBusterTestSuite) fillTOBLane(numTxs int) {
	// Insert a bunch of auction transactions into the global mempool and auction mempool
	for i := 0; i < numTxs; i++ {
		// randomly select a bidder to create the tx
		randomIndex := suite.random.Intn(len(suite.accounts))
		acc := suite.accounts[randomIndex]

		// create a randomized auction transaction
		nonce := suite.nonces[acc.Address.String()]
		bidAmount := sdk.NewInt(int64(suite.random.Intn(1000) + 1))
		bid := sdk.NewCoin("foo", bidAmount)
		tx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, acc, bid, nonce, 1000, nil)
		suite.Require().NoError(err)

		// insert the auction tx into the global mempool
		suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
		suite.nonces[acc.Address.String()]++
	}
}

func (suite *BlockBusterTestSuite) anteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	if _, ok := suite.txCache[tx]; ok {
		return ctx, fmt.Errorf("tx already seen")
	}

	suite.txCache[tx] = struct{}{}

	signer := tx.GetMsgs()[0].GetSigners()[0]
	suite.bankKeeper.EXPECT().GetAllBalances(ctx, signer).AnyTimes().Return(
		sdk.NewCoins(
			sdk.NewCoin("foo", sdk.NewInt(100000000000000)),
		),
	)

	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	return suite.builderDecorator.AnteHandle(ctx, tx, false, next)
}

func (suite *BlockBusterTestSuite) resetLanes() {
	suite.tobLane = auction.NewTOBLane(
		suite.logger,
		suite.encodingConfig.TxConfig.TxDecoder(),
		suite.encodingConfig.TxConfig.TxEncoder(),
		0, // No bound on the number of transactions in the lane
		suite.anteHandler,
		suite.auctionFactory,
		suite.tobBlockSpace,
	)

	suite.baseLane = base.NewDefaultLane(
		suite.logger,
		suite.encodingConfig.TxConfig.TxDecoder(),
		suite.encodingConfig.TxConfig.TxEncoder(),
		suite.anteHandler,
		suite.baseBlockSpace,
	)

	suite.lanes = []blockbuster.Lane{suite.tobLane, suite.baseLane}
	suite.mempool = blockbuster.NewMempool(suite.lanes...)
}
