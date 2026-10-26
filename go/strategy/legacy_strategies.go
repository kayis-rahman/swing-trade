package strategy

import (
	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// Stable strategy names matching Java's NAME constants; backtest selection
// depends on them.
const (
	PullbackInUptrendName        = "PULLBACK_UPTREND"
	VolatilitySqueezeName        = "VOLATILITY_SQUEEZE"
	FiftyTwoWeekHighBreakoutName = "52W_HIGH_BREAKOUT"
	Rsi2MeanReversionName        = "RSI2_MEAN_REVERSION"
	PriceActionName              = "PRICE_ACTION"
)

// rsi2EntryThreshold mirrors Rsi2MeanReversionStrategy.RSI_ENTRY_THRESHOLD.
var rsi2EntryThreshold = mustRuleDecimal("10")

// PullbackInUptrend thresholds.
var (
	pullbackMinPriceRatio = mustRuleDecimal("0.97")
	pullbackMaxPriceRatio = mustRuleDecimal("1.01")
	pullbackMinHighRatio  = mustRuleDecimal("0.90")
	pullbackMinRSI        = mustRuleDecimal("40")
	pullbackMaxRSI        = mustRuleDecimal("58")
	pullbackMaxVolume     = mustRuleDecimal("1.20")
	pullbackCloseFactor   = mustRuleDecimal("0.95")
)

// VolatilitySqueeze thresholds.
var (
	squeezeMaxVolumeRatio = mustRuleDecimal("0.80")
	squeezeMinRSI         = mustRuleDecimal("50")
	squeezeMaxRSI         = mustRuleDecimal("70")
	squeezeMinHighRatio   = mustRuleDecimal("0.97")
)

// FiftyTwoWeekHighBreakout threshold.
var fiftyTwoWeekVolumeMultiplier = mustRuleDecimal("2.0")

func mustRuleDecimal(value string) coredecimal.Decimal {
	decimalValue, err := coredecimal.Parse(value)
	if err != nil {
		panic(err)
	}
	return decimalValue
}

// PullbackInUptrendStrategy mirrors Java's bounded pullback strategy: a
// controlled pullback near EMA20 while the primary trend stays positive.
type PullbackInUptrendStrategy struct{ TradingStrategyDefaults }

// Name mirrors the Java NAME constant.
func (PullbackInUptrendStrategy) Name() string { return PullbackInUptrendName }

// RegimeFilterEnabled mirrors the Java override: the overlay is on.
func (PullbackInUptrendStrategy) RegimeFilterEnabled() bool { return true }

// IsEntrySignal mirrors the Java five-condition entry (the default four plus
// the private pullback-near-EMA20 ratio band).
func (strategy PullbackInUptrendStrategy) IsEntrySignal(indicators *Indicators) bool {
	return strategy.TrendAligned(indicators) &&
		strategy.RsiInEntryRange(indicators) &&
		strategy.VolumeSurge(indicators) &&
		strategy.pullbackNearEma20(indicators) &&
		strategy.NearWeeklyHigh(indicators)
}

// IsSignalExit mirrors Java's default body for this strategy.
func (strategy PullbackInUptrendStrategy) IsSignalExit(indicators *Indicators) bool {
	return DefaultIsSignalExit(strategy, indicators)
}

// TrendAligned mirrors the Java rule: price and EMA20 above EMA50.
func (PullbackInUptrendStrategy) TrendAligned(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema50) > 0 && indicators.Ema20.Cmp(indicators.Ema50) > 0
}

// RsiInEntryRange mirrors the Java [40,58] band.
func (PullbackInUptrendStrategy) RsiInEntryRange(indicators *Indicators) bool {
	return indicators.Rsi.Cmp(pullbackMinRSI) >= 0 && indicators.Rsi.Cmp(pullbackMaxRSI) <= 0
}

// VolumeSurge mirrors the Java rule: volume at most1.20x the average.
func (PullbackInUptrendStrategy) VolumeSurge(indicators *Indicators) bool {
	return CompareProduct(indicators.VolumeMA, pullbackMaxVolume, indicators.Volume) >= 0
}

// NearWeeklyHigh mirrors the Java rule: price at least90% of the weekly high.
func (PullbackInUptrendStrategy) NearWeeklyHigh(indicators *Indicators) bool {
	return CompareProduct(indicators.WeeklyHigh, pullbackMinHighRatio, indicators.Price) <= 0
}

// pullbackNearEma20 mirrors the Java private rule: round(price/EMA20, 8,
// HALF_UP) inside [0.97, 1.01].
func (PullbackInUptrendStrategy) pullbackNearEma20(indicators *Indicators) bool {
	return CompareRoundedRatio(indicators.Price, indicators.Ema20, pullbackMinPriceRatio, 8) >= 0 &&
		CompareRoundedRatio(indicators.Price, indicators.Ema20, pullbackMaxPriceRatio, 8) <= 0
}

// CloseBelowEma20 mirrors the Java rule: price below95% of EMA20.
func (PullbackInUptrendStrategy) CloseBelowEma20(indicators *Indicators) bool {
	return CompareProduct(indicators.Ema20, pullbackCloseFactor, indicators.Price) > 0
}

// Ema20BelowEma50 mirrors the Java rule.
func (PullbackInUptrendStrategy) Ema20BelowEma50(indicators *Indicators) bool {
	return indicators.Ema20.Cmp(indicators.Ema50) < 0
}

// RsiBelowLowerBound mirrors the Java rule: RSI below40.
func (PullbackInUptrendStrategy) RsiBelowLowerBound(indicators *Indicators) bool {
	return indicators.Rsi.Cmp(pullbackMinRSI) < 0
}

// VolatilitySqueezeStrategy mirrors Java's contracted-volume breakout setup.
type VolatilitySqueezeStrategy struct{ TradingStrategyDefaults }

// Name mirrors the Java NAME constant.
func (VolatilitySqueezeStrategy) Name() string { return VolatilitySqueezeName }

// RegimeFilterEnabled mirrors the Java override: the overlay is on.
func (VolatilitySqueezeStrategy) RegimeFilterEnabled() bool { return true }

// IsEntrySignal mirrors the Java override, whose body is exactly the default
// four-rule conjunction.
func (strategy VolatilitySqueezeStrategy) IsEntrySignal(indicators *Indicators) bool {
	return DefaultIsEntrySignal(strategy, indicators)
}

// IsSignalExit mirrors Java's default body for this strategy.
func (strategy VolatilitySqueezeStrategy) IsSignalExit(indicators *Indicators) bool {
	return DefaultIsSignalExit(strategy, indicators)
}

// TrendAligned mirrors the Java rule: price at or above EMA20 and EMA20
// above EMA50.
func (VolatilitySqueezeStrategy) TrendAligned(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) >= 0 && indicators.Ema20.Cmp(indicators.Ema50) > 0
}

// RsiInEntryRange mirrors the Java [50,70] band.
func (VolatilitySqueezeStrategy) RsiInEntryRange(indicators *Indicators) bool {
	return indicators.Rsi.Cmp(squeezeMinRSI) >= 0 && indicators.Rsi.Cmp(squeezeMaxRSI) <= 0
}

// VolumeSurge mirrors the Java rule: volume at most80% of the average.
func (VolatilitySqueezeStrategy) VolumeSurge(indicators *Indicators) bool {
	return CompareProduct(indicators.VolumeMA, squeezeMaxVolumeRatio, indicators.Volume) >= 0
}

// NearWeeklyHigh mirrors the Java rule: price at least97% of the weekly high.
func (VolatilitySqueezeStrategy) NearWeeklyHigh(indicators *Indicators) bool {
	return CompareProduct(indicators.WeeklyHigh, squeezeMinHighRatio, indicators.Price) <= 0
}

// CloseBelowEma20 mirrors the Java rule.
func (VolatilitySqueezeStrategy) CloseBelowEma20(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) < 0
}

// Ema20BelowEma50 mirrors the Java rule.
func (VolatilitySqueezeStrategy) Ema20BelowEma50(indicators *Indicators) bool {
	return indicators.Ema20.Cmp(indicators.Ema50) < 0
}

// RsiBelowLowerBound mirrors the Java rule: RSI below50.
func (VolatilitySqueezeStrategy) RsiBelowLowerBound(indicators *Indicators) bool {
	return indicators.Rsi.Cmp(squeezeMinRSI) < 0
}

// FiftyTwoWeekHighBreakoutStrategy mirrors Java's new-high breakout setup.
type FiftyTwoWeekHighBreakoutStrategy struct{ TradingStrategyDefaults }

// Name mirrors the Java NAME constant.
func (FiftyTwoWeekHighBreakoutStrategy) Name() string { return FiftyTwoWeekHighBreakoutName }

// RegimeFilterEnabled mirrors the Java override: the overlay is on.
func (FiftyTwoWeekHighBreakoutStrategy) RegimeFilterEnabled() bool { return true }

// IsEntrySignal mirrors the Java three-rule entry (no RSI condition).
func (strategy FiftyTwoWeekHighBreakoutStrategy) IsEntrySignal(indicators *Indicators) bool {
	return strategy.TrendAligned(indicators) &&
		strategy.VolumeSurge(indicators) &&
		strategy.NearWeeklyHigh(indicators)
}

// IsSignalExit mirrors Java's default body for this strategy.
func (strategy FiftyTwoWeekHighBreakoutStrategy) IsSignalExit(indicators *Indicators) bool {
	return DefaultIsSignalExit(strategy, indicators)
}

// TrendAligned mirrors the Java rule: price above EMA20 and EMA20 above
// EMA50.
func (FiftyTwoWeekHighBreakoutStrategy) TrendAligned(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) > 0 && indicators.Ema20.Cmp(indicators.Ema50) > 0
}

// RsiInEntryRange mirrors the Java rule: always true (no RSI gate).
func (FiftyTwoWeekHighBreakoutStrategy) RsiInEntryRange(*Indicators) bool { return true }

// VolumeSurge mirrors the Java rule: volume at least2.0x the average.
func (FiftyTwoWeekHighBreakoutStrategy) VolumeSurge(indicators *Indicators) bool {
	return CompareProduct(indicators.VolumeMA, fiftyTwoWeekVolumeMultiplier, indicators.Volume) <= 0
}

// NearWeeklyHigh mirrors the Java rule: price strictly above the weekly high.
func (FiftyTwoWeekHighBreakoutStrategy) NearWeeklyHigh(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.WeeklyHigh) > 0
}

// CloseBelowEma20 mirrors the Java rule.
func (FiftyTwoWeekHighBreakoutStrategy) CloseBelowEma20(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) < 0
}

// Ema20BelowEma50 mirrors the Java rule.
func (FiftyTwoWeekHighBreakoutStrategy) Ema20BelowEma50(indicators *Indicators) bool {
	return indicators.Ema20.Cmp(indicators.Ema50) < 0
}

// RsiBelowLowerBound mirrors the Java rule: never fires (always false).
func (FiftyTwoWeekHighBreakoutStrategy) RsiBelowLowerBound(*Indicators) bool { return false }

// Rsi2MeanReversionStrategy mirrors Java's RSI(2) mean reversion for stocks
// above their long-term trend, exiting when price reclaims the short-term
// average.
type Rsi2MeanReversionStrategy struct{ TradingStrategyDefaults }

// Name mirrors the Java NAME constant.
func (Rsi2MeanReversionStrategy) Name() string { return Rsi2MeanReversionName }

// RegimeFilterEnabled mirrors the Java override: the overlay is on.
func (Rsi2MeanReversionStrategy) RegimeFilterEnabled() bool { return true }

// IsEntrySignal mirrors the Java entry: trend aligned and RSI(2) below10.
func (strategy Rsi2MeanReversionStrategy) IsEntrySignal(indicators *Indicators) bool {
	return strategy.TrendAligned(indicators) && indicators.Rsi.Cmp(rsi2EntryThreshold) < 0
}

// IsSignalExit mirrors the Java exit: price reclaims the short average.
func (Rsi2MeanReversionStrategy) IsSignalExit(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) > 0
}

// TrendAligned mirrors the Java rule: price above EMA50.
func (Rsi2MeanReversionStrategy) TrendAligned(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema50) > 0
}

// RsiInEntryRange mirrors the Java entry band: RSI below10.
func (Rsi2MeanReversionStrategy) RsiInEntryRange(indicators *Indicators) bool {
	return indicators.Rsi.Cmp(rsi2EntryThreshold) < 0
}

// VolumeSurge mirrors the Java rule: always true.
func (Rsi2MeanReversionStrategy) VolumeSurge(*Indicators) bool { return true }

// NearWeeklyHigh mirrors the Java rule: always true.
func (Rsi2MeanReversionStrategy) NearWeeklyHigh(*Indicators) bool { return true }

// CloseBelowEma20 mirrors the Java rule: never fires.
func (Rsi2MeanReversionStrategy) CloseBelowEma20(*Indicators) bool { return false }

// Ema20BelowEma50 mirrors the Java rule: never fires.
func (Rsi2MeanReversionStrategy) Ema20BelowEma50(*Indicators) bool { return false }

// RsiBelowLowerBound mirrors the Java rule: never fires.
func (Rsi2MeanReversionStrategy) RsiBelowLowerBound(*Indicators) bool { return false }

// PriceActionStrategy mirrors Java's production four-rule strategy over the
// shared StrategyParams thresholds.
type PriceActionStrategy struct{ TradingStrategyDefaults }

// Name mirrors the Java NAME constant.
func (PriceActionStrategy) Name() string { return PriceActionName }

// IsEntrySignal mirrors Java's default four-rule confluence (PriceAction does
// not override it).
func (strategy PriceActionStrategy) IsEntrySignal(indicators *Indicators) bool {
	return DefaultIsEntrySignal(strategy, indicators)
}

// IsSignalExit mirrors Java's default exit (PriceAction does not override it).
func (strategy PriceActionStrategy) IsSignalExit(indicators *Indicators) bool {
	return DefaultIsSignalExit(strategy, indicators)
}

// TrendAligned mirrors the Java rule: price above EMA20 and EMA20 above
// EMA50.
func (PriceActionStrategy) TrendAligned(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) > 0 && indicators.Ema20.Cmp(indicators.Ema50) > 0
}

// RsiInEntryRange mirrors the Java [50,65] band from StrategyParams.
func (PriceActionStrategy) RsiInEntryRange(indicators *Indicators) bool {
	return indicators.Rsi.Cmp(domain.RSILower) >= 0 && indicators.Rsi.Cmp(domain.RSIUpper) <= 0
}

// VolumeSurge mirrors the Java rule: volume strictly above1.5x the average.
func (PriceActionStrategy) VolumeSurge(indicators *Indicators) bool {
	return CompareProduct(indicators.VolumeMA, domain.VolumeMultiplier, indicators.Volume) < 0
}

// NearWeeklyHigh mirrors the Java rule: price at least97% of the weekly high.
func (PriceActionStrategy) NearWeeklyHigh(indicators *Indicators) bool {
	return CompareProduct(indicators.WeeklyHigh, domain.HighProximity, indicators.Price) <= 0
}

// CloseBelowEma20 mirrors the Java rule.
func (PriceActionStrategy) CloseBelowEma20(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) < 0
}

// Ema20BelowEma50 mirrors the Java rule.
func (PriceActionStrategy) Ema20BelowEma50(indicators *Indicators) bool {
	return indicators.Ema20.Cmp(indicators.Ema50) < 0
}

// RsiBelowLowerBound mirrors the Java rule: RSI below the lower band.
func (PriceActionStrategy) RsiBelowLowerBound(indicators *Indicators) bool {
	return indicators.Rsi.Cmp(domain.RSILower) < 0
}
