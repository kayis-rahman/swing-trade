package strategy

import (
	"fmt"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// DailyEquityPoint mirrors Java record (plan §6.1).
type DailyEquityPoint struct {
	Date   time.Time
	Equity coredecimal.Decimal
}

// NewDailyEquityPoint mirrors the Java record constructor.
func NewDailyEquityPoint(date time.Time, equity coredecimal.Decimal) (DailyEquityPoint, error) {
	if date.IsZero() {
		return DailyEquityPoint{}, fmt.Errorf("date and equity cannot be null")
	}
	return DailyEquityPoint{Date: date, Equity: equity}, nil
}
