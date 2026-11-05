package strategy

import (
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// PortfolioEquityPoint mirrors Java record.
type PortfolioEquityPoint struct {
	Date                time.Time
	Equity              coredecimal.Decimal
	SettledCash         coredecimal.Decimal
	UnsettledProceeds   coredecimal.Decimal
	PositionMarketValue coredecimal.Decimal
}

func NewPortfolioEquityPointTotal(date time.Time, equity coredecimal.Decimal) PortfolioEquityPoint {
	return PortfolioEquityPoint{
		Date: date, Equity: equity, SettledCash: equity,
		UnsettledProceeds: coredecimal.Decimal{}, PositionMarketValue: coredecimal.Decimal{},
	}
}
