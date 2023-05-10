package e2e

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/builder/types"
)

func (s *IntegrationTestSuite) TestInit() {
	// ensure that all accounts have some funds
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
	balanceBefore := s.queryBalancesOf(s.valResources[0], s.accounts[0].Address)

	from := s.accounts[0]
	to := s.accounts[1]
	amount := sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000)))
	sequenceOffset := uint64(0)
	timeout := uint64(1000000)
	tx := s.createMsgSendTx(from, to.Address.String(), amount, sequenceOffset, timeout)

	// send tx
	s.broadcastTx(s.valResources[0], tx)

	// check balances
	balanceAfter := s.queryBalancesOf(s.valResources[0], s.accounts[0].Address)

	s.Require().True(balanceAfter.IsAllLTE(balanceBefore))
}
