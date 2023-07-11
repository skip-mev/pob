package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
)

// NewGenesisState creates a new GenesisState instance.
func NewGenesisState(params Params) *GenesisState {
	return &GenesisState{
		Params: params,
	}
}

// DefaultGenesisState returns the default GenesisState instance.
//
// Deprecated: Please use `DefaultGenesisStateWithAddressPrefix` instead.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// DefaultGenesisStateWithAddressPrefix returns the default GenesisState instance with a custom address prefix.
func DefaultGenesisStateWithAddressPrefix(bech32AddressPrefix string) *GenesisState {
	return &GenesisState{
		Params: DefaultParamsWithAddressPrefix(bech32AddressPrefix),
	}
}

// Validate performs basic validation of the builder module genesis state.
func (gs GenesisState) Validate() error {
	return gs.Params.Validate()
}

// GetGenesisStateFromAppState returns x/builder GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.Codec, appState map[string]json.RawMessage) GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return genesisState
}
