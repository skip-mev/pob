package base_test

import (
	"fmt"
	"math/rand"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/lanes/base"
	"github.com/skip-mev/pob/blockbuster/utils/mocks"
	testutils "github.com/skip-mev/pob/testutils"
	"github.com/stretchr/testify/suite"
)

type BaseTestSuite struct {
	suite.Suite

	encodingConfig testutils.EncodingConfig
	random         *rand.Rand
	accounts       []testutils.Account
}

func TestBaseTestSuite(t *testing.T) {
	suite.Run(t, new(BaseTestSuite))
}

func (suite *BaseTestSuite) SetupTest() {
	// Set up basic TX encoding config.
	suite.encodingConfig = testutils.CreateTestEncodingConfig()

	// Create a few random accounts
	suite.random = rand.New(rand.NewSource(1))
	suite.accounts = testutils.RandomAccounts(suite.random, 5)
}

func (suite *BaseTestSuite) initLane(
	maxBlockSpace math.LegacyDec,
	expectedExecution map[sdk.Tx]bool,
) *base.DefaultLane {
	anteHandler := mocks.NewAnteHandler(suite.T())

	for tx, pass := range expectedExecution {
		var err error
		if !pass {
			err = fmt.Errorf("transaction predetermined to fail")
		}

		anteHandler.On("AnteHandler", sdk.Context{}, tx, false).
			Maybe().
			Return(
				sdk.Context{},
				err,
			)
	}

	config := blockbuster.NewBaseLaneConfig(
		log.NewTestLogger(suite.T()),
		suite.encodingConfig.TxConfig.TxEncoder(),
		suite.encodingConfig.TxConfig.TxDecoder(),
		anteHandler.AnteHandler,
		maxBlockSpace,
	)

	return base.NewDefaultLane(config)
}

func (suite *BaseTestSuite) TestPrepareLane() {
	suite.Run("should not build a proposal when amount configured to lane is too small", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[0],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := suite.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		suite.Require().NoError(lane.Insert(sdk.Context{}, tx))

		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
		suite.Require().NoError(err)

		// Create a proposal
		maxTxBytes := int64(len(txBz) - 1)
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		suite.Require().NoError(err)

		// Ensure the proposal is empty
		suite.Require().Equal(0, proposal.GetNumTxs())
		suite.Require().Equal(int64(0), proposal.GetTotalTxBytes())
	})

	suite.Run("should be able to build a proposal with a tx that just fits in", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[0],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: true,
		}
		lane := suite.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		suite.Require().NoError(lane.Insert(sdk.Context{}, tx))

		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
		suite.Require().NoError(err)

		// Create a proposal
		maxTxBytes := int64(len(txBz))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		suite.Require().NoError(err)

		// Ensure the proposal is not empty and contains the transaction
		suite.Require().Equal(1, proposal.GetNumTxs())
		suite.Require().Equal(maxTxBytes, proposal.GetTotalTxBytes())
		suite.Require().Equal(txBz, proposal.GetTxs()[0])
	})

	suite.Run("should not build a proposal with a that fails verify tx", func() {
		// Create a basic transaction that should not in the proposal
		tx, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[0],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx: false,
		}
		lane := suite.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		suite.Require().NoError(lane.Insert(sdk.Context{}, tx))

		// Create a proposal
		txBz, err := suite.encodingConfig.TxConfig.TxEncoder()(tx)
		suite.Require().NoError(err)

		maxTxBytes := int64(len(txBz))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		suite.Require().NoError(err)

		// Ensure the proposal is empty
		suite.Require().Equal(0, proposal.GetNumTxs())
		suite.Require().Equal(int64(0), proposal.GetTotalTxBytes())

		// Ensure the transaction is removed from the lane
		suite.Require().False(lane.Contains(tx))
		suite.Require().Equal(0, lane.CountTx())
	})

	suite.Run("should order transactions correctly in the proposal", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[0],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[1],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := suite.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		suite.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(10), tx1))
		suite.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(5), tx2))

		txBz1, err := suite.encodingConfig.TxConfig.TxEncoder()(tx1)
		suite.Require().NoError(err)

		txBz2, err := suite.encodingConfig.TxConfig.TxEncoder()(tx2)
		suite.Require().NoError(err)

		maxTxBytes := int64(len(txBz1)) + int64(len(txBz2))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		suite.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		suite.Require().Equal(2, proposal.GetNumTxs())
		suite.Require().Equal(maxTxBytes, proposal.GetTotalTxBytes())
		suite.Require().Equal([][]byte{txBz1, txBz2}, proposal.GetTxs())
	})

	suite.Run("should order transactions correctly in the proposal (with different insertion)", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[0],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[1],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := suite.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		suite.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(5), tx1))
		suite.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(10), tx2))

		txBz1, err := suite.encodingConfig.TxConfig.TxEncoder()(tx1)
		suite.Require().NoError(err)

		txBz2, err := suite.encodingConfig.TxConfig.TxEncoder()(tx2)
		suite.Require().NoError(err)

		maxTxBytes := int64(len(txBz1)) + int64(len(txBz2))
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		suite.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		suite.Require().Equal(2, proposal.GetNumTxs())
		suite.Require().Equal(maxTxBytes, proposal.GetTotalTxBytes())
		suite.Require().Equal([][]byte{txBz2, txBz1}, proposal.GetTxs())
	})

	suite.Run("should include tx that fits in proposal when other does not", func() {
		// Create a basic transaction that should not in the proposal
		tx1, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[0],
			0,
			1,
			0,
		)
		suite.Require().NoError(err)

		tx2, err := testutils.CreateRandomTx(
			suite.encodingConfig.TxConfig,
			suite.accounts[1],
			0,
			10, // This tx is too large to fit in the proposal
			0,
		)
		suite.Require().NoError(err)

		// Create a lane with a max block space of 1 but a proposal that is smaller than the tx
		expectedExecution := map[sdk.Tx]bool{
			tx1: true,
			tx2: true,
		}
		lane := suite.initLane(math.LegacyMustNewDecFromStr("1"), expectedExecution)

		// Insert the transaction into the lane
		suite.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(10), tx1))
		suite.Require().NoError(lane.Insert(sdk.Context{}.WithPriority(5), tx2))

		txBz1, err := suite.encodingConfig.TxConfig.TxEncoder()(tx1)
		suite.Require().NoError(err)

		txBz2, err := suite.encodingConfig.TxConfig.TxEncoder()(tx2)
		suite.Require().NoError(err)

		maxTxBytes := int64(len(txBz1)) + int64(len(txBz2)) - 1
		proposal, err := lane.PrepareLane(sdk.Context{}, blockbuster.NewProposal(maxTxBytes), maxTxBytes, blockbuster.NoOpPrepareLanesHandler())
		suite.Require().NoError(err)

		// Ensure the proposal is ordered correctly
		suite.Require().Equal(1, proposal.GetNumTxs())
		suite.Require().Equal(int64(len(txBz1)), proposal.GetTotalTxBytes())
		suite.Require().Equal([][]byte{txBz1}, proposal.GetTxs())
	})
}

func (suite *BaseTestSuite) TestProcessLane() {}

func (suite *BaseTestSuite) TestProcessLaneBasic() {}
