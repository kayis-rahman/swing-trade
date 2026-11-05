package strategy

import (
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// IntradayBacktestTrade mirrors Java record.
type IntradayBacktestTrade struct {
	Strategy  string
	EntryTime time.Time
	ExitTime  time.Time
	EntryPrice coredecimal.Decimal
	ExitPrice  coredecimal.Decimal
	Quantity   int
	Costs      coredecimal.Decimal
	PnL        coredecimal.Decimal
	ExitReason string
}
