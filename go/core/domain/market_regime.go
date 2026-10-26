package domain

import (
	"fmt"
	"strings"
	"time"
)

// MarketRegime mirrors Java's com.swingtrade.domain.MarketRegime enum.
type MarketRegime string

// Market regime values matching Java's enum constants.
const (
	MarketRegimeBullish MarketRegime = "BULLISH"
	MarketRegimeBearish MarketRegime = "BEARISH"
	MarketRegimeUnknown MarketRegime = "UNKNOWN"
)

// MarketRegimeAssessment mirrors Java's record: the result of evaluating an
// index candle series against a market-regime policy.
type MarketRegimeAssessment struct {
	Regime   MarketRegime
	Eligible bool
	AsOf     *time.Time
	Reason   string
}

// NewMarketRegimeAssessment mirrors the Java compact constructor's validation.
func NewMarketRegimeAssessment(regime MarketRegime, eligible bool, asOf *time.Time, reason string) (MarketRegimeAssessment, error) {
	if regime == "" {
		return MarketRegimeAssessment{}, fmt.Errorf("Market regime cannot be null") //nolint:staticcheck // exact Java message
	}
	if strings.TrimSpace(reason) == "" {
		return MarketRegimeAssessment{}, fmt.Errorf("Assessment reason cannot be blank") //nolint:staticcheck // exact Java message
	}
	return MarketRegimeAssessment{Regime: regime, Eligible: eligible, AsOf: asOf, Reason: reason}, nil
}

// MarketRegimeUnavailable mirrors MarketRegimeAssessment.unavailable.
func MarketRegimeUnavailable(reason string) MarketRegimeAssessment {
	assessment, _ := NewMarketRegimeAssessment(MarketRegimeUnknown, false, nil, reason)
	return assessment
}

// RelativeStrengthAssessment mirrors Java's record: a stock's bounded return
// compared with an index's. Return percentages are doubles in Java, so they
// stay float here under G4's float-path rules.
type RelativeStrengthAssessment struct {
	Eligible        bool
	StockReturnPct  float64
	IndexReturnPct  float64
	ExcessReturnPct float64
	AsOf            *time.Time
	Reason          string
}

// NewRelativeStrengthAssessment mirrors the Java compact constructor's
// validation.
func NewRelativeStrengthAssessment(
	eligible bool,
	stockReturnPct, indexReturnPct, excessReturnPct float64,
	asOf *time.Time,
	reason string,
) (RelativeStrengthAssessment, error) {
	if strings.TrimSpace(reason) == "" {
		return RelativeStrengthAssessment{}, fmt.Errorf("Assessment reason cannot be blank") //nolint:staticcheck // exact Java message
	}
	return RelativeStrengthAssessment{
		Eligible: eligible, StockReturnPct: stockReturnPct,
		IndexReturnPct: indexReturnPct, ExcessReturnPct: excessReturnPct,
		AsOf: asOf, Reason: reason,
	}, nil
}

// RelativeStrengthUnavailable mirrors RelativeStrengthAssessment.unavailable.
func RelativeStrengthUnavailable(reason string) RelativeStrengthAssessment {
	assessment, _ := NewRelativeStrengthAssessment(false, 0, 0, 0, nil, reason)
	return assessment
}
