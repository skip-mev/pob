package rewards_address_provider

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/pob/x/builder/types"
)

// Provides auction profits to the block proposer.
type ProposerRewardsAddressProvider struct {
	distrKeeper   types.DistributionKeeper
	stakingKeeper types.StakingKeeper
}

func NewProposerRewardsAddressProvider(
	distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,
) RewardsAddressProvider {
	return &ProposerRewardsAddressProvider{
		distrKeeper:   distrKeeper,
		stakingKeeper: stakingKeeper,
	}
}

func (p *ProposerRewardsAddressProvider) GetRewardsAddress(context sdk.Context) sdk.AccAddress {
	prevPropConsAddr := p.distrKeeper.GetPreviousProposerConsAddr(context)
	prevProposer := p.stakingKeeper.ValidatorByConsAddr(context, prevPropConsAddr)

	return sdk.AccAddress(prevProposer.GetOperator())
}
