package constructor

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

func DefaultMatchHandler() blockbuster.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		return true
	}
}
