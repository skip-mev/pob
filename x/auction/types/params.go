package types

import (
	fmt "fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	// TODO: Choose reasonable default values.
	//
	// Ref: https://github.com/skip-mev/pob/issues/7
	DefaultMaxBundleSize        uint32
	DefaultEscrowAccountAddress string
	DefaultReserveFee           = sdk.Coins{}
	DefaultMinBuyInFee          = sdk.Coins{}
	DefaultMinBidIncrement      = sdk.Coins{}
	DefaultProposerFee          = sdk.ZeroDec()
)

// NewParams returns a new Params instance with the provided values.
func NewParams(maxBundleSize uint32, escrowAccountAddress string, reserveFee, minBuyInFee, minBidIncrement sdk.Coins, proposerFee sdk.Dec) Params {
	return Params{
		MaxBundleSize:        maxBundleSize,
		EscrowAccountAddress: escrowAccountAddress,
		ReserveFee:           reserveFee,
		MinBuyInFee:          minBuyInFee,
		MinBidIncrement:      minBidIncrement,
		ProposerFee:          proposerFee,
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
		DefaultProposerFee,
	)
}

// Validate performs basic validation on the parameters.
func (p Params) Validate() error {
	if err := validateEscrowAccountAddress(p.EscrowAccountAddress); err != nil {
		return err
	}

	if err := p.ReserveFee.Validate(); err != nil {
		return fmt.Errorf("invalid reserve fee (%s)", err)
	}

	if err := p.MinBuyInFee.Validate(); err != nil {
		return fmt.Errorf("invalid minimum buy-in fee (%s)", err)
	}

	if err := p.MinBidIncrement.Validate(); err != nil {
		return fmt.Errorf("invalid minimum bid increment (%s)", err)
	}

	if err := validateProposerFee(p.ProposerFee); err != nil {
		return err
	}

	return nil
}

func validateProposerFee(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("proposer fee cannot be nil: %s", v)
	}
	if v.IsNegative() {
		return fmt.Errorf("proposer fee cannot be negative: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("proposer fee too large: %s", v)
	}

	return nil
}

// validateEscrowAccountAddress ensures the escrow account address is a valid
// address (if set).
func validateEscrowAccountAddress(account string) error {
	// If the escrow account address is set, ensure it is a valid address.
	if _, err := sdk.AccAddressFromBech32(account); err != nil {
		return fmt.Errorf("invalid escrow account address (%s)", err)
	}

	return nil
}
