package strategy

import (
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// PortfolioTrade mirrors Java record.
type PortfolioTrade struct {
	Symbol                string
	EntryDate             time.Time
	ExitDate              time.Time
	EntryPrice            coredecimal.Decimal
	ExitPrice             coredecimal.Decimal
	StopLoss              coredecimal.Decimal
	Target                coredecimal.Decimal
	Quantity              int
	ExitReason            ExitReason
	RiskPerShare          coredecimal.Decimal
	PnL                   coredecimal.Decimal
	PnLPct                float64
	HoldingDays           int
	MaxAdverseExcursion   coredecimal.Decimal
	MaxFavorableExcursion coredecimal.Decimal
	StrategyScoreAtEntry  coredecimal.Decimal
}

// RMultiple returns the trade profit/loss expressed in initial risk units.
func (t PortfolioTrade) RMultiple() float64 {
	if t.RiskPerShare.Cmp(coredecimal.Decimal{}) <= 0 || t.Quantity == 0 {
		return 0.0
	}
	riskUnits := float64(t.RiskPerShare.Units().Int64())
	pnlUnits := float64(t.PnL.Units().Int64())
	initialRisk := riskUnits * float64(t.Quantity)
	if initialRisk == 0 {
		return 0.0
	}
	return pnlUnits / initialRisk
}

// IsWin reports whether the trade closed with positive PnL.
func (t PortfolioTrade) IsWin() bool {
	return t.PnL.Cmp(coredecimal.Decimal{}) > 0
}
