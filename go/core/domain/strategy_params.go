package domain

import coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"

// mustDomainDecimal panics only for programmer-written literals that can
// never fail to parse.
func mustDomainDecimal(value string) coredecimal.Decimal {
	decimalValue, err := coredecimal.Parse(value)
	if err != nil {
		panic(err)
	}
	return decimalValue
}

// Strategy parameter constants mirroring Java's StrategyParams: all strategy
// engines reference these to stay in sync.
const (
	// EMA periods.
	EMAFast = 20
	EMASlow = 50

	// Common indicator periods.
	RSIPeriod      = 14
	ATRPeriod      = 14
	VolumeMAPeriod = 20

	// 52-week trading days (~252).
	FiftyTwoWeekTradingDays = 252

	// Candle requirements.
	MinCandles = EMASlow
)

// Strategy threshold constants mirroring Java's BigDecimal fields.
var (
	// RSILower and RSIUpper bound the RSI entry band.
	RSILower = mustDomainDecimal("50")
	RSIUpper = mustDomainDecimal("65")

	// VolumeMultiplier and HighProximity are the shared entry thresholds.
	VolumeMultiplier = mustDomainDecimal("1.5")
	HighProximity    = mustDomainDecimal("0.97")
)
