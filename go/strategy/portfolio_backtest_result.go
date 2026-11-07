package strategy

import (
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// PortfolioBacktestResult mirrors Java's com.swingtrade.strategy.PortfolioBacktestResult.
type PortfolioBacktestResult struct {
	EvaluationStart     time.Time
	EvaluationEnd       time.Time
	InitialCapital      coredecimal.Decimal
	FinalCapital        coredecimal.Decimal
	TotalReturn         coredecimal.Decimal
	MaxDrawdownPct      coredecimal.Decimal
	SharpeRatio         float64
	CAGRPct             float64
	SortinoRatio        float64
	CalmarRatio         float64
	TotalTrades         int
	WinningTrades       int
	RejectedTrades      int
	Trades              []BacktestTrade
	EquityCurve         []PortfolioEquityPoint
	RejectionReasons    []string
	BenchmarkComparison BenchmarkComparison
	StrategyVariantID   *string
}
