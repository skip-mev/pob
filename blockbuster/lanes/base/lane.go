package base

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/skip-mev/pob/blockbuster"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "default"
)

var _ blockbuster.Lane = (*DefaultLane)(nil)

// DefaultLane defines a default lane implementation. The default lane orders
// transactions by the sdk.Context priority. The default lane will accept any
// transaction that is not a part of the lane's IgnoreList. By default, the IgnoreList
// is empty and the default lane will accept any transaction. The default lane on its
// own implements the same functionality as the pre v0.47.0 tendermint mempool and proposal
// handlers.
type DefaultLane struct {
	// Mempool defines the mempool for the lane.
	Mempool

	// LaneConfig defines the base lane configuration.
	Cfg blockbuster.BaseLaneConfig

	// Name defines the name of the lane.
	laneName string

	// txPriority maintains how the mempool determines relative ordering
	// of transactions
	txPriority blockbuster.TxPriority[math.Int]

	// gasTokenDenom defines the gas token denom.
	gasTokenDenom string
}

// NewDefaultLane returns a new default lane.
func NewDefaultLane(cfg blockbuster.BaseLaneConfig, gasTokenDenom string) *DefaultLane {
	if err := cfg.ValidateBasic(); err != nil {
		panic(err)
	}

	lane := &DefaultLane{
		Mempool:       NewDefaultMempool(cfg.TxEncoder, cfg.MaxTxs, gasTokenDenom),
		Cfg:           cfg,
		laneName:      LaneName,
		txPriority:    TxPriority(gasTokenDenom),
		gasTokenDenom: gasTokenDenom,
	}

	return lane
}

// WithName returns a lane option that sets the lane's name.
func (l *DefaultLane) WithName(name string) *DefaultLane {
	l.laneName = name
	return l
}

// Compare determines the relative priority of two transactions belonging in the same lane.
// In the default case, priority is determined by the priority of the context passed down
// to the API.
func (l *DefaultLane) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) int {
	firstPriority := l.txPriority.GetTxPriority(ctx, this)
	secondPriority := l.txPriority.GetTxPriority(ctx, other)

	return l.txPriority.Compare(firstPriority, secondPriority)
}

// Match returns true if the transaction belongs to this lane. Since
// this is the default lane, it always returns true except for transactions
// that belong to lanes in the ignore list.
func (l *DefaultLane) Match(ctx sdk.Context, tx sdk.Tx) bool {
	return !l.MatchIgnoreList(ctx, tx)
}

// MatchIgnoreList returns true if any of the lanes that are in the ignore list
// match the current transaction.
func (l *DefaultLane) MatchIgnoreList(ctx sdk.Context, tx sdk.Tx) bool {
	for _, lane := range l.Cfg.IgnoreList {
		if lane.Match(ctx, tx) {
			return true
		}
	}

	return false
}

// Name returns the name of the lane.
func (l *DefaultLane) Name() string {
	return l.laneName
}

// Logger returns the lane's logger.
func (l *DefaultLane) Logger() log.Logger {
	return l.Cfg.Logger
}

// SetAnteHandler sets the lane's antehandler.
func (l *DefaultLane) SetAnteHandler(anteHandler sdk.AnteHandler) {
	l.Cfg.AnteHandler = anteHandler
}

// GetMaxBlockSpace returns the maximum block space for the lane as a relative percentage.
func (l *DefaultLane) GetMaxBlockSpace() math.LegacyDec {
	return l.Cfg.MaxBlockSpace
}

// GetIgnoreList returns the lane's ignore list.
func (l *DefaultLane) GetIgnoreList() []blockbuster.Lane {
	return l.Cfg.IgnoreList
}
