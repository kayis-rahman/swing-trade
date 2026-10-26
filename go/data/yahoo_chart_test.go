package data

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestParseYahooChartPreservesExchangeDatesDecimalsAndAdjustedClose(t *testing.T) {
	// Friday and Saturday timestamps near UTC midnight cross into the next IST date.
	friday := time.Date(2026, 10, 22, 18, 45, 0, 0, time.UTC).Unix()
	saturday := time.Date(2026, 10, 24, 18, 45, 0, 0, time.UTC).Unix()
	payload := `{"chart":{"result":[{"timestamp":[` + itoa(friday) + `,` + itoa(saturday) + `],"meta":{"timezone":"Asia/Kolkata"},"indicators":{"quote":[{"open":[100.12345,100],"high":[105,105],"low":[99,99],"close":[103.98765,103],"volume":[1200,1200]}],"adjclose":[{"adjclose":[102.5,102]}]}}]}}`
	candles, err := ParseYahooChart([]byte(payload), "ABC.NS", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), "1d")
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 {
		t.Fatalf("got %d candles, want Friday only: %+v", len(candles), candles)
	}
	candle := candles[0]
	if candle.Symbol != "ABC.NS" || candle.Date.Format("2006-01-02") != "2026-10-23" {
		t.Fatalf("candle identity = %s %s", candle.Symbol, candle.Date.Format("2006-01-02"))
	}
	if candle.Open.String() != "100.1235" || candle.Close.String() != "103.9877" || candle.AdjustedClose.String() != "102.5000" {
		t.Fatalf("unexpected decimal values: %+v", candle)
	}
	if !candle.BarTime.Equal(time.Date(2026, 10, 23, 0, 0, 0, 0, candle.Date.Location())) {
		t.Fatalf("daily bar time = %v, want midnight exchange time", candle.BarTime)
	}
}

func TestParseYahooChartFallsBackToCloseAndRejectsInvalidBars(t *testing.T) {
	friday := time.Date(2026, 10, 23, 4, 0, 0, 0, time.UTC).Unix()
	saturday := time.Date(2026, 10, 24, 4, 0, 0, 0, time.UTC).Unix()
	payload := `{"chart":{"result":[{"timestamp":[` + itoa(friday) + `,` + itoa(saturday) + `],"meta":{"timezone":"Asia/Kolkata"},"indicators":{"quote":[{"open":[10,null],"high":[12,12],"low":[9,9],"close":[11,11],"volume":[5,5]}]}}]}}`
	candles, err := ParseYahooChart([]byte(payload), "ABC", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), "15m")
	if err != nil {
		t.Fatal(err)
	}
	if len(candles) != 1 {
		t.Fatalf("got %d candles, want one valid weekday bar", len(candles))
	}
	if candles[0].AdjustedClose.String() != candles[0].Close.String() {
		t.Fatalf("adjusted-close fallback = %s, close = %s", candles[0].AdjustedClose, candles[0].Close)
	}
	if candles[0].BarTime.Hour() == 0 && candles[0].BarTime.Minute() == 0 {
		t.Fatalf("intraday bar time was normalized: %v", candles[0].BarTime)
	}
}

func TestParseYahooChartReportsMalformedJSON(t *testing.T) {
	_, err := ParseYahooChart([]byte("{"), "ABC", time.Time{}, time.Time{}, "1d")
	if err == nil || !strings.Contains(err.Error(), "decode Yahoo chart response") {
		t.Fatalf("error = %v", err)
	}
}

func itoa(value int64) string {
	return fmt.Sprintf("%d", value)
}
