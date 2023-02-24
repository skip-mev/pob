package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	// Default auction module parameters
	DefaultMaxBundleSize        uint32 = 0
	DefaultEscrowAccountAddress        = ""
	DefaultReserveFee                  = sdk.Coins{}
	DefaultMinBuyInFee                 = sdk.Coins{}
	DefaultMinBidIncrement             = sdk.Coins{}

	// Parameter store keys
	ParamStoreKeyMaxBundleSize        = []byte("DefaultMaxBundleSize")
	ParamStoreKeyEscrowAccountAddress = []byte("DefaultEscrowAccountAddress")
	ParamStoreKeyReserveFee           = []byte("DefaultReserveFee")
	ParamStoreKeyMinBuyInFee          = []byte("DefaultMinBuyInFee")
	ParamStoreKeyMinBidIncrement      = []byte("DefaultMinBidIncrement")
)

// NewParams returns a new Params instance with the provided values.
func NewParams(maxBundleSize uint32, escrowAccountAddress string, reserveFee, minBuyInFee, minBidIncrement sdk.Coins) Params {
	return Params{
		MaxBundleSize:        maxBundleSize,
		EscrowAccountAddress: escrowAccountAddress,
		ReserveFee:           reserveFee,
		MinBuyInFee:          minBuyInFee,
		MinBidIncrement:      minBidIncrement,
	}
}

// DefaultParams returns the default x/auction parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultMaxBundleSize,
		DefaultEscrowAccountAddress,
		DefaultReserveFee,
		DefaultMinBuyInFee,
		DefaultMinBidIncrement,
	)
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyMaxBundleSize, &p.MaxBundleSize, validateMaxBundleSize),
		paramtypes.NewParamSetPair(ParamStoreKeyEscrowAccountAddress, &p.EscrowAccountAddress, validateEscrowAccountAddress),
		paramtypes.NewParamSetPair(ParamStoreKeyReserveFee, &p.ReserveFee, validateCoins),
		paramtypes.NewParamSetPair(ParamStoreKeyMinBuyInFee, &p.MinBuyInFee, validateCoins),
		paramtypes.NewParamSetPair(ParamStoreKeyMinBidIncrement, &p.MinBidIncrement, validateCoins),
	}
}

// Validate performs a basic validation of x/auction parameters.
func (p Params) Validate() error {
	if err := validateMaxBundleSize(p.MaxBundleSize); err != nil {
		return err
	}

	if err := validateEscrowAccountAddress(p.EscrowAccountAddress); err != nil {
		return err
	}

	if err := validateCoins(p.ReserveFee); err != nil {
		return err
	}

	if err := validateCoins(p.MinBuyInFee); err != nil {
		return err
	}

	if err := validateCoins(p.MinBidIncrement); err != nil {
		return err
	}

	return nil
}

// validateMaxBundleSize ensures the max bundle size is a valid uint32.
func validateMaxBundleSize(i interface{}) error {
	_, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

// validateEscrowAccountAddress ensures the escrow account address is a valid address (if set).
func validateEscrowAccountAddress(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// If the escrow account address is set, ensure it is a valid address.
	if len(v) != 0 {
		if _, err := sdk.AccAddressFromBech32(v); err != nil {
			return fmt.Errorf("invalid escrow account address (%s)", err)
		}
	}

	return nil
}

// validateCoins ensures the coins are valid.
func validateCoins(i interface{}) error {
	v, ok := i.(sdk.Coins)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return v.Validate()
}
