package strategy

import (
	"fmt"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// RelativeStrengthMomentumName is the stable type identifier.
const RelativeStrengthMomentumName = "RS_NIFTY_MOMENTUM"

// RelativeStrengthMomentumStrategy mirrors Java's relative-strength-vs-index
// momentum setup.
// RelativeStrengthMomentumStrategy mirrors Java.
type RelativeStrengthMomentumStrategy struct {
	TradingStrategyDefaults
	rsThresholdPct       coredecimal.Decimal
	requireHighProximity bool
	highProximityPct     coredecimal.Decimal
}

// NewRelativeStrengthMomentumStrategy mirrors Java's constructor validations.
// NewRelativeStrengthMomentumStrategy mirrors Java.
func NewRelativeStrengthMomentumStrategy(rsThresholdPct coredecimal.Decimal, requireHighProximity bool, highProximityPct coredecimal.Decimal) (RelativeStrengthMomentumStrategy, error) {
	zero := coredecimal.Decimal{}
	one := mustRuleDecimal("1")
	if highProximityPct.Cmp(zero) <= 0 || highProximityPct.Cmp(one) > 0 {
		return RelativeStrengthMomentumStrategy{}, fmt.Errorf("High-proximity fraction must be in (0, 1]") //nolint:staticcheck
	}
	return RelativeStrengthMomentumStrategy{
		rsThresholdPct:       rsThresholdPct,
		requireHighProximity: requireHighProximity,
		highProximityPct:     highProximityPct,
	}, nil
}

// rsThreshold mirrors Java.
func (s RelativeStrengthMomentumStrategy) rsThreshold() coredecimal.Decimal {
	if s.rsThresholdPct == (coredecimal.Decimal{}) {
		return mustRuleDecimal("0")
	}
	return s.rsThresholdPct
}

// highProximity mirrors Java.
func (s RelativeStrengthMomentumStrategy) highProximity() coredecimal.Decimal {
	if s.highProximityPct == (coredecimal.Decimal{}) {
		return mustRuleDecimal("0.90")
	}
	return s.highProximityPct
}

// Name mirrors Java.
func (RelativeStrengthMomentumStrategy) Name() string { return RelativeStrengthMomentumName }

// RegimeFilterEnabled mirrors Java.
func (RelativeStrengthMomentumStrategy) RegimeFilterEnabled() bool { return true }

// RelativeStrengthFilterEnabled mirrors Java.
func (RelativeStrengthMomentumStrategy) RelativeStrengthFilterEnabled() bool { return true }

// IsEntrySignal mirrors Java.
func (s RelativeStrengthMomentumStrategy) IsEntrySignal(indicators *Indicators) bool {
	return s.TrendAligned(indicators) && (!s.requireHighProximity || s.NearWeeklyHigh(indicators))
}

// IsSignalExit mirrors Java.
func (s RelativeStrengthMomentumStrategy) IsSignalExit(indicators *Indicators) bool {
	return s.CloseBelowEma20(indicators) || s.Ema20BelowEma50(indicators) || s.RsiBelowLowerBound(indicators)
}

// TrendAligned mirrors Java.
func (s RelativeStrengthMomentumStrategy) TrendAligned(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema50) > 0
}

// RsiInEntryRange mirrors Java.
func (RelativeStrengthMomentumStrategy) RsiInEntryRange(*Indicators) bool { return true }

// VolumeSurge mirrors Java.
func (RelativeStrengthMomentumStrategy) VolumeSurge(*Indicators) bool { return true }

// NearWeeklyHigh mirrors Java.
func (s RelativeStrengthMomentumStrategy) NearWeeklyHigh(indicators *Indicators) bool {
	return CompareProduct(indicators.WeeklyHigh, s.highProximity(), indicators.Price) <= 0
}

// CloseBelowEma20 mirrors Java.
func (s RelativeStrengthMomentumStrategy) CloseBelowEma20(indicators *Indicators) bool {
	return indicators.Price.Cmp(indicators.Ema20) < 0
}

// Ema20BelowEma50 mirrors Java.
func (s RelativeStrengthMomentumStrategy) Ema20BelowEma50(indicators *Indicators) bool {
	return indicators.Ema50.Cmp(coredecimal.Decimal{}) != 0 && indicators.Price.Cmp(indicators.Ema50) < 0
}

// RsiBelowLowerBound mirrors Java.
func (RelativeStrengthMomentumStrategy) RsiBelowLowerBound(*Indicators) bool { return false }

// IsEntryEligible mirrors Java.
func (s RelativeStrengthMomentumStrategy) IsEntryEligible(indicators *Indicators, regime *domain.MarketRegimeAssessment, relativeStrength *domain.RelativeStrengthAssessment) bool {
	if !s.IsEntrySignal(indicators) {
		return false
	}
	if relativeStrength == nil {
		return false
	}
	excess, err := coredecimal.Parse(fmt.Sprintf("%.4f", relativeStrength.ExcessReturnPct))
	if err != nil {
		return false
	}
	if excess.Cmp(s.rsThreshold()) < 0 {
		return false
	}
	return regime != nil && regime.Eligible && containsRegime(s.AllowedMarketRegimes(), regime.Regime)
}
