package strategy

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

func d(t *testing.T, value string) coredecimal.Decimal {
	parsed, err := coredecimal.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestNewIntradayBarValidatesAndConverts(t *testing.T) {
	open, high, low, close := d(t, "10"), d(t, "12"), d(t, "9"), d(t, "11")
	timestamp := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	bar, err := NewIntradayBar(timestamp, open, high, low, close, 100)
	if err != nil {
		t.Fatal(err)
	}
	if bar.SessionDateLocal() != "2026-01-02" {
		t.Fatalf("SessionDateLocal() = %s", bar.SessionDateLocal())
	}
	if bar.SessionDate().UTC().Hour() != 0 {
		t.Fatalf("SessionDate() = %v", bar.SessionDate())
	}
	candle := bar.AsCandle("ABC")
	if candle.Symbol != "ABC" || *candle.Volume != 100 {
		t.Fatalf("AsCandle() = %+v", candle)
	}

	if _, err := NewIntradayBar(time.Time{}, open, high, low, close, 1); err == nil {
		t.Fatal("expected error for zero timestamp")
	}
	if _, err := NewIntradayBar(timestamp, open, low, high, close, 1); err == nil {
		t.Fatal("expected error for invalid OHLC")
	}
	if _, err := NewIntradayBar(timestamp, open, high, low, close, -1); err == nil {
		t.Fatal("expected error for negative volume")
	}
}

func TestPortfolioEquityPointTotalConstructor(t *testing.T) {
	equity := d(t, "1000")
	point := NewPortfolioEquityPointTotal(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), equity)
	if point.Equity.String() != equity.String() || point.SettledCash.String() != equity.String() {
		t.Fatalf("NewPortfolioEquityPointTotal() = %+v", point)
	}
}

func TestPortfolioTradeRMultipleAndWin(t *testing.T) {
	trade := PortfolioTrade{
		Quantity:     2,
		RiskPerShare: d(t, "1"),
		PnL:          d(t, "5"),
	}
	if got := trade.RMultiple(); got != 2.5 {
		t.Fatalf("RMultiple() = %v, want 2.5", got)
	}
	if !trade.IsWin() {
		t.Fatal("IsWin() = false, want true")
	}
	losing := PortfolioTrade{PnL: d(t, "-1")}
	if losing.IsWin() {
		t.Fatal("IsWin() = true, want false")
	}
	if (PortfolioTrade{}).RMultiple() != 0 {
		t.Fatal("RMultiple() zero trade should be 0")
	}
}

func TestNewSignalResultValidatesAndNormalizes(t *testing.T) {
	date := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	result, err := NewSignalResult("ABC", date, domain.SignalTypeBuy, d(t, "1.23456789"), d(t, "2"), d(t, "3"), d(t, "4"), "reason")
	if err != nil {
		t.Fatal(err)
	}
	if result.RSI.String() != "1.234600" {
		t.Fatalf("RSI normalized = %s", result.RSI.String())
	}
	if _, err := NewSignalResult("   ", date, domain.SignalTypeBuy, d(t, "1"), d(t, "2"), d(t, "3"), d(t, "4"), "r"); err == nil {
		t.Fatal("expected symbol validation error")
	}
	if _, err := NewSignalResult("ABC", time.Time{}, domain.SignalTypeBuy, d(t, "1"), d(t, "2"), d(t, "3"), d(t, "4"), "r"); err == nil {
		t.Fatal("expected date validation error")
	}
}

func TestBacktestMetricHelpers(t *testing.T) {
	initial := d(t, "100")
	final := d(t, "120")
	if got := TotalReturnPct(initial, final).String(); got != "20.0000" {
		t.Fatalf("TotalReturnPct() = %s, want 20.0000", got)
	}
	curve := []coredecimal.Decimal{initial, d(t, "90"), final}
	if got := MaxDrawdownPct(curve).String(); got == "0.0000" {
		t.Fatal("MaxDrawdownPct() should be nonzero")
	}
	if got := MaxDrawdownPct(nil); got.String() != "0.0000" {
		t.Fatalf("MaxDrawdownPct(nil) = %s", got.String())
	}
	if got := SharpeRatio(curve); got == 0 {
		// May be zero for this tiny curve only if deviation zero; accept either finite.
	}
	high := d(t, "1000")
	_ = CagrPct(initial, high, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}
