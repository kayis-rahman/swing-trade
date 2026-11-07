package strategy

import (
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// BacktestTrade mirrors Java record.
type BacktestTrade struct {
	Symbol      string
	EntryDate   time.Time
	ExitDate    time.Time
	EntryPrice  coredecimal.Decimal
	ExitPrice   coredecimal.Decimal
	StopLoss    coredecimal.Decimal
	Target      coredecimal.Decimal
	Quantity    int
	ExitReason  ExitReason
	PnL         coredecimal.Decimal
	PnLPct      coredecimal.Decimal
	HoldingDays int
}
