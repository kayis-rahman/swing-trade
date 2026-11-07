package strategy

import (
	"math"
	"strconv"
	"time"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// SignalOverlapAnalyzer mirrors Java diagnostics.
type SignalOverlapAnalyzer struct{}

type entryKey struct {
	symbol    string
	entryDate time.Time
}

// EntryJaccardSimilarity computes Jaccard overlap of entry keys.
func EntryJaccardSimilarity(a, b []PortfolioTrade) float64 {
	setA := entryKeys(a)
	setB := entryKeys(b)
	if len(setA) == 0 && len(setB) == 0 {
		return 1.0
	}
	inter := 0
	for k := range setA {
		if _, ok := setB[k]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0.0
	}
	return float64(inter) / float64(union)
}

func entryKeys(trades []PortfolioTrade) map[entryKey]struct{} {
	keys := map[entryKey]struct{}{}
	for _, t := range trades {
		keys[entryKey{symbol: t.Symbol, entryDate: t.EntryDate}] = struct{}{}
	}
	return keys
}

// DailyReturnCorrelation mirrors Java Pearson correlation of aligned daily returns.
func DailyReturnCorrelation(a, b []DailyEquityPoint) float64 {
	returnsA := returnsByDate(a)
	returnsB := returnsByDate(b)
	xs := make([]float64, 0, len(returnsA))
	ys := make([]float64, 0, len(returnsA))
	for date, xa := range returnsA {
		yb, ok := returnsB[date]
		if !ok {
			continue
		}
		xs = append(xs, xa)
		ys = append(ys, yb)
	}
	if len(xs) < 2 {
		return 0.0
	}
	return pearsonFloat(xs, ys)
}

func returnsByDate(curve []DailyEquityPoint) map[time.Time]float64 {
	result := map[time.Time]float64{}
	for i := 1; i < len(curve); i++ {
		prev := curve[i-1].Equity
		cur := curve[i].Equity
		var ret float64
		if prev.Cmp(coredecimal.Decimal{}) == 0 {
			ret = 0.0
		} else {
			// use decimal as string to float
			pv, _ := strconv.ParseFloat(prev.String(), 64)
			cv, _ := strconv.ParseFloat(cur.String(), 64)
			ret = (cv - pv) / pv
		}
		result[curve[i].Date] = ret
	}
	return result
}

func pearsonFloat(x, y []float64) float64 {
	n := len(x)
	if n == 0 {
		return 0.0
	}
	meanX := meanFloat(x)
	meanY := meanFloat(y)
	cov, varX, varY := 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		cov += dx * dy
		varX += dx * dx
		varY += dy * dy
	}
	if varX == 0.0 || varY == 0.0 {
		return 0.0
	}
	return cov / math.Sqrt(varX*varY)
}

func meanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
