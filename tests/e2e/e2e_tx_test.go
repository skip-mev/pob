package e2e

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
<<<<<<< HEAD
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
=======
>>>>>>> tags/v1.0.1
	"github.com/ory/dockertest/v3/docker"
	"github.com/skip-mev/pob/tests/app"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

// execMsgSendTx executes a send transaction on the given validator given the provided
// recipient and amount. This function returns the transaction hash. It does not wait for the
// transaction to be committed.
func (s *IntegrationTestSuite) execMsgSendTx(valIdx int, to sdk.AccAddress, amount sdk.Coin) string {
	address, err := s.chain.validators[valIdx].keyInfo.GetAddress()
	s.Require().NoError(err)

	s.T().Logf(
		"sending %s from %s to %s",
		amount, address, to,
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.valResources[valIdx].Container.ID,
		User:         "root",
		Cmd: []string{
			"testappd",
			"tx",
			"bank",
			"send",
			address.String(), // sender
			to.String(),      // receiver
			amount.String(),  // amount
			fmt.Sprintf("--%s=%s", flags.FlagFrom, s.chain.validators[valIdx].keyInfo.Name),
			fmt.Sprintf("--%s=%s", flags.FlagChainID, s.chain.id),
			fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoin(app.BondDenom, sdk.NewInt(1000000000)).String()),
			"--keyring-backend=test",
			"--broadcast-mode=sync",
			"-y",
		},
	})
	s.Require().NoError(err)

	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
	})
	s.Require().NoErrorf(err, "stdout: %s, stderr: %s", outBuf.String(), errBuf.String())

	output := outBuf.String()
	resp := strings.Split(output, ":")
	txHash := strings.TrimSpace(resp[len(resp)-1])

	return txHash
}

// createAuctionBidTx creates a transaction that bids on an auction given the provided bidder, bid, and transactions.
<<<<<<< HEAD
func (s *IntegrationTestSuite) createAuctionBidTx(account TestAccount, bid sdk.Coin, transactions [][]byte, sequenceOffset, height, gasLimit uint64, fees sdk.Coins) []byte {
=======
func (s *IntegrationTestSuite) createAuctionBidTx(account TestAccount, bid sdk.Coin, transactions [][]byte, sequenceOffset, height uint64) []byte {
>>>>>>> tags/v1.0.1
	msgs := []sdk.Msg{
		&buildertypes.MsgAuctionBid{
			Bidder:       account.Address.String(),
			Bid:          bid,
			Transactions: transactions,
		},
	}

<<<<<<< HEAD
	return s.createTx(account, msgs, sequenceOffset, height, gasLimit, fees)
=======
	return s.createTx(account, msgs, sequenceOffset, height)
>>>>>>> tags/v1.0.1
}

// createMsgSendTx creates a send transaction given the provided signer, recipient, amount, sequence number offset, and block height timeout.
// This function is primarily used to create bundles of transactions.
<<<<<<< HEAD
func (s *IntegrationTestSuite) createMsgSendTx(account TestAccount, toAddress string, amount sdk.Coins, sequenceOffset, height, gasLimit uint64, fees sdk.Coins) []byte {
=======
func (s *IntegrationTestSuite) createMsgSendTx(account TestAccount, toAddress string, amount sdk.Coins, sequenceOffset, height uint64) []byte {
>>>>>>> tags/v1.0.1
	msgs := []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: account.Address.String(),
			ToAddress:   toAddress,
			Amount:      amount,
		},
	}

<<<<<<< HEAD
	return s.createTx(account, msgs, sequenceOffset, height, gasLimit, fees)
}

// createMsgDelegateTx creates a delegate transaction given the provided signer, validator, amount, sequence number offset
// and block height timeout.
func (s *IntegrationTestSuite) createMsgDelegateTx(account TestAccount, validator string, amount sdk.Coin, sequenceOffset, height, gasLimit uint64, fees sdk.Coins) []byte {
	msgs := []sdk.Msg{
		&stakingtypes.MsgDelegate{
			DelegatorAddress: account.Address.String(),
			ValidatorAddress: validator,
			Amount:           amount,
		},
	}

	return s.createTx(account, msgs, sequenceOffset, height, gasLimit, fees)
}

// createTx creates a transaction given the provided messages, sequence number offset, and block height timeout.
func (s *IntegrationTestSuite) createTx(account TestAccount, msgs []sdk.Msg, sequenceOffset, height, gasLimit uint64, fees sdk.Coins) []byte {
=======
	return s.createTx(account, msgs, sequenceOffset, height)
}

// createTx creates a transaction given the provided messages, sequence number offset, and block height timeout.
func (s *IntegrationTestSuite) createTx(account TestAccount, msgs []sdk.Msg, sequenceOffset, height uint64) []byte {
>>>>>>> tags/v1.0.1
	txConfig := encodingConfig.TxConfig
	txBuilder := txConfig.NewTxBuilder()

	// Get account info of the sender to set the account number and sequence number
	baseAccount := s.queryAccount(account.Address)
	sequenceNumber := baseAccount.Sequence + sequenceOffset

	// Set the messages, fees, and timeout.
	txBuilder.SetMsgs(msgs...)
	txBuilder.SetGasLimit(5000000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(150000))))
	txBuilder.SetTimeoutHeight(height)

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivateKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: sequenceNumber,
	}

	s.Require().NoError(txBuilder.SetSignatures(sigV2))

	signerData := authsigning.SignerData{
		ChainID:       s.chain.id,
		AccountNumber: baseAccount.AccountNumber,
		Sequence:      sequenceNumber,
	}

	sigV2, err := clienttx.SignWithPrivKey(
		txConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		account.PrivateKey,
		txConfig,
		sequenceNumber,
	)
	s.Require().NoError(err)
	s.Require().NoError(txBuilder.SetSignatures(sigV2))

	bz, err := txConfig.TxEncoder()(txBuilder.GetTx())
	s.Require().NoError(err)

	return bz
}
