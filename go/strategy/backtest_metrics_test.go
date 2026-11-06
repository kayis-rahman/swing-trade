package strategy

import (
	"math"
	"testing"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

func TestBacktestMetricsCagrAnnualizesReturnUsingCalendarDates(t *testing.T) {
	cagr := CagrPct(mustDecimalValue(t, "100"), mustDecimalValue(t, "121"),
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
	if cagr < 9.98 || cagr > 10.02 {
		t.Fatalf("cagr = %v, want 10", cagr)
	}
}

func TestBacktestMetricsSortinoUsesNegativeReturnsForDownsideDeviation(t *testing.T) {
	curve := []coredecimal.Decimal{
		mustDecimalValue(t, "100"), mustDecimalValue(t, "110"), mustDecimalValue(t, "105"), mustDecimalValue(t, "115"),
	}
	expected := ((0.10 - (5.0 / 110.0) + (10.0 / 105.0)) / 3.0) /
		(5.0 / 110.0 / 3.0) * 0 // not exact expected computed below differently
	_ = expected
	got := SortinoRatio(curve)
	// Recompute expected per Java formula:
	exp := ((0.10 - (5.0 / 110.0) + (10.0 / 105.0)) / 3.0) /
		math.Sqrt((5.0/110.0)*(5.0/110.0)/3.0) * math.Sqrt(252.0)
	_ = exp
	if got < exp-1e-9 || got > exp+1e-9 {
		t.Fatalf("sortino = %v, want %v", got, exp)
	}
}

func TestBacktestMetricsSortinoWithNoDownsideReturnsZero(t *testing.T) {
	curve := []coredecimal.Decimal{mustDecimalValue(t, "100"), mustDecimalValue(t, "101"), mustDecimalValue(t, "102")}
	if SortinoRatio(curve) != 0.0 {
		t.Fatalf("sortino = %v, want 0", SortinoRatio(curve))
	}
}

func TestBacktestMetricsCalmarDividesCagrByDrawdown(t *testing.T) {
	if CalmarRatio(10.0, mustDecimalValue(t, "20")) != 0.5 {
		t.Fatalf("calmar 1")
	}
	if CalmarRatio(10.0, mustDecimalValue(t, "0")) != 0.0 {
		t.Fatalf("calmar 2")
	}
}

func TestBacktestMetricsBuyAndHoldUsesPriceReturnAndReportsExcessReturn(t *testing.T) {
	cmp := BuyAndHoldComparison(mustDecimalValue(t, "18"), mustDecimalValue(t, "100"), mustDecimalValue(t, "125"))
	if cmp.BenchmarkReturnPct < 24.999999 || cmp.BenchmarkReturnPct > 25.000001 {
		t.Fatalf("benchmark return = %v", cmp.BenchmarkReturnPct)
	}
	if cmp.ExcessReturnPct < -7.000001 || cmp.ExcessReturnPct > -6.999999 {
		t.Fatalf("excess = %v", cmp.ExcessReturnPct)
	}
}

func TestBacktestMetricsBuyAndHoldIsUnavailableForInvalidPrices(t *testing.T) {
	cmp := BuyAndHoldComparison(mustDecimalValue(t, "18"), mustDecimalValue(t, "0"), mustDecimalValue(t, "125"))
	if cmp.BenchmarkName != "BUY_AND_HOLD" {
		t.Fatalf("name = %s", cmp.BenchmarkName)
	}
	if cmp.BenchmarkReturnPct != 0.0 || cmp.ExcessReturnPct != 0.0 {
		t.Fatalf("returns = %v %v", cmp.BenchmarkReturnPct, cmp.ExcessReturnPct)
	}
}
