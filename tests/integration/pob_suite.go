package integration

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	initBalance = 1000000000000
)

// POBIntegrationTestSuite runs the POB integration test-suite against a given interchaintest specification
type POBIntegrationTestSuite struct {
	suite.Suite
	// spec
	spec *interchaintest.ChainSpec
	// chain
	chain ibc.Chain
	// interchain
	ic *interchaintest.Interchain
	// users
	user1, user2, user3 ibc.Wallet
}

func NewPOBIntegrationTestSuiteFromSpec(spec *interchaintest.ChainSpec) *POBIntegrationTestSuite {
	return &POBIntegrationTestSuite{
		spec: spec,
	}
}

func (s *POBIntegrationTestSuite) SetupSuite() {
	// build the chain
	s.T().Log("building chain with spec", s.spec)
	s.chain = ChainBuilderFromChainSpec(s.T(), s.spec)

	// build the interchain
	s.T().Log("building interchain")
	ctx := context.Background()
	s.ic = BuildPOBInterchain(s.T(), ctx, s.chain)

	// get the users
	s.user1 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, s.chain)[0]
	s.user2 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, s.chain)[0]
	s.user3 = interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), initBalance, s.chain)[0]
}

func (s *POBIntegrationTestSuite) TearDownSuite() {
	// close the interchain
	s.ic.Close()
}

func (s *POBIntegrationTestSuite) SetupSubTest() {
	// wait for 1 block height
	// query height
	height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
	require.NoError(s.T(), err)
	WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height + 1)
}

func (s *POBIntegrationTestSuite) TestQueryParams() {
	// query params
	params := QueryBuilderParams(s.T(), s.chain)

	// check default params are correct
	s.Require().Equal(buildertypes.DefaultMaxBundleSize, params.MaxBundleSize)
	s.Require().Equal(buildertypes.DefaultEscrowAccountAddress, sdk.AccAddress(params.EscrowAccountAddress))
	s.Require().Equal(buildertypes.DefaultReserveFee, params.ReserveFee)
	s.Require().Equal(buildertypes.DefaultMinBidIncrement, params.MinBidIncrement)
	s.Require().Equal(buildertypes.DefaultFrontRunningProtection, params.FrontRunningProtection)
	s.Require().Equal(buildertypes.DefaultProposerFee, params.ProposerFee)
}

// TestValidBids tests the execution of various valid auction bids. There are a few
// invariants that are tested:
//
//  1. The order of transactions in a bundle is preserved when bids are valid.
//  2. All transactions execute as expected.
//  3. The balance of the escrow account should be updated correctly.
//  4. Top of block bids will be included in block proposals before other transactions
func (s *POBIntegrationTestSuite) TestValidBids() {
	params := QueryBuilderParams(s.T(), s.chain)
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("Valid Auction Bid", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bundle w/ a single tx
		// create message send tx
		tx := banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))

		// create the MsgAuctioBid
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User: s.user1,
				Msgs: []sdk.Msg{
					tx,
				},
				SequenceIncrement: 1,
			},
		})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast + wait for the tx to be included in a block
		res := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid},
				Height: height + 1,
			},
		})
		height = height + 1

		// wait for next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height)

		// query + verify the block
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height))
		VerifyBlock(s.T(), block, 0, TxHash(res[0]), bundledTxs)

		// ensure that the escrow account has the correct balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Valid bid with multiple other transactions", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create the bundle w/ a single tx
		// bank-send msg
		msgs := make([]sdk.Msg, 2)
		msgs[0] = banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))
		msgs[1] = banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))

		// create the MsgAuctionBid
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []Tx{
			{
				User:              s.user1,
				Msgs:              msgs[0:1],
				SequenceIncrement: 1,
			},
		})

		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// create the messages to be broadcast
		msgsToBcast := make([]Tx, 0)
		msgsToBcast = append(msgsToBcast, Tx{
			User:   s.user1,
			Msgs:   []sdk.Msg{bid},
			Height: height + 1,
		})

		msgsToBcast = append(msgsToBcast, Tx{
			User:   s.user2,
			Msgs:   msgs[1:2],
			Height: height + 1,
		})

		regular_txs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), msgsToBcast)

		// get the block at the next height
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// verify the block
		bidTxHash := TxHash(regular_txs[0])
		VerifyBlock(s.T(), block, 0, bidTxHash, append(bundledTxs, regular_txs[1:]...))

		// ensure that escrow account has the correct balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("iterative bidding from the same account", func() {
		// get escrow account balance before bid
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create multi-tx valid bundle
		// bank-send msg
		txs := make([]Tx, 2)
		txs[0] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		txs[1] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 2,
		}
		// create bundle
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, txs)
		// create 2 more bundle w same txs from same user
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), txs)
		bid3, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement).Add(params.MinBidIncrement), txs)

		// query height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast all bids
		broadcastedTxs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid},
				Height:             height + 1,
				SkipInclusionCheck: true,
			},
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid2},
				Height:             height + 1,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid3},
				Height: height + 1,
			},
		})

		// Verify the block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		VerifyBlock(s.T(), Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1)), 0, TxHash(broadcastedTxs[2]), bundledTxs)

		//  check escrow account balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement.Add(params.MinBidIncrement)).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("bid with a bundle with transactions that are already in the mempool", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create valid bundle
		// bank-send msg
		txs := make([]Tx, 2)
		txs[0] = Tx{
			User: s.user1,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
		}
		txs[1] = Tx{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}

		// create bundle
		bidAmt := params.ReserveFee
		bid, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt, txs)

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast txs in the bundle to network + bundle + extra
		broadcastedTxs := BroadcastTxs(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []Tx{txs[0], txs[1], {
			User:   s.user2,
			Msgs:   []sdk.Msg{bid},
			Height: height + 1,
		}, {
			User: s.user3,
			Msgs: []sdk.Msg{banktypes.NewMsgSend(s.user3.Address(), s.user1.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
		}})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check block
		VerifyBlock(s.T(), block, 0, TxHash(broadcastedTxs[2]), append(bundledTxs, broadcastedTxs[3]))

		// check escrow account balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})
}

// TestMultipleBids tests the execution of various valid auction bids in the same block. There are a few
// invariants that are tested:
//
//  1. The order of transactions in a bundle is preserved when bids are valid.
//  2. All transactions execute as expected.
//  3. The balance of the escrow account should be updated correctly.
//  4. Top of block bids will be included in block proposals before other transactions
//     that are included in the same block.
//  5. If there is a block that has multiple valid bids with timeouts that are sufficiently far apart,
//     the bids should be executed respecting the highest bids until the timeout is reached.
func (s *POBIntegrationTestSuite) TestMultipleBids() {
	params := QueryBuilderParams(s.T(), s.chain)
	escrowAddr := sdk.AccAddress(params.EscrowAccountAddress).String()

	s.Run("broadcasting bids to two different validators (both should execute over several blocks) with same bid", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := MessagesForUser{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg})

		// create bid 2
		msg2 := MessagesForUser{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid2 w/ higher bid than bid1
		bid2, bundledTxs2 := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []MessagesForUser{msg2})

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastMsgsPerUser(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []MessagesForUser{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid1},
				Height:             height + 2,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, TxHash(txs[1]), bundledTxs2)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement.Add(bidAmt)).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)

		// check next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+2)
		block = Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+2))

		// check bid1 was included second
		VerifyBlock(s.T(), block, TxHash(txs[0]), bundledTxs)
	})

	s.Run("Multiple bid transactions with second bid being smaller than min bid increment", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := MessagesForUser{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg})

		// create bid 2
		msg2 := MessagesForUser{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg2})

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastMsgsPerUser(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []MessagesForUser{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid1},
				Height: height + 1,
			},
			{
				User:       s.user2,
				Msgs:       []sdk.Msg{bid2},
				Height:     height + 1,
				ExpectFail: true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple bid transactions from diff accounts with second bid being smaller than min bid increment", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := MessagesForUser{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg})

		// create bid 2
		msg2 := MessagesForUser{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg2})

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastMsgsPerUser(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []MessagesForUser{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid1},
				Height: height + 1,
			},
			{
				User:              s.user1,
				Msgs:              []sdk.Msg{bid2},
				SequenceIncrement: 1,
				Height:            height + 1,
				ExpectFail:        true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple transactions with increasing bids but first bid has same bundle so it should fail in later block", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := MessagesForUser{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg})

		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []MessagesForUser{msg})

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastMsgsPerUser(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []MessagesForUser{
			{
				User:   s.user1,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
			{
				User:              s.user1,
				Msgs:              []sdk.Msg{bid1},
				Height:            height + 1,
				SequenceIncrement: 1,
				ExpectFail:        true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple transactions from diff. account with increasing bids but first bid has same bundle so it should fail in later block", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := MessagesForUser{
			User:              s.user3,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user3.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
		}

		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg})

		// create bid2 w/ higher bid than bid1
		bid2, _ := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []MessagesForUser{msg})

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastMsgsPerUser(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []MessagesForUser{
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
			{
				User:       s.user1,
				Msgs:       []sdk.Msg{bid1},
				Height:     height + 1,
				ExpectFail: true,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})

	s.Run("Multiple transactions with increasing bids and different bundles", func() {
		// escrow account balance
		escrowAcctBalanceBeforeBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)

		// create bid 1
		// bank-send msg
		msg := MessagesForUser{
			User:              s.user1,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user1.Address(), s.user2.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}
		// create bid1
		bidAmt := params.ReserveFee
		bid1, bundledTxs := CreateAuctionBidMsg(s.T(), context.Background(), s.user1, s.chain.(*cosmos.CosmosChain), bidAmt, []MessagesForUser{msg})

		// create bid2
		// create a second message
		msg2 := MessagesForUser{
			User:              s.user2,
			Msgs:              []sdk.Msg{banktypes.NewMsgSend(s.user2.Address(), s.user3.Address(), sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))))},
			SequenceIncrement: 1,
		}

		// create bid2 w/ higher bid than bid1
		bid2, bundledTxs2 := CreateAuctionBidMsg(s.T(), context.Background(), s.user2, s.chain.(*cosmos.CosmosChain), bidAmt.Add(params.MinBidIncrement), []MessagesForUser{msg2})

		// get chain height
		height, err := s.chain.(*cosmos.CosmosChain).Height(context.Background())
		require.NoError(s.T(), err)

		// broadcast both bids
		txs := BroadcastMsgsPerUser(s.T(), context.Background(), s.chain.(*cosmos.CosmosChain), []MessagesForUser{
			{
				User:               s.user1,
				Msgs:               []sdk.Msg{bid1},
				Height:             height + 2,
				SkipInclusionCheck: true,
			},
			{
				User:   s.user2,
				Msgs:   []sdk.Msg{bid2},
				Height: height + 1,
			},
		})

		// query next block
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+1)
		block := Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+1))

		// check bid2 was included first
		VerifyBlock(s.T(), block, TxHash(txs[1]), bundledTxs2)

		// query next block and check tx inclusion
		WaitForHeight(s.T(), s.chain.(*cosmos.CosmosChain), height+2)
		block = Block(s.T(), s.chain.(*cosmos.CosmosChain), int64(height+2))

		// check bid1 was included second
		VerifyBlock(s.T(), block, TxHash(txs[0]), bundledTxs)

		// check escrow balance
		escrowAcctBalanceAfterBid := QueryAccountBalance(s.T(), s.chain, escrowAddr, params.ReserveFee.Denom)
		expectedIncrement := escrowAddressIncrement(bidAmt.Add(params.MinBidIncrement.Add(bidAmt)).Amount, params.ProposerFee)
		require.Equal(s.T(), escrowAcctBalanceBeforeBid+expectedIncrement, escrowAcctBalanceAfterBid)
	})
}

func escrowAddressIncrement(bid sdk.Int, proposerFee sdk.Dec) int64 {
	return int64(bid.Sub(sdk.NewDecFromInt(bid).Mul(proposerFee).RoundInt()).Int64())
}
