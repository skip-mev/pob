package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
<<<<<<< HEAD
	"github.com/skip-mev/pob/blockbuster"
	"github.com/skip-mev/pob/blockbuster/utils"
=======
	"github.com/skip-mev/pob/mempool"
>>>>>>> tags/v1.0.1
	builderante "github.com/skip-mev/pob/x/builder/ante"
	builderkeeper "github.com/skip-mev/pob/x/builder/keeper"
)

type POBHandlerOptions struct {
	BaseOptions   ante.HandlerOptions
<<<<<<< HEAD
	Mempool       blockbuster.Mempool
	TOBLane       builderante.TOBLane
	TxDecoder     sdk.TxDecoder
	TxEncoder     sdk.TxEncoder
	BuilderKeeper builderkeeper.Keeper
	FreeLane      blockbuster.Lane
=======
	Mempool       mempool.Mempool
	TxDecoder     sdk.TxDecoder
	TxEncoder     sdk.TxEncoder
	BuilderKeeper builderkeeper.Keeper
>>>>>>> tags/v1.0.1
}

// NewPOBAnteHandler wraps all of the default Cosmos SDK AnteDecorators with the POB AnteHandler.
func NewPOBAnteHandler(options POBHandlerOptions) sdk.AnteHandler {
	if options.BaseOptions.AccountKeeper == nil {
		panic("account keeper is required for ante builder")
	}

	if options.BaseOptions.BankKeeper == nil {
		panic("bank keeper is required for ante builder")
	}

	if options.BaseOptions.SignModeHandler == nil {
		panic("sign mode handler is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(options.BaseOptions.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.BaseOptions.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.BaseOptions.AccountKeeper),
<<<<<<< HEAD
		utils.NewIgnoreDecorator(
			ante.NewDeductFeeDecorator(
				options.BaseOptions.AccountKeeper,
				options.BaseOptions.BankKeeper,
				options.BaseOptions.FeegrantKeeper,
				options.BaseOptions.TxFeeChecker,
			),
			options.FreeLane,
=======
		ante.NewDeductFeeDecorator(
			options.BaseOptions.AccountKeeper,
			options.BaseOptions.BankKeeper,
			options.BaseOptions.FeegrantKeeper,
			options.BaseOptions.TxFeeChecker,
>>>>>>> tags/v1.0.1
		),
		ante.NewSetPubKeyDecorator(options.BaseOptions.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.BaseOptions.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.BaseOptions.AccountKeeper, options.BaseOptions.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.BaseOptions.AccountKeeper, options.BaseOptions.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.BaseOptions.AccountKeeper),
<<<<<<< HEAD
		builderante.NewBuilderDecorator(options.BuilderKeeper, options.TxEncoder, options.TOBLane, options.Mempool),
=======
		builderante.NewBuilderDecorator(options.BuilderKeeper, options.TxEncoder, options.Mempool),
>>>>>>> tags/v1.0.1
	}

	return sdk.ChainAnteDecorators(anteDecorators...)
}
