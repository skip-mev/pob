package e2e

import (
	"context"
	"fmt"

	tmclient "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	buildertypes "github.com/skip-mev/pob/x/builder/types"
)

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
