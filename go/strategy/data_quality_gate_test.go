package strategy

import (
	"strconv"
	"strings"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/data"
)

func mustDecimalValue(t *testing.T, value string) coredecimal.Decimal {
	t.Helper()
	decimalValue, err := coredecimal.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return decimalValue
}

// weekdayCandles mirrors DataQualityGateTest.weekdayCandles: one candle per
// weekday in [start, end] with a constant or drifting base price.
func weekdayCandles(t *testing.T, start, end time.Time, basePrice, drift float64) []data.YahooCandle {
	t.Helper()
	var candles []data.YahooCandle
	price := basePrice
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		base := mustDecimalValue(t, formatFloat(price))
		high := mustDecimalValue(t, formatFloat(price*1.01))
		low := mustDecimalValue(t, formatFloat(price*0.99))
		candles = append(candles, data.YahooCandle{
			Symbol: "SYM", Timeframe: "D", Date: day,
			Open: base, High: high, Low: low, Close: base,
			AdjustedClose: base, Volume: 1_000_000,
		})
		price += drift
	}
	return candles
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func TestEvaluateDataQualityPassesCleanDailyCandles(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 28)
	candles := weekdayCandles(t, start, end, 100.0, 0.0)
	if reason, excluded := EvaluateDataQuality("CLEAN", candles, start, end); excluded {
		t.Fatalf("clean data excluded: %q", reason)
	}
}

func TestEvaluateDataQualityBlocksNonPositivePrice(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 14)
	candles := weekdayCandles(t, start, end, 100.0, 0.0)
	candles[3].Close = coredecimal.Decimal{}
	reason, excluded := EvaluateDataQuality("BADPRICE", candles, start, end)
	if !excluded {
		t.Fatal("zero close price was not excluded")
	}
	if !strings.Contains(reason, "Zero or negative price on 2024-01-04") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestEvaluateDataQualityBlocksMissingBars(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 56)
	candles := weekdayCandles(t, start, end, 100.0, 0.0)
	sparse := make([]data.YahooCandle, 0, len(candles))
	for i, candle := range candles {
		if i%7 != 0 {
			sparse = append(sparse, candle)
		}
	}
	reason, excluded := EvaluateDataQuality("SPARSE", sparse, start, end)
	if !excluded || !strings.Contains(reason, "Missing bars") {
		t.Fatalf("sparse data result = %q, %v", reason, excluded)
	}
}

func TestEvaluateDataQualityMissingBarBoundaryIsStrictlyAboveThreshold(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC) // 25 weekdays
	candles := weekdayCandles(t, start, end, 100.0, 0.0)
	if len(candles) != 25 {
		t.Fatalf("fixture weekdays = %d, want25", len(candles))
	}
	// Exactly8% missing (2 of25) stays under the strict greater-than cutoff.
	exactlyThreshold := append([]data.YahooCandle(nil), candles[2:]...)
	if reason, excluded := EvaluateDataQuality("EDGE", exactlyThreshold, start, end); excluded {
		t.Fatalf("8.0%% missing was excluded: %q", reason)
	}
	// One bar more missing flips the gate, with Java's exact message.
	oneMore := append([]data.YahooCandle(nil), candles[3:]...)
	reason, excluded := EvaluateDataQuality("WORSE", oneMore, start, end)
	if !excluded {
		t.Fatal("16% missing data passed")
	}
	want := "Missing bars 12.0% exceeds 8.0% threshold (expected ~25, found 22)"
	if reason != want {
		t.Fatalf("reason = %q, want %q", reason, want)
	}
}

func TestEvaluateDataQualityFlagsSuspectedSplit(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 14)
	candles := weekdayCandles(t, start, end, 100.0, 0.0)
	halved := mustDecimalValue(t, "50")
	candles[5].Open = halved
	candles[5].High = halved
	candles[5].Low = halved
	candles[5].Close = halved
	candles[5].AdjustedClose = halved
	reason, excluded := EvaluateDataQuality("SPLIT", candles, start, end)
	if !excluded || !strings.Contains(reason, "Suspected unadjusted split") {
		t.Fatalf("split result = %q, %v", reason, excluded)
	}
	want := "Suspected unadjusted split/bonus: 50.0% single-bar move on 2024-01-08 (heuristic, no corporate-action feed available to confirm)"
	if reason != want {
		t.Fatalf("reason = %q, want %q", reason, want)
	}
}

func TestEvaluateDataQualityRejectsEmptyHistory(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for name, candles := range map[string][]data.YahooCandle{
		"nil":   nil,
		"empty": {},
	} {
		reason, excluded := EvaluateDataQuality("EMPTY", candles, start, start.AddDate(0, 0, 7))
		if !excluded || reason != "No candles in evaluation window" {
			t.Fatalf("%s input = %q, %v", name, reason, excluded)
		}
	}
}

func TestEvaluateDataQualityWeekendOnlyWindowNeedsNoBars(t *testing.T) {
	// A weekend window expects zero weekday bars; a candle outside the window
	// still participates in the split scan, mirroring Java's order.
	start := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC) // Saturday
	end := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)   // Sunday
	candles := weekdayCandles(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 100.0, 0.0)
	if reason, excluded := EvaluateDataQuality("WEEKEND", candles, start, end); excluded {
		t.Fatalf("weekend window excluded: %q", reason)
	}
}

func TestUnadjustedSplitScanSkipsZeroPreviousClose(t *testing.T) {
	zero := data.YahooCandle{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)}
	jump := data.YahooCandle{Date: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
		Close: mustDecimalValue(t, "10")}
	if reason := checkUnadjustedSplitJumps([]data.YahooCandle{zero, jump}); reason != "" {
		t.Fatalf("zero previous close produced %q", reason)
	}
}

func TestDivideHalfUpToScaleMatchesBigDecimal(t *testing.T) {
	for name, tc := range map[string]struct {
		numerator   string
		denominator string
		scale       int
		want        string
	}{
		"simple":            {"0.5", "1", 6, "0.500000"},
		"round half up":     {"0.125", "1", 2, "0.13"},
		"round half down":   {"0.124", "1", 2, "0.12"},
		"negative half up":  {"-0.125", "1", 2, "-0.13"},
		"recurring decimal": {"1", "3", 6, "0.333333"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := divideHalfUpToScale(tc.numerator, tc.denominator, tc.scale); got != tc.want {
				t.Fatalf("divideHalfUpToScale(%s, %s, %d) = %s, want %s",
					tc.numerator, tc.denominator, tc.scale, got, tc.want)
			}
		})
	}
	if got := divideHalfUpToScale("1", "0", 6); got != "0.000000" {
		t.Fatalf("zero denominator = %s", got)
	}
}
