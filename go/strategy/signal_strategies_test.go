package strategy

import (
	"strconv"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func testBar(symbol string, date time.Time, open, close float64, highMult, lowMult float64, volume int64) domain.OhlcvCandle {
	high := open
	if close > high {
		high = close
	}
	low := open
	if close < low {
		low = close
	}
	high *= highMult
	low *= lowMult
	return domain.OhlcvCandleOf(
		symbol,
		date,
		coredecimalFromFloat(open),
		coredecimalFromFloat(high),
		coredecimalFromFloat(low),
		coredecimalFromFloat(close),
		volume,
	)
}

func coredecimalFromFloat(value float64) coredecimal.Decimal {
	parsed, err := coredecimal.Parse(strconv.FormatFloat(value, 'f', -1, 64))
	if err != nil {
		panic(err)
	}
	return parsed
}

func ruleByKey(t *testing.T, decision Decision, key string) RuleOutcome {
	t.Helper()
	for _, rule := range decision.Rules {
		if rule.Key == key {
			return rule
		}
	}
	t.Fatalf("no rule outcome for %q", key)
	return RuleOutcome{}
}

func pullbackFixture() []domain.OhlcvCandle {
	candles := make([]domain.OhlcvCandle, 0, 97)
	price := 100.0
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 90; i++ {
		close := price * 1.004
		candles = append(candles, testBar("PBTEST", date, price, close, 1.002, 0.998, 1_000_000))
		price = close
		date = date.AddDate(0, 0, 1)
	}
	for i := 0; i < 6; i++ {
		close := price * 0.985
		candles = append(candles, testBar("PBTEST", date, price, close, 1.002, 0.998, 1_000_000))
		price = close
		date = date.AddDate(0, 0, 1)
	}
	candles = append(candles, testBar("PBTEST", date, price, price*1.02, 1.002, 0.998, 1_000_000))
	return candles
}

func pullbackParams() StrategyParamsView {
	return StrategyParamsViewOf(map[string]any{
		"trendEma":            50,
		"pullbackEma":         20,
		"touchPct":            decimalDefault("0.02"),
		"touchLookback":       8,
		"rsiPeriod":           14,
		"rsiDip":              decimalDefault("40"),
		"rsiTrigger":          decimalDefault("45"),
		"slopeLookback":       10,
		"requireConfirmation": false,
		"entryScoreThreshold": decimalDefault("0.5"),
		"atrPeriod":           14,
		"atrStopMult":         decimalDefault("2"),
		"rewardRisk":          decimalDefault("2"),
		"maxHoldDays":         20,
		"trailAtrMult":        zeroDecimal(),
	})
}

func squeezeFixture() []domain.OhlcvCandle {
	candles := make([]domain.OhlcvCandle, 0, 111)
	price := 100.0
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 70; i++ {
		close := price * 1.003
		candles = append(candles, testBar("SQTEST", date, price, close, 1.001, 0.999, 1_000_000))
		price = close
		date = date.AddDate(0, 0, 1)
	}
	for i := 0; i < 40; i++ {
		factor := 1.0005
		if i%2 == 1 {
			factor = 0.9995
		}
		close := price * factor
		candles = append(candles, testBar("SQTEST", date, price, close, 1.0005, 0.9995, 400_000))
		price = close
		date = date.AddDate(0, 0, 1)
	}
	candles = append(candles, testBar("SQTEST", date, price, price*1.06, 1.001, 0.999, 5_000_000))
	return candles
}

func squeezeParams() StrategyParamsView {
	return StrategyParamsViewOf(map[string]any{
		"bbPeriod":            20,
		"trendEma":            50,
		"squeezeDefinition":   "BB_PCTILE",
		"squeezeLookback":     60,
		"squeezePctile":       decimalDefault("20"),
		"squeezeRecency":      5,
		"volumeMaPeriod":      20,
		"volMult":             decimalDefault("1.5"),
		"entryScoreThreshold": decimalDefault("0.75"),
		"atrPeriod":           14,
		"atrStopMult":         decimalDefault("2"),
		"rewardRisk":          decimalDefault("2"),
		"maxHoldDays":         20,
		"trailAtrMult":        zeroDecimal(),
	})
}

func TestLegacyPriceActionAdapterTypeIsBreakout(t *testing.T) {
	adapter := NewLegacyPriceActionAdapter(PriceActionStrategy{})
	if adapter.Type() != "BREAKOUT" {
		t.Fatalf("Type() = %q, want BREAKOUT", adapter.Type())
	}
}

func TestLegacyDefaultParamsMatchStrategyParams(t *testing.T) {
	params := DefaultLegacyPriceActionParams()
	if got := params.GetInt("emaFast"); got != domain.EMAFast {
		t.Errorf("emaFast = %d, want %d", got, domain.EMAFast)
	}
	if got := params.GetInt("emaSlow"); got != domain.EMASlow {
		t.Errorf("emaSlow = %d, want %d", got, domain.EMASlow)
	}
	if got := params.GetDecimal("rsiMin"); got.Cmp(domain.RSILower) != 0 {
		t.Errorf("rsiMin = %s, want %s", got.String(), domain.RSILower.String())
	}
	if got := params.GetDecimal("rsiMax"); got.Cmp(domain.RSIUpper) != 0 {
		t.Errorf("rsiMax = %s, want %s", got.String(), domain.RSIUpper.String())
	}
	if got := params.GetDecimal("volMult"); got.Cmp(domain.VolumeMultiplier) != 0 {
		t.Errorf("volMult = %s, want %s", got.String(), domain.VolumeMultiplier.String())
	}
	if got := params.GetDecimal("highProximity"); got.Cmp(domain.HighProximity) != 0 {
		t.Errorf("highProximity = %s, want %s", got.String(), domain.HighProximity.String())
	}
	if got := params.GetDecimal("entryScoreThreshold"); got.Cmp(oneDecimal()) != 0 {
		t.Errorf("entryScoreThreshold = %s, want 1", got.String())
	}
}

func TestLegacyWarmupBarsIsMaxOfEmaSlowAndWeeklyHighPeriod(t *testing.T) {
	adapter := NewLegacyPriceActionAdapter(PriceActionStrategy{})
	want := maxInt(domain.EMASlow, domain.FiftyTwoWeekTradingDays)
	if got := adapter.WarmupBars(DefaultLegacyPriceActionParams()); got != want {
		t.Fatalf("WarmupBars() = %d, want %d", got, want)
	}
}

func TestLegacyRequiredIndicatorsIsNonEmpty(t *testing.T) {
	adapter := NewLegacyPriceActionAdapter(PriceActionStrategy{})
	if len(adapter.RequiredIndicators(DefaultLegacyPriceActionParams())) == 0 {
		t.Fatal("RequiredIndicators() is empty")
	}
}

func TestLegacyClassifyLegacyStyleReproducesSignalExitWiring(t *testing.T) {
	candles := make([]domain.OhlcvCandle, 0, 80)
	price := 100.0
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 80; i++ {
		close := price * 0.985
		candles = append(candles, testBar("CRASH", date, price, close, 1.001, 0.999, 1_000_000))
		price = close
		date = date.AddDate(0, 0, 1)
	}
	ctx := MarketContextOf("CRASH", candles)
	adapter := NewLegacyPriceActionAdapter(PriceActionStrategy{})
	got, err := adapter.ClassifyLegacyStyle(ctx, ctx.BarCount()-1, DefaultLegacyPriceActionParams())
	if err != nil {
		t.Fatalf("ClassifyLegacyStyle: %v", err)
	}
	if got != domain.SignalTypeSell {
		t.Fatalf("classify = %q, want SELL", got)
	}
}

func TestLegacyParamSchemaDescribesEveryDefaultParam(t *testing.T) {
	adapter := NewLegacyPriceActionAdapter(PriceActionStrategy{})
	schema := adapter.ParamSchema()
	if len(schema.Params) == 0 {
		t.Fatal("schema is empty")
	}
	required := []string{
		"emaFast", "emaSlow", "rsiMin", "rsiMax", "volMult", "highProximity",
		"entryScoreThreshold", "atrStopMult", "rewardRisk", "maxHoldDays", "trailAtrMult",
	}
	for _, name := range required {
		found := false
		for _, def := range schema.Params {
			if def.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("schema missing %q", name)
		}
	}
}

func TestLegacyCrossFieldRulesRejectBadPairs(t *testing.T) {
	adapter := NewLegacyPriceActionAdapter(PriceActionStrategy{})
	rules := adapter.CrossFieldRules()
	badEma := map[string]any{"emaFast": 60, "emaSlow": 20, "rsiMin": 50, "rsiMax": 65}
	badRsi := map[string]any{"emaFast": 20, "emaSlow": 50, "rsiMin": 70, "rsiMax": 65}
	for _, rule := range rules {
		switch {
		case contains(rule.Message, "emaFast") && rule.Test(badEma):
			t.Errorf("ema rule %q passed with emaFast > emaSlow", rule.Message)
		case contains(rule.Message, "rsiMin") && rule.Test(badRsi):
			t.Errorf("rsi rule %q passed with rsiMin > rsiMax", rule.Message)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func TestPullbackTrendUpRulePassesInUptrend(t *testing.T) {
	ctx := MarketContextOf("PBTEST", pullbackFixture())
	decision, err := NewPullbackStrategy().EvaluateEntry(ctx, 60, pullbackParams())
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	if !ruleByKey(t, decision, "trendUp").Passed {
		t.Fatal("trendUp rule did not pass in a steady uptrend")
	}
}

func TestPullbackTouchOrRsiRecoveredOnBounceBar(t *testing.T) {
	candles := pullbackFixture()
	ctx := MarketContextOf("PBTEST", candles)
	decision, err := NewPullbackStrategy().EvaluateEntry(ctx, len(candles)-1, pullbackParams())
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	touched := ruleByKey(t, decision, "touchedPullbackEma").Passed
	recovered := ruleByKey(t, decision, "rsiRecovered").Passed
	if !touched && !recovered {
		t.Fatal("neither pullback touch nor RSI recovery fired on the bounce bar")
	}
}

func TestPullbackTrendUpRuleFailsInDowntrend(t *testing.T) {
	candles := make([]domain.OhlcvCandle, 0, 70)
	price := 100.0
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 70; i++ {
		close := price * 0.996
		candles = append(candles, testBar("PBTEST", date, price, close, 1.002, 0.998, 1_000_000))
		price = close
		date = date.AddDate(0, 0, 1)
	}
	ctx := MarketContextOf("PBTEST", candles)
	decision, err := NewPullbackStrategy().EvaluateEntry(ctx, len(candles)-1, pullbackParams())
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	if decision.Type != domain.SignalTypeHold {
		t.Fatalf("decision = %q, want HOLD", decision.Type)
	}
	if ruleByKey(t, decision, "trendUp").Passed {
		t.Fatal("trendUp rule passed in a downtrend")
	}
}

func TestPullbackSuggestedStopIsMinOfSwingLowAndAtrStop(t *testing.T) {
	candles := pullbackFixture()
	ctx := MarketContextOf("PBTEST", candles)
	params := StrategyParamsViewOf(map[string]any{
		"trendEma":            50,
		"pullbackEma":         20,
		"touchPct":            decimalDefault("0.02"),
		"touchLookback":       8,
		"rsiPeriod":           14,
		"rsiDip":              decimalDefault("40"),
		"rsiTrigger":          decimalDefault("45"),
		"slopeLookback":       10,
		"requireConfirmation": false,
		"entryScoreThreshold": zeroDecimal(),
		"atrPeriod":           14,
		"atrStopMult":         decimalDefault("2"),
		"rewardRisk":          decimalDefault("2"),
		"maxHoldDays":         20,
		"trailAtrMult":        zeroDecimal(),
	})
	lastBar := len(candles) - 1
	decision, err := NewPullbackStrategy().EvaluateEntry(ctx, lastBar, params)
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	if decision.Type != domain.SignalTypeBuy {
		return
	}
	view := ctx.View(lastBar)
	swingLow := view.LowestLow(8)
	atrStop := view.Close().Sub(decimalProduct(view.ATR(14), decimalDefault("2")))
	want := swingLow
	if atrStop.Cmp(swingLow) < 0 {
		want = atrStop
	}
	if decision.SuggestedStop == nil || decision.SuggestedStop.Cmp(want) != 0 {
		t.Fatalf("suggestedStop = %v, want %s", decision.SuggestedStop, want.String())
	}
}

func TestSqueezeSqueezedRulePassesAfterConsolidation(t *testing.T) {
	candles := squeezeFixture()
	ctx := MarketContextOf("SQTEST", candles)
	decision, err := NewSqueezeStrategy().EvaluateEntry(ctx, len(candles)-1, squeezeParams())
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	if !ruleByKey(t, decision, "squeezed").Passed {
		t.Fatal("squeezed rule did not pass after a tight consolidation")
	}
}

func TestSqueezeExpansionAndVolumeSurgeFireOnBreakoutBar(t *testing.T) {
	candles := squeezeFixture()
	ctx := MarketContextOf("SQTEST", candles)
	decision, err := NewSqueezeStrategy().EvaluateEntry(ctx, len(candles)-1, squeezeParams())
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	for _, key := range []string{"expansion", "volumeSurge", "trendAligned"} {
		if !ruleByKey(t, decision, key).Passed {
			t.Errorf("%s rule did not pass on the breakout bar", key)
		}
	}
}

func TestSqueezeBbInKcDefinitionAlsoDetectsSqueeze(t *testing.T) {
	candles := squeezeFixture()
	ctx := MarketContextOf("SQTEST", candles)
	params := StrategyParamsViewOf(map[string]any{
		"bbPeriod":            20,
		"trendEma":            50,
		"squeezeDefinition":   "BB_IN_KC",
		"squeezeLookback":     60,
		"squeezePctile":       decimalDefault("20"),
		"squeezeRecency":      5,
		"volumeMaPeriod":      20,
		"volMult":             decimalDefault("1.5"),
		"entryScoreThreshold": decimalDefault("0.75"),
		"atrPeriod":           14,
		"atrStopMult":         decimalDefault("2"),
		"rewardRisk":          decimalDefault("2"),
		"maxHoldDays":         20,
		"trailAtrMult":        zeroDecimal(),
	})
	decision, err := NewSqueezeStrategy().EvaluateEntry(ctx, len(candles)-1, params)
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	if !ruleByKey(t, decision, "squeezed").Passed {
		t.Fatal("BB_IN_KC squeeze was not detected")
	}
}

func TestSqueezeNoSqueezeWhenVolatilityNeverContracts(t *testing.T) {
	candles := make([]domain.OhlcvCandle, 0, 130)
	price := 100.0
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 130; i++ {
		changePct := 0.02
		if i%2 == 1 {
			changePct = -0.015
		}
		close := price * (1 + changePct)
		candles = append(candles, testBar("SQTEST", date, price, close, 1.01, 0.99, 1_000_000))
		price = close
		date = date.AddDate(0, 0, 1)
	}
	ctx := MarketContextOf("SQTEST", candles)
	decision, err := NewSqueezeStrategy().EvaluateEntry(ctx, len(candles)-1, squeezeParams())
	if err != nil {
		t.Fatalf("EvaluateEntry: %v", err)
	}
	if ruleByKey(t, decision, "squeezed").Passed {
		t.Fatal("squeezed rule passed without a volatility contraction")
	}
}

func TestResolverSignalTypeResolvesWithDefaultsFilledIn(t *testing.T) {
	resolver := newTestResolver()
	config := testStrategyConfig(t, "PULLBACK", map[string]any{"trendEma": 60})
	resolved := resolver.Resolve(&config)
	if resolved.Kind != ResolvedStrategySignal {
		t.Fatalf("kind = %v, want signal (%s)", resolved.Kind, resolved.Reason)
	}
	if resolved.Strategy.Type() != "PULLBACK" {
		t.Fatalf("type = %q", resolved.Strategy.Type())
	}
	if got := resolved.Params.GetInt("trendEma"); got != 60 {
		t.Fatalf("trendEma = %d, want 60", got)
	}
	if got := resolved.Params.GetInt("pullbackEma"); got != 20 {
		t.Fatalf("pullbackEma = %d, want default 20", got)
	}
}

func TestResolverSignalAndLegacyNamesBothResolve(t *testing.T) {
	resolver := newTestResolver()
	breakout := testStrategyConfig(t, "BREAKOUT", map[string]any{})
	if resolved := resolver.Resolve(&breakout); resolved.Kind != ResolvedStrategySignal {
		t.Fatalf("BREAKOUT kind = %v, want signal (%s)", resolved.Kind, resolved.Reason)
	}
	legacy := testStrategyConfig(t, PriceActionName, map[string]any{})
	if resolved := resolver.Resolve(&legacy); resolved.Kind != ResolvedStrategyLegacy {
		t.Fatalf("legacy kind = %v, want legacy (%s)", resolved.Kind, resolved.Reason)
	}
}

func TestResolverUnknownTypeIsUnresolvedWithReason(t *testing.T) {
	resolver := newTestResolver()
	config := testStrategyConfig(t, "NOPE", map[string]any{})
	resolved := resolver.Resolve(&config)
	if resolved.Kind != ResolvedStrategyUnresolved {
		t.Fatalf("kind = %v, want unresolved", resolved.Kind)
	}
	if !contains(resolved.Reason, "NOPE") {
		t.Fatalf("reason = %q, want it to contain NOPE", resolved.Reason)
	}
}

func TestResolverInvalidParamsAreUnresolvedNotThrown(t *testing.T) {
	resolver := newTestResolver()
	unknownKey := testStrategyConfig(t, "PULLBACK", map[string]any{"bogus": 1})
	if resolved := resolver.Resolve(&unknownKey); resolved.Kind != ResolvedStrategyUnresolved {
		t.Fatalf("unknown key kind = %v, want unresolved", resolved.Kind)
	}
	outOfRange := testStrategyConfig(t, "PULLBACK", map[string]any{"trendEma": 100000})
	if resolved := resolver.Resolve(&outOfRange); resolved.Kind != ResolvedStrategyUnresolved {
		t.Fatalf("out of range kind = %v, want unresolved", resolved.Kind)
	}
}

func TestResolverNilConfigIsUnresolved(t *testing.T) {
	resolver := newTestResolver()
	if resolved := resolver.Resolve(nil); resolved.Kind != ResolvedStrategyUnresolved {
		t.Fatalf("kind = %v, want unresolved", resolved.Kind)
	}
}

func newTestResolver() *StrategyResolver {
	types := NewStrategyTypeRegistry(
		NewLegacyPriceActionAdapter(PriceActionStrategy{}),
		NewPullbackStrategy(),
		NewSqueezeStrategy(),
	)
	legacy := NewLegacyStrategyRegistry(PriceActionStrategy{})
	return NewStrategyResolver(types, legacy)
}

func testStrategyConfig(t *testing.T, strategyType string, params map[string]any) domain.StrategyConfig {
	t.Helper()
	config, err := domain.CreateStrategyConfig(
		"test-v1", 1, strategyType, params, map[string]any{},
		domain.StrategyConfigModeShadow, decimalDefault("100000"), true, nil, time.Now(),
	)
	if err != nil {
		t.Fatalf("CreateStrategyConfig: %v", err)
	}
	return config
}
