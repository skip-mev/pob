package constructor

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

func DefaultMatchHandler() blockbuster.MatchHandler {
	return func(ctx sdk.Context, tx sdk.Tx) bool {
		return true
	}
}

// TxPriority returns a TxPriority over the base lane transactions. It prioritizes
// transactions by their fee.
func DefaultTxPriority() blockbuster.TxPriority[string] {
	return blockbuster.TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			feeTx, ok := tx.(sdk.FeeTx)
			if !ok {
				return ""
			}

			return feeTx.GetFee().String()
		},
		Compare: func(a, b string) int {
			aCoins, _ := sdk.ParseCoinsNormalized(a)
			bCoins, _ := sdk.ParseCoinsNormalized(b)

			switch {
			case aCoins == nil && bCoins == nil:
				return 0

			case aCoins == nil:
				return -1

			case bCoins == nil:
				return 1

			default:
				switch {
				case aCoins.IsAllGT(bCoins):
					return 1

				case aCoins.IsAllLT(bCoins):
					return -1

				default:
					return 0
				}
			}
		},
		MinValue: "",
	}
}
