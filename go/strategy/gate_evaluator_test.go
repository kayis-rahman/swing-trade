package strategy

import (
	"strings"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

// gateDate mirrors GateEvaluatorTest's pinned LocalDate.of(2026, 1, 15).
var gateDate = time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

func mustParseDecisionDecimal(t *testing.T, value string) coredecimal.Decimal {
	t.Helper()
	decimalValue, err := coredecimal.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return decimalValue
}

func buyDecision(t *testing.T) Decision {
	t.Helper()
	one := mustParseDecisionDecimal(t, "1")
	stop := mustParseDecisionDecimal(t, "90")
	target := mustParseDecisionDecimal(t, "110")
	decision, err := NewDecision(domain.SignalTypeBuy, one, nil, &stop, &target, "test buy")
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func holdDecision(t *testing.T) Decision {
	t.Helper()
	decision, err := NewDecision(domain.SignalTypeHold, mustParseDecisionDecimal(t, "0"), nil, nil, nil, "test hold")
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

// fakeIndexSeries mirrors GateEvaluatorTest's FakeIndexSeries.
type fakeIndexSeries struct {
	hasData     bool
	close       coredecimal.Decimal
	ema         coredecimal.Decimal
	emaLookback coredecimal.Decimal
}

func closeAboveEmaIndex(t *testing.T) fakeIndexSeries {
	t.Helper()
	return fakeIndexSeries{hasData: true,
		close: mustParseDecisionDecimal(t, "20000"), ema: mustParseDecisionDecimal(t, "19000"),
		emaLookback: mustParseDecisionDecimal(t, "18500")}
}

func closeBelowEmaIndex(t *testing.T) fakeIndexSeries {
	t.Helper()
	return fakeIndexSeries{hasData: true,
		close: mustParseDecisionDecimal(t, "18000"), ema: mustParseDecisionDecimal(t, "19000"),
		emaLookback: mustParseDecisionDecimal(t, "18500")}
}

func flatEmaSlopeIndex(t *testing.T) fakeIndexSeries {
	t.Helper()
	return fakeIndexSeries{hasData: true,
		close: mustParseDecisionDecimal(t, "20000"), ema: mustParseDecisionDecimal(t, "19000"),
		emaLookback: mustParseDecisionDecimal(t, "19000")}
}

func noDataIndex() fakeIndexSeries { return fakeIndexSeries{hasData: false} }

func (f fakeIndexSeries) HasDataOn(time.Time) bool { return f.hasData }

func (f fakeIndexSeries) Close(time.Time) coredecimal.Decimal { return f.close }

func (f fakeIndexSeries) EMA(int, time.Time) coredecimal.Decimal { return f.ema }

func (f fakeIndexSeries) EMALookback(int, time.Time, int) coredecimal.Decimal { return f.emaLookback }

// --- regimeGate (ported from GateEvaluatorTest) ---

func TestRegimeGateAllowsBuyWhenIndexCloseAboveEma(t *testing.T) {
	result := GateEvaluator{}.ApplyRegimeGate(buyDecision(t), gateDate, closeAboveEmaIndex(t), ptrRegime(RegimeGateDefaults()))
	if result.AnyBlocked() {
		t.Fatal("close above EMA blocked the BUY")
	}
	if result.Decision.Type != domain.SignalTypeBuy {
		t.Fatalf("decision = %s, want BUY", result.Decision.Type)
	}
	if len(result.Outcomes) != 1 || result.Outcomes[0].Blocked {
		t.Fatalf("outcomes = %+v", result.Outcomes)
	}
}

func TestRegimeGateBlocksBuyWhenIndexCloseBelowEma(t *testing.T) {
	result := GateEvaluator{}.ApplyRegimeGate(buyDecision(t), gateDate, closeBelowEmaIndex(t), ptrRegime(RegimeGateDefaults()))
	if !result.AnyBlocked() {
		t.Fatal("close below EMA did not block the BUY")
	}
	if result.Decision.Type != domain.SignalTypeHold {
		t.Fatalf("decision = %s, want HOLD", result.Decision.Type)
	}
	if result.Outcomes[0].Gate != "regimeGate" {
		t.Fatalf("outcome gate = %q", result.Outcomes[0].Gate)
	}
	if !strings.HasPrefix(result.Decision.Reasoning, "Gate-blocked: Index regime is down") {
		t.Fatalf("reasoning = %q", result.Decision.Reasoning)
	}
}

func TestRegimeGateEMASlopeUpModeBlocksFlatSlope(t *testing.T) {
	config, err := NewRegimeGateConfig(200, RegimeGateModeEMASlopeUp, 10)
	if err != nil {
		t.Fatal(err)
	}
	result := GateEvaluator{}.ApplyRegimeGate(buyDecision(t), gateDate, flatEmaSlopeIndex(t), ptrRegime(config))
	if !result.AnyBlocked() {
		t.Fatal("flat EMA slope did not block the BUY")
	}
}

func TestRegimeGateNeverBlocksANonBuyDecision(t *testing.T) {
	result := GateEvaluator{}.ApplyRegimeGate(holdDecision(t), gateDate, closeBelowEmaIndex(t), ptrRegime(RegimeGateDefaults()))
	if result.AnyBlocked() || len(result.Outcomes) != 0 {
		t.Fatalf("HOLD decision was gated: %+v", result)
	}
}

func TestRegimeGateSkipsGracefullyWhenNoIndexDataAvailable(t *testing.T) {
	result := GateEvaluator{}.ApplyRegimeGate(buyDecision(t), gateDate, noDataIndex(), ptrRegime(RegimeGateDefaults()))
	if result.AnyBlocked() || result.Decision.Type != domain.SignalTypeBuy {
		t.Fatalf("missing index data blocked the BUY: %+v", result)
	}
	if len(result.Outcomes) != 1 || result.Outcomes[0].Blocked ||
		result.Outcomes[0].Reason != "No index data available; gate skipped" {
		t.Fatalf("skip outcome = %+v", result.Outcomes)
	}
}

func TestRegimeGateWithoutIndexRecordsSkipAndWithoutConfigPassesThrough(t *testing.T) {
	// Java records a skip outcome when the index series is absent.
	skip := GateEvaluator{}.ApplyRegimeGate(buyDecision(t), gateDate, nil, ptrRegime(RegimeGateDefaults()))
	if skip.AnyBlocked() || len(skip.Outcomes) != 1 || skip.Outcomes[0].Blocked {
		t.Fatalf("nil index result = %+v", skip)
	}
	// A nil config means the overlay is not configured: no outcomes at all.
	passThrough := GateEvaluator{}.ApplyRegimeGate(buyDecision(t), gateDate, closeBelowEmaIndex(t), nil)
	if passThrough.AnyBlocked() || len(passThrough.Outcomes) != 0 {
		t.Fatalf("nil config result = %+v", passThrough)
	}
}

// --- sentimentGate (ported from GateEvaluatorTest) ---

func sentimentResult(t *testing.T, score domain.SentimentScore, date time.Time) *domain.SentimentResult {
	t.Helper()
	summary := "sentiment summary"
	confidence := 0.8
	result := domain.CreateSentimentResult("SYM", date, score, &summary, nil, &confidence, nil, nil)
	return &result
}

func TestSentimentGateAllowsBuyWhenSentimentMeetsMinimum(t *testing.T) {
	result := GateEvaluator{}.ApplySentimentGate(buyDecision(t), gateDate,
		sentimentResult(t, domain.SentimentScorePositive, gateDate), ptrSentiment(SentimentGateDefaults()))
	if result.AnyBlocked() || result.Decision.Type != domain.SignalTypeBuy {
		t.Fatalf("positive sentiment blocked the BUY: %+v", result)
	}
}

func TestSentimentGateBlocksBuyWhenSentimentIsNegative(t *testing.T) {
	result := GateEvaluator{}.ApplySentimentGate(buyDecision(t), gateDate,
		sentimentResult(t, domain.SentimentScoreNegative, gateDate), ptrSentiment(SentimentGateDefaults()))
	if !result.AnyBlocked() || result.Decision.Type != domain.SignalTypeHold {
		t.Fatalf("negative sentiment result = %+v", result)
	}
}

func TestSentimentGateBlocksBuyWhenSentimentIsStale(t *testing.T) {
	staleDate := gateDate.AddDate(0, 0, -10)
	result := GateEvaluator{}.ApplySentimentGate(buyDecision(t), gateDate,
		sentimentResult(t, domain.SentimentScorePositive, staleDate),
		ptrSentiment(NewSentimentGateConfig(domain.SentimentScoreNeutral, 3)))
	if !result.AnyBlocked() {
		t.Fatal("stale sentiment did not block the BUY")
	}
	if !strings.Contains(result.Outcomes[0].Reason, "stale") {
		t.Fatalf("reason = %q", result.Outcomes[0].Reason)
	}
	want := "Sentiment is 10 days stale (max 3); BUY blocked"
	if result.Outcomes[0].Reason != want {
		t.Fatalf("reason = %q, want %q", result.Outcomes[0].Reason, want)
	}
}

func TestSentimentGateBlocksBuyWhenNoSentimentAvailable(t *testing.T) {
	result := GateEvaluator{}.ApplySentimentGate(buyDecision(t), gateDate, nil, ptrSentiment(SentimentGateDefaults()))
	if !result.AnyBlocked() {
		t.Fatal("missing sentiment did not block the BUY")
	}
	if result.Outcomes[0].Reason != "No sentiment available; BUY blocked" {
		t.Fatalf("reason = %q", result.Outcomes[0].Reason)
	}
}

func TestSentimentGateNeverBlocksANonBuyDecision(t *testing.T) {
	result := GateEvaluator{}.ApplySentimentGate(holdDecision(t), gateDate, nil, ptrSentiment(SentimentGateDefaults()))
	if result.AnyBlocked() || len(result.Outcomes) != 0 {
		t.Fatalf("HOLD decision was gated: %+v", result)
	}
}

func TestSentimentGateBelowMinimumBlocks(t *testing.T) {
	result := GateEvaluator{}.ApplySentimentGate(buyDecision(t), gateDate,
		sentimentResult(t, domain.SentimentScoreNegative, gateDate), ptrSentiment(SentimentGateDefaults()))
	if !result.AnyBlocked() {
		t.Fatal("negative sentiment below the NEUTRAL minimum passed")
	}
	want := "Sentiment NEGATIVE below minimum NEUTRAL; BUY blocked"
	if result.Outcomes[0].Reason != want {
		t.Fatalf("reason = %q, want %q", result.Outcomes[0].Reason, want)
	}
}

func TestSentimentGateTreatsUnknownAsNeutralRanked(t *testing.T) {
	// UNKNOWN ranks with NEUTRAL (1), so it clears a NEUTRAL minimum exactly
	// like Java's SentimentGateConfig.rank mapping.
	result := GateEvaluator{}.ApplySentimentGate(buyDecision(t), gateDate,
		sentimentResult(t, domain.SentimentScoreUnknown, gateDate), ptrSentiment(SentimentGateDefaults()))
	if result.AnyBlocked() {
		t.Fatalf("UNKNOWN sentiment ranked below its neutral group: %+v", result)
	}
}

// --- combined (ported from GateEvaluatorTest) ---

func TestApplyAllAggregatesOutcomesFromBothGates(t *testing.T) {
	result := GateEvaluator{}.ApplyAll(buyDecision(t), gateDate, closeAboveEmaIndex(t),
		ptrRegime(RegimeGateDefaults()),
		sentimentResult(t, domain.SentimentScorePositive, gateDate),
		ptrSentiment(SentimentGateDefaults()))
	if len(result.Outcomes) != 2 || result.AnyBlocked() || result.Decision.Type != domain.SignalTypeBuy {
		t.Fatalf("aggregated result = %+v", result)
	}
}

func TestApplyAllBlocksIfEitherGateBlocks(t *testing.T) {
	result := GateEvaluator{}.ApplyAll(buyDecision(t), gateDate, closeAboveEmaIndex(t),
		ptrRegime(RegimeGateDefaults()), nil, ptrSentiment(SentimentGateDefaults()))
	if !result.AnyBlocked() || result.Decision.Type != domain.SignalTypeHold {
		t.Fatalf("missing sentiment did not block the combined run: %+v", result)
	}
	// The regime pass is still recorded before the sentiment block.
	if len(result.Outcomes) != 2 || result.Outcomes[0].Blocked || !result.Outcomes[1].Blocked {
		t.Fatalf("outcome order = %+v", result.Outcomes)
	}
}

// --- configs, decision, and outcome validation ---

func TestRegimeGateConfigDefaultsAndValidation(t *testing.T) {
	defaults := RegimeGateDefaults()
	if defaults.IndexEma != 200 || defaults.Mode != RegimeGateModeCloseAbove || defaults.SlopeLookbackDays != 10 {
		t.Fatalf("defaults = %+v", defaults)
	}
	normalized, err := NewRegimeGateConfig(50, "", -1)
	if err != nil || normalized.Mode != RegimeGateModeCloseAbove || normalized.SlopeLookbackDays != 10 {
		t.Fatalf("normalized config = %+v, %v", normalized, err)
	}
	if _, err := NewRegimeGateConfig(0, RegimeGateModeCloseAbove, 10); err == nil ||
		err.Error() != "indexEma must be positive" {
		t.Fatalf("indexEma validation = %v", err)
	}
}

func TestSentimentGateConfigDefaultsAndRankOrdering(t *testing.T) {
	defaults := SentimentGateDefaults()
	if defaults.MinScore != domain.SentimentScoreNeutral || defaults.MaxAgeDays != 3 {
		t.Fatalf("defaults = %+v", defaults)
	}
	normalized := NewSentimentGateConfig("", 0)
	if normalized.MinScore != domain.SentimentScoreNeutral || normalized.MaxAgeDays != 3 {
		t.Fatalf("normalized config = %+v", normalized)
	}
}

func TestDecisionValidationMessages(t *testing.T) {
	one := mustParseDecisionDecimal(t, "1")
	if _, err := NewDecision("", one, nil, nil, nil, "x"); err == nil ||
		err.Error() != "Decision type cannot be null" {
		t.Fatalf("type validation = %v", err)
	}
	negative := mustParseDecisionDecimal(t, "-0.1")
	if _, err := NewDecision(domain.SignalTypeBuy, negative, nil, nil, nil, "x"); err == nil ||
		!strings.HasPrefix(err.Error(), "Score must be in [0,1]:") {
		t.Fatalf("score validation = %v", err)
	}
	tooHigh := mustParseDecisionDecimal(t, "1.1")
	if _, err := NewDecision(domain.SignalTypeBuy, tooHigh, nil, nil, nil, "x"); err == nil {
		t.Fatal("score above one accepted")
	}
	// Rules are copied defensively like List.copyOf.
	rule, err := NewRuleOutcome("trendAligned", true, mustParseDecisionDecimal(t, "2"), true, "ok")
	if err != nil {
		t.Fatal(err)
	}
	decision, err := NewDecision(domain.SignalTypeBuy, one, []RuleOutcome{rule}, nil, nil, "x")
	if err != nil {
		t.Fatal(err)
	}
	rule.Passed = false
	if !decision.Rules[0].Passed {
		t.Fatal("rules slice aliasing Java's immutable copy")
	}
}

func TestRuleOutcomeValidationMessages(t *testing.T) {
	if _, err := NewRuleOutcome("  ", true, mustParseDecisionDecimal(t, "1"), false, ""); err == nil ||
		err.Error() != "Rule key cannot be null or blank" {
		t.Fatalf("key validation = %v", err)
	}
	if _, err := NewRuleOutcome("key", true, mustParseDecisionDecimal(t, "-1"), false, ""); err == nil ||
		err.Error() != "Rule weight must be non-negative" {
		t.Fatalf("weight validation = %v", err)
	}
}

func TestGateOutcomeValidationAndFactories(t *testing.T) {
	if _, err := NewGateOutcome(" ", false, "x", nil); err == nil ||
		err.Error() != "gate cannot be null or blank" {
		t.Fatalf("gate validation = %v", err)
	}
	pass := GateOutcomePass("regimeGate", "ok")
	block := GateOutcomeBlock("sentimentGate", "blocked")
	if pass.Blocked || !block.Blocked || pass.Score != nil || block.Score != nil {
		t.Fatalf("factories = %+v / %+v", pass, block)
	}
	result := NewGateResult(buyDecisionForOutcomeTest(t), nil)
	if len(result.Outcomes) != 0 || result.AnyBlocked() {
		t.Fatalf("nil outcomes = %+v", result)
	}
}

func buyDecisionForOutcomeTest(t *testing.T) Decision { return buyDecision(t) }

func ptrRegime(config RegimeGateConfig) *RegimeGateConfig { return &config }

func ptrSentiment(config SentimentGateConfig) *SentimentGateConfig { return &config }
