package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/auction/types"
)

var (
	_ types.QueryServer = QueryServer{}
)

// QueryServer defines the auction module's gRPC querier service.
type QueryServer struct {
	keeper Keeper
}

// NewQueryServer creates a new gRPC query server for the auction module.
func NewQueryServer(keeper Keeper) *QueryServer {
	return &QueryServer{keeper: keeper}
}

// Params queries all parameters of the auction module.
func (q QueryServer) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.keeper.GetParams(ctx)

	return &types.QueryParamsResponse{Params: params}, nil
}
