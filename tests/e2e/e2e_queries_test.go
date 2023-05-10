package e2e

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ory/dockertest/v3"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

func (s *IntegrationTestSuite) queryBuilderParams(node *dockertest.Resource) *buildertypes.Params {
	queryClient := buildertypes.NewQueryClient(s.createClientContext(node))

	request := &buildertypes.QueryParamsRequest{}
	response, err := queryClient.Params(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	return &response.Params
}

func (s *IntegrationTestSuite) queryBalancesOf(node *dockertest.Resource, address sdk.AccAddress) sdk.Coins {
	queryClient := banktypes.NewQueryClient(s.createClientContext(node))

	request := &banktypes.QueryAllBalancesRequest{Address: address.String()}
	response, err := queryClient.AllBalances(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	return response.Balances
}

func (s *IntegrationTestSuite) queryAccount(node *dockertest.Resource, address sdk.AccAddress) *authtypes.BaseAccount {
	queryClient := authtypes.NewQueryClient(s.createClientContext(node))

	response, err := queryClient.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address.String(),
	})
	s.Require().NoError(err)

	account := &authtypes.BaseAccount{}
	err = account.Unmarshal(response.Account.Value)
	if err != nil {
		panic(err)
	}

	return account
}
