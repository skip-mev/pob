package e2e

import (
	"context"

	tmclient "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ory/dockertest/v3"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

// queryBuilderParams returns the params of the builder module.
func (s *IntegrationTestSuite) queryBuilderParams(node *dockertest.Resource) *buildertypes.Params {
	queryClient := buildertypes.NewQueryClient(s.createClientContext())

	request := &buildertypes.QueryParamsRequest{}
	response, err := queryClient.Params(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	return &response.Params
}

// queryBalancesOf returns the balances of an account.
func (s *IntegrationTestSuite) queryBalancesOf(node *dockertest.Resource, address sdk.AccAddress) sdk.Coins {
	queryClient := banktypes.NewQueryClient(s.createClientContext())

	request := &banktypes.QueryAllBalancesRequest{Address: address.String()}
	response, err := queryClient.AllBalances(context.Background(), request)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	return response.Balances
}

// queryAccount returns the account of an address.
func (s *IntegrationTestSuite) queryAccount(node *dockertest.Resource, address sdk.AccAddress) *authtypes.BaseAccount {
	queryClient := authtypes.NewQueryClient(s.createClientContext())

	response, err := queryClient.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address.String(),
	})
	s.Require().NoError(err)
	s.Require().NotNil(response)

	account := &authtypes.BaseAccount{}
	err = account.Unmarshal(response.Account.Value)
	s.Require().NoError(err)

	return account
}

// queryCurrentHeight returns the current block height.
func (s *IntegrationTestSuite) queryCurrentHeight() int64 {
	client := tmclient.NewServiceClient(s.createClientContext())

	resp, err := client.GetLatestBlock(context.Background(), &tmclient.GetLatestBlockRequest{})
	s.Require().NoError(err)

	return resp.Block.Header.Height
}

// queryBlock returns the block at the given height.
func (s *IntegrationTestSuite) queryBlock(height int64) *tmclient.Block {
	client := tmclient.NewServiceClient(s.createClientContext())

	resp, err := client.GetBlockByHeight(context.Background(), &tmclient.GetBlockByHeightRequest{Height: height})
	s.Require().NoError(err)

	return resp.GetSdkBlock()
}

// queryBlockTxs returns the txs of the block at the given height.
func (s *IntegrationTestSuite) queryBlockTxs(height int64) [][]byte {
	block := s.queryBlock(height)

	return block.Data.Txs
}
