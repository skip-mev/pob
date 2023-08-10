package constructor

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the constructor lane.
	LaneName = "constructor"
)

type LaneConstructor[C comparable] struct {
	Cfg      blockbuster.BaseLaneConfig
	laneName string
	blockbuster.LaneMempool
	matchHandler blockbuster.MatchHandler
}

func NewLaneConstructor[C comparable](
	cfg blockbuster.BaseLaneConfig,
	laneName string,
	txPriority blockbuster.TxPriority[C],
	matchHandlerFn blockbuster.MatchHandler,
) *LaneConstructor[C] {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	lane := &LaneConstructor[C]{
		Cfg:          cfg,
		laneName:     LaneName,
		LaneMempool:  NewConstructorMempool[C](txPriority, cfg.TxEncoder, cfg.MaxTxs),
		matchHandler: matchHandlerFn,
	}

	return lane
}

func (l *LaneConstructor[C]) Match(ctx sdk.Context, tx sdk.Tx) bool {
	return l.matchHandler(ctx, tx) && !l.CheckIgnoreList(ctx, tx)
}

func (l *LaneConstructor[C]) CheckIgnoreList(ctx sdk.Context, tx sdk.Tx) bool {
	for _, lane := range l.Cfg.IgnoreList {
		if lane.Match(ctx, tx) {
			return true
		}
	}

	return false
}

func (l *LaneConstructor[C]) Name() string {
	return l.laneName
}

func (l *LaneConstructor[C]) SetAnteHandler(anteHandler sdk.AnteHandler) {
	l.Cfg.AnteHandler = anteHandler
}

func (l *LaneConstructor[C]) Logger() log.Logger {
	return l.Cfg.Logger
}

func (l *LaneConstructor[C]) GetMaxBlockSpace() math.LegacyDec {
	return l.Cfg.MaxBlockSpace
}
