package data

import (
	"fmt"
	"sort"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// CandleDifference describes a row-level mismatch between two historical datasets.
type CandleDifference struct {
	Identity string
	Field    string
	Expected string
	Actual   string
}

// CompareCandleSets compares candle rows independent of input ordering.
// The first slice is expected; the second is actual.
func CompareCandleSets(expected, actual []YahooCandle) []CandleDifference {
	want := indexCandles(expected)
	got := indexCandles(actual)
	differences := make([]CandleDifference, 0)
	for key, duplicate := range want.duplicates {
		differences = append(differences, CandleDifference{Identity: key, Field: "duplicate", Expected: fmt.Sprint(duplicate)})
	}
	for key, duplicate := range got.duplicates {
		differences = append(differences, CandleDifference{Identity: key, Field: "duplicate", Actual: fmt.Sprint(duplicate)})
	}
	for key, expectedCandle := range want.rows {
		actualCandle, ok := got.rows[key]
		if !ok {
			differences = append(differences, CandleDifference{Identity: key, Field: "row", Expected: "present", Actual: "missing"})
			continue
		}
		compareCandleFields(&differences, key, expectedCandle, actualCandle)
	}
	for key := range got.rows {
		if _, ok := want.rows[key]; !ok {
			differences = append(differences, CandleDifference{Identity: key, Field: "row", Expected: "missing", Actual: "present"})
		}
	}
	sort.Slice(differences, func(i, j int) bool {
		if differences[i].Identity != differences[j].Identity {
			return differences[i].Identity < differences[j].Identity
		}
		return differences[i].Field < differences[j].Field
	})
	return differences
}

type candleIndex struct {
	rows       map[string]YahooCandle
	duplicates map[string]int
}

func indexCandles(candles []YahooCandle) candleIndex {
	index := candleIndex{rows: make(map[string]YahooCandle, len(candles)), duplicates: make(map[string]int)}
	for _, candle := range candles {
		key := candleIdentity(candle)
		if _, exists := index.rows[key]; exists {
			index.duplicates[key]++
			continue
		}
		index.rows[key] = candle
	}
	return index
}

func candleIdentity(candle YahooCandle) string {
	return fmt.Sprintf("%s|%s|%s|%s", candle.Symbol, candle.Timeframe, candle.Date.Format("2006-01-02"), candle.BarTime.Format("15:04:05"))
}

func compareCandleFields(differences *[]CandleDifference, identity string, expected, actual YahooCandle) {
	compareDecimal(differences, identity, "open", expected.Open, actual.Open)
	compareDecimal(differences, identity, "high", expected.High, actual.High)
	compareDecimal(differences, identity, "low", expected.Low, actual.Low)
	compareDecimal(differences, identity, "close", expected.Close, actual.Close)
	compareDecimal(differences, identity, "adjustedClose", expected.AdjustedClose, actual.AdjustedClose)
	if expected.Volume != actual.Volume {
		*differences = append(*differences, CandleDifference{Identity: identity, Field: "volume", Expected: fmt.Sprint(expected.Volume), Actual: fmt.Sprint(actual.Volume)})
	}
}

func compareDecimal(differences *[]CandleDifference, identity, field string, expected, actual coredecimal.Decimal) {
	if expected.Cmp(actual) != 0 {
		*differences = append(*differences, CandleDifference{Identity: identity, Field: field, Expected: expected.String(), Actual: actual.String()})
	}
}
