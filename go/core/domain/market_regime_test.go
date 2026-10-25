package domain

import (
	"testing"
	"time"
)

func TestMarketRegimeAssessmentValidationAndUnavailable(t *testing.T) {
	asOf := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	assessment, err := NewMarketRegimeAssessment(MarketRegimeBullish, true, &asOf, "index above EMA")
	if err != nil || !assessment.Eligible || assessment.Regime != MarketRegimeBullish ||
		assessment.AsOf == nil || !assessment.AsOf.Equal(asOf) {
		t.Fatalf("assessment = %+v, %v", assessment, err)
	}
	if _, err := NewMarketRegimeAssessment("", false, nil, "reason"); err == nil ||
		err.Error() != "Market regime cannot be null" {
		t.Fatalf("regime validation = %v", err)
	}
	if _, err := NewMarketRegimeAssessment(MarketRegimeBearish, false, nil, " "); err == nil ||
		err.Error() != "Assessment reason cannot be blank" {
		t.Fatalf("reason validation = %v", err)
	}
	unavailable := MarketRegimeUnavailable("no index data")
	if unavailable.Regime != MarketRegimeUnknown || unavailable.Eligible || unavailable.AsOf != nil ||
		unavailable.Reason != "no index data" {
		t.Fatalf("unavailable = %+v", unavailable)
	}
}

func TestRelativeStrengthAssessmentValidationAndUnavailable(t *testing.T) {
	assessment, err := NewRelativeStrengthAssessment(true, 12.5, 8.0, 4.5, nil, "outperform")
	if err != nil || !assessment.Eligible || assessment.StockReturnPct != 12.5 ||
		assessment.ExcessReturnPct != 4.5 {
		t.Fatalf("assessment = %+v, %v", assessment, err)
	}
	if _, err := NewRelativeStrengthAssessment(false, 0, 0, 0, nil, ""); err == nil ||
		err.Error() != "Assessment reason cannot be blank" {
		t.Fatalf("reason validation = %v", err)
	}
	unavailable := RelativeStrengthUnavailable("no index")
	if unavailable.Eligible || unavailable.StockReturnPct != 0 || unavailable.Reason != "no index" {
		t.Fatalf("unavailable = %+v", unavailable)
	}
}

func TestMustDomainDecimalPanicsOnBadLiteral(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("bad literal did not panic")
		}
	}()
	mustDomainDecimal("not-a-decimal")
}

func TestMarketRegimeConstantsStayStable(t *testing.T) {
	if MarketRegimeBullish != MarketRegime("BULLISH") || MarketRegimeBearish != MarketRegime("BEARISH") || MarketRegimeUnknown != MarketRegime("UNKNOWN") {
		t.Fatal("market regime wire names changed")
	}
}
