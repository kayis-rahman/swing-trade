package strategy

import (
	"fmt"
	"math"
)

// CandleWithPrices is the shared OHLCV input for the Java BigDecimal and
// Double variants: both feed the same math after doubleValue().
type CandleWithPrices struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// StochasticValues mirrors Java's StochasticValues record (K and D lines).
type StochasticValues struct {
	K float64
	D float64
}

// BollingerBands mirrors Java's BollingerBands record.
type BollingerBands struct {
	Upper  float64
	Middle float64
	Lower  float64
}

// CalculateMACD ports TechnicalIndicators.calculateMACD: the last value of
// TA4J's MACDIndicator, EMA(fast) - EMA(slow). Java's signalPeriod parameter
// is accepted and ignored, mirroring the wrapper.
func CalculateMACD(closePrices []float64, fastPeriod, slowPeriod, signalPeriod int) (*float64, error) {
	if len(closePrices) == 0 {
		return nil, nil
	}
	if fastPeriod <= 0 || slowPeriod <= 0 {
		return nil, fmt.Errorf("MACD periods must be positive")
	}
	if fastPeriod >= slowPeriod {
		return nil, fmt.Errorf("Fast period must be less than slow period") //nolint:staticcheck // exact Java IllegalArgumentException message for parity
	}
	if len(closePrices) < slowPeriod {
		return nil, nil
	}
	closes := ta4jSeries(closePrices)
	return asFloat(last(ta4jEMA(closes, fastPeriod)).sub(last(ta4jEMA(closes, slowPeriod)))), nil
}

// CalculateATR ports TechnicalIndicators.calculateATR: TA4J's ATRIndicator,
// the MMA of TRIndicator.
func CalculateATR(candles []CandleWithPrices, period int) (*float64, error) {
	if len(candles) == 0 {
		return nil, nil
	}
	if period <= 0 {
		return nil, fmt.Errorf("ATR period must be positive")
	}
	if len(candles) < period {
		return nil, nil
	}
	return asFloat(ta4jATR(candles, period)), nil
}

// CalculateStochastic ports TechnicalIndicators.calculateStochastic: TA4J's
// StochasticOscillatorKIndicator over kPeriod and the D indicator, an SMA of
// the K values over three bars. A flat window divides by zero, which TA4J
// turns into NaN.
func CalculateStochastic(candles []CandleWithPrices, kPeriod, dPeriod int) (*StochasticValues, error) {
	if len(candles) == 0 {
		return nil, nil
	}
	if kPeriod <= 0 || dPeriod <= 0 {
		return nil, fmt.Errorf("Stochastic periods must be positive") //nolint:staticcheck // exact Java IllegalArgumentException message for parity
	}
	if len(candles) < kPeriod {
		return nil, nil
	}
	highs, lows, closes := candleSeries(candles)
	highest := computeHighest(highs, kPeriod)
	lowest := computeLowest(lows, kPeriod)
	kValues := make([]decimalNum, len(candles))
	kIsNaN := make([]bool, len(candles))
	for i := range candles {
		quotient, err := closes[i].sub(lowest[i]).divide(highest[i].sub(lowest[i]))
		if err != nil {
			kIsNaN[i] = true
			continue
		}
		kValues[i] = quotient.multiply(decimalNumFromInt(100))
	}
	lastIndex := len(candles) - 1
	k := math.NaN()
	if !kIsNaN[lastIndex] {
		k = *asFloat(kValues[lastIndex])
	}
	start := max(0, lastIndex-2)
	for i := start; i <= lastIndex; i++ {
		if kIsNaN[i] {
			return &StochasticValues{K: k, D: math.NaN()}, nil
		}
	}
	return &StochasticValues{K: k, D: *asFloat(computeSMA(kValues[start:], 3)[lastIndex-start])}, nil
}

// CalculateBollingerBands ports TechnicalIndicators.calculateBollingerBands:
// the TA4J SMA middle band and Java's double-precision population standard
// deviation, scaled by the multiplier. Explicit float64 conversions keep Go
// from fusing multiply-add, which Java never does.
func CalculateBollingerBands(closePrices []float64, period int, multiplier float64) (*BollingerBands, error) {
	if len(closePrices) == 0 {
		return nil, nil
	}
	if period <= 0 {
		return nil, fmt.Errorf("Bollinger Bands period must be positive") //nolint:staticcheck // exact Java IllegalArgumentException message for parity
	}
	if len(closePrices) < period {
		return nil, nil
	}
	mean := *lastAsFloat(computeSMA(ta4jSeries(closePrices), period))
	var squared float64
	for _, price := range closePrices[len(closePrices)-period:] {
		diff := price - mean
		squared += float64(diff * diff)
	}
	stdDev := math.Sqrt(squared / float64(period))
	return &BollingerBands{
		Upper:  mean + float64(stdDev*multiplier),
		Middle: mean,
		Lower:  mean - float64(stdDev*multiplier),
	}, nil
}

// CalculateVWAP ports TechnicalIndicators.calculateVWAP: the volume-weighted
// average of typical prices in plain double arithmetic, null when no volume
// traded.
func CalculateVWAP(candles []CandleWithPrices) (*float64, error) {
	if len(candles) == 0 {
		return nil, nil
	}
	var totalValue, totalVolume float64
	for _, candle := range candles {
		typicalPrice := (candle.High + candle.Low + candle.Close) / 3.0
		totalValue += float64(typicalPrice * candle.Volume)
		totalVolume += candle.Volume
	}
	if totalVolume == 0 {
		return nil, nil
	}
	value := totalValue / totalVolume
	return &value, nil
}

// CalculateADX ports TechnicalIndicators.calculateADX: double-precision
// directional movement averaged over the last period values, divided by the
// TA4J ATR, then DX. Java's calculateSmoothedADX is an identity, so no second
// smoothing happens.
func CalculateADX(candles []CandleWithPrices, period int) (*float64, error) {
	if len(candles) == 0 {
		return nil, nil
	}
	if period <= 0 {
		return nil, fmt.Errorf("ADX period must be positive")
	}
	if len(candles) < period+1 {
		return nil, nil
	}
	plusDM := make([]float64, 0, len(candles)-1)
	minusDM := make([]float64, 0, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		upMove := candles[i].High - candles[i-1].High
		downMove := candles[i-1].Low - candles[i].Low
		switch {
		case upMove > downMove && upMove > 0:
			plusDM = append(plusDM, upMove)
			minusDM = append(minusDM, 0)
		case downMove > upMove && downMove > 0:
			plusDM = append(plusDM, 0)
			minusDM = append(minusDM, downMove)
		default:
			plusDM = append(plusDM, 0)
			minusDM = append(minusDM, 0)
		}
	}
	atrValue := *asFloat(ta4jATR(candles, period))
	if atrValue == 0 {
		return nil, nil
	}
	smoothedPlus := lastPeriodSum(plusDM, period) / float64(period)
	smoothedMinus := lastPeriodSum(minusDM, period) / float64(period)
	plusDI := math.Abs(smoothedPlus / atrValue * 100)
	minusDI := math.Abs(smoothedMinus / atrValue * 100)
	sumDI := plusDI + minusDI
	if sumDI == 0 {
		return nil, nil
	}
	dx := math.Abs(plusDI-minusDI) / sumDI * 100
	return &dx, nil
}

// ta4jATR is the last value of TA4J's ATRIndicator: MMA over TRIndicator,
// where bar 0 is |high-low| and later bars take the largest absolute gap
// against the previous close.
func ta4jATR(candles []CandleWithPrices, period int) decimalNum {
	highs, lows, closes := candleSeries(candles)
	trueRanges := make([]decimalNum, len(candles))
	for i := range candles {
		trueRange := absDecimalNum(highs[i].sub(lows[i]))
		if i > 0 {
			trueRange = maxDecimal(trueRange, absDecimalNum(highs[i].sub(closes[i-1])))
			trueRange = maxDecimal(trueRange, absDecimalNum(closes[i-1].sub(lows[i])))
		}
		trueRanges[i] = trueRange
	}
	return last(ta4jMMA(trueRanges, period))
}

func candleSeries(candles []CandleWithPrices) (highs, lows, closes []decimalNum) {
	highs = make([]decimalNum, len(candles))
	lows = make([]decimalNum, len(candles))
	closes = make([]decimalNum, len(candles))
	for i, candle := range candles {
		highs[i], lows[i], closes[i] = ta4jNum(candle.High), ta4jNum(candle.Low), ta4jNum(candle.Close)
	}
	return highs, lows, closes
}

func absDecimalNum(value decimalNum) decimalNum {
	if value.coefficient.Sign() < 0 {
		return decimalNumFromInt(0).sub(value)
	}
	return value
}

// lastPeriodSum sums the final period entries the way Java's
// calculateSmoothedValue does, including a partial window.
func lastPeriodSum(values []float64, period int) float64 {
	var sum float64
	for _, value := range values[max(0, len(values)-period):] {
		sum += value
	}
	return sum
}
