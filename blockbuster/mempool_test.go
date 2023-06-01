package blockbuster_test

import "fmt"

func (suite *BlockBusterTestSuite) TestInsert() {
	cases := []struct {
		name       string
		numTobTxs  int
		numBaseTxs int
	}{
		{
			"insert 1 tob tx",
			1,
			0,
		},
		{
			"insert 10 tob txs",
			10,
			0,
		},
		{
			"insert 1 base tx",
			0,
			1,
		},
		{
			"insert 10 base txs and 10 tob txs",
			10,
			10,
		},
		{
			"insert 100 base txs and 100 tob txs",
			100,
			100,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Fill the base lane with numBaseTxs transactions
			suite.fillBaseLane(tc.numBaseTxs)

			// Fill the TOB lane with numTobTxs transactions
			suite.fillTOBLane(tc.numTobTxs)

			// Validate the mempool
			suite.Require().Equal(suite.mempool.CountTx(), tc.numBaseTxs+tc.numTobTxs)

			// Validate the lanes
			suite.Require().Equal(suite.baseLane.CountTx(), tc.numBaseTxs)
			suite.Require().Equal(suite.tobLane.CountTx(), tc.numTobTxs)

			// Validate the lane counts
			laneCounts := suite.mempool.GetTxDistribution()
			fmt.Println(laneCounts)

			// Ensure that the lane counts are correct
			suite.Require().Equal(laneCounts[suite.tobLane.Name()], tc.numTobTxs)
			suite.Require().Equal(laneCounts[suite.baseLane.Name()], tc.numBaseTxs)
		})
	}
}

func (suite *BlockBusterTestSuite) TestRemove() {
	cases := []struct {
		name       string
		numTobTxs  int
		numBaseTxs int
	}{
		{
			"insert 1 tob tx",
			1,
			0,
		},
		{
			"insert 10 tob txs",
			10,
			0,
		},
		{
			"insert 1 base tx",
			0,
			1,
		},
		{
			"insert 10 base txs and 10 tob txs",
			10,
			10,
		},
		{
			"insert 100 base txs and 100 tob txs",
			100,
			100,
		},
	}

	for _, tc := range cases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Fill the base lane with numBaseTxs transactions
			suite.fillBaseLane(tc.numBaseTxs)

			// Fill the TOB lane with numTobTxs transactions
			suite.fillTOBLane(tc.numTobTxs)

			// Remove all transactions from the lanes
			tobCount := tc.numTobTxs
			baseCount := tc.numBaseTxs
			for iterator := suite.baseLane.Select(suite.ctx, nil); iterator != nil; {
				tx := iterator.Tx()

				// Remove the transaction from the mempool
				suite.Require().NoError(suite.mempool.Remove(tx))

				// Ensure that the transaction is no longer in the mempool
				contains, err := suite.mempool.Contains(tx)
				suite.Require().NoError(err)
				suite.Require().False(contains)

				// Ensure the number of transactions in the lane is correct
				baseCount--
				suite.Require().Equal(suite.baseLane.CountTx(), baseCount)

				distribution := suite.mempool.GetTxDistribution()
				suite.Require().Equal(distribution[suite.baseLane.Name()], baseCount)

				iterator = suite.baseLane.Select(suite.ctx, nil)
			}

			suite.Require().Equal(0, suite.baseLane.CountTx())
			suite.Require().Equal(tobCount, suite.tobLane.CountTx())

			// Remove all transactions from the lanes
			for iterator := suite.tobLane.Select(suite.ctx, nil); iterator != nil; {
				tx := iterator.Tx()

				// Remove the transaction from the mempool
				suite.Require().NoError(suite.mempool.Remove(tx))

				// Ensure that the transaction is no longer in the mempool
				contains, err := suite.mempool.Contains(tx)
				suite.Require().NoError(err)
				suite.Require().False(contains)

				// Ensure the number of transactions in the lane is correct
				tobCount--
				suite.Require().Equal(suite.tobLane.CountTx(), tobCount)

				distribution := suite.mempool.GetTxDistribution()
				suite.Require().Equal(distribution[suite.tobLane.Name()], tobCount)

				iterator = suite.tobLane.Select(suite.ctx, nil)
			}

			suite.Require().Equal(0, suite.tobLane.CountTx())
			suite.Require().Equal(0, suite.baseLane.CountTx())
			suite.Require().Equal(0, suite.mempool.CountTx())

			// Validate the lane counts
			distribution := suite.mempool.GetTxDistribution()

			// Ensure that the lane counts are correct
			suite.Require().Equal(distribution[suite.tobLane.Name()], 0)
			suite.Require().Equal(distribution[suite.baseLane.Name()], 0)
		})
	}
}
