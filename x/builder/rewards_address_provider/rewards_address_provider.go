package rewardsaddressprovider

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RewardsAddressProvider is an interface that provides an address where auction profits are sent.
type RewardsAddressProvider interface {
	GetRewardsAddress(context sdk.Context) sdk.AccAddress
}
