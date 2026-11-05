package strategy

import (
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// IntradayBacktestResult mirrors Java record.
type IntradayBacktestResult struct {
	InitialCapital       coredecimal.Decimal
	FinalCapital         coredecimal.Decimal
	NetPnL               coredecimal.Decimal
	Trades               []IntradayBacktestTrade
	OpenPosition         *OpenPosition
	OpenPositionEntryTime *time.Time
}
