package e2e

import (
	"context"

	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (s *IntegrationTestSuite) createMsgSendTx(account TestAccount, toAddress string, amount sdk.Coins, sequenceOffset, height int) []byte {
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

func (s *IntegrationTestSuite) createTx(account TestAccount, msgs []sdk.Msg, sequenceOffset, height int) []byte {
	// Get the searcher account that will be used to sign the bundle transactions
	baseAccount := s.queryAccount(s.valResources[0], account.Address)

	txConfig := authtx.NewTxConfig(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), authtx.DefaultSignModes)
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
		Sequence: uint64(baseAccount.Sequence) + uint64(sequenceOffset),
	}

	txBuilder.SetTimeoutHeight(uint64(height))

	err := txBuilder.SetSignatures(sigV2)
	if err != nil {
		panic(err)
	}

	signerData := authsigning.SignerData{
		ChainID:       s.chain.id,
		AccountNumber: baseAccount.AccountNumber,
		Sequence:      uint64(baseAccount.Sequence) + uint64(sequenceOffset),
	}

	sigV2, err = clienttx.SignWithPrivKey(
		txConfig.SignModeHandler().DefaultMode(),
		signerData,
		txBuilder,
		account.PrivateKey,
		txConfig,
		uint64(baseAccount.Sequence)+uint64(sequenceOffset),
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

// broadcastTx broadcasts a transaction to the given grpc url
func (s *IntegrationTestSuite) broadcastTx(txbz []byte) {
	txClient := tx.NewServiceClient(s.createClientContext())

	req := &tx.BroadcastTxRequest{
		Mode:    tx.BroadcastMode_BROADCAST_MODE_SYNC,
		TxBytes: txbz,
	}
	resp, err := txClient.BroadcastTx(context.Background(), req)
	s.Require().NoError(err)
	s.Require().Equal(resp.TxResponse.Code, 0)
}
