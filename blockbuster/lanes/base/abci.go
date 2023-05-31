package base

import sdk "github.com/cosmos/cosmos-sdk/types"

func (l *BaseLane) PrepareLane(sdk.Context, int64, map[string][]byte) ([][]byte, error) {
	panic("implement me")
}

func (l *BaseLane) ProcessLane(sdk.Context, [][]byte) error {
	panic("implement me")
}

func (l *BaseLane) VerifyTx(sdk.Context, sdk.Tx) error {
	panic("implement me")
}
