//go:build e2e

package e2e

import (
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/pob/x/builder/types"
)

func (s *IntegrationTestSuite) TestInit() {
	for _, account := range s.accounts {
		node := s.valResources[0]
		balances := s.queryBalancesOf(node, account.Address)
		s.Require().NotEmpty(balances)
	}
}

func (s *IntegrationTestSuite) TestGetBuilderParams() {
	params := s.queryBuilderParams(s.valResources[0])

	s.Require().Equal(params.FrontRunningProtection, types.DefaultFrontRunningProtection)
	s.Require().Equal(params.ProposerFee, types.DefaultProposerFee)
	s.Require().Equal(params.MaxBundleSize, types.DefaultMaxBundleSize)
}

func (s *IntegrationTestSuite) TestSimpleTx() {
	// balanceBefore := s.queryBalancesOf(s.valResources[0], s.accounts[0].Address)

	from := s.accounts[0]
	to := s.accounts[1]
	amount := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))

	msg := &banktypes.MsgSend{
		FromAddress: from.Address.String(),
		ToAddress:   to.Address.String(),
		Amount:      amount,
	}

	ctx := s.createClientContext(s.valResources[0])
	ctx = ctx.WithBroadcastMode(flags.BroadcastSync).
		WithSkipConfirmation(true).
		WithFrom(s.chain.validators[0].keyInfo.Name).
		WithOutputFormat("json").
		WithKeyring(s.chain.validators[0].keyring)

	txFactory := tx.Factory{}.
		WithChainID(s.chain.id).
		WithKeybase(s.chain.validators[0].keyring).
		WithTimeoutHeight(10000).
		WithSignMode(signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithTxConfig(encodingConfig.TxConfig)

	err := tx.BroadcastTx(ctx, txFactory, msg)
	s.Require().NoError(err)

	// TODO/XXX: Get tx hash from ctx.Output and confirm tx was successful, for now,
	// we just sleep for a bit.
	time.Sleep(3 * time.Second)

	// // check balances
	// balanceAfter := s.queryBalancesOf(s.valResources[0], s.accounts[0].Address)

	// s.Require().True(balanceAfter.IsAllLTE(balanceBefore))
}
