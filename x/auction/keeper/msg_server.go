package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/x/auction/types"
)

var _ types.MsgServer = MsgServer{}

// MsgServer is the wrapper for the auction module's msg service.
type MsgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the auction MsgServer interface.
func NewMsgServerImpl(keeper Keeper) *MsgServer {
	return &MsgServer{Keeper: keeper}
}

func (m MsgServer) AuctionBid(goCtx context.Context, msg *types.MsgAuctionBid) (*types.MsgAuctionBidResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// This should never return an error because the address was validated when
	// the message was ingressed.
	bidder, err := sdk.AccAddressFromBech32(msg.Bidder)
	if err != nil {
		return nil, err
	}

	// Ensure that the number of transactions is less than or equal to the maximum
	// allowed.
	maxBundleSize, err := m.GetMaxBundleSize(ctx)
	if err != nil {
		return nil, err
	}

	if uint32(len(msg.Transactions)) > maxBundleSize {
		return nil, fmt.Errorf("the number of transactions in the bid is greater than the maximum allowed; expected <= %d, got %d", maxBundleSize, len(msg.Transactions))
	}

	proposerFee, err := m.GetProposerFee(ctx)
	if err != nil {
		return nil, err
	}

	if proposerFee.IsZero() {
		// send the entire bid to the escrow account when no proposer fee is set
		if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, types.ModuleName, msg.Bid); err != nil {
			return nil, err
		}
	} else {
		prevPropConsAddr := m.distrKeeper.GetPreviousProposerConsAddr(ctx)
		prevProposer := m.stakingKeeper.ValidatorByConsAddr(ctx, prevPropConsAddr)

		// Determine the amount of the bid that goes to the (previous) proposer. If
		// a remainder exists, it'll go to the escrow account.
		bid := sdk.NewDecCoinsFromCoins(msg.Bid...)
		proposerReward, proposerRemainder := bid.MulDecTruncate(proposerFee).TruncateDecimal()

		if err := m.bankKeeper.SendCoins(ctx, bidder, sdk.AccAddress(prevProposer.GetOperator()), proposerReward); err != nil {
			return nil, err
		}

		// Determine the amount of the remaining bid that goes to the escrow account.
		// If a decimal remainder exists, it'll stay with the bidding account.
		escrowTotal := bid.Sub(sdk.NewDecCoinsFromCoins(proposerReward...)).Add(proposerRemainder...)
		escrowReward, _ := escrowTotal.TruncateDecimal()

		if err := m.bankKeeper.SendCoinsFromAccountToModule(ctx, bidder, types.ModuleName, escrowReward); err != nil {
			return nil, err
		}
	}

	return &types.MsgAuctionBidResponse{}, nil
}

func (m MsgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ensure that the message signer is the authority
	if msg.Authority != m.Keeper.GetAuthority() {
		return nil, fmt.Errorf("this message can only be executed by the authority; expected %s, got %s", m.Keeper.GetAuthority(), msg.Authority)
	}

	if err := m.Keeper.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
