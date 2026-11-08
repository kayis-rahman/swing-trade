package domain

import (
	"fmt"
	"strings"
	"time"
)

// BenchmarkCandleSeries mirrors Java com.swingtrade.domain.BenchmarkCandleSeries.
type BenchmarkCandleSeries struct {
	Benchmark string
	From      time.Time
	To        time.Time
	Candles   []OhlcvCandle
}

// NewBenchmarkCandleSeries validates and copies input.
func NewBenchmarkCandleSeries(benchmark string, from, to time.Time, candles []OhlcvCandle) (BenchmarkCandleSeries, error) {
	if strings.TrimSpace(benchmark) == "" {
		return BenchmarkCandleSeries{}, fmt.Errorf("Benchmark cannot be blank")
	}
	if from.IsZero() || to.IsZero() || from.After(to) {
		return BenchmarkCandleSeries{}, fmt.Errorf("Benchmark window must be valid")
	}
	copied := []OhlcvCandle{}
	if candles != nil {
		copied = append(copied, candles...)
	}
	return BenchmarkCandleSeries{Benchmark: benchmark, From: from, To: to, Candles: copied}, nil
}
