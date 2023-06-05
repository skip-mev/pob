package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IgnoreDecorator is an AnteDecorator that wraps an existing AnteDecorator. It allows
// EthTransactions to skip said Decorator by checking to see if the transaction
// contains a message of the given type.
type (
	Lane interface {
		Match(tx sdk.Tx) bool
	}

	IgnoreDecorator struct {
		decorator sdk.AnteDecorator
		lanes     []Lane
	}
)

// NewIgnoreDecorator returns a new IgnoreDecorator[D, M] instance.
func NewIgnoreDecorator(decorator sdk.AnteDecorator, lanes ...Lane) *IgnoreDecorator {
	return &IgnoreDecorator{
		decorator: decorator,
		lanes:     lanes,
	}
}

// AnteHandle implements the sdk.AnteDecorator interface, it is handle the
// type check for the message type.
func (sd IgnoreDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	for _, lane := range sd.lanes {
		if lane.Match(tx) {
			return next(ctx, tx, simulate)
		}
	}

	return sd.decorator.AnteHandle(ctx, tx, simulate, next)
}
