package rewards_address_provider

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Provides auction profits to a fixed address
type FixedAddressRewardsAddressProvider struct {
	rewardsAddress sdk.AccAddress
}

func NewFixedAddressRewardsAddressProvider(
	rewardsAddress sdk.AccAddress,
) RewardsAddressProvider {
	return &FixedAddressRewardsAddressProvider{
		rewardsAddress: rewardsAddress,
	}
}

func (p *FixedAddressRewardsAddressProvider) GetRewardsAddress(context sdk.Context) sdk.AccAddress {
	return p.rewardsAddress
}