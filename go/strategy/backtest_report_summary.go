package strategy

import "time"

// BacktestReportSummary mirrors Java record.
type BacktestReportSummary struct {
	GeneratedAt       time.Time
	SymbolsBacktested int
	Top10ByWinRate    []any
	Top10ByTotalReturn []any
	OverallWinRate     float64
	OverallSharpeRatio float64
	Results            []any
}
