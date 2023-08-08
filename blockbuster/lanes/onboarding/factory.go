package onboarding

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type (
	// Factory defines the API responsible for processing transactions in the onboarding lane.
	// In particular, it is responsible for determining if a transaction is an onboarding transaction.
	Factory interface {
		// IsOnboardingTx defines a function that checks if a transaction qualifies as onboarding.
		IsOnboardingTx(ctx sdk.Context, tx sdk.Tx) bool
	}

	// BankKeeper defines the required functionality for the bank keeper.
	BankKeeper interface {
		// GetBalance returns the balance of the specified address for the specified denomination.
		GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	}

	// AccountKeeper defines the required functionality for the account keeper.
	AccountKeeper interface {
		// GetAccount returns the account for the given address.
		GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	}

	// StakingKeeper defines the required functionality for the staking keeper.
	StakingKeeper interface {
		// BondDenom returns the denomination used to bond coins. We assume that this is the native token.
		BondDenom(ctx context.Context) (string, error)
	}

	// DefaultOnboardingFactory defines a default implmentation for the onboarding factory
	// interface for processing onboarding transactions.
	DefaultOnboardingFactory struct {
		bankKeeper    BankKeeper
		accountKeeper AccountKeeper
		stakingKeeper StakingKeeper

		// maxNumTxs defines the maximum number of transactions that a user can have before
		// they are considered onboarded.
		maxNumTxs uint64
	}
)

var _ Factory = (*DefaultOnboardingFactory)(nil)

// NewDefaultOnboardingFactory returns a default onboarding factory interface implementation.
func NewDefaultOnboardingFactory(
	bankKeeper BankKeeper,
	accountKeeper AccountKeeper,
	stakingKeeper StakingKeeper,
	maxNumTxs uint64,
) Factory {
	return &DefaultOnboardingFactory{
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
		stakingKeeper: stakingKeeper,
		maxNumTxs:     maxNumTxs,
	}
}

// IsOnboardingTx defines a default function that checks if a transaction is onboarding. In this case,
// we consider a transaction to be an onboarding transaction iff:
// 1. The user has less than 5 total transactions on chain.
// 2. The user does NOT have the native token in their account balance.
func (config *DefaultOnboardingFactory) IsOnboardingTx(ctx sdk.Context, tx sdk.Tx) bool {
	// Retrieve the signers of the transaction
	sigTx, ok := tx.(signing.SigVerifiableTx)
	if !ok {
		return false
	}

	// There must be valid signers on the transaction
	signers, err := sigTx.GetSigners()
	if err != nil || len(signers) == 0 {
		return false
	}

	account := signers[0]

	// Invarient #1
	accountInfo := config.accountKeeper.GetAccount(ctx, account)
	if accountInfo.GetSequence() > config.maxNumTxs {
		return false
	}

	// Invarient #2
	bondDenom, err := config.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return false
	}

	if config.bankKeeper.GetBalance(ctx, account, bondDenom).IsPositive() {
		return false
	}

	return true
}
