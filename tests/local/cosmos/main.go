package main

// This file contains a script that developers can execute
// after spinning up a localnet to test the auction module.
// The script will execute a series of transactions that
// will test the auction module. The script will
//
//  1. Initialize accounts with a balance to allow for
//     transactions to be sent.
//  2. Create a series of transactions that will test
//     the auction module.
//  3. Print out the results of the transactions and pseudo
//     test cases.
//
// NOTE: THIS SCRIPT IS NOT MEANT TO BE RUN IN PRODUCTION
// AND IS NOT A REPLACEMENT FOR UNIT or E2E TESTS.
//
// TO USE THE SCRIPT:
//  1. Create a wallet you can retrieve the private key from
//  2. Add the wallet some balance in the genesis file (or
//     send it some balance after spinning up the localnet)
//  3. Update the CONFIG variable below with the appropriate
//     values. In particular, update the private key to be your key.
//     Since this is NOT a production script, it is okay to hardcode
//     the private key into the DefaultConfig function below. Update
//    	the CosmosRPCURL to be the URL of the Cosmos RPC endpoint and so forth.
//  4. Run the script with `go run main.go`

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	cmtclient "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type (
	Account struct {
		Address    sdk.AccAddress
		PrivateKey *secp256k1.PrivKey
	}

	EncodingConfig struct {
		InterfaceRegistry codectypes.InterfaceRegistry
		Codec             codec.Codec
		TxConfig          client.TxConfig
		Amino             *codec.LegacyAmino
	}

	ScriptConfig struct {
		// CosmosRPCURL is the URL of the Cosmos RPC endpoint
		CosmosRPCURL string
		// SearcherPrivateKey is the private key of the account that will init accounts and bid
		Searcher Account
		// ChainID is the chain ID of the network
		ChainID string
		// NumAccounts is the number of accounts to init
		NumAccounts int
		// InitBalance is the initial balance of each account
		InitBalance math.Int
		// TestAccounts
		TestAccounts []Account
		// EncodingConfig is the encoding config for the application
		EncodingConfig EncodingConfig
		// ChainPrefix is the bech32 prefix of the chain
		ChainPrefix string
		// Denom is the denom of the chain
		Denom string
	}
)

var (
	CONFIG = DefaultConfig()
)

func main() {
	// Initialize the accounts that will be used in the bidding simulation
	initAccounts()

	// Retrieve the auction parameters
	params := getAuctionParams()
	reserveFee := params.ReserveFee.Amount
	maxBundleSize := params.MaxBundleSize

	// Config defaults here
	defaultSendAmount := sdk.NewCoins(sdk.NewCoin(CONFIG.Denom, sdk.NewInt(10000)))

	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "Valid auction bid",
			test: func() {
				bundle := [][]byte{
					createMsgSendTx(
						CONFIG.Searcher.PrivateKey,
						CONFIG.Searcher.Address,
						CONFIG.TestAccounts[0].Address,
						defaultSendAmount,
						1,
					),
				}

				bidTx := createBundleTx(
					CONFIG.Searcher.PrivateKey,
					CONFIG.Searcher.Address,
					sdk.NewCoin(CONFIG.Denom, reserveFee),
					bundle,
					0,
				)

				displayExpectedBundle("Valid auction bid", bidTx, bundle)

				// broadcast the transaction and wait for it to be included in a block
				waitForABlock()
				broadcastTx(bidTx)
				height := getCurrentBlockHeight()
				waitForABlock()

				bundleHashes := bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: true,
					bundleHashes[1]: true,
				}

				verifyBlock(height+1, bundleHashes, expectedExecution)
			},
		},
		{
			name: "Valid bid with multiple other transactions",
			test: func() {
				// Create a bundle with a multiple transaction that is valid
				bundle := make([][]byte, maxBundleSize)
				for i := 0; i < int(maxBundleSize); i++ {
					bundle[i] = createMsgSendTx(
						CONFIG.TestAccounts[0].PrivateKey,
						CONFIG.TestAccounts[0].Address,
						CONFIG.TestAccounts[1].Address,
						defaultSendAmount,
						uint64(i),
					)
				}

				// Wait for a block to ensure all transactions are included in the same block
				waitForABlock()

				// Create a bid transaction that includes the bundle and is valid
				bidTx := createBundleTx(
					CONFIG.Searcher.PrivateKey,
					CONFIG.Searcher.Address,
					sdk.NewCoin(CONFIG.Denom, reserveFee),
					bundle,
					0,
				)

				displayExpectedBundle("gud auction bid", bidTx, bundle)

				// Execute a few other messages to be included in the block after the bid and bundle
				normalTxs := make([][]byte, 3)
				normalTxs[0] = createMsgSendTx(
					CONFIG.TestAccounts[1].PrivateKey,
					CONFIG.TestAccounts[1].Address,
					CONFIG.TestAccounts[2].Address,
					defaultSendAmount,
					uint64(0),
				)
				normalTxs[1] = createMsgSendTx(
					CONFIG.TestAccounts[1].PrivateKey,
					CONFIG.TestAccounts[1].Address,
					CONFIG.TestAccounts[2].Address,
					defaultSendAmount,
					uint64(1),
				)
				normalTxs[2] = createMsgSendTx(
					CONFIG.TestAccounts[1].PrivateKey,
					CONFIG.TestAccounts[1].Address,
					CONFIG.TestAccounts[2].Address,
					defaultSendAmount,
					uint64(2),
				)

				// Broadcast the bid and normal transactions
				broadcastTx(bidTx)
				for _, tx := range normalTxs {
					broadcastTx(tx)
				}

				height := getCurrentBlockHeight()

				// Wait for a block to be created
				waitForABlock()

				// Ensure that the block was correctly created and executed in the order expected
				bundleHashes := bundleToTxHashes(bidTx, bundle)

				expectedExecution := map[string]bool{}
				for _, hash := range bundleHashes {
					expectedExecution[hash] = true
				}

				txHashes := normalTxsToTxHashes(normalTxs)
				for _, hash := range txHashes {
					expectedExecution[hash] = true
				}

				totalExpectedBlock := append(bundleHashes, txHashes...)
				verifyBlock(height+1, totalExpectedBlock, expectedExecution)
			},
		},
		{
			name: "Invalid auction bid",
			test: func() {
				bundle := [][]byte{
					createMsgSendTx(
						CONFIG.Searcher.PrivateKey,
						CONFIG.Searcher.Address,
						CONFIG.TestAccounts[0].Address,
						defaultSendAmount,
						1,
					),
				}

				bidTx := createBundleTx(
					CONFIG.Searcher.PrivateKey,
					CONFIG.Searcher.Address,
					sdk.NewCoin(CONFIG.Denom, sdk.ZeroInt()),
					bundle,
					0,
				)

				displayExpectedBundle("Invalid auction bid", bidTx, bundle)

				// broadcast the transaction and wait for it to be included in a block
				waitForABlock()
				broadcastTx(bidTx)
				height := getCurrentBlockHeight()
				waitForABlock()

				bundleHashes := bundleToTxHashes(bidTx, bundle)
				expectedExecution := map[string]bool{
					bundleHashes[0]: false,
					bundleHashes[1]: false,
				}

				verifyBlock(height+1, nil, expectedExecution)
			},
		},
	}

	for _, testCase := range testCases {
		waitForABlock()
		wrapTestCase(testCase.name, testCase.test)
	}
}

// ------------------------------------------ TRANSACTION HELPERS ------------------------------------------ //

// createTx will create a transaction with the given parameters:
// - privateKey: the private key of the sender (passed in so the tx can be signed)
// - msgs: the messages to send
// - sequenceOffset: the sequence offset (used to create transactions that are not in sequence)
func createTx(privateKey *secp256k1.PrivKey, msgs []sdk.Msg, sequenceOffset uint64) []byte {
	// Get the searcher account that will be used to sign the bundle transactions
	account := getAccount(privateKey)

	txConfig := authtx.NewTxConfig(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), authtx.DefaultSignModes)
	txBuilder := txConfig.NewTxBuilder()

	txBuilder.SetMsgs(msgs...)
	txBuilder.SetGasLimit(5000000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(75000))))

	sequence := account.GetSequence() + sequenceOffset
	sigV2 := signing.SignatureV2{
		PubKey: privateKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: sequence,
	}

	height := uint64(getCurrentBlockHeight())
	txBuilder.SetTimeoutHeight(height + 3)

	err := txBuilder.SetSignatures(sigV2)
	if err != nil {
		panic(err)
	}

	signerData := authsigning.SignerData{
		ChainID:       CONFIG.ChainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      sequence,
	}

	sigV2, err = clienttx.SignWithPrivKey(
		txConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		privateKey,
		txConfig,
		sequence,
	)
	if err != nil {
		panic(err)
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		panic(err)
	}

	bz, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		panic(err)
	}

	return bz
}

// createBundleTx creates a bundle transaction (auction tx) with the given parameters:
// - privateKey: the private key of the sender (passed in so the tx can be signed)
// - bidder: the address of the bidder
// - bid: the bid amount
// - transactions: the transactions to include in the bundle
// - sequenceOffset: the sequence offset (used to create transactions that are not in sequence)
func createBundleTx(
	privateKey *secp256k1.PrivKey,
	bidder sdk.AccAddress,
	bid sdk.Coin,
	transactions [][]byte,
	sequenceOffset uint64,
) []byte {
	msgs := []sdk.Msg{
		&buildertypes.MsgAuctionBid{
			Bidder:       bidder.String(),
			Bid:          bid,
			Transactions: transactions,
		},
	}

	return createTx(privateKey, msgs, sequenceOffset)
}

// createMsgSendTx creates a MsgSend transaction with the given parameters:
// - privateKey: the private key of the sender (passed in so the tx can be signed)
// - from: the address of the sender
// - toAddress: the address of the recipient
// - amount: the amount of coins to send
// - sequenceOffset: the sequence offset (used to create transactions that are not in sequence)
func createMsgSendTx(
	privateKey *secp256k1.PrivKey,
	from, toAddress sdk.AccAddress,
	amount sdk.Coins,
	sequenceOffset uint64,
) []byte {
	msgs := []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: from.String(),
			ToAddress:   toAddress.String(),
			Amount:      amount,
		},
	}

	return createTx(privateKey, msgs, sequenceOffset)
}

// broadcastTx will broadcast the given transaction to the blockchain.
func broadcastTx(tx []byte) {
	grpcConn := getCosmosClient(CONFIG.CosmosRPCURL)

	_, err := txtypes.NewServiceClient(grpcConn).BroadcastTx(
		context.Background(),
		&txtypes.BroadcastTxRequest{
			Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
			TxBytes: tx,
		},
	)
	if err != nil {
		panic(err)
	}
}

// ------------------------------------------ SET UP HELPERS ------------------------------------------ //

// initAccounts initializes the accounts that will be used in the bidding simulation.
// The accounts are created and seeded some balance from the searcher account that should
// already have a balance.
func initAccounts() {
	log := fmt.Sprintf("Initializing %d accounts...", CONFIG.NumAccounts)
	fmt.Println(log)

	for i := 0; i < CONFIG.NumAccounts; i++ {
		// Create a new account
		account := createAccount()

		fmt.Println("Creating account:", account.Address.String())

		// Seed the account with some balance
		sendTx := createMsgSendTx(
			CONFIG.Searcher.PrivateKey,
			CONFIG.Searcher.Address,
			account.Address,
			sdk.NewCoins(sdk.NewCoin(CONFIG.Denom, CONFIG.InitBalance)),
			0,
		)

		// Broadcast the transaction and wait for it to be included in a block
		broadcastTx(sendTx)
		waitForABlock()
		waitForABlock()

		balances := getBalanceOf(account.Address)

		if balances.AmountOf(CONFIG.Denom).IsZero() {
			panic("account balance not initialized correctly")
		}

		CONFIG.TestAccounts = append(CONFIG.TestAccounts, account)
	}
}

// createAccount randomly creates a new private key and returns
// an Account struct that contains the private key (which will be used
// to sign transactions) and the address
func createAccount() Account {
	// Generate a new private key and the associated address
	key := secp256k1.GenPrivKey()
	address, err := sdk.Bech32ifyAddressBytes(CONFIG.ChainPrefix, key.PubKey().Address().Bytes())
	if err != nil {
		panic(err)
	}

	sdkAddress, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}

	return Account{
		PrivateKey: key,
		Address:    sdkAddress,
	}
}

// normalTxsToTxHashes converts a slice of normal transactions to a slice of transaction hashes.
func normalTxsToTxHashes(txs [][]byte) []string {
	hashes := make([]string, len(txs))

	for i, tx := range txs {
		hashBz := sha256.Sum256(tx)
		hash := hex.EncodeToString(hashBz[:])
		hashes[i] = hash
	}

	return hashes
}

// bundleToTxHashes converts a bundle to a slice of transaction hashes.
func bundleToTxHashes(bidTx []byte, bundle [][]byte) []string {
	hashes := make([]string, len(bundle)+1)

	// encode the bid transaction into a hash
	hashBz := sha256.Sum256(bidTx)
	hash := hex.EncodeToString(hashBz[:])
	hashes[0] = hash

	for i, hash := range normalTxsToTxHashes(bundle) {
		hashes[i+1] = hash
	}

	return hashes
}

// waitForABlock will wait for a block to be created
func waitForABlock() {
	curr := getCurrentBlockHeight()

	for getCurrentBlockHeight() <= curr {
		time.Sleep(time.Second)
	}
}

// createEncodingConfig creates a new EncodingConfig for testing purposes.
func createEncodingConfig() EncodingConfig {
	cdc := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()

	cryptocodec.RegisterInterfaces(interfaceRegistry)
	buildertypes.RegisterInterfaces(interfaceRegistry)
	codec := codec.NewProtoCodec(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             codec,
		TxConfig:          authtx.NewTxConfig(codec, authtx.DefaultSignModes),
		Amino:             cdc,
	}
}

// Creates a default configuration for testing the script
func DefaultConfig() ScriptConfig {
	cfg := ScriptConfig{
		CosmosRPCURL:   "localhost:9090",
		ChainID:        "chain-id-0",
		NumAccounts:    3,
		InitBalance:    sdk.NewInt(10000000),
		TestAccounts:   []Account{},
		EncodingConfig: createEncodingConfig(),
		ChainPrefix:    "cosmos",
		Denom:          "stake",
	}

	privateKeys := []string{
		"dba619dd3a2a75193053d8bbfc7cc572ffd92e7d166a3d7c4eedcd2c9df24982",
	}

	accounts := make([]Account, 0, len(privateKeys))
	for _, key := range privateKeys {
		privateKey, err := hex.DecodeString(key)
		if err != nil {
			panic(err)
		}

		privKey := &secp256k1.PrivKey{Key: privateKey}
		address, err := sdk.Bech32ifyAddressBytes(cfg.ChainPrefix, privKey.PubKey().Address().Bytes())
		if err != nil {
			panic(err)
		}

		sdkAddress, err := sdk.AccAddressFromBech32(address)
		if err != nil {
			panic(err)
		}

		accounts = append(accounts, Account{
			PrivateKey: privKey,
			Address:    sdkAddress,
		})
	}

	cfg.Searcher = accounts[0]

	return cfg
}

// wrapTestCase wraps the test case in a function that will be executed
// in a tidy way (i.e. with a header and footer). It also waits for a block
// to be mined before executing the test case.
func wrapTestCase(name string, testCase func()) {
	fmt.Println("--------------------------------------------------")
	waitForABlock()
	log := fmt.Sprintf("Running test case: %s", name)
	fmt.Println(log)
	testCase()
	fmt.Print("--------------------------------------------------\n\n\n\n")
}

// ------------------------------------------ QUERY/VERIFICATION HELPERS ------------------------------------------ //

// getCosmosClient returns an grpc.ClientConn that connects to the local
// Cosmos node.
func getCosmosClient(rpc string) *grpc.ClientConn {
	grpcConn, err := grpc.Dial(
		rpc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	return grpcConn
}

// getSearcherAccount returns the address, private key, sequence number, and account number of the searcher account
func getAccount(privateKey *secp256k1.PrivKey) *authtypes.BaseAccount {
	signerAddress, err := sdk.Bech32ifyAddressBytes("cosmos", privateKey.PubKey().Address().Bytes())
	if err != nil {
		panic(err)
	}

	// Query node to get the account number and sequence number
	grpcConn := getCosmosClient(CONFIG.CosmosRPCURL)

	response, err := authtypes.NewQueryClient(grpcConn).Account(
		context.Background(),
		&authtypes.QueryAccountRequest{
			Address: signerAddress,
		},
	)
	if err != nil {
		panic(err)
	}

	account := &authtypes.BaseAccount{}
	err = account.Unmarshal(response.Account.Value)
	if err != nil {
		panic(err)
	}

	return account
}

// getAuctionParams returns the current builder module parameters
func getAuctionParams() *buildertypes.Params {
	// Get the grpc connection used to query account info and broadcast transactions
	grpcConn := getCosmosClient(CONFIG.CosmosRPCURL)

	// Query the current block height
	res, err := buildertypes.NewQueryClient(grpcConn).Params(context.Background(), &buildertypes.QueryParamsRequest{})
	if err != nil {
		panic(err)
	}

	return &res.Params
}

// getBalanceOf returns the balance of the given address
func getBalanceOf(address sdk.AccAddress) sdk.Coins {
	// Get the grpc connection used to query account info and broadcast transactions
	grpcConn := getCosmosClient(CONFIG.CosmosRPCURL)

	// Query the current block height
	grpcRes, err := banktypes.NewQueryClient(grpcConn).AllBalances(
		context.Background(),
		&banktypes.QueryAllBalancesRequest{
			Address: address.String(),
		},
	)
	if err != nil {
		panic(err)
	}

	return grpcRes.Balances
}

// getCurrentBlockHeight returns the current block height
func getCurrentBlockHeight() int64 {
	grpcConn := getCosmosClient(CONFIG.CosmosRPCURL)

	// Query the current block height
	grpcRes, err := cmtclient.NewServiceClient(grpcConn).GetLatestBlock(
		context.Background(),
		&cmtclient.GetLatestBlockRequest{},
	)
	if err != nil {
		panic(err)
	}

	return grpcRes.GetSdkBlock().Header.Height
}

// queryBlockTxs returns the txs of the block at the given height.
func getBlockTxs(height int64) [][]byte {
	queryClient := cmtclient.NewServiceClient(getCosmosClient(CONFIG.CosmosRPCURL))

	req := &cmtclient.GetBlockByHeightRequest{Height: height}
	resp, err := queryClient.GetBlockByHeight(context.Background(), req)
	if err != nil {
		panic(err)
	}

	return resp.GetSdkBlock().Data.Txs
}

// displayExpectedBundle displays the expected order of the bid and bundled transactions.
func displayExpectedBundle(prefix string, bidTx []byte, bundle [][]byte) {
	// encode the bid transaction into a hash
	hashes := bundleToTxHashes(bidTx, bundle)

	expectedBundle := fmt.Sprintf("%s expected bundle:\n\t(%d, %s)\n", prefix, 0, hashes[0])
	for index, bundleTx := range hashes[1:] {
		expectedBundle += fmt.Sprintf("\t(%d, %s)\n", index+1, bundleTx)
	}

	fmt.Println(expectedBundle)
}

// verifyBlock verifies that the transactions in the block at the given height were seen
// and executed in the order they were submitted.
func verifyBlock(height int64, txs []string, expectedExecution map[string]bool) {
	waitForABlock()
	fmt.Println("Verifying block", height)

	// Get the block's transactions and display the expected and actual block for debugging.
	blockTxs := getBlockTxs(height)
	displayBlock(blockTxs, txs)

	// Check that the block contains the expected transactions in the expected order.
	if len(txs) != len(blockTxs) {
		panic(fmt.Sprintf("Block %d does not contain the expected number of transactions", height))
	}

	hashBlockTxs := normalTxsToTxHashes(blockTxs)
	for index, tx := range txs {
		currTx := strings.ToUpper(tx)
		blockTx := strings.ToUpper(hashBlockTxs[index])

		if currTx != blockTx {
			panic(fmt.Sprintf("Block %d does not contain the expected transactions in the expected order", height))
		}
	}

	fmt.Println("Block contains the expected transactions in the expected order", height)
}

// displayExpectedBlock displays the expected and actual blocks.
func displayBlock(txs [][]byte, expectedTxs []string) {
	if len(expectedTxs) != 0 {
		expectedBlock := fmt.Sprintf("Expected block:\n\t(%d, %s)\n", 0, expectedTxs[0])
		for index, expectedTx := range expectedTxs[1:] {
			expectedBlock += fmt.Sprintf("\t(%d, %s)\n", index+1, expectedTx)
		}

		fmt.Println(expectedBlock)
	}

	// Display the actual block.
	if len(txs) == 0 {
		fmt.Println("Actual block is empty")
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

	fmt.Println(actualBlock)
}
