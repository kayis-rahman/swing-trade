package strategy

import (
	"testing"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// indicators builds one bar's readings from Java's StrategyFamilyTest order:
// price, ema20, ema50, rsi, volume, volumeMa, weeklyHigh.
func familyIndicators(t *testing.T, price, ema20, ema50, rsi, volume, volumeMA, weeklyHigh string) *Indicators {
	t.Helper()
	return &Indicators{
		Price:      mustParseDecisionDecimal(t, price),
		Ema20:      mustParseDecisionDecimal(t, ema20),
		Ema50:      mustParseDecisionDecimal(t, ema50),
		Rsi:        mustParseDecisionDecimal(t, rsi),
		Volume:     mustParseDecisionDecimal(t, volume),
		VolumeMA:   mustParseDecisionDecimal(t, volumeMA),
		WeeklyHigh: mustParseDecisionDecimal(t, weeklyHigh),
	}
}

// Ported from StrategyFamilyTest.pullbackInUptrend_acceptsControlledPullback.
func TestPullbackInUptrendAcceptsControlledPullback(t *testing.T) {
	strategy := PullbackInUptrendStrategy{}
	if !strategy.IsEntrySignal(familyIndicators(t, "98", "100", "95", "52", "110", "100", "105")) {
		t.Fatal("controlled pullback was rejected")
	}
	if strategy.IsEntrySignal(familyIndicators(t, "92", "100", "95", "52", "110", "100", "105")) {
		t.Fatal("deep pullback passed the ratio band")
	}
}

// Ported from StrategyFamilyTest.volatilitySqueeze_requiresContractedVolumeAndBreakoutContext.
func TestVolatilitySqueezeRequiresContractedVolume(t *testing.T) {
	strategy := VolatilitySqueezeStrategy{}
	if !strategy.IsEntrySignal(familyIndicators(t, "99", "98", "95", "60", "75", "100", "102")) {
		t.Fatal("contracted volume breakout was rejected")
	}
	if strategy.IsEntrySignal(familyIndicators(t, "99", "98", "95", "60", "85", "100", "102")) {
		t.Fatal("expanded volume passed the squeeze gate")
	}
}

// Ported from StrategyFamilyTest.fiftyTwoWeekHighBreakout_requiresHighCloseAndTwoTimesVolume.
func TestFiftyTwoWeekHighBreakoutRequiresHighCloseAndDoubleVolume(t *testing.T) {
	strategy := FiftyTwoWeekHighBreakoutStrategy{}
	if !strategy.IsEntrySignal(familyIndicators(t, "106", "100", "95", "70", "200", "100", "105")) {
		t.Fatal("new-high breakout was rejected")
	}
	if strategy.IsEntrySignal(familyIndicators(t, "106", "100", "95", "70", "199", "100", "105")) {
		t.Fatal("volume below2x passed")
	}
	if strategy.IsEntrySignal(familyIndicators(t, "105", "100", "95", "70", "200", "100", "105")) {
		t.Fatal("close at the weekly high passed")
	}
	if strategy.IsEntrySignal(familyIndicators(t, "104", "100", "95", "70", "200", "100", "105")) {
		t.Fatal("close below the weekly high passed")
	}
}

// Ported from StrategyFamilyTest.rsi2MeanReversion_requiresOversoldReadingAboveLongTermTrend.
func TestRsi2MeanReversionRequiresOversoldAboveTrend(t *testing.T) {
	strategy := Rsi2MeanReversionStrategy{}
	oversold := familyIndicators(t, "101", "100", "95", "9", "100", "100", "110")
	if !strategy.IsEntrySignal(oversold) {
		t.Fatal("oversold RSI2 entry was rejected")
	}
	if strategy.IsEntrySignal(familyIndicators(t, "101", "100", "95", "10", "100", "100", "110")) {
		t.Fatal("RSI at the threshold passed")
	}
	if !strategy.IsSignalExit(oversold) {
		t.Fatal("price reclaiming EMA20 did not exit")
	}
}

// Ported from StrategyFamilyTest.familyNamesAreStableAndProductionStrategyIsUnchanged.
func TestStrategyFamilyNamesAreStable(t *testing.T) {
	for strategy, want := range map[TradingStrategy]string{
		PullbackInUptrendStrategy{}:        "PULLBACK_UPTREND",
		VolatilitySqueezeStrategy{}:        "VOLATILITY_SQUEEZE",
		FiftyTwoWeekHighBreakoutStrategy{}: "52W_HIGH_BREAKOUT",
		Rsi2MeanReversionStrategy{}:        "RSI2_MEAN_REVERSION",
		PriceActionStrategy{}:              "PRICE_ACTION",
	} {
		if strategy.Name() != want {
			t.Fatalf("name = %q, want %q", strategy.Name(), want)
		}
	}
}

// overlayTestStrategy drives the Java default overlays (regime on/off,
// relative strength on/off) that the five legacy strategies inherit.
type overlayTestStrategy struct {
	TradingStrategyDefaults
	entry    bool
	regimeOn bool
	rsOn     bool
	regimes  []domain.MarketRegime
}

func (s overlayTestStrategy) Name() string                                { return "OVERLAY_TEST" }
func (s overlayTestStrategy) RegimeFilterEnabled() bool                   { return s.regimeOn }
func (s overlayTestStrategy) AllowedMarketRegimes() []domain.MarketRegime { return s.regimes }
func (s overlayTestStrategy) RelativeStrengthFilterEnabled() bool         { return s.rsOn }
func (s overlayTestStrategy) IsEntrySignal(*Indicators) bool              { return s.entry }
func (s overlayTestStrategy) IsSignalExit(*Indicators) bool               { return false }
func (s overlayTestStrategy) TrendAligned(*Indicators) bool               { return false }
func (s overlayTestStrategy) RsiInEntryRange(*Indicators) bool            { return false }
func (s overlayTestStrategy) VolumeSurge(*Indicators) bool                { return false }
func (s overlayTestStrategy) NearWeeklyHigh(*Indicators) bool             { return false }
func (s overlayTestStrategy) CloseBelowEma20(*Indicators) bool            { return false }
func (s overlayTestStrategy) Ema20BelowEma50(*Indicators) bool            { return false }
func (s overlayTestStrategy) RsiBelowLowerBound(*Indicators) bool         { return false }

func TestEntryEligibilityFailsClosedOnMissingFacts(t *testing.T) {
	indicators := familyIndicators(t, "101", "100", "95", "55", "100", "100", "110")
	bullish := domain.MarketRegimeAssessment{Regime: domain.MarketRegimeBullish, Eligible: true, Reason: "up"}
	bearish := domain.MarketRegimeAssessment{Regime: domain.MarketRegimeBearish, Eligible: true, Reason: "down"}
	strength := domain.RelativeStrengthAssessment{Eligible: true, Reason: "outperform"}
	weak := domain.RelativeStrengthAssessment{Eligible: false, Reason: "underperform"}

	regimeStrategy := overlayTestStrategy{entry: true, regimeOn: true, regimes: []domain.MarketRegime{domain.MarketRegimeBullish}}
	if !IsEntryEligible(regimeStrategy, indicators, &bullish) {
		t.Fatal("eligible bullish regime blocked the entry")
	}
	if IsEntryEligible(regimeStrategy, indicators, &bearish) {
		t.Fatal("disallowed bearish regime passed")
	}
	if IsEntryEligible(regimeStrategy, indicators, nil) {
		t.Fatal("missing regime data failed open")
	}

	rsStrategy := overlayTestStrategy{entry: true, rsOn: true}
	if !IsEntryEligibleWithRelativeStrength(rsStrategy, indicators, nil, &strength) {
		t.Fatal("eligible relative strength blocked the entry")
	}
	if IsEntryEligibleWithRelativeStrength(rsStrategy, indicators, nil, &weak) {
		t.Fatal("weak relative strength passed")
	}
	if IsEntryEligibleWithRelativeStrength(rsStrategy, indicators, nil, nil) {
		t.Fatal("missing relative strength failed open")
	}

	// With overlays off, no regime or strength facts are required.
	plain := overlayTestStrategy{entry: true}
	if !IsEntryEligible(plain, indicators, nil) || !IsEntryEligibleWithRelativeStrength(plain, indicators, nil, nil) {
		t.Fatal("disabled overlays should not gate the entry")
	}
	// A failed entry signal never qualifies.
	if IsEntryEligible(overlayTestStrategy{entry: false}, indicators, nil) {
		t.Fatal("failed entry signal passed eligibility")
	}
}

func TestTradingStrategyDefaultsMatchJava(t *testing.T) {
	priceAction := PriceActionStrategy{}
	if priceAction.RegimeFilterEnabled() || priceAction.RelativeStrengthFilterEnabled() {
		t.Fatal("PriceAction must keep both overlays off by default")
	}
	if len(priceAction.AllowedMarketRegimes()) != 1 || priceAction.AllowedMarketRegimes()[0] != domain.MarketRegimeBullish {
		t.Fatalf("default regimes = %v", priceAction.AllowedMarketRegimes())
	}
	if priceAction.RequiredEntryRules() != 4 ||
		priceAction.EntryConfluenceDescription() != "All entry rules passed" ||
		priceAction.EntryRsiDescription() != "RSI between 50-65" {
		t.Fatalf("default metadata = %d / %q / %q",
			priceAction.RequiredEntryRules(), priceAction.EntryConfluenceDescription(), priceAction.EntryRsiDescription())
	}
	// The four legacy overlay strategies opt in explicitly.
	for _, strategy := range []TradingStrategy{
		PullbackInUptrendStrategy{}, VolatilitySqueezeStrategy{},
		FiftyTwoWeekHighBreakoutStrategy{}, Rsi2MeanReversionStrategy{},
	} {
		if !strategy.RegimeFilterEnabled() {
			t.Fatalf("%s must opt into the regime overlay", strategy.Name())
		}
	}
}

func TestPriceActionStrategyRulesUseStrategyParams(t *testing.T) {
	strategy := PriceActionStrategy{}
	if !strategy.RsiInEntryRange(familyIndicators(t, "101", "100", "95", "65", "100", "100", "110")) {
		t.Fatal("RSI at the upper band must pass")
	}
	if strategy.RsiInEntryRange(familyIndicators(t, "101", "100", "95", "66", "100", "100", "110")) {
		t.Fatal("RSI above the upper band passed")
	}
	if !strategy.VolumeSurge(familyIndicators(t, "101", "100", "95", "55", "151", "100", "110")) {
		t.Fatal("volume above1.5x must pass")
	}
	if strategy.VolumeSurge(familyIndicators(t, "101", "100", "95", "55", "150", "100", "110")) {
		t.Fatal("volume at exactly1.5x must fail the strict comparison")
	}
	if !strategy.NearWeeklyHigh(familyIndicators(t, "104", "100", "95", "55", "100", "100", "105")) {
		t.Fatal("price at97% of the weekly high must pass")
	}
	if !strategy.RsiBelowLowerBound(familyIndicators(t, "101", "100", "95", "49", "100", "100", "110")) {
		t.Fatal("RSI below the lower band must report the exit rule")
	}
	if !strategy.IsEntrySignal(familyIndicators(t, "101", "100", "95", "55", "200", "100", "103")) {
		t.Fatal("full four-rule confluence was rejected")
	}
}

func TestStrategyParamConstantsMatchJava(t *testing.T) {
	if domain.EMAFast != 20 || domain.EMASlow != 50 || domain.MinCandles != 50 ||
		domain.FiftyTwoWeekTradingDays != 252 || domain.RSIPeriod != 14 {
		t.Fatalf("period constants drifted: fast=%d slow=%d min=%d days=%d rsi=%d",
			domain.EMAFast, domain.EMASlow, domain.MinCandles, domain.FiftyTwoWeekTradingDays, domain.RSIPeriod)
	}
	if domain.RSILower.String() != "50.0000" || domain.RSIUpper.String() != "65.0000" ||
		domain.VolumeMultiplier.String() != "1.5000" || domain.HighProximity.String() != "0.9700" {
		t.Fatalf("threshold constants drifted: %s %s %s %s",
			domain.RSILower, domain.RSIUpper, domain.VolumeMultiplier, domain.HighProximity)
	}
}

func TestRoundQuotientMatchesHalfUp(t *testing.T) {
	for _, tc := range []struct {
		num, den string
		scale    int
		want     string
	}{
		{"0.125", "1", 2, "13"},
		{"-0.125", "1", 2, "-13"},
		{"1", "3", 6, "333333"},
		{"98", "100", 8, "98000000"},
	} {
		if got := roundQuotient(tc.num, tc.den, tc.scale); got != tc.want {
			t.Fatalf("roundQuotient(%s, %s, %d) = %s, want %s", tc.num, tc.den, tc.scale, got, tc.want)
		}
	}
	zero := mustParseDecisionDecimal(t, "0")
	if got := CompareRoundedRatio(mustParseDecisionDecimal(t, "1"), zero, mustParseDecisionDecimal(t, "1"), 8); got != 0 {
		t.Fatalf("zero denominator comparison = %d, want 0", got)
	}
}
