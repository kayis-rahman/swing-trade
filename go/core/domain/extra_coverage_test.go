package domain

import (
	"math"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestBenchmarkCandleSeriesAndComparisonHelpers(t *testing.T) {
	candles := []OhlcvCandle{{Symbol: "NIFTY", Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
	series, err := NewBenchmarkCandleSeries("NIFTY50", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), candles)
	if err != nil {
		t.Fatalf("NewBenchmarkCandleSeries: %v", err)
	}
	if series.Benchmark != "NIFTY50" || len(series.Candles) != 1 {
		t.Fatalf("series not populated: %+v", series)
	}
	if _, err := NewBenchmarkCandleSeries("", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), nil); err == nil {
		t.Fatal("expected blank benchmark error")
	}
	if _, err := NewBenchmarkCandleSeries("X", time.Time{}, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), nil); err == nil {
		t.Fatal("expected invalid window error")
	}

	if bad, err := NewBenchmarkComparison("", 1, 2, BenchmarkAvailable); err == nil || bad.BenchmarkName != "" {
		t.Fatalf("expected name error, got %+v %v", bad, err)
	}
	bh := BuyAndHoldBenchmark(10, 5)
	if bh.ExcessReturnPct != 5 || bh.Availability != BenchmarkAvailable {
		t.Fatalf("buy and hold wrong: %+v", bh)
	}
	nf := Nifty50PriceBenchmark(8, 3)
	if nf.BenchmarkName != BenchmarkNifty50 || nf.ExcessReturnPct != 5 {
		t.Fatalf("nifty wrong: %+v", nf)
	}
	un := UnavailableBenchmark(7)
	if un.Availability != BenchmarkUnavailable || un.BenchmarkReturnPct != 0 {
		t.Fatalf("unavailable wrong: %+v", un)
	}
}

func TestOhlcvCandleOf(t *testing.T) {
	open, _ := coredecimal.Parse("10")
	high, _ := coredecimal.Parse("12")
	low, _ := coredecimal.Parse("9")
	close, _ := coredecimal.Parse("11")
	c := OhlcvCandleOf("TCS", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), open, high, low, close, 100)
	if c.Close.String() != "11.0000" || c.AdjClose.String() != "11.0000" {
		t.Fatalf("OhlcvCandleOf adj close wrong: %+v", c)
	}
	if !c.IsBullish() {
		t.Fatal("expected bullish")
	}
}

func TestStrategyConfigOverlays(t *testing.T) {
	config, err := CreateStrategyConfig("v1", 1, "TYPE", map[string]any{"a": 1}, map[string]any{"o": 2}, StrategyConfigModeChampion, coredecimal.Decimal{}, true, nil, time.Now())
	if err != nil {
		t.Fatalf("CreateStrategyConfig: %v", err)
	}
	over := config.Overlays()
	if over["o"] != 2 {
		t.Fatalf("overlays wrong: %+v", over)
	}
	over["o"] = 999
	if config.Overlays()["o"] != 2 {
		t.Fatal("Overlays did not copy")
	}
	params := config.Params()
	if params["a"] != 1 {
		t.Fatalf("params wrong: %+v", params)
	}
}

func TestSentimentUnknownPredicate(t *testing.T) {
	u := SentimentScoreUnknown
	if !u.IsUnknown() || u.SupportsEntry() {
		t.Fatal("unknown score predicate wrong")
	}
	res := createTestResult(t, SentimentScoreUnknown, &testConfidence)
	if !res.IsUnknown() || res.IsPositive() || res.IsNeutral() || res.IsNegative() {
		t.Fatalf("unknown result predicate wrong: %+v", res.Score)
	}
	if res.SupportsEntry() {
		t.Fatal("unknown result must not support entry")
	}
}

func TestPortfolioExposureDecisionHelpers(t *testing.T) {
	acc := PortfolioExposureDecisionAccept()
	if !acc.AcceptedDecision() || !acc.Accepted || acc.Reason != "" {
		t.Fatalf("accept wrong: %+v", acc)
	}
	rej := PortfolioExposureDecisionReject("R")
	if rej.AcceptedDecision() || rej.Accepted || rej.Reason != "R" {
		t.Fatalf("reject wrong: %+v", rej)
	}
	none := PortfolioExposurePolicyNone()
	dec := none(PortfolioExposureContext{})
	if !dec.Accepted {
		t.Fatal("none policy should accept")
	}

	sector, err := NewSectorExposureLimit(2, 0.5)
	if err != nil {
		t.Fatalf("NewSectorExposureLimit: %v", err)
	}
	if _, err := NewSectorExposureLimit(-1, 0.5); err == nil {
		t.Fatal("expected sector validation error")
	}
	if _, err := NewSectorExposureLimit(1, 1.5); err == nil {
		t.Fatal("expected sector fraction error")
	}
	corr, err := NewCorrelationExposureLimit(0.5, 5)
	if err != nil {
		t.Fatalf("NewCorrelationExposureLimit: %v", err)
	}
	if _, err := NewCorrelationExposureLimit(2, 5); err == nil {
		t.Fatal("expected corr validation error")
	}
	if _, err := NewCorrelationExposureLimit(0.5, 1); err == nil {
		t.Fatal("expected lookback validation error")
	}
	policy := PortfolioExposurePolicyLimits(sector, corr)
	if policy == nil {
		t.Fatal("expected policy")
	}

	// panic on invalid direct struct
	func() {
		defer func() { _ = recover() }()
		_ = PortfolioExposurePolicyLimits(SectorExposureLimit{MaxPositions: -1, MaxCapitalFraction: 0.5}, corr)
	}()
}

func TestPortfolioExposurePolicyReturnsAndLimits(t *testing.T) {
	held, _ := coredecimal.Parse("100")
	candidate, _ := coredecimal.Parse("10")
	capital, _ := coredecimal.Parse("1000")
	ctx := PortfolioExposureContext{
		CandidateSymbol:   "A",
		CandidateSector:   strPtr("Tech"),
		CandidateNotional: candidate,
		InitialCapital:    capital,
		OpenHoldings: []PortfolioExposureContextHolding{
			{Symbol: "B", Sector: strPtr("Tech"), Notional: held},
		},
		ClosingPrices: map[string][]float64{
			"A": {10, 11, 12, 13, 14, 15},
			"B": {10, 10.5, 11, 11.5, 12, 12.5},
		},
	}
	sector, _ := NewSectorExposureLimit(0, 0.5) // max 0 positions -> reject
	corr, _ := NewCorrelationExposureLimit(0.1, 5)
	policy := PortfolioExposurePolicyLimits(sector, corr)
	dec := policy(ctx)
	if dec.Accepted {
		t.Fatal("expected sector position rejection")
	}
	// capital limit path
	sector2, _ := NewSectorExposureLimit(5, 0.05)
	policy2 := PortfolioExposurePolicyLimits(sector2, CorrelationExposureLimit{LookbackDays: 2, MaxAbsoluteCorrelation: 1.0})
	dec2 := policy2(ctx)
	if dec2.Accepted {
		t.Fatal("expected sector capital rejection")
	}
	// correlation rejection with high corr
	sector3, _ := NewSectorExposureLimit(5, 1.0)
	corr2, _ := NewCorrelationExposureLimit(0.01, 5)
	policy3 := PortfolioExposurePolicyLimits(sector3, corr2)
	dec3 := policy3(ctx)
	if dec3.Accepted {
		t.Fatal("expected correlation rejection")
	}
	// accept all
	sector4, _ := NewSectorExposureLimit(5, 1.0)
	corr3, _ := NewCorrelationExposureLimit(1.0, 5)
	policy4 := PortfolioExposurePolicyLimits(sector4, corr3)
	dec4 := policy4(ctx)
	if !dec4.Accepted {
		t.Fatalf("expected accept, got %s", dec4.Reason)
	}
}

func TestPortfolioExposureReturnsAndCorrelation(t *testing.T) {
	sector4, _ := NewSectorExposureLimit(5, 1.0)
	// cancellation/NaN guard in returns
	r := returns([]float64{1, 0, 2}, 10)
	if r != nil {
		// returns returns empty slice on bad price; len should be 0
		if len(r) != 0 {
			t.Fatalf("returns should reject non-positive: %v", r)
		}
	}
	r2 := returns([]float64{1, math.NaN(), 2}, 10)
	if len(r2) != 0 {
		t.Fatalf("returns should reject nan: %v", r2)
	}
	r3 := returns(nil, 10)
	if r3 != nil {
		t.Fatalf("nil returns should be nil: %v", r3)
	}
	// correlation of empty
	if !math.IsNaN(correlation([]float64{}, []float64{})) {
		t.Fatal("expected nan correlation for empty")
	}
	c := correlation([]float64{1, 2, 3}, []float64{1, 2, 3})
	if math.Abs(c-1) > 1e-9 {
		t.Fatalf("perfect correlation expected 1, got %f", c)
	}

	// direct constructor validation panic branch already tested via assignment? exercise policyLimits panic for corr
	func() {
		defer func() { _ = recover() }()
		_ = PortfolioExposurePolicyLimits(sector4, CorrelationExposureLimit{LookbackDays: 0, MaxAbsoluteCorrelation: 0.5})
	}()
}

func strPtr(s string) *string { return &s }
