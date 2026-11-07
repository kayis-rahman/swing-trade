package strategy

import (
	"fmt"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// IntradayBacktestConfig mirrors Java record.
type IntradayBacktestConfig struct {
	InitialCapital  coredecimal.Decimal
	Quantity        int
	MaxTradesPerDay int
	MaxDailyLoss    coredecimal.Decimal
}

// NewIntradayBacktestConfig mirrors the Java record constructor.
func NewIntradayBacktestConfig(initialCapital coredecimal.Decimal, quantity, maxTradesPerDay int, maxDailyLoss coredecimal.Decimal) (IntradayBacktestConfig, error) {
	if initialCapital.Cmp(coredecimal.Decimal{}) <= 0 || quantity <= 0 || maxTradesPerDay <= 0 || maxDailyLoss.Cmp(coredecimal.Decimal{}) <= 0 {
		return IntradayBacktestConfig{}, fmt.Errorf("capital, quantity and limits must be positive")
	}
	return IntradayBacktestConfig{
		InitialCapital:  initialCapital,
		Quantity:        quantity,
		MaxTradesPerDay: maxTradesPerDay,
		MaxDailyLoss:    maxDailyLoss,
	}, nil
}

// IntradayBacktestConfigDefaults mirrors Java defaults().
func IntradayBacktestConfigDefaults() IntradayBacktestConfig {
	ic, _ := coredecimal.Parse("100000")
	mdl, _ := coredecimal.Parse("1000")
	c, _ := NewIntradayBacktestConfig(ic, 1, 3, mdl)
	return c
}
