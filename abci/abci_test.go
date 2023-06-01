package abci_test

// import (
// 	"math/big"
// 	"math/rand"
// 	"testing"
// 	"time"

// 	comettypes "github.com/cometbft/cometbft/abci/types"
// 	"github.com/cometbft/cometbft/libs/log"
// 	storetypes "github.com/cosmos/cosmos-sdk/store/types"
// 	"github.com/cosmos/cosmos-sdk/testutil"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/golang/mock/gomock"
// 	"github.com/skip-mev/pob/abci"
// 	"github.com/skip-mev/pob/blockbuster"
// 	"github.com/skip-mev/pob/blockbuster/lanes/auction"
// 	"github.com/skip-mev/pob/blockbuster/lanes/base"
// 	"github.com/skip-mev/pob/mempool"
// 	testutils "github.com/skip-mev/pob/testutils"
// 	"github.com/skip-mev/pob/x/builder/ante"
// 	"github.com/skip-mev/pob/x/builder/keeper"
// 	buildertypes "github.com/skip-mev/pob/x/builder/types"
// 	"github.com/stretchr/testify/suite"
// )

// type ABCITestSuite struct {
// 	suite.Suite
// 	ctx sdk.Context

// 	// mempool and lane set up
// 	mempool  *blockbuster.Mempool
// 	tobLane  *auction.TOBLane
// 	baseLane *base.DefaultLane
// 	lanes    []blockbuster.Lane

// 	logger               log.Logger
// 	encodingConfig       testutils.EncodingConfig
// 	proposalHandler      *abci.ProposalHandler
// 	voteExtensionHandler *abci.VoteExtensionHandler
// 	config               mempool.AuctionFactory
// 	txs                  map[string]struct{}

// 	// auction bid setup
// 	auctionBidAmount sdk.Coin
// 	minBidIncrement  sdk.Coin

// 	// builder setup
// 	builderKeeper    keeper.Keeper
// 	bankKeeper       *testutils.MockBankKeeper
// 	accountKeeper    *testutils.MockAccountKeeper
// 	distrKeeper      *testutils.MockDistributionKeeper
// 	stakingKeeper    *testutils.MockStakingKeeper
// 	builderDecorator ante.BuilderDecorator
// 	key              *storetypes.KVStoreKey
// 	authorityAccount sdk.AccAddress

// 	// account set up
// 	accounts []testutils.Account
// 	balances sdk.Coins
// 	random   *rand.Rand
// 	nonces   map[string]uint64
// }

// func TestABCISuite(t *testing.T) {
// 	suite.Run(t, new(ABCITestSuite))
// }

// func (suite *ABCITestSuite) SetupTest() {
// 	// General config
// 	suite.encodingConfig = testutils.CreateTestEncodingConfig()
// 	suite.random = rand.New(rand.NewSource(time.Now().Unix()))
// 	suite.key = storetypes.NewKVStoreKey(buildertypes.StoreKey)
// 	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, storetypes.NewTransientStoreKey("transient_test"))
// 	suite.ctx = testCtx.Ctx.WithBlockHeight(1)

// 	// Mempool set up
// 	// Lanes configuration
// 	//
// 	// TOB lane set up
// 	suite.tobLane = auction.NewTOBLane(
// 		suite.ctx.Logger(),
// 		suite.encodingConfig.TxConfig.TxDecoder(),
// 		suite.encodingConfig.TxConfig.TxEncoder(),
// 		0, // No bound on the number of transactions in the lane
// 		suite.anteHandler,
// 		auction.NewDefaultAuctionFactory(suite.encodingConfig.TxConfig.TxDecoder()),
// 		sdk.NewDecFromBigIntWithPrec(big.NewInt(1), 1), // 10% of the block space
// 	)

// 	// Base lane set up
// 	suite.baseLane = base.NewDefaultLane(
// 		suite.ctx.Logger(),
// 		suite.encodingConfig.TxConfig.TxDecoder(),
// 		suite.encodingConfig.TxConfig.TxEncoder(),
// 		suite.anteHandler,
// 		sdk.ZeroDec(),
// 	)

// 	// Mempool set up
// 	suite.lanes = []blockbuster.Lane{suite.tobLane, suite.baseLane}
// 	suite.mempool = blockbuster.NewMempool(suite.lanes...)

// 	suite.txs = make(map[string]struct{})
// 	suite.auctionBidAmount = sdk.NewCoin("foo", sdk.NewInt(1000000000))
// 	suite.minBidIncrement = sdk.NewCoin("foo", sdk.NewInt(1000))

// 	// Mock keepers set up
// 	ctrl := gomock.NewController(suite.T())
// 	suite.accountKeeper = testutils.NewMockAccountKeeper(ctrl)
// 	suite.accountKeeper.EXPECT().GetModuleAddress(buildertypes.ModuleName).Return(sdk.AccAddress{}).AnyTimes()
// 	suite.bankKeeper = testutils.NewMockBankKeeper(ctrl)
// 	suite.distrKeeper = testutils.NewMockDistributionKeeper(ctrl)
// 	suite.stakingKeeper = testutils.NewMockStakingKeeper(ctrl)
// 	suite.authorityAccount = sdk.AccAddress([]byte("authority"))

// 	// Builder keeper / decorator set up
// 	suite.builderKeeper = keeper.NewKeeper(
// 		suite.encodingConfig.Codec,
// 		suite.key,
// 		suite.accountKeeper,
// 		suite.bankKeeper,
// 		suite.distrKeeper,
// 		suite.stakingKeeper,
// 		suite.authorityAccount.String(),
// 	)
// 	err := suite.builderKeeper.SetParams(suite.ctx, buildertypes.DefaultParams())
// 	suite.Require().NoError(err)
// 	suite.builderDecorator = ante.NewBuilderDecorator(suite.builderKeeper, suite.encodingConfig.TxConfig.TxEncoder(), suite.tobLane, suite.mempool)

// 	// Accounts set up
// 	suite.accounts = testutils.RandomAccounts(suite.random, 10)
// 	suite.balances = sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1000000000000000000)))
// 	suite.nonces = make(map[string]uint64)
// 	for _, acc := range suite.accounts {
// 		suite.nonces[acc.Address.String()] = 0
// 	}

// 	// Proposal handler set up
// 	suite.logger = log.NewNopLogger()
// 	suite.proposalHandler = abci.NewProposalHandler(suite.logger, suite.mempool, suite.encodingConfig.TxConfig.TxEncoder())
// 	suite.voteExtensionHandler = NewVoteExtensionHandler(suite.mempool, suite.encodingConfig.TxConfig.TxDecoder(), suite.encodingConfig.TxConfig.TxEncoder(), suite.anteHandler)
// }

// func (suite *ABCITestSuite) anteHandler(ctx sdk.Context, tx sdk.Tx, _ bool) (sdk.Context, error) {
// 	signer := tx.GetMsgs()[0].GetSigners()[0]
// 	suite.bankKeeper.EXPECT().GetAllBalances(ctx, signer).AnyTimes().Return(suite.balances)

// 	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
// 		return ctx, nil
// 	}

// 	ctx, err := suite.builderDecorator.AnteHandle(ctx, tx, false, next)
// 	if err != nil {
// 		return ctx, err
// 	}

// 	return ctx, nil
// }

// // fillBaseLane fills the base lane with numTxs transactions that are randomly created.
// func (suite *ABCITestSuite) fillBaseLane(numTxs int) {
// 	for i := 0; i < numTxs; i++ {
// 		// randomly select an account to create the tx
// 		randomIndex := suite.random.Intn(len(suite.accounts))
// 		acc := suite.accounts[randomIndex]

// 		// create a few random msgs and construct the tx
// 		nonce := suite.nonces[acc.Address.String()]
// 		randomMsgs := testutils.CreateRandomMsgs(acc.Address, 3)
// 		tx, err := testutils.CreateTx(suite.encodingConfig.TxConfig, acc, nonce, 1000, randomMsgs)
// 		suite.Require().NoError(err)

// 		// insert the tx into the lane and update the account
// 		suite.nonces[acc.Address.String()]++
// 		priority := suite.random.Int63n(100) + 1
// 		suite.Require().NoError(suite.mempool.Insert(suite.ctx.WithPriority(priority), tx))
// 	}
// }

// // fillTOBLane fills the TOB lane with numTxs transactions that are randomly created.
// func (suite *ABCITestSuite) fillTOBLane(numTxs int) {
// 	// Insert a bunch of auction transactions into the global mempool and auction mempool
// 	for i := 0; i < numTxs; i++ {
// 		// randomly select a bidder to create the tx
// 		randomIndex := suite.random.Intn(len(suite.accounts))
// 		acc := suite.accounts[randomIndex]

// 		// create a randomized auction transaction
// 		nonce := suite.nonces[acc.Address.String()]
// 		bidAmount := sdk.NewInt(int64(suite.random.Intn(1000) + 1))
// 		bid := sdk.NewCoin("foo", bidAmount)
// 		tx, err := testutils.CreateAuctionTxWithSigners(suite.encodingConfig.TxConfig, acc, bid, nonce, 1000, nil)
// 		suite.Require().NoError(err)

// 		// insert the auction tx into the global mempool
// 		suite.Require().NoError(suite.mempool.Insert(suite.ctx, tx))
// 		suite.nonces[acc.Address.String()]++
// 	}
// }

// func (suite *ABCITestSuite) exportMempool() [][]byte {
// 	txs := make([][]byte, 0)
// 	seenTxs := make(map[string]bool)

// 	iterator := suite.mempool.Select(suite.ctx, nil)
// 	for ; iterator != nil; iterator = iterator.Next() {
// 		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(iterator.Tx())
// 		suite.Require().NoError(err)

// 		if !seenTxs[string(txBz)] {
// 			txs = append(txs, txBz)
// 		}
// 	}

// 	return txs
// }

// func (suite *ABCITestSuite) createPrepareProposalRequest(maxBytes int64) comettypes.RequestPrepareProposal {
// 	voteExtensions := make([]comettypes.ExtendedVoteInfo, 0)

// 	auctionIterator := suite.mempool.AuctionBidSelect(suite.ctx)
// 	for ; auctionIterator != nil; auctionIterator = auctionIterator.Next() {
// 		tx := auctionIterator.Tx()

// 		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
// 		suite.Require().NoError(err)

// 		voteExtensions = append(voteExtensions, comettypes.ExtendedVoteInfo{
// 			VoteExtension: txBz,
// 		})
// 	}

// 	return comettypes.RequestPrepareProposal{
// 		MaxTxBytes: maxBytes,
// 		LocalLastCommit: comettypes.ExtendedCommitInfo{
// 			Votes: voteExtensions,
// 		},
// 	}
// }

// func (suite *ABCITestSuite) createExtendedCommitInfoFromTxBzs(txs [][]byte) []byte {
// 	voteExtensions := make([]comettypes.ExtendedVoteInfo, 0)

// 	for _, txBz := range txs {
// 		voteExtensions = append(voteExtensions, comettypes.ExtendedVoteInfo{
// 			VoteExtension: txBz,
// 		})
// 	}

// 	commitInfo := comettypes.ExtendedCommitInfo{
// 		Votes: voteExtensions,
// 	}

// 	commitInfoBz, err := commitInfo.Marshal()
// 	suite.Require().NoError(err)

// 	return commitInfoBz
// }

// func (suite *ABCITestSuite) createAuctionInfoFromTxBzs(txs [][]byte, numTxs uint64) []byte {
// 	auctionInfo := abci.AuctionInfo{
// 		ExtendedCommitInfo: suite.createExtendedCommitInfoFromTxBzs(txs),
// 		NumTxs:             numTxs,
// 		MaxTxBytes:         int64(len(txs[0])),
// 	}

// 	auctionInfoBz, err := auctionInfo.Marshal()
// 	suite.Require().NoError(err)

// 	return auctionInfoBz
// }

// func (suite *ABCITestSuite) getAllAuctionTxs() ([]sdk.Tx, [][]byte) {
// 	auctionIterator := suite.mempool.AuctionBidSelect(suite.ctx)
// 	txs := make([]sdk.Tx, 0)
// 	txBzs := make([][]byte, 0)

// 	for ; auctionIterator != nil; auctionIterator = auctionIterator.Next() {
// 		txs = append(txs, auctionIterator.Tx())

// 		bz, err := suite.encodingConfig.TxConfig.TxEncoder()(auctionIterator.Tx())
// 		suite.Require().NoError(err)

// 		txBzs = append(txBzs, bz)
// 	}

// 	return txs, txBzs
// }

// func (suite *ABCITestSuite) createExtendedCommitInfoFromTxs(txs []sdk.Tx) comettypes.ExtendedCommitInfo {
// 	voteExtensions := make([][]byte, 0)
// 	for _, tx := range txs {
// 		bz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
// 		suite.Require().NoError(err)

// 		voteExtensions = append(voteExtensions, bz)
// 	}

// 	return suite.createExtendedCommitInfo(voteExtensions)
// }

// func (suite *ABCITestSuite) createExtendedVoteInfo(voteExtensions [][]byte) []comettypes.ExtendedVoteInfo {
// 	commitInfo := make([]comettypes.ExtendedVoteInfo, 0)
// 	for _, voteExtension := range voteExtensions {
// 		info := comettypes.ExtendedVoteInfo{
// 			VoteExtension: voteExtension,
// 		}

// 		commitInfo = append(commitInfo, info)
// 	}

// 	return commitInfo
// }

// func (suite *ABCITestSuite) createExtendedCommitInfo(voteExtensions [][]byte) comettypes.ExtendedCommitInfo {
// 	commitInfo := comettypes.ExtendedCommitInfo{
// 		Votes: suite.createExtendedVoteInfo(voteExtensions),
// 	}

// 	return commitInfo
// }
