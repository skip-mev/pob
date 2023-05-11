package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/skip-mev/pob/mempool"
	builderante "github.com/skip-mev/pob/x/builder/ante"
	builderkeeper "github.com/skip-mev/pob/x/builder/keeper"
)

type POBHandlerOptions struct {
	BaseOptions   ante.HandlerOptions
	Mempool       mempool.Mempool
	TxDecoder     sdk.TxDecoder
	TxEncoder     sdk.TxEncoder
	BuilderKeeper builderkeeper.Keeper
}

// NewPOBAnteHandler wraps all of the default Cosmos SDK AnteDecorators with the POB AnteHandler.
func NewPOBAnteHandler(options POBHandlerOptions) sdk.AnteHandler {
	baseHandler, err := ante.NewAnteHandler(options.BaseOptions)
	if err != nil {
		panic(err)
	}

	builderDecorator := builderante.NewBuilderDecorator(options.BuilderKeeper, options.TxDecoder, options.TxEncoder, options.Mempool)

	return func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return builderDecorator.AnteHandle(ctx, tx, simulate, baseHandler)
	}
}