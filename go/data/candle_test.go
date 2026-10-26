package data

import (
	"testing"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestIsValidCandleMatchesJavaOhlcvRules(t *testing.T) {
	candle := validCandle(t)
	if !IsValidCandle(&candle) {
		t.Fatal("valid OHLCV candle rejected")
	}
	cases := []struct {
		name   string
		mutate func(*CandleData)
	}{
		{"null open", func(c *CandleData) { c.Open = nil }},
		{"null high", func(c *CandleData) { c.High = nil }},
		{"null low", func(c *CandleData) { c.Low = nil }},
		{"null close", func(c *CandleData) { c.Close = nil }},
		{"zero volume", func(c *CandleData) { c.Volume = 0 }},
		{"negative volume", func(c *CandleData) { c.Volume = -1 }},
		{"zero price", func(c *CandleData) { c.Open = decimalPtr(t, "0") }},
		{"negative price", func(c *CandleData) { c.Close = decimalPtr(t, "-1") }},
		{"high below low", func(c *CandleData) { c.High = decimalPtr(t, "99") }},
		{"high below open", func(c *CandleData) { c.High = decimalPtr(t, "100") }},
		{"high below close", func(c *CandleData) { c.High = decimalPtr(t, "100") }},
		{"low above open", func(c *CandleData) { c.Low = decimalPtr(t, "101") }},
		{"low above close", func(c *CandleData) { c.Low = decimalPtr(t, "101") }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := validCandle(t)
			test.mutate(&candidate)
			if IsValidCandle(&candidate) {
				t.Fatal("invalid candle accepted")
			}
		})
	}
	if IsValidCandle(nil) {
		t.Fatal("nil candle accepted")
	}
}

func validCandle(t *testing.T) CandleData {
	t.Helper()
	return CandleData{
		Symbol: "TCS",
		Open:   decimalPtr(t, "100.00"), High: decimalPtr(t, "102.00"),
		Low: decimalPtr(t, "99.00"), Close: decimalPtr(t, "101.00"), Volume: 1,
	}
}

func decimalPtr(t *testing.T, value string) *coredecimal.Decimal {
	t.Helper()
	parsed, err := coredecimal.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return &parsed
}
