package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	tmclient "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createClientContext creates a client.Context for use in integration tests.
// Note, it assumes all queries and broadcasts go to the first node.
func (s *IntegrationTestSuite) createClientContext() client.Context {
	node := s.valResources[0]

	rpcURI := node.GetHostPort("26657/tcp")
	gRPCURI := node.GetHostPort("9090/tcp")

	rpcClient, err := client.NewClientFromNode(rpcURI)
	s.Require().NoError(err)

	grpcClient, err := grpc.Dial(gRPCURI, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	s.Require().NoError(err)

	return client.Context{}.
		WithNodeURI(rpcURI).
		WithClient(rpcClient).
		WithGRPCClient(grpcClient).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithCodec(encodingConfig.Codec).
		WithChainID(s.chain.id).
		WithBroadcastMode(flags.BroadcastSync)
}

// createTestAccounts creates and funds test accounts with a balance.
func (s *IntegrationTestSuite) createTestAccounts(numAccounts int, balance sdk.Coin) []TestAccount {
	accounts := make([]TestAccount, numAccounts)

	for i := 0; i < numAccounts; i++ {
		// Generate a new account with private key that will be used to sign transactions.
		privKey := secp256k1.GenPrivKey()
		pubKey := privKey.PubKey()
		addr := sdk.AccAddress(pubKey.Address())

		account := TestAccount{
			PrivateKey: privKey,
			Address:    addr,
		}

		// Fund the account.
		s.execMsgSendTx(0, account.Address, balance)

		// Wait for the balance to be updated.
		s.Require().Eventually(func() bool {
			return !s.queryBalancesOf(addr.String()).IsZero()
		},
			10*time.Second,
			1*time.Second,
		)

		accounts[i] = account
	}

	return accounts
}

// calculateProposerEscrowSplit calculates the amount of a bid that should go to the escrow account
// and the amount that should go to the proposer. The simulation e2e environment does not support
// checking the proposer's balance, it only validates that the escrow address has the correct balance.
func (s *IntegrationTestSuite) calculateProposerEscrowSplit(bid sdk.Coin) sdk.Coin {
	// Get the params to determine the proposer fee.
	params := s.queryBuilderParams()
	proposerFee := params.ProposerFee

	var proposerReward sdk.Coins
	if proposerFee.IsZero() {
		// send the entire bid to the escrow account when no proposer fee is set
		return bid
	}

	// determine the amount of the bid that goes to the (previous) proposer
	bidDec := sdk.NewDecCoinsFromCoins(bid)
	proposerReward, _ = bidDec.MulDecTruncate(proposerFee).TruncateDecimal()

	// Determine the amount of the remaining bid that goes to the escrow account.
	// If a decimal remainder exists, it'll stay with the bidding account.
	escrowTotal := bidDec.Sub(sdk.NewDecCoinsFromCoins(proposerReward...))
	escrowReward, _ := escrowTotal.TruncateDecimal()

	return sdk.NewCoin(bid.Denom, escrowReward.AmountOf(bid.Denom))
}

// waitForABlock will wait until the current block height has increased by a single block.
func (s *IntegrationTestSuite) waitForABlock() {
	height := s.queryCurrentHeight()
	s.Require().Eventually(
		func() bool {
			return s.queryCurrentHeight() >= height+1
		},
		10*time.Second,
		50*time.Millisecond,
	)
}

// bundleToTxHashes converts a bundle to a slice of transaction hashes.
func (s *IntegrationTestSuite) bundleToTxHashes(bundle []string) []string {
	hashes := make([]string, len(bundle))

	for i, tx := range bundle {
		hashBz, err := hex.DecodeString(tx)
		s.Require().NoError(err)

		shaBz := sha256.Sum256(hashBz)
		hashes[i] = hex.EncodeToString(shaBz[:])
	}

	return hashes
}

// verifyBlock verifies that the transactions in the block at the given height were seen
// and executed in the order they were submitted i.e. how they are broadcasted in the bundle.
func (s *IntegrationTestSuite) verifyBlock(height int64, bidTx string, bundle []string, expectedExecution map[string]bool) {
	s.T().Logf("Verifying block %d", height)

	// Get the block's transactions and display the expected and actual block for debugging.
	txs := s.queryBlockTxs(height)
	s.displayBlock(txs, bidTx, bundle)

	// Ensure that all transactions executed as expected (i.e. landed or failed to land).
	for tx, landed := range expectedExecution {
		s.T().Logf("Verifying tx %s executed as %t", tx, landed)
		s.Require().Equal(landed, s.queryTxPassed(tx) == nil)
	}
	s.T().Logf("All txs executed as expected")

	// Check that the block contains the expected transactions in the expected order
	// iff the bid transaction was expected to execute.
	if expectedExecution[bidTx] {
		hashBz := sha256.Sum256(txs[0])
		hash := hex.EncodeToString(hashBz[:])
		s.Require().Equal(strings.ToUpper(bidTx), strings.ToUpper(hash))

		for index, bundleTx := range bundle {
			hashBz := sha256.Sum256(txs[index+1])
			txHash := hex.EncodeToString(hashBz[:])

			s.Require().Equal(strings.ToUpper(bundleTx), strings.ToUpper(txHash))
		}
	}
}

// displayExpectedBlock displays the expected and actual blocks.
func (s *IntegrationTestSuite) displayBlock(txs [][]byte, bidTx string, bundle []string) {
	expectedBlock := fmt.Sprintf("Expected block:\n\t(%d, %s)\n", 0, bidTx)
	for index, bundleTx := range bundle {
		expectedBlock += fmt.Sprintf("\t(%d, %s)\n", index+1, bundleTx)
	}

	s.T().Logf(expectedBlock)

	// Display the actual block.
	if len(txs) == 0 {
		s.T().Logf("Actual block is empty")
		return
	}

	hashBz := sha256.Sum256(txs[0])
	hash := hex.EncodeToString(hashBz[:])
	actualBlock := fmt.Sprintf("Actual block:\n\t(%d, %s)\n", 0, hash)
	for index, tx := range txs[1:] {
		hashBz := sha256.Sum256(tx)
		txHash := hex.EncodeToString(hashBz[:])

		actualBlock += fmt.Sprintf("\t(%d, %s)\n", index+1, txHash)
	}

	s.T().Logf(actualBlock)
}

// displayExpectedBundle displays the expected order of the bid and bundled transactions.
func (s *IntegrationTestSuite) displayExpectedBundle(prefix, bidTx string, bundle []string) {
	expectedBundle := fmt.Sprintf("%s expected bundle:\n\t(%d, %s)\n", prefix, 0, bidTx)
	for index, bundleTx := range s.bundleToTxHashes(bundle) {
		expectedBundle += fmt.Sprintf("\t(%d, %s)\n", index+1, bundleTx)
	}

	s.T().Logf(expectedBundle)
}

// queryTx queries a transaction by its hash and returns whether there was an
// error in including the transaction in a block.
func (s *IntegrationTestSuite) queryTxPassed(txHash string) error {
	queryClient := txtypes.NewServiceClient(s.createClientContext())

	req := &txtypes.GetTxRequest{Hash: txHash}
	resp, err := queryClient.GetTx(context.Background(), req)
	if err != nil {
		return err
	}

	if resp.TxResponse.Code != 0 {
		return fmt.Errorf("tx failed: %s", resp.TxResponse.RawLog)
	}

	return nil
}

// queryBuilderParams returns the params of the builder module.
func (s *IntegrationTestSuite) queryBuilderParams() buildertypes.Params {
	queryClient := buildertypes.NewQueryClient(s.createClientContext())

	req := &buildertypes.QueryParamsRequest{}
	resp, err := queryClient.Params(context.Background(), req)
	s.Require().NoError(err)

	return resp.Params
}

// queryBalancesOf returns the balances of an account.
func (s *IntegrationTestSuite) queryBalancesOf(address string) sdk.Coins {
	queryClient := banktypes.NewQueryClient(s.createClientContext())

	req := &banktypes.QueryAllBalancesRequest{Address: address}
	resp, err := queryClient.AllBalances(context.Background(), req)
	s.Require().NoError(err)

	return resp.Balances
}

// queryBalanceOf returns the balance of an account for a specific denom.
func (s *IntegrationTestSuite) queryBalanceOf(address string, denom string) sdk.Coin {
	queryClient := banktypes.NewQueryClient(s.createClientContext())

	req := &banktypes.QueryBalanceRequest{Address: address, Denom: denom}
	resp, err := queryClient.Balance(context.Background(), req)
	s.Require().NoError(err)

	return *resp.Balance
}

// queryAccount returns the account of an address.
func (s *IntegrationTestSuite) queryAccount(address sdk.AccAddress) *authtypes.BaseAccount {
	queryClient := authtypes.NewQueryClient(s.createClientContext())

	req := &authtypes.QueryAccountRequest{Address: address.String()}
	resp, err := queryClient.Account(context.Background(), req)
	s.Require().NoError(err)

	account := &authtypes.BaseAccount{}
	err = account.Unmarshal(resp.Account.Value)
	s.Require().NoError(err)

	return account
}

// queryCurrentHeight returns the current block height.
func (s *IntegrationTestSuite) queryCurrentHeight() int64 {
	queryClient := tmclient.NewServiceClient(s.createClientContext())

	req := &tmclient.GetLatestBlockRequest{}
	resp, err := queryClient.GetLatestBlock(context.Background(), req)
	s.Require().NoError(err)

	return resp.SdkBlock.Header.Height
}

// queryBlockTxs returns the txs of the block at the given height.
func (s *IntegrationTestSuite) queryBlockTxs(height int64) [][]byte {
	queryClient := tmclient.NewServiceClient(s.createClientContext())

	req := &tmclient.GetBlockByHeightRequest{Height: height}
	resp, err := queryClient.GetBlockByHeight(context.Background(), req)
	s.Require().NoError(err)

	return resp.GetSdkBlock().Data.Txs
}
