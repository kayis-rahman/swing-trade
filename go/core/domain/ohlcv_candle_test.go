package domain

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestOhlcvCandleConstructorsAndBasicMethods(t *testing.T) {
	open, _ := coredecimal.Parse("10")
	high, _ := coredecimal.Parse("12")
	low, _ := coredecimal.Parse("9")
	close, _ := coredecimal.Parse("11")
	candle := NewOhlcvCandle("TCS", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), open, high, low, close, 100, close)
	if candle.Symbol != "TCS" || candle.Volume == nil || *candle.Volume != 100 {
		t.Fatalf("constructor did not populate candle: %#v", candle)
	}
	if got := candle.GetRange(); got.String() != "3.0000" {
		t.Fatalf("GetRange() = %s, want 3.0000", got.String())
	}
	if !candle.IsBullish() || candle.IsBearish() {
		t.Fatal("unexpected bullish/bearish classification")
	}
	adjusted := candle.AdjustedForAnalysis()
	if adjusted.Close.String() != "11.0000" {
		t.Fatalf("adjusted close = %s, want 11", adjusted.Close.String())
	}
	_ = OhlcvCandleOf
}
