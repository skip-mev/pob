package types

import (
	fmt "fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	DefaultMaxBundleSize          uint32 = 2
	DefaultEscrowAccountAddress   string = "cosmos13aysrj3fnmscsfshkxhrskeu6q6x837cvs78qd"
	DefaultReserveFee                    = sdk.NewCoin("stake", sdk.NewInt(10_000_000))
	DefaultMinBuyInFee                   = sdk.NewCoin("stake", sdk.NewInt(1_000_000))
	DefaultMinBidIncrement               = sdk.NewCoin("stake", sdk.NewInt(20_000_000))
	DefaultFrontRunningProtection        = true
	DefaultProposerFee                   = sdk.NewDecWithPrec(1, 2)
)

// NewParams returns a new Params instance with the provided values.
func NewParams(
	maxBundleSize uint32,
	escrowAccountAddress string,
	reserveFee, minBuyInFee, minBidIncrement sdk.Coin,
	frontRunningProtection bool,
	proposerFee sdk.Dec,
) Params {
	return Params{
		MaxBundleSize:          maxBundleSize,
		EscrowAccountAddress:   escrowAccountAddress,
		ReserveFee:             reserveFee,
		MinBuyInFee:            minBuyInFee,
		MinBidIncrement:        minBidIncrement,
		FrontRunningProtection: frontRunningProtection,
		ProposerFee:            proposerFee,
	}
}

// DefaultParams returns the default x/builder parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultMaxBundleSize,
		DefaultEscrowAccountAddress,
		DefaultReserveFee,
		DefaultMinBuyInFee,
		DefaultMinBidIncrement,
		DefaultFrontRunningProtection,
		DefaultProposerFee,
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

	if err := validateFee(p.MinBuyInFee); err != nil {
		return fmt.Errorf("invalid minimum buy-in fee (%s)", err)
	}

	if err := validateFee(p.MinBidIncrement); err != nil {
		return fmt.Errorf("invalid minimum bid increment (%s)", err)
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

// validateEscrowAccountAddress ensures the escrow account address is a valid
// address.
func validateEscrowAccountAddress(account string) error {
	// If the escrow account address is set, ensure it is a valid address.
	if _, err := sdk.AccAddressFromBech32(account); err != nil {
		return fmt.Errorf("invalid escrow account address (%s)", err)
	}

	return nil
}
