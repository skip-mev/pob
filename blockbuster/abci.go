package blockbuster

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ProposalHandler struct {
	mempool Mempool
}

func (h *ProposalHandler) PrepareProposalHandler() sdk.PrepareProposalHandler {
	panic("not implemented")
}

func (h *ProposalHandler) ProcessProposalHandler() sdk.ProcessProposalHandler {
	panic("not implemented")
}
