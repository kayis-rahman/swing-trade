package strategy

import (
	"math"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	domain "github.com/kayis-rahman/swing-trade/go/core/domain"
)

const (
	tradingDaysPerYear  = 252.0
	calendarDaysPerYear = 365.25
)

func decimalToFloat64(d coredecimal.Decimal) float64 {
	return float64(d.Units().Int64()) / 10000.0
}

// CagrPct computes the compound annual growth rate as a percentage.
func CagrPct(initialCapital, finalCapital coredecimal.Decimal, start, end time.Time) float64 {
	if initialCapital.Cmp(coredecimal.Decimal{}) <= 0 || finalCapital.Cmp(coredecimal.Decimal{}) < 0 ||
		start.IsZero() || end.IsZero() || !start.Before(end) {
		return 0.0
	}
	years := end.Sub(start).Hours() / 24.0 / calendarDaysPerYear
	if years <= 0 {
		return 0.0
	}
	if finalCapital.Cmp(coredecimal.Decimal{}) == 0 {
		return -100.0
	}
	growth := decimalToFloat64(finalCapital) / decimalToFloat64(initialCapital)
	return (math.Pow(growth, 1.0/years) - 1.0) * 100.0
}

// TotalReturnPct returns the total return as a percentage of initial capital.
func TotalReturnPct(initialCapital, finalCapital coredecimal.Decimal) coredecimal.Decimal {
	return coredecimal.PercentOf(finalCapital.Sub(initialCapital), initialCapital)
}

// MaxDrawdownPct returns the largest peak-to-trough drop of the equity curve, in percent.
func MaxDrawdownPct(capitalCurve []coredecimal.Decimal) coredecimal.Decimal {
	if len(capitalCurve) == 0 {
		return coredecimal.Decimal{}
	}
	peak := capitalCurve[0]
	maxDrawdown := coredecimal.Decimal{}
	for _, value := range capitalCurve {
		if value.Cmp(peak) > 0 {
			peak = value
		}
		if peak.Cmp(coredecimal.Decimal{}) > 0 {
			drawdown := coredecimal.PercentOf(peak.Sub(value), peak)
			if drawdown.Cmp(maxDrawdown) > 0 {
				maxDrawdown = drawdown
			}
		}
	}
	return maxDrawdown
}

// SortinoRatio computes the annualized Sortino ratio.
func SortinoRatio(capitalCurve []coredecimal.Decimal) float64 {
	if len(capitalCurve) < 3 {
		return 0.0
	}
	sumReturns := 0.0
	sumDownsideSquares := 0.0
	observations := 0
	for i := 1; i < len(capitalCurve); i++ {
		previous := capitalCurve[i-1]
		if previous.Cmp(coredecimal.Decimal{}) == 0 {
			continue
		}
		dailyReturn := (decimalToFloat64(capitalCurve[i]) - decimalToFloat64(previous)) / decimalToFloat64(previous)
		sumReturns += dailyReturn
		downside := math.Min(dailyReturn, 0.0)
		sumDownsideSquares += math.Pow(downside, 2)
		observations++
	}
	if observations == 0 {
		return 0.0
	}
	downsideDeviation := math.Sqrt(sumDownsideSquares / float64(observations))
	if downsideDeviation == 0.0 {
		return 0.0
	}
	return (sumReturns / float64(observations)) / downsideDeviation * math.Sqrt(tradingDaysPerYear)
}

// SharpeRatio computes the annualized Sharpe ratio.
func SharpeRatio(capitalCurve []coredecimal.Decimal) float64 {
	if len(capitalCurve) < 3 {
		return 0.0
	}
	returns := make([]float64, len(capitalCurve)-1)
	for i := 1; i < len(capitalCurve); i++ {
		previous := capitalCurve[i-1]
		if previous.Cmp(coredecimal.Decimal{}) == 0 {
			returns[i-1] = 0.0
		} else {
			returns[i-1] = (decimalToFloat64(capitalCurve[i]) - decimalToFloat64(previous)) / decimalToFloat64(previous)
		}
	}
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))
	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	variance /= float64(len(returns))
	deviation := math.Sqrt(variance)
	if deviation == 0.0 {
		return 0.0
	}
	return mean / deviation * math.Sqrt(tradingDaysPerYear)
}

// CalmarRatio divides CAGR by drawdown.
func CalmarRatio(cagrPct float64, maxDrawdownPct coredecimal.Decimal) float64 {
	if maxDrawdownPct.Cmp(coredecimal.Decimal{}) == 0 {
		return 0.0
	}
	return cagrPct / decimalToFloat64(maxDrawdownPct)
}

// BuyAndHoldComparison compares strategy return to a buy-and-hold benchmark.
func BuyAndHoldComparison(strategyReturnPct coredecimal.Decimal, startingClose, endingClose coredecimal.Decimal) domain.BenchmarkComparison {
	if startingClose.Cmp(coredecimal.Decimal{}) <= 0 || endingClose.Cmp(coredecimal.Decimal{}) < 0 {
		return domain.UnavailableBenchmark(decimalToFloat64(strategyReturnPct))
	}
	benchmarkReturnPct := (decimalToFloat64(endingClose) - decimalToFloat64(startingClose)) / decimalToFloat64(startingClose) * 100.0
	return domain.BuyAndHoldBenchmark(decimalToFloat64(strategyReturnPct), benchmarkReturnPct)
}
