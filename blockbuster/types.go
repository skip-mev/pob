package blockbuster

import sdk "github.com/cosmos/cosmos-sdk/types"

type (
	MatchHandler func(ctx sdk.Context, tx sdk.Tx) bool
)
