package rewardsaddressprovider

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/skip-mev/pob/x/builder/types"
)

// ProposerRewardsAddressProvider provides auction profits to the block proposer.
type ProposerRewardsAddressProvider struct {
	distrKeeper   types.DistributionKeeper
	stakingKeeper types.StakingKeeper
}

// NewFixedAddressRewardsAddressProvider creates a reward provider for a fixed address.
func NewProposerRewardsAddressProvider(
	distrKeeper types.DistributionKeeper,
	stakingKeeper types.StakingKeeper,
) types.RewardsAddressProvider {
	return &ProposerRewardsAddressProvider{
		distrKeeper:   distrKeeper,
		stakingKeeper: stakingKeeper,
	}
}

func (p *ProposerRewardsAddressProvider) GetRewardsAddress(context sdk.Context) (sdk.AccAddress, error) {
	prevPropConsAddr, err := p.distrKeeper.GetPreviousProposerConsAddr(context)
	if err != nil {
		return nil, err
	}

	prevProposer, err := p.stakingKeeper.GetValidatorByConsAddr(context, prevPropConsAddr)
	if err != nil {
		return nil, err
	}

	return sdk.AccAddress(prevProposer.GetOperator()), nil
}
