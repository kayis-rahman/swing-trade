package strategy

import (
	"fmt"
	"math"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

func ppDate(y, m, d int) time.Time { return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC) }

func ppDecimal(v float64) coredecimal.Decimal {
	d, _ := coredecimal.Parse(fmt.Sprintf("%.10g", v))
	return d
}

func ppTrade(symbol string, entry, exit time.Time, entryPrice float64, qty int, pnl float64) BacktestTrade {
	entryD := ppDecimal(entryPrice)
	return BacktestTrade{
		Symbol:      symbol,
		EntryDate:   entry,
		ExitDate:    exit,
		EntryPrice:  entryD,
		ExitPrice:   entryD,
		StopLoss:    ppDecimal(entryPrice - 1),
		Target:      ppDecimal(entryPrice + 1),
		Quantity:    qty,
		ExitReason:  ExitReasonTimeStop,
		PnL:         ppDecimal(pnl),
		PnLPct:      ppDecimal(pnl),
		HoldingDays: 1,
	}
}

func ppResult(symbol string, trade BacktestTrade) BacktestResult {
	win := 0
	lose := 0
	if trade.PnL.Cmp(coredecimal.Decimal{}) > 0 {
		win = 1
	} else {
		lose = 1
	}
	return BacktestResult{
		Symbol:        symbol,
		TotalTrades:   1,
		WinningTrades: win,
		LosingTrades:  lose,
		Trades:        []BacktestTrade{trade},
	}
}

func ppCandle(symbol string, d time.Time, close float64, low float64) domain.OhlcvCandle {
	p := ppDecimal(close)
	hi := ppDecimal(close)
	if low > close {
		hi = ppDecimal(low)
	}
	lo := ppDecimal(low)
	if low > close {
		lo = ppDecimal(close)
	}
	return domain.OhlcvCandleOf(symbol, d, p, hi, lo, p, 1)
}

func TestPortfolioLiveSharesCapitalAndRejectsEntry(t *testing.T) {
	held := ppTrade("AAA", ppDate(2024, 1, 2), ppDate(2024, 1, 5), 60, 1, 10)
	rejected := ppTrade("BBB", ppDate(2024, 1, 3), ppDate(2024, 1, 6), 60, 1, 10)
	cfg := PortfolioBacktestConfigDefaults(ppDecimal(100))
	res, err := PortfolioBacktestEngineSimulate(
		[]BacktestResult{ppResult("AAA", held), ppResult("BBB", rejected)},
		cfg, ppDate(2024, 1, 1), ppDate(2024, 1, 6))
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalTrades != 1 {
		t.Fatalf("total trades %d", res.TotalTrades)
	}
	if res.RejectedTrades != 1 {
		t.Fatalf("rejected %d", res.RejectedTrades)
	}
	if len(res.Trades) != 1 || res.Trades[0].Symbol != "AAA" {
		t.Fatalf("trades %+v", res.Trades)
	}
	if res.FinalCapital.Cmp(ppDecimal(110)) != 0 {
		t.Fatalf("final capital %s", res.FinalCapital)
	}
	if len(res.EquityCurve) != 6 {
		t.Fatalf("eq len %d", len(res.EquityCurve))
	}
}

func TestPortfolioLiveBenchmarkVariantAndMetrics(t *testing.T) {
	held := ppTrade("AAA", ppDate(2024, 1, 2), ppDate(2024, 1, 4), 100, 1, 10)
	candles := map[string][]domain.OhlcvCandle{
		"AAA": {ppCandle("AAA", ppDate(2024, 1, 2), 100, 100), ppCandle("AAA", ppDate(2024, 1, 3), 110, 110), ppCandle("AAA", ppDate(2024, 1, 4), 110, 110)},
	}
	nifty := domain.BenchmarkCandleSeries{
		Benchmark: "NIFTY50",
		From:      ppDate(2024, 1, 2), To: ppDate(2024, 1, 4),
		Candles: []domain.OhlcvCandle{
			ppCandle("NIFTY50", ppDate(2024, 1, 2), 100, 100),
			ppCandle("NIFTY50", ppDate(2024, 1, 4), 105, 105),
		},
	}
	cfg := PortfolioBacktestConfigDefaults(ppDecimal(100))
	variant := ""
	res, err := PortfolioBacktestEngineSimulate(
		[]BacktestResult{ppResult("AAA", held)}, cfg,
		ppDate(2024, 1, 2), ppDate(2024, 1, 4),
		candles, map[string]string{}, nifty, variant)
	if err != nil {
		t.Fatal(err)
	}
	if res.StrategyVariantID == nil || *res.StrategyVariantID != "" {
		t.Fatalf("variant %+v", res.StrategyVariantID)
	}
	if res.BenchmarkComparison.BenchmarkReturnPct != 5.0 {
		t.Fatalf("bench %v", res.BenchmarkComparison.BenchmarkReturnPct)
	}
	if res.TotalReturn.Cmp(coredecimal.Decimal{}) == 0 {
		t.Fatalf("total return zero unexpectedly")
	}
}

func TestPortfolioLiveInvalidInputs(t *testing.T) {
	cfg := PortfolioBacktestConfigDefaults(ppDecimal(100))
	_, err := PortfolioBacktestEngineSimulate(nil, cfg, ppDate(2024, 1, 2), ppDate(2024, 1, 1))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPortfolioLiveFiniteGuardExposure(t *testing.T) {
	_, err := domain.NewSectorExposureLimit(1, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = domain.NewSectorExposureLimit(1, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	// finite guard: NaN should error
	_, err = domain.NewSectorExposureLimit(1, math.NaN())
	if err == nil {
		t.Fatal("expected NaN error")
	}
}

func TestPortfolioLiveSettlementFallbackLastObservation(t *testing.T) {
	held := ppTrade("AAA", ppDate(2024, 1, 2), ppDate(2024, 1, 4), 100, 1, 10)
	candles := map[string][]domain.OhlcvCandle{
		"AAA": {
			ppCandle("AAA", ppDate(2024, 1, 2), 100, 100),
			ppCandle("AAA", ppDate(2024, 1, 3), 110, 110),
			ppCandle("AAA", ppDate(2024, 1, 4), 110, 110),
		},
	}
	cfg := PortfolioBacktestConfigDefaults(ppDecimal(100))
	res, err := PortfolioBacktestEngineSimulate(
		[]BacktestResult{ppResult("AAA", held)}, cfg,
		ppDate(2024, 1, 2), ppDate(2024, 1, 4), candles)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.EquityCurve) == 0 {
		t.Fatal("no equity")
	}
	// proceeds from exit on last observation settle next day if no map entry -> exit date+1
}
