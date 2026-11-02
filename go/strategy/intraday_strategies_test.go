package strategy

import (
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func intradayCandles() ([]domain.OhlcvCandle, []time.Time) {
	date := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	candles := []domain.OhlcvCandle{
		domain.OhlcvCandleOf("T", date, mustRuleDecimal("100"), mustRuleDecimal("101"), mustRuleDecimal("99"), mustRuleDecimal("100"), 1000),
		domain.OhlcvCandleOf("T", date, mustRuleDecimal("100"), mustRuleDecimal("102"), mustRuleDecimal("99"), mustRuleDecimal("101"), 1000),
		domain.OhlcvCandleOf("T", date, mustRuleDecimal("101"), mustRuleDecimal("103"), mustRuleDecimal("100"), mustRuleDecimal("102"), 1000),
	}
	ts := []time.Time{
		time.Date(2024, 1, 2, 9, 15, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 9, 30, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 9, 45, 0, 0, time.UTC),
	}
	return candles, ts
}

func TestIntradayStrategiesBasic(t *testing.T) {
	candles, ts := intradayCandles()
	ctx, err := NewMarketContextIntraday("T", candles, ts)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []SignalStrategy{IntradayOpeningRangeBreakoutStrategy{}, IntradayVwapPullbackStrategy{}, IntradayRsi2Strategy{}} {
		dec, err := s.EvaluateEntry(ctx, 2, StrategyParamsViewOf(map[string]any{}))
		if err != nil {
			t.Fatal(err)
		}
		_, _ = s.ParamSchema(), dec
		if s.WarmupBars(StrategyParamsView{}) == 0 {
			t.Fatal("warmup")
		}
		if s.RequiredIndicators(StrategyParamsView{}) == nil {
			t.Fatal("req")
		}
		if s.CrossFieldRules() == nil {
			t.Fatal("cross")
		}
	}
	// exit paths
	pos, _ := OpenPositionAt(0, candles[0].Date, mustRuleDecimal("100"), mustRuleDecimal("99"), mustRuleDecimal("102"), 1)
	for _, s := range []SignalStrategy{IntradayOpeningRangeBreakoutStrategy{}, IntradayVwapPullbackStrategy{}, IntradayRsi2Strategy{}} {
		ed, err := s.EvaluateExit(ctx, 2, pos, StrategyParamsViewOf(map[string]any{}))
		if err != nil {
			t.Fatal(err)
		}
		_ = ed
	}
	// unknown utility
	if intradayEmptySchema().Params == nil {
		t.Fatal("schema")
	}
	_ = intradayDecision(false, "k", "d", mustRuleDecimal("1"))
	_ = mulDecimal(mustRuleDecimal("2"), mustRuleDecimal("3"))
}

func TestRSNiftyMomentumBasic(t *testing.T) {
	s := RelativeStrengthMomentumStrategy{}
	ind := familyIndicators(t, "101", "100", "95", "55", "100", "100", "110")
	_ = s.IsEntrySignal(ind)
	_ = s.TrendAligned(ind)
	_ = s.RsiInEntryRange(ind)
	_ = s.VolumeSurge(ind)
	_ = s.NearWeeklyHigh(ind)
	_ = s.CloseBelowEma20(ind)
	_ = s.Ema20BelowEma50(ind)
	_ = s.RsiBelowLowerBound(ind)
	_ = s.IsSignalExit(ind)
	_ = s.Name()
	_ = s.RegimeFilterEnabled()
	_ = s.RelativeStrengthFilterEnabled()
	_ = s.AllowedMarketRegimes()
	ns, _ := NewRelativeStrengthMomentumStrategy(mustRuleDecimal("0"), false, mustRuleDecimal("0.90"))
	_ = ns
}
