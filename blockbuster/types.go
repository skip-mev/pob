package blockbuster

import sdk "github.com/cosmos/cosmos-sdk/types"

type (
	MatchHandler       func(ctx sdk.Context, tx sdk.Tx) bool
	PrepareLaneHandler func(ctx sdk.Context, proposal BlockProposal, maxTxBytes int64) (txs [][]byte, txsToRemove []sdk.Tx, err error)
	ProcessLaneHandler func(ctx sdk.Context, proposal BlockProposal) (txs [][]byte, err error)
)
