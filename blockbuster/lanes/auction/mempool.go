package auction

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

// TxPriority returns a TxPriority over auction bid transactions only. It
// is to be used in the auction index only.
func TxPriority(config Factory) blockbuster.TxPriority[string] {
	return blockbuster.TxPriority[string]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) string {
			bidInfo, err := config.GetAuctionBidInfo(tx)
			if err != nil {
				panic(err)
			}

			return bidInfo.Bid.String()
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

// Insert inserts a transaction into the auction mempool.
func (l *TOBLane) Insert(ctx context.Context, tx sdk.Tx) error {
	unwrappedCtx := sdk.UnwrapSDKContext(ctx)
	if !l.Match(unwrappedCtx, tx) {
		return fmt.Errorf("transaction does not match lane")
	}

	return l.LaneConstructor.Insert(ctx, tx)
}

// GetTopAuctionTx returns the highest bidding transaction in the auction mempool.
func (l *TOBLane) GetTopAuctionTx(ctx context.Context) sdk.Tx {
	iterator := l.Select(ctx, nil)
	if iterator == nil {
		return nil
	}

	return iterator.Tx()
}
