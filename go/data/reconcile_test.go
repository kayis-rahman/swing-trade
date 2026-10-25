package data

import (
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestCompareCandleSetsIgnoresRowOrder(t *testing.T) {
	first := validYahooCandle()
	second := first
	second.Date = second.Date.AddDate(0, 0, 1)
	differences := CompareCandleSets([]YahooCandle{first, second}, []YahooCandle{second, first})
	if len(differences) != 0 {
		t.Fatalf("differences = %+v", differences)
	}
}

func TestCompareCandleSetsReportsMissingDuplicateAndFieldDifferencesDeterministically(t *testing.T) {
	want := validYahooCandle()
	got := want
	got.Close, _ = coredecimal.Parse("11.01")
	got.Volume++
	missing := validYahooCandle()
	missing.Date = time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	unexpected := validYahooCandle()
	unexpected.Date = time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	expected := []YahooCandle{want, want, missing}
	actual := []YahooCandle{got, unexpected, got}
	first := CompareCandleSets(expected, actual)
	second := CompareCandleSets(expected, actual)
	if len(first) != 6 {
		t.Fatalf("got %d differences: %+v", len(first), first)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("difference order is unstable: %v vs %v", first, second)
		}
	}
	fields := map[string]bool{}
	for _, diff := range first {
		fields[diff.Field] = true
	}
	for _, field := range []string{"close", "volume", "row", "duplicate"} {
		if !fields[field] {
			t.Errorf("missing %q difference in %+v", field, first)
		}
	}
}

func TestCompareCandleSetsIdentityIncludesTimeframeAndBarTime(t *testing.T) {
	first := validYahooCandle()
	second := first
	second.Timeframe = "D"
	third := first
	third.BarTime = third.BarTime.Add(15 * time.Minute)
	differences := CompareCandleSets([]YahooCandle{first}, []YahooCandle{second, third})
	if len(differences) != 3 {
		t.Fatalf("differences = %+v", differences)
	}
}
