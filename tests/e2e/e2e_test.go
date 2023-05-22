//go:build e2e

package e2e

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/tests/app"
)

func (s *IntegrationTestSuite) TestGetBuilderParams() {
	params := s.queryBuilderParams()
	s.Require().NotNil(params)
}

// TestBundles tests the execution of various auction bids. There are a few
// invariants that are tested:
//
//  1. The order of transactions in a bundle is preserved when bids are valid.
//  2. All transactions execute as expected.
//  3. The balance of the escrow account should be updated correctly.
//  4. Top of block bids will be included in block proposals before other transactions
//     that are included in the same block.
func (s *IntegrationTestSuite) TestBundles() {
	// Create the accounts that will create transactions to be included in bundles
	initBalance := sdk.NewInt64Coin(app.BondDenom, 10000000000)
	numAccounts := 4
	accounts := s.createTestAccounts(numAccounts, initBalance)

	// basic send amount
	defaultSendAmount := sdk.NewCoins(sdk.NewCoin(app.BondDenom, sdk.NewInt(10)))

	// auction parameters
	params := s.queryBuilderParams()
	reserveFee := params.ReserveFee
	minBidIncrement := params.MinBidIncrement
	maxBundleSize := params.MaxBundleSize
	escrowAddress := params.EscrowAccountAddress

	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "Valid auction bid",
			test: func() {
				// Create a bundle with a single transaction
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 1, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[0], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("Valid auction bid", bidTx, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: true,
					bundleHashes[1]: true,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(expectedEscrowFee, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "invalid auction bid with a bid smaller than the reserve fee",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction (this should not be included in the block proposal)
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 1, 1000),
				}

				// Create a bid transaction that includes a bid that is smaller than the reserve fee
				bid := reserveFee.Sub(sdk.NewInt64Coin(app.BondDenom, 1))
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[0], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("invalid auction bid", bidTx, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that no transactions were executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: false,
					bundleHashes[1]: false,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "invalid auction bid with too many transactions in the bundle",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with too many transactions
				bundle := [][]byte{}
				for i := 0; i < int(maxBundleSize)+1; i++ {
					bundle = append(bundle, s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, uint64(i+1), 1000))
				}

				// Create a bid transaction that includes the bundle
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[0], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("invalid auction bid", bidTx, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that no transactions were executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := make(map[string]bool)

				for _, hash := range bundleHashes {
					expectedExecution[hash] = false
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "invalid auction bid that has an invalid timeout",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and has a bad timeout
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[0], bid, bundle, 0, height)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("invalid auction bid", bidTx, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that no transactions were executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: false,
					bundleHashes[1]: false,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Multiple transactions with second bid being smaller than min bid increment (same account)",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 1, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[0], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("bid 1", bidTx, bundle)

				// Create a second bid transaction that includes the bundle and is valid (but smaller than the min bid increment)
				badBid := reserveFee.Add(sdk.NewInt64Coin(app.BondDenom, 10))
				bidTx2 := s.createAuctionBidTx(accounts[0], badBid, bundle, 0, height+1)
				s.broadcastTx(bidTx2, 0)
				s.displayExpectedBundle("bid 2", bidTx2, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the first bid was executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				bundleHashes2 := s.bundleToTxHashes(bidTx2, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]:  true,
					bundleHashes[1]:  true,
					bundleHashes2[0]: false,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))

				// Wait another block to make sure the second bid is not executed
				s.waitForABlock()
				s.verifyBlock(height+2, bundleHashes2, expectedExecution)
			},
		},
		{
			name: "Multiple transactions with second bid being smaller than min bid increment (different account)",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := [][]byte{
					s.createMsgSendTx(accounts[2], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[0], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("bid 1", bidTx, bundle)

				// Create a second bid transaction that includes the bundle and is valid (but smaller than the min bid increment)
				badBid := reserveFee.Add(sdk.NewInt64Coin(app.BondDenom, 10))
				bidTx2 := s.createAuctionBidTx(accounts[1], badBid, bundle, 0, height+1)
				s.broadcastTx(bidTx2, 0)
				s.displayExpectedBundle("bid 2", bidTx2, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the first bid was executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				bundleHashes2 := s.bundleToTxHashes(bidTx2, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]:  true,
					bundleHashes[1]:  true,
					bundleHashes2[0]: false,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))

				// Wait another block to make sure the second bid is not executed
				s.waitForABlock()
				s.verifyBlock(height+2, bundleHashes2, expectedExecution)
			},
		},
		{
			name: "Multiple transactions with increasing bids but first bid has same bundle so it should fail (same account)",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 1, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[0], bid, bundle, 0, height+2)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("bid 1", bidTx, bundle)

				// Create a second bid transaction that includes the bundle and is valid
				bid2 := reserveFee.Add(minBidIncrement)
				bidTx2 := s.createAuctionBidTx(accounts[0], bid2, bundle, 0, height+1)
				s.broadcastTx(bidTx2, 0)
				s.displayExpectedBundle("bid 2", bidTx2, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the second bid was executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				bundleHashes2 := s.bundleToTxHashes(bidTx2, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]:  false,
					bundleHashes2[0]: true,
					bundleHashes2[1]: true,
				}
				s.verifyBlock(height+1, bundleHashes2, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid2)
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))

				// Wait for a block to be created and ensure that the first bid was not executed
				s.waitForABlock()
				s.verifyBlock(height+2, bundleHashes, expectedExecution)
			},
		},
		{
			name: "Multiple transactions with increasing bids but first bid has same bundle so it should fail (different account)",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[2], bid, bundle, 0, height+2)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("bid 1", bidTx, bundle)

				// Create a second bid transaction that includes the bundle and is valid
				bid2 := reserveFee.Add(minBidIncrement)
				bidTx2 := s.createAuctionBidTx(accounts[1], bid2, bundle, 0, height+1)
				s.broadcastTx(bidTx2, 0)
				s.displayExpectedBundle("bid 2", bidTx2, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the second bid was executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				bundleHashes2 := s.bundleToTxHashes(bidTx2, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]:  false,
					bundleHashes2[0]: true,
					bundleHashes2[1]: true,
				}
				s.verifyBlock(height+1, bundleHashes2, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid2)
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))

				// Wait for a block to be created and ensure that the first bid was not executed
				s.waitForABlock()
				s.verifyBlock(height+2, bundleHashes, expectedExecution)
			},
		},
		{
			name: "Multiple transactions with increasing bids and different bundles (one should execute)",
			test: func() {
				// Get escrow account balance
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction
				firstBundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bundle with a single transaction
				secondBundle := [][]byte{
					s.createMsgSendTx(accounts[1], accounts[0].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[2], bid, firstBundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("bid 1", bidTx, firstBundle)

				// Create a second bid transaction that includes the bundle and is valid
				bid2 := reserveFee.Add(minBidIncrement)
				bidTx2 := s.createAuctionBidTx(accounts[3], bid2, secondBundle, 0, height+1)
				s.broadcastTx(bidTx2, 0)
				s.displayExpectedBundle("bid 2", bidTx2, secondBundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure only the second bid was executed
				bundleHashes := s.bundleToTxHashes(bidTx, firstBundle)
				bundleHashes2 := s.bundleToTxHashes(bidTx2, secondBundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]:  false,
					bundleHashes[1]:  false,
					bundleHashes2[0]: true,
					bundleHashes2[1]: true,
				}
				s.verifyBlock(height+1, bundleHashes2, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid2)
				s.Require().Equal(expectedEscrowFee.Add(escrowBalance), s.queryBalanceOf(escrowAddress, app.BondDenom))

				// Wait for a block to be created and ensure that the second bid is executed
				s.waitForABlock()
				s.verifyBlock(height+2, bundleHashes, expectedExecution)
			},
		},
		{
			name: "Invalid bid that includes an invalid bundle tx",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction that is invalid (sequence number is wrong)
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 1000, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[1], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("invalid auction bid", bidTx, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: false,
					bundleHashes[1]: false,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Invalid bid that is attempting to front-run/sandwich",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a front-running bundle
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
					s.createMsgSendTx(accounts[1], accounts[0].Address.String(), defaultSendAmount, 0, 1000),
					s.createMsgSendTx(accounts[2], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[1], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("front-running auction bid", bidTx, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: false,
					bundleHashes[1]: false,
					bundleHashes[2]: false,
					bundleHashes[3]: false,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Invalid bid that is attempting to bid more than their balance",
			test: func() {
				// Get escrow account balance to ensure that it is not changed
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a single transaction that is valid
				bundle := [][]byte{
					s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, 0, 1000),
				}

				// Create a bid transaction that includes the bundle and is valid
				bid := sdk.NewCoin(app.BondDenom, sdk.NewInt(999999999999999999))
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[1], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("bad auction bid", bidTx, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: false,
					bundleHashes[1]: false,
				}
				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "Valid bid with multiple other transactions",
			test: func() {
				// Get escrow account balance to ensure that it is updated correctly
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a multiple transaction that is valid
				bundle := make([][]byte, maxBundleSize)
				for i := 0; i < int(maxBundleSize); i++ {
					bundle[i] = s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, uint64(i), 1000)
				}

				// Wait for a block to ensure all transactions are included in the same block
				s.waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[1], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("gud auction bid", bidTx, bundle)

				// Execute a few other messages to be included in the block after the bid and bundle
				normalTxs := make([][]byte, 3)
				normalTxs[0] = s.createMsgSendTx(accounts[2], accounts[1].Address.String(), defaultSendAmount, 0, 1000)
				normalTxs[1] = s.createMsgSendTx(accounts[2], accounts[1].Address.String(), defaultSendAmount, 1, 1000)
				normalTxs[2] = s.createMsgSendTx(accounts[2], accounts[1].Address.String(), defaultSendAmount, 2, 1000)

				for _, tx := range normalTxs {
					s.broadcastTx(tx, 0)
				}

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: true,
				}

				for _, hash := range bundleHashes[1:] {
					expectedExecution[hash] = true
				}

				for _, hash := range s.normalTxsToTxHashes(normalTxs) {
					expectedExecution[hash] = true
				}

				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(escrowBalance.Add(expectedEscrowFee), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "attack vector, searcher attempts to include several txs in the same block to invalidate auction",
			test: func() {
				// Get escrow account balance to ensure that it is updated correctly
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a multiple transaction that is valid
				bundle := make([][]byte, maxBundleSize)
				for i := 0; i < int(maxBundleSize); i++ {
					bundle[i] = s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, uint64(i), 1000)
				}

				// Wait for a block to ensure all transactions are included in the same block
				s.waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[1], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)

				// Execute a few other messages to be included in the block after the bid and bundle
				normalTxs := make([][]byte, 3)
				normalTxs[0] = s.createMsgSendTx(accounts[1], accounts[1].Address.String(), defaultSendAmount, 0, 1000)
				normalTxs[1] = s.createMsgSendTx(accounts[1], accounts[1].Address.String(), defaultSendAmount, 1, 1000)
				normalTxs[2] = s.createMsgSendTx(accounts[1], accounts[1].Address.String(), defaultSendAmount, 2, 1000)

				for _, tx := range normalTxs {
					s.broadcastTx(tx, 0)
				}

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: true,
				}

				// The entire bundle should land irrespective of the transactions submitted by the searcher
				for _, hash := range bundleHashes[1:] {
					expectedExecution[hash] = true
				}

				// We expect only the first normal transaction to not be executed (due to a sequence number mismatch)
				normalHashes := s.normalTxsToTxHashes(normalTxs)
				expectedExecution[normalHashes[0]] = false
				for _, hash := range normalHashes[1:] {
					expectedExecution[hash] = true
				}

				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(escrowBalance.Add(expectedEscrowFee), s.queryBalanceOf(escrowAddress, app.BondDenom))

			},
		},
		{
			name: "iterative bidding from the same account",
			test: func() {
				// Get escrow account balance to ensure that it is updated correctly
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a multiple transaction that is valid
				bundle := make([][]byte, maxBundleSize)
				for i := 0; i < int(maxBundleSize); i++ {
					bundle[i] = s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, uint64(i), 1000)
				}

				// Wait for a block to ensure all transactions are included in the same block
				s.waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[1], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("gud auction bid 1", bidTx, bundle)

				// Create another bid transaction that includes the bundle and is valid from the same account
				// to verify that user can bid with the same account multiple times in the same block
				bid2 := bid.Add(minBidIncrement)
				bidTx2 := s.createAuctionBidTx(accounts[1], bid2, bundle, 0, height+1)
				s.broadcastTx(bidTx2, 0)
				s.displayExpectedBundle("gud auction bid 2", bidTx2, bundle)

				// Create a third bid
				bid3 := bid2.Add(minBidIncrement)
				bidTx3 := s.createAuctionBidTx(accounts[1], bid3, bundle, 0, height+1)
				s.broadcastTx(bidTx3, 0)
				s.displayExpectedBundle("gud auction bid 3", bidTx3, bundle)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				bundleHashes2 := s.bundleToTxHashes(bidTx2, bundle)
				bundleHashes3 := s.bundleToTxHashes(bidTx3, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]:  false,
					bundleHashes2[0]: false,
					bundleHashes3[0]: true,
				}

				for _, hash := range bundleHashes3[1:] {
					expectedExecution[hash] = true
				}

				s.verifyBlock(height+1, bundleHashes3, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid3)
				s.Require().Equal(escrowBalance.Add(expectedEscrowFee), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "multi-block auction bids",
			test: func() {
				// Get escrow account balance to ensure that it is updated correctly
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a multiple transaction that is valid
				bundle := make([][]byte, maxBundleSize)
				for i := 0; i < int(maxBundleSize); i++ {
					bundle[i] = s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, uint64(i), 1000)
				}

				bundle2 := make([][]byte, maxBundleSize)
				for i := 0; i < int(maxBundleSize); i++ {
					bundle2[i] = s.createMsgSendTx(accounts[1], accounts[0].Address.String(), defaultSendAmount, uint64(i), 1000)
				}

				// Wait for a block to ensure all transactions are included in the same block
				s.waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[2], bid, bundle, 0, height+2)
				s.broadcastTx(bidTx, 0)
				s.displayExpectedBundle("gud auction bid 1", bidTx, bundle)

				// Create another bid transaction that includes the bundle and is valid from a different account
				bid2 := bid.Add(minBidIncrement)
				bidTx2 := s.createAuctionBidTx(accounts[3], bid2, bundle2, 0, height+1)
				s.broadcastTx(bidTx2, 0)
				s.displayExpectedBundle("gud auction bid 2", bidTx2, bundle2)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				bundleHashes2 := s.bundleToTxHashes(bidTx2, bundle2)
				expectedExecution := map[string]bool{
					bundleHashes2[0]: true,
				}

				for _, hash := range bundleHashes2[1:] {
					expectedExecution[hash] = true
				}

				s.verifyBlock(height+1, bundleHashes2, expectedExecution)

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				expectedExecution = map[string]bool{
					bundleHashes[0]: true,
				}

				for _, hash := range bundleHashes[1:] {
					expectedExecution[hash] = true
				}

				s.verifyBlock(height+2, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance (both bids should have been extracted by this point)
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid).Add(s.calculateProposerEscrowSplit(bid2))
				s.Require().Equal(escrowBalance.Add(expectedEscrowFee), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "bid with a bundle with transactions that are already in the mempool",
			test: func() {
				// Get escrow account balance to ensure that it is updated correctly
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a multiple transaction that is valid
				bundle := make([][]byte, maxBundleSize)
				for i := 0; i < int(maxBundleSize); i++ {
					bundle[i] = s.createMsgSendTx(accounts[0], accounts[1].Address.String(), defaultSendAmount, uint64(i), 1000)
				}

				// Wait for a block to ensure all transactions are included in the same block
				s.waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[2], bid, bundle, 0, height+1)

				// Broadcast all of the transactions in the bundle to the mempool
				for _, tx := range bundle {
					s.broadcastTx(tx, 0)
				}

				// Broadcast the bid transaction
				s.broadcastTx(bidTx, 0)

				// Broadcast some other transactions to the mempool
				normalTxs := make([][]byte, 10)
				for i := 0; i < 10; i++ {
					normalTxs[i] = s.createMsgSendTx(accounts[1], accounts[3].Address.String(), defaultSendAmount, uint64(i), 1000)
					s.broadcastTx(normalTxs[i], 0)
				}

				// Wait for a block to be created
				s.waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: true,
				}

				for _, hash := range bundleHashes[1:] {
					expectedExecution[hash] = true
				}

				for _, hash := range s.normalTxsToTxHashes(normalTxs) {
					expectedExecution[hash] = true
				}

				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				expectedEscrowFee := s.calculateProposerEscrowSplit(bid)
				s.Require().Equal(escrowBalance.Add(expectedEscrowFee), s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
		{
			name: "searcher is attempting to submit a bundle that includes a bid",
			test: func() {
				// Get escrow account balance to ensure that it is updated correctly
				escrowBalance := s.queryBalanceOf(escrowAddress, app.BondDenom)

				// Create a bundle with a multiple transaction that is valid
				bundle := [][]byte{
					s.createAuctionBidTx(accounts[0], reserveFee, nil, 0, 1000),
				}

				// Wait for a block to ensure all transactions are included in the same block
				s.waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bid := reserveFee
				height := s.queryCurrentHeight()
				bidTx := s.createAuctionBidTx(accounts[2], bid, bundle, 0, height+1)
				s.broadcastTx(bidTx, 0)

				// Ensure that the block was built correctly and that the bid was not executed
				bundleHashes := s.bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: false,
					bundleHashes[1]: false,
				}

				s.verifyBlock(height+1, bundleHashes, expectedExecution)

				// Ensure that the escrow account has the correct balance
				s.Require().Equal(escrowBalance, s.queryBalanceOf(escrowAddress, app.BondDenom))
			},
		},
	}

	for _, tc := range testCases {
		s.waitForABlock()
		s.Run(tc.name, tc.test)
	}
}
