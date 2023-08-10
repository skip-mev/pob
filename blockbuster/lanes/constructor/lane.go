package constructor

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

type LaneConstructor[C comparable] struct {
	Cfg      blockbuster.BaseLaneConfig
	laneName string
	blockbuster.LaneMempool
	matchHandler            blockbuster.MatchHandler
	prepareLaneHandler      blockbuster.PrepareLaneHandler
	processLaneHandler      blockbuster.ProcessLaneHandler
	processLaneBasicHandler blockbuster.ProcessLaneBasicHandler
}

func NewLaneConstructor[C comparable](
	cfg blockbuster.BaseLaneConfig,
	laneName string,
	laneMempool blockbuster.LaneMempool,
	matchHandlerFn blockbuster.MatchHandler,
) *LaneConstructor[C] {
	lane := &LaneConstructor[C]{
		Cfg:          cfg,
		laneName:     laneName,
		LaneMempool:  laneMempool,
		matchHandler: matchHandlerFn,
	}

	if err := lane.ValidateBasic(); err != nil {
		panic(err)
	}

	return lane
}

func (l *LaneConstructor[C]) ValidateBasic() error {
	if err := l.Cfg.ValidateBasic(); err != nil {
		return err
	}

	if l.laneName == "" {
		return fmt.Errorf("lane name cannot be empty")
	}

	if l.LaneMempool == nil {
		return fmt.Errorf("lane mempool cannot be nil")
	}

	if l.matchHandler == nil {
		return fmt.Errorf("match handler cannot be nil")
	}

	// TODO: This is sort of redundant rn
	if l.prepareLaneHandler == nil {
		l.prepareLaneHandler = l.DefaultPrepareLaneHandler()
	}

	if l.processLaneHandler == nil {
		l.processLaneHandler = l.DefaultProcessLaneHandler()
	}

	if l.processLaneBasicHandler == nil {
		l.processLaneBasicHandler = l.DefaultProcessLaneBasicHandler()
	}

	return nil
}

func (l *LaneConstructor[C]) SetPrepareLaneHandler(prepareLaneHandler blockbuster.PrepareLaneHandler) {
	l.prepareLaneHandler = prepareLaneHandler
}

func (l *LaneConstructor[C]) SetProcessLaneHandler(processLaneHandler blockbuster.ProcessLaneHandler) {
	l.processLaneHandler = processLaneHandler
}

func (l *LaneConstructor[C]) SetProcessLaneBasicHandler(processLaneBasicHandler blockbuster.ProcessLaneBasicHandler) {
	l.processLaneBasicHandler = processLaneBasicHandler
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
