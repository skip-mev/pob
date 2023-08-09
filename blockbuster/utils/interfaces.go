package utils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

//go:generate mockery --name AnteHandler --output ./mocks --case underscore
type AnteHandler interface {
	AnteHandler(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error)
}
