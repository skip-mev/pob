package e2e

import (
	"context"

	client "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	ctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ory/dockertest/v3"
)

func (s *IntegrationTestSuite) createMsgSendTx(account TestAccount, toAddress string, amount sdk.Coins, sequenceOffset, height uint64) []byte {
	msgs := []sdk.Msg{
		&banktypes.MsgSend{
			FromAddress: account.Address.String(),
			ToAddress:   toAddress,
			Amount:      amount,
		},
	}

	tx := s.createTx(account, msgs, sequenceOffset, height)

	return tx
}

func (s *IntegrationTestSuite) createTx(account TestAccount, msgs []sdk.Msg, sequenceOffset, height uint64) []byte {
	// Get the searcher account that will be used to sign the bundle transactions
	baseAccount := s.queryAccount(s.valResources[0], account.Address)

	txConfig := authtx.NewTxConfig(codec.NewProtoCodec(ctypes.NewInterfaceRegistry()), authtx.DefaultSignModes)
	txBuilder := txConfig.NewTxBuilder()

	txBuilder.SetMsgs(msgs...)
	txBuilder.SetGasLimit(5000000)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(75000))))

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivateKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  txConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: baseAccount.Sequence + sequenceOffset,
	}

	txBuilder.SetTimeoutHeight(height)

	err := txBuilder.SetSignatures(sigV2)
	if err != nil {
		panic(err)
	}

	signerData := authsigning.SignerData{
		ChainID:       s.chain.id,
		AccountNumber: baseAccount.AccountNumber,
		Sequence:      baseAccount.Sequence + sequenceOffset,
	}

	sigV2, err = client.SignWithPrivKey(
		txConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		account.PrivateKey,
		txConfig,
		baseAccount.Sequence+sequenceOffset,
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

func (s *IntegrationTestSuite) broadcastTx(node *dockertest.Resource, txbz []byte) {
	txClient := tx.NewServiceClient(s.createClientContext(node))

	req := &tx.BroadcastTxRequest{
		Mode:    tx.BroadcastMode_BROADCAST_MODE_SYNC,
		TxBytes: txbz,
	}
	_, err := txClient.BroadcastTx(context.Background(), req)
	s.Require().NoError(err)
}
