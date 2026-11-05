package strategy

import (
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

// SignalResult mirrors Java record.
type SignalResult struct {
	Symbol    string
	Date      time.Time
	Type      domain.SignalType
	RSI       coredecimal.Decimal
	EMA20     coredecimal.Decimal
	EMA50     coredecimal.Decimal
	ATR       coredecimal.Decimal
	Reasoning string
}
