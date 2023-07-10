package rewards_address_provider

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Provides address for where to send auction profits.
type RewardsAddressProvider interface {
	GetRewardsAddress(context sdk.Context) sdk.AccAddress
}
