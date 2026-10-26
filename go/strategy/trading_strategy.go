package strategy

import (
	"math/big"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// Indicators mirrors Java's Indicators record: one bar's readings shared by
// the live signal engine and the backtest simulator so rules cannot drift.
type Indicators struct {
	Price      coredecimal.Decimal
	Ema20      coredecimal.Decimal
	Ema50      coredecimal.Decimal
	Rsi        coredecimal.Decimal
	Volume     coredecimal.Decimal
	VolumeMA   coredecimal.Decimal
	WeeklyHigh coredecimal.Decimal
}

// TradingStrategy mirrors Java's TradingStrategy interface: a named set of
// entry/exit rules evaluated against a single bar. Java's default methods
// that strategies rarely override are provided by embedding
// TradingStrategyDefaults; the polymorphic defaults that strategies do
// override (entry/exit confluences) are composed through
// DefaultIsEntrySignal and DefaultIsSignalExit so implementations stay
// explicit about whether they override.
type TradingStrategy interface {
	Name() string
	RegimeFilterEnabled() bool
	AllowedMarketRegimes() []domain.MarketRegime
	RelativeStrengthFilterEnabled() bool
	RequiredEntryRules() int
	EntryConfluenceDescription() string
	EntryRsiDescription() string
	IsEntrySignal(*Indicators) bool
	IsSignalExit(*Indicators) bool
	TrendAligned(*Indicators) bool
	RsiInEntryRange(*Indicators) bool
	VolumeSurge(*Indicators) bool
	NearWeeklyHigh(*Indicators) bool
	CloseBelowEma20(*Indicators) bool
	Ema20BelowEma50(*Indicators) bool
	RsiBelowLowerBound(*Indicators) bool
}

// TradingStrategyDefaults provides Java's interface default bodies for the
// overlay flags and descriptions; embed it to inherit them.
type TradingStrategyDefaults struct{}

// RegimeFilterEnabled mirrors the Java default: overlays are opt-in.
func (TradingStrategyDefaults) RegimeFilterEnabled() bool { return false }

// AllowedMarketRegimes mirrors the Java default: only BULLISH.
func (TradingStrategyDefaults) AllowedMarketRegimes() []domain.MarketRegime {
	return []domain.MarketRegime{domain.MarketRegimeBullish}
}

// RelativeStrengthFilterEnabled mirrors the Java default: opt-in.
func (TradingStrategyDefaults) RelativeStrengthFilterEnabled() bool { return false }

// RequiredEntryRules mirrors the Java default of four.
func (TradingStrategyDefaults) RequiredEntryRules() int { return 4 }

// EntryConfluenceDescription mirrors the Java default text.
func (TradingStrategyDefaults) EntryConfluenceDescription() string { return "All entry rules passed" }

// EntryRsiDescription mirrors the Java default text.
func (TradingStrategyDefaults) EntryRsiDescription() string { return "RSI between 50-65" }

// DefaultIsEntrySignal mirrors Java's default isEntrySignal: the four rules
// combined with AND.
func DefaultIsEntrySignal(strategy TradingStrategy, indicators *Indicators) bool {
	return strategy.TrendAligned(indicators) &&
		strategy.RsiInEntryRange(indicators) &&
		strategy.VolumeSurge(indicators) &&
		strategy.NearWeeklyHigh(indicators)
}

// DefaultIsSignalExit mirrors Java's default isSignalExit: the three exit
// rules combined with OR - deliberately looser than entry.
func DefaultIsSignalExit(strategy TradingStrategy, indicators *Indicators) bool {
	return strategy.CloseBelowEma20(indicators) ||
		strategy.Ema20BelowEma50(indicators) ||
		strategy.RsiBelowLowerBound(indicators)
}

// IsEntryEligible mirrors Java's two-argument isEntryEligible default: entry
// signal plus the optional overlays, failing closed on missing facts when an
// overlay is enabled.
func IsEntryEligible(strategy TradingStrategy, indicators *Indicators, regime *domain.MarketRegimeAssessment) bool {
	return IsEntryEligibleWithRelativeStrength(strategy, indicators, regime, nil)
}

// IsEntryEligibleWithRelativeStrength mirrors the three-argument default
// including the optional index-relative-strength assessment.
func IsEntryEligibleWithRelativeStrength(
	strategy TradingStrategy,
	indicators *Indicators,
	regime *domain.MarketRegimeAssessment,
	relativeStrength *domain.RelativeStrengthAssessment,
) bool {
	if !strategy.IsEntrySignal(indicators) {
		return false
	}
	regimeEligible := !strategy.RegimeFilterEnabled() ||
		(regime != nil && regime.Eligible && containsRegime(strategy.AllowedMarketRegimes(), regime.Regime))
	relativeStrengthEligible := !strategy.RelativeStrengthFilterEnabled() ||
		(relativeStrength != nil && relativeStrength.Eligible)
	return regimeEligible && relativeStrengthEligible
}

func containsRegime(regimes []domain.MarketRegime, want domain.MarketRegime) bool {
	for _, regime := range regimes {
		if regime == want {
			return true
		}
	}
	return false
}

// CompareProduct mirrors BigDecimal.compareTo(a.multiply(b), c) exactly:
// the product of two four-decimal values has eight decimals, so the
// comparison happens on aligned big integers instead of materializing a
// fixed-scale Decimal.
func CompareProduct(left, multiplier, right coredecimal.Decimal) int {
	product := new(big.Rat).Mul(ratOf(left), ratOf(multiplier))
	return product.Cmp(ratOf(right))
}

// CompareRoundedRatio mirrors BigDecimal.compareTo(a.divide(b, scale,
// HALF_UP), bound): the quotient is rounded half-up at the requested scale
// before the comparison, exactly like Java's pullback-ratio check.
func CompareRoundedRatio(numerator, denominator, bound coredecimal.Decimal, scale int) int {
	if denominator.Cmp(coredecimal.Decimal{}) == 0 {
		return 0
	}
	// roundQuotient returns the rounded quotient scaled by 10^scale; scale the
	// bound the same way so the comparison stays exact.
	rounded := roundQuotient(numerator.String(), denominator.String(), scale)
	roundedRat, ok := new(big.Rat).SetString(rounded)
	if !ok {
		return 0
	}
	boundScaled := new(big.Rat).Mul(ratOf(bound), pow10Rat(scale))
	return roundedRat.Cmp(boundScaled)
}

// ratOf converts a fixed-scale decimal string to an exact rational.
func ratOf(value coredecimal.Decimal) *big.Rat {
	rat := new(big.Rat)
	rat.SetString(value.String())
	return rat
}

// roundQuotient computes roundHalfUp(|num|/den * 10^scale)/10^scale as a
// decimal string, keeping the sign of the numerator.
func roundQuotient(numerator, denominator string, scale int) string {
	num, numOK := new(big.Rat).SetString(numerator)
	den, denOK := new(big.Rat).SetString(denominator)
	if !numOK || !denOK || den.Sign() == 0 {
		return "0"
	}
	negative := num.Sign() < 0
	num.Abs(num)
	scaled := new(big.Rat).Quo(new(big.Rat).Mul(num, pow10Rat(scale)), den)
	whole, remainder := new(big.Int).QuoRem(scaled.Num(), scaled.Denom(), new(big.Int))
	if new(big.Int).Lsh(remainder, 1).Cmp(scaled.Denom()) >= 0 {
		whole.Add(whole, big.NewInt(1))
	}
	if negative {
		whole.Neg(whole)
	}
	return whole.String()
}

func pow10Rat(exponent int) *big.Rat {
	rat := new(big.Rat)
	rat.SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil))
	return rat
}
