package keeper

import (
	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/skip-mev/pob/x/auction/types"
)

type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	paramstore paramtypes.Subspace

	bankkeeper types.BankKeeper

	// The address that is capable of executing a MsgUpdateParams message. Typically this will be the
	// governance module's address.
	authority string
}

// NewKeeper creates a new keeper instance.
func NewKeeper(cdc codec.BinaryCodec, storeKey storetypes.StoreKey, ps paramtypes.Subspace, accountKeeper types.AccountKeeper, bankkeeper types.BankKeeper, authority string) Keeper {
	// Ensure that the authority address is valid.
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(err)
	}

	// Ensure that the auction module account exists.
	if accountKeeper.GetModuleAddress(types.ModuleName) == nil {
		panic("auction module account has not been set")
	}

	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		paramstore: ps,
		bankkeeper: bankkeeper,
		authority:  authority,
	}
}

// Logger returns an auction module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetAuthority returns the address that is capable of executing a MsgUpdateParams message.
func (k Keeper) GetAuthority() string {
	return k.authority
}
