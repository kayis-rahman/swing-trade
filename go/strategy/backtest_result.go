package strategy

import (

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// BacktestResult mirrors Java's com.swingtrade.strategy.BacktestResult.
type BacktestResult struct {
	Symbol             string
	TotalTrades        int
	WinningTrades      int
	LosingTrades       int
	WinRate            float64
	AvgGainPct         coredecimal.Decimal
	AvgLossPct         coredecimal.Decimal
	MaxDrawdownPct     coredecimal.Decimal
	SharpeRatio        float64
	TotalReturn        coredecimal.Decimal
	Expectancy         float64
	Trades             []BacktestTrade
	CAGRPct            float64
	SortinoRatio       float64
	CalmarRatio        float64
	BenchmarkComparison BenchmarkComparison
}
