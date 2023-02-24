package types

import (
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgAuctionBid{}
)

// GetSignBytes implements the LegacyMsg interface.
func (m MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	if err := m.Params.Validate(); err != nil {
		return err
	}

	return nil
}

// GetSignBytes implements the LegacyMsg interface.
func (m MsgAuctionBid) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&m))
}

// GetSigners returns the expected signers for a MsgAuctionBid message.
func (m MsgAuctionBid) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Bidder)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (m MsgAuctionBid) ValidateBasic() error {
	// TODO: Implement validation.
	//
	// Ref: https://github.com/skip-mev/pob/issues/8
	if _, err := sdk.AccAddressFromBech32(msg.Bidder); err != nil {
		return fmt.Errorf("invalid bidder address (%s)", err)
	}

	// Validate the bid.
	if len(msg.Bid) == 0 {
		return fmt.Errorf("no bid included")
	}

	if err := msg.Bid.Validate(); err != nil {
		return fmt.Errorf("invalid bid (%s)", err)
	}

	// Validate the transactions.
	if len(msg.Transactions) == 0 {
		return fmt.Errorf("no transactions included")
	}

	for _, tx := range msg.Transactions {
		if len(tx) == 0 {
			return fmt.Errorf("empty transaction included")
		}
	}

	return nil
}
