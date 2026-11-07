package strategy

import (
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

type intradayBuyAlways struct{}

func (intradayBuyAlways) Type() string             { return "BUY_ALWAYS" }
func (intradayBuyAlways) ParamSchema() ParamSchema { return intradayEmptySchema() }
func (intradayBuyAlways) RequiredIndicators(StrategyParamsView) []IndicatorKey {
	return []IndicatorKey{}
}
func (intradayBuyAlways) WarmupBars(StrategyParamsView) int { return 0 }
func (intradayBuyAlways) CrossFieldRules() []CrossFieldRule { return []CrossFieldRule{} }
func (intradayBuyAlways) EvaluateEntry(ctx MarketContext, index int, params StrategyParamsView) (Decision, error) {
	return intradayDecision(true, "buy", "buy", ctx.View(index).Close()), nil
}
func (intradayBuyAlways) EvaluateExit(ctx MarketContext, index int, position OpenPosition, params StrategyParamsView) (ExitDecision, error) {
	return ExitHold(), nil
}

func intradayEngineBars() []IntradayBar {
	ist := time.FixedZone("IST", 5*60*60+30*60)
	specs := []struct{ o, h, l, c string }{
		{"100", "101", "99", "100"},
		{"100", "102", "99", "101"},
		{"101", "103", "100", "102"},
		{"102", "104", "101", "103"},
		{"103", "105", "102", "104"},
	}
	bars := make([]IntradayBar, 0, len(specs))
	for i, s := range specs {
		o := mustParseDecimal(s.o)
		h := mustParseDecimal(s.h)
		l := mustParseDecimal(s.l)
		c := mustParseDecimal(s.c)
		b, err := NewIntradayBar(time.Date(2024, 1, 2, 9, 15+i*15, 0, 0, ist), o, h, l, c, 1000)
		if err != nil {
			panic(err)
		}
		bars = append(bars, b)
	}
	return bars
}

func TestIntradayBacktestEngineRun(t *testing.T) {
	engine := NewIntradayBacktestEngine(func(time.Time) bool { return true }, DefaultIntradayCostModel(), IntradayBacktestConfigDefaults())
	res, err := engine.Run("T", intradayEngineBars(), intradayBuyAlways{}, DefaultIntradayBacktestParams())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trades) == 0 {
		t.Fatal("expected trade")
	}
	// zero bars path is inside run but bars filtered? Use one bar triggers no position? coverage enough.
	_, err = engine.Run("", intradayEngineBars(), intradayBuyAlways{}, DefaultIntradayBacktestParams())
	if err == nil {
		t.Fatal("expected symbol error")
	}
	_, err = engine.Run("T", intradayEngineBars(), nil, DefaultIntradayBacktestParams())
	if err == nil {
		t.Fatal("expected strategy error")
	}
	// empty input yields empty result, not error
	empty, err := engine.Run("T", []IntradayBar{}, intradayBuyAlways{}, DefaultIntradayBacktestParams())
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Trades) != 0 {
		t.Fatal("expected no trades")
	}
}

func TestIntradayBacktestEngineRunAsOf(t *testing.T) {
	engine := NewIntradayBacktestEngine(func(time.Time) bool { return true }, DefaultIntradayCostModel(), IntradayBacktestConfigDefaults())
	ist := time.FixedZone("IST", 5*60*60+30*60)
	bars := intradayEngineBars()
	sessionDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	asOf := time.Date(2024, 1, 2, 9, 30, 0, 0, ist)
	res, err := engine.RunAsOf("T", bars, intradayBuyAlways{}, DefaultIntradayBacktestParams(), sessionDate, asOf)
	if err != nil {
		t.Fatal(err)
	}
	_ = res
	_, err = engine.RunAsOf("T", nil, intradayBuyAlways{}, DefaultIntradayBacktestParams(), sessionDate, asOf)
	if err == nil {
		t.Fatal("expected nil input error")
	}
	_, err = engine.RunAsOf("T", bars, intradayBuyAlways{}, DefaultIntradayBacktestParams(), time.Time{}, asOf)
	if err == nil {
		t.Fatal("expected zero session date error")
	}
	_, err = engine.RunAsOf("T", bars, intradayBuyAlways{}, DefaultIntradayBacktestParams(), sessionDate, time.Time{})
	if err == nil {
		t.Fatal("expected zero asof error")
	}
	_, err = engine.RunAsOf("T", bars, intradayBuyAlways{}, DefaultIntradayBacktestParams(), sessionDate, asOf.Add(5*time.Second))
	if err == nil {
		t.Fatal("expected nonzero second error")
	}
	_, err = engine.RunAsOf("T", bars, intradayBuyAlways{}, DefaultIntradayBacktestParams(), sessionDate, time.Date(2024, 1, 2, 10, 5, 0, 0, ist))
	if err == nil {
		t.Fatal("expected unknown start error")
	}
	// missing as-of bar: filter bars to exclude 9:30
	filtered := []IntradayBar{bars[0], bars[2], bars[3], bars[4]}
	_, err = engine.RunAsOf("T", filtered, intradayBuyAlways{}, DefaultIntradayBacktestParams(), sessionDate, asOf)
	if err == nil {
		t.Fatal("expected missing as-of error")
	}
}

func TestIntradayCostModelAndHelpers(t *testing.T) {
	m := DefaultIntradayCostModel()
	b := m.BuyFill(mustParseDecimal("100"))
	s := m.SellFill(mustParseDecimal("100"))
	c, err := m.RoundTripCost(b, s, 2)
	if err != nil {
		t.Fatal(err)
	}
	_ = c
	_, err = m.RoundTripCost(b, s, 0)
	if err == nil {
		t.Fatal("expected qty error")
	}
	_, err = NewIntradayCostModel(mustParseDecimal("-1"), mustParseDecimal("0.001"))
	if err == nil {
		t.Fatal("expected cost error")
	}
	if !isNormalSessionMinutes(9*60 + 15) {
		t.Fatal("session minutes")
	}
	// barStartMatches exact equality
	ist := time.FixedZone("IST", 5*60*60+30*60)
	bar := intradayEngineBars()[0]
	starts := data.ExpectedNSESessionBarStarts(bar.SessionDate(), true)
	if !barStartMatches(starts, bar) {
		t.Fatal("barStartMatches")
	}
	_ = ist
}
