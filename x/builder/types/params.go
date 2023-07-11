package types

import (
	fmt "fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

var (
	DefaultMaxBundleSize             uint32 = 2
	DefaultEscrowAccountAddressBytes []byte = address.Module(ModuleName)
	DefaultReserveFee                       = sdk.NewCoin("stake", sdk.NewInt(1))
	DefaultMinBidIncrement                  = sdk.NewCoin("stake", sdk.NewInt(1))
	DefaultFrontRunningProtection           = true
	DefaultProposerFee                      = sdk.ZeroDec()
)

// NewParams returns a new Params instance with the provided values.
func NewParams(
	maxBundleSize uint32,
	escrowAccountAddressBytes []byte,
	reserveFee, minBidIncrement sdk.Coin,
	frontRunningProtection bool,
	proposerFee sdk.Dec,
	bech32AddressPrefix string,
) Params {
	escrowAccountAddress, err := bech32.ConvertAndEncode(bech32AddressPrefix, escrowAccountAddressBytes)
	if err != nil {
		panic("Could not encode escrow account address")
	}

	return Params{
		MaxBundleSize:          maxBundleSize,
		EscrowAccountAddress:   escrowAccountAddress,
		ReserveFee:             reserveFee,
		MinBidIncrement:        minBidIncrement,
		FrontRunningProtection: frontRunningProtection,
		ProposerFee:            proposerFee,
	}
}

// DefaultParams returns the default x/builder parameters.
//
// Deprecated: Please use `DefaultParamsWithAddressPrefix` instead.
func DefaultParams() Params {
	return DefaultParamsWithAddressPrefix(
		"cosmos",
	)
}

// DefaultParamsWithAddressPrefix returns the default x/builder parameters with a custom address prefix.
func DefaultParamsWithAddressPrefix(bech32AddressPrefix string) Params {
	return NewParams(
		DefaultMaxBundleSize,
		DefaultEscrowAccountAddressBytes,
		DefaultReserveFee,
		DefaultMinBidIncrement,
		DefaultFrontRunningProtection,
		DefaultProposerFee,
		bech32AddressPrefix,
	)
}

// Validate performs basic validation on the parameters.
func (p Params) Validate() error {
	if err := validateEscrowAccountAddress(p.EscrowAccountAddress); err != nil {
		return err
	}
	if err := validateFee(p.ReserveFee); err != nil {
		return fmt.Errorf("invalid reserve fee (%s)", err)
	}
	if err := validateFee(p.MinBidIncrement); err != nil {
		return fmt.Errorf("invalid minimum bid increment (%s)", err)
	}

	// Minimum bid increment must always be greater than 0.
	if p.MinBidIncrement.IsLTE(sdk.NewCoin(p.MinBidIncrement.Denom, sdk.ZeroInt())) {
		return fmt.Errorf("minimum bid increment cannot be zero")
	}

	denoms := map[string]struct{}{
		p.ReserveFee.Denom:      {},
		p.MinBidIncrement.Denom: {},
	}

	if len(denoms) != 1 {
		return fmt.Errorf("mismatched auction fee denoms: minimum bid increment (%s), reserve fee (%s)", p.MinBidIncrement, p.ReserveFee)
	}

	return validateProposerFee(p.ProposerFee)
}

func validateFee(fee sdk.Coin) error {
	if fee.IsNil() {
		return fmt.Errorf("fee cannot be nil: %s", fee)
	}

	return fee.Validate()
}

func validateProposerFee(v sdk.Dec) error {
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

func validateEscrowAccountAddress(account string) error {
	if _, err := sdk.AccAddressFromBech32(account); err != nil {
		return fmt.Errorf("invalid escrow account address (%s)", err)
	}

	return nil
}
