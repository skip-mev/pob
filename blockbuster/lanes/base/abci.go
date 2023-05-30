package base

import sdk "github.com/cosmos/cosmos-sdk/types"

func (l *BaseLane) VerifyTx(sdk.Context, sdk.Tx) error {
	return nil
}

func (l *BaseLane) PrepareLane(sdk.Context, int64, map[string][]byte) ([][]byte, error) {
	return nil, nil
}

func (l *BaseLane) ProcessLane(sdk.Context, [][]byte) error {
	return nil
}
