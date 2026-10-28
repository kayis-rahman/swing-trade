package strategy

import (
	"fmt"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

// computeSMA mirrors TA4J's SMAIndicator: the trailing window is clamped to
// the available bars, so early values are partial averages rather than zero.
func computeSMA(series []decimalNum, period int) []decimalNum {
	out := make([]decimalNum, len(series))
	for i := range series {
		start := maxInt(0, i-period+1)
		sum := decimalNumFromInt(0)
		for j := start; j <= i; j++ {
			sum = sum.add(series[j])
		}
		observations := minInt(period, i+1)
		q, _ := sum.divide(decimalNumFromInt(observations))
		out[i] = q
	}
	return out
}

func computeEMA(series []decimalNum, period int) []decimalNum {
	out := make([]decimalNum, len(series))
	if len(series) == 0 || period <= 0 {
		return out
	}
	mult, _ := newDecimalNum(fmtFloat(2.0 / float64(period+1)))
	value := series[0]
	for i, p := range series {
		if i == 0 {
			value = p
		} else {
			// previous + (price-previous)*mult
			diff := p.sub(value)
			value = value.add(diff.multiply(mult))
		}
		out[i] = value
	}
	return out
}

func computeRSI(series []decimalNum, period int) []decimalNum {
	out := make([]decimalNum, len(series))
	if len(series) == 0 || period <= 0 {
		return out
	}
	runGain, runLoss := decimalNumFromInt(0), decimalNumFromInt(0)
	for i := 1; i < len(series); i++ {
		change := series[i].sub(series[i-1])
		gain, loss := decimalNumFromInt(0), decimalNumFromInt(0)
		if change.coefficient.Sign() > 0 {
			gain = change
		} else if change.coefficient.Sign() < 0 {
			loss = decimalNumFromInt(0).sub(change)
		}
		runGain = addDiv(runGain, gain, period)
		runLoss = addDiv(runLoss, loss, period)
		if runLoss.coefficient.Sign() == 0 {
			if runGain.coefficient.Sign() == 0 {
				out[i] = decimalNumFromInt(0)
			} else {
				out[i] = decimalNumFromInt(100)
			}
		} else {
			rs, _ := runGain.divide(runLoss)
			out[i] = sub100(rs)
		}
	}
	return out
}

func computeATR(highs, lows, closes []decimalNum, period int) []decimalNum {
	out := make([]decimalNum, len(highs))
	trs := make([]decimalNum, len(highs))
	for i := range highs {
		if i == 0 {
			trs[i] = highs[i].sub(lows[i])
		} else {
			tr1 := highs[i].sub(lows[i])
			tr2 := highs[i].sub(closes[i-1])
			if tr2.coefficient.Sign() < 0 {
				tr2 = tr2.sub(tr2).sub(tr2) // abs
			}
			tr3 := lows[i].sub(closes[i-1])
			if tr3.coefficient.Sign() < 0 {
				tr3 = tr3.sub(tr3).sub(tr3)
			}
			trs[i] = maxDecimal(tr1, maxDecimal(tr2, tr3))
		}
	}
	// smooth like MMA with 1/period
	run := decimalNumFromInt(0)
	for i := range trs {
		if i == 0 {
			run = trs[i]
		} else {
			run = addDiv(run, trs[i], period)
		}
		out[i] = run
	}
	return out
}

func maxDecimal(a, b decimalNum) decimalNum {
	if a.coefficient.Cmp(b.coefficient) >= 0 {
		return a
	}
	return b
}

func computeHighest(series []decimalNum, period int) []decimalNum {
	out := make([]decimalNum, len(series))
	for i := range series {
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		mx := series[start]
		for j := start + 1; j <= i; j++ {
			if series[j].coefficient.Cmp(mx.coefficient) > 0 {
				mx = series[j]
			}
		}
		out[i] = mx
	}
	return out
}

func computeLowest(series []decimalNum, period int) []decimalNum {
	out := make([]decimalNum, len(series))
	for i := range series {
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		mn := series[start]
		for j := start + 1; j <= i; j++ {
			if series[j].coefficient.Cmp(mn.coefficient) < 0 {
				mn = series[j]
			}
		}
		out[i] = mn
	}
	return out
}

func computeBBUpper(series []decimalNum, period int, k decimalNum) []decimalNum {
	mid := computeSMA(series, period)
	sd := computeStdDev(series, period)
	out := make([]decimalNum, len(series))
	for i := range series {
		out[i] = mid[i].add(sd[i].multiply(k))
	}
	return out
}
func computeBBLower(series []decimalNum, period int, k decimalNum) []decimalNum {
	mid := computeSMA(series, period)
	sd := computeStdDev(series, period)
	out := make([]decimalNum, len(series))
	for i := range series {
		out[i] = mid[i].sub(sd[i].multiply(k))
	}
	return out
}

// computeStdDev mirrors TA4J's StandardDeviationIndicator over
// VarianceIndicator: a population variance over the clamped trailing window,
// square-rooted. Early bars use the observations available so far.
func computeStdDev(series []decimalNum, period int) []decimalNum {
	out := make([]decimalNum, len(series))
	for i := range series {
		start := maxInt(0, i-period+1)
		observations := minInt(period, i+1)
		mean, _ := sumRange(series, start, i).divide(decimalNumFromInt(observations))
		sq := decimalNumFromInt(0)
		for j := start; j <= i; j++ {
			d := series[j].sub(mean)
			sq = sq.add(d.multiply(d))
		}
		variance, _ := sq.divide(decimalNumFromInt(observations))
		out[i] = sqrtDecimal(variance)
	}
	return out
}

func sumRange(series []decimalNum, start, end int) decimalNum {
	sum := decimalNumFromInt(0)
	for j := start; j <= end; j++ {
		sum = sum.add(series[j])
	}
	return sum
}

func sqrtDecimal(d decimalNum) decimalNum {
	// newton
	x, _ := newDecimalNum("1")
	for k := 0; k < 20; k++ {
		// x = (x + d/x)/2
		q, _ := d.divide(x)
		next, _ := x.add(q).divide(decimalNumFromInt(2))
		x = next
	}
	return x
}

func computeKeltnerUpper(seriesClose, seriesHigh, seriesLow []decimalNum, period int, mult coredecimal.Decimal) []decimalNum {
	// TA4J 0.16: KeltnerChannelMiddleIndicator is an EMA of the typical price
	// ((high+low+close)/3) and KeltnerChannelUpperIndicator adds ATR*mult.
	typical := make([]decimalNum, len(seriesClose))
	for i := range seriesClose {
		sum := seriesHigh[i].add(seriesLow[i]).add(seriesClose[i])
		typical[i], _ = sum.divide(decimalNumFromInt(3))
	}
	mid := computeEMA(typical, period)
	atr := computeATR(seriesHigh, seriesLow, seriesClose, period)
	km, _ := newDecimalNum(mult.String())
	out := make([]decimalNum, len(seriesClose))
	for i := range seriesClose {
		out[i] = mid[i].add(atr[i].multiply(km))
	}
	return out
}

func fmtFloat(v float64) string {
	return fmtSprintfFloat(v)
}

func fmtSprintfFloat(v float64) string {
	// use standard fmt via fmt package
	importFmt := ""
	_ = importFmt
	return fmt.Sprintf("%g", v)
}

func addDiv(acc, val decimalNum, period int) decimalNum {
	diff := val.sub(acc)
	q, _ := diff.divide(decimalNumFromInt(period))
	return acc.add(q)
}

func sub100(rs decimalNum) decimalNum {
	onePlus := decimalNumFromInt(1).add(rs)
	q, _ := decimalNumFromInt(100).divide(onePlus)
	return decimalNumFromInt(100).sub(q)
}
