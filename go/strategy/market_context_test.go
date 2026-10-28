package strategy

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/core/domain"
)

func buildSimpleCandles(count int) []domain.OhlcvCandle {
	candles := make([]domain.OhlcvCandle, 0, count)
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		candles = append(candles, domain.OhlcvCandleOf("TESTCO", date,
			mustDecimal("100"), mustDecimal("101"),
			mustDecimal("99"), mustDecimal("100"), 1000000))
		date = date.AddDate(0, 0, 1)
	}
	return candles
}

func TestMarketContextBounds(t *testing.T) {
	ctx := MarketContextOf("TESTCO", buildSimpleCandles(10))
	if ctx.BarCount() != 10 {
		t.Fatalf("bar count")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	ctx.View(10)
}

func TestMarketContextViewDefault(t *testing.T) {
	ctx := MarketContextOf("TESTCO", buildSimpleCandles(60))
	view := ctx.View(40)
	if view.Close().Cmp(view.Close(40)) != 0 {
		t.Fatalf("default close mismatch")
	}
	if view.EMA(20).Cmp(view.EMA(20, 40)) != 0 {
		t.Fatalf("default ema mismatch")
	}
	// look-ahead
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic on look-ahead")
			}
		}()
		view.Close(41)
	}()
}

func TestMarketContextCache(t *testing.T) {
	ctx := MarketContextOf("TESTCO", buildSimpleCandles(300))
	barIndex := 250
	for i := 0; i < 12; i++ {
		view := ctx.View(barIndex)
		_ = view.EMA(20)
		_ = view.RSI(14)
	}
	if ctx.CachedIndicatorCount() != 2 {
		t.Fatalf("expected 2 cached, got %d", ctx.CachedIndicatorCount())
	}
	ctx2 := MarketContextOf("TESTCO", buildSimpleCandles(300))
	view2 := ctx2.View(250)
	_ = view2.EMA(20)
	_ = view2.EMA(20)
	_ = view2.EMA(50)
	_ = view2.RSI(14)
	if ctx2.CachedIndicatorCount() != 3 {
		t.Fatalf("expected 3 cached, got %d", ctx2.CachedIndicatorCount())
	}
}

func mustDecimal(s string) coredecimal.Decimal {
	d,_ := coredecimal.Parse(s)
	return d
}
