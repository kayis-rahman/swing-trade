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
// EMA(fast) - EMA(slow) from TA4J's MACDIndicator. Java's signalPeriod
// parameter is accepted and ignored, mirroring the wrapper.
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
	fast, err := emaValue(closePrices, fastPeriod)
	if err != nil {
		return nil, err
	}
	slow, err := emaValue(closePrices, slowPeriod)
	if err != nil {
		return nil, err
	}
	value := fast - slow
	return &value, nil
}

// CalculateATR ports TechnicalIndicators.calculateATR: TA4J's true range
// (bar0 is |high-low|, later bars take the max of the three gaps against the
// previous close) smoothed with Wilder's MMA (multiplier 1/period, seeded at
// the first true range).
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
	trueRanges := make([]float64, len(candles))
	for i := range candles {
		trueRanges[i] = trueRange(candles, i)
	}
	value := wilderAverage(trueRanges, period)
	return &value, nil
}

// CalculateStochastic ports TechnicalIndicators.calculateStochastic: TA4J's
// oscillator K over the kPeriod window (D is the SMA of the last three K
// values, TA4J's StochasticOscillatorDIndicator).
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
	kValues := make([]float64, len(candles))
	for i := range candles {
		highest := highestHigh(candles, i, kPeriod)
		lowest := lowestLow(candles, i, kPeriod)
		kValues[i] = (candles[i].Close - lowest) / (highest - lowest) * 100
	}
	k := kValues[len(kValues)-1]
	window := kValues
	if len(kValues) > 3 {
		window = kValues[len(kValues)-3:]
	}
	var sum float64
	for _, value := range window {
		sum += value
	}
	// TA4J's D uses SMAIndicator semantics: divide by the full three-value
	// period even when fewer K values exist, mirroring the wrapper bytecode.
	return &StochasticValues{K: k, D: sum / 3}, nil
}

// CalculateBollingerBands ports TechnicalIndicators.calculateBollingerBands:
// SMA middle band with the population standard deviation (divided by the
// period) of the same window, scaled by the multiplier.
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
	window := closePrices[len(closePrices)-period:]
	var sum float64
	for _, price := range window {
		sum += price
	}
	mean := sum / float64(period)
	var squared float64
	for _, price := range window {
		diff := price - mean
		squared += diff * diff
	}
	stdDev := math.Sqrt(squared / float64(period))
	return &BollingerBands{
		Upper:  mean + stdDev*multiplier,
		Middle: mean,
		Lower:  mean - stdDev*multiplier,
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
		totalValue += typicalPrice * candle.Volume
		totalVolume += candle.Volume
	}
	if totalVolume == 0 {
		return nil, nil
	}
	value := totalValue / totalVolume
	return &value, nil
}

// CalculateADX ports TechnicalIndicators.calculateADX: directional movement
// over the window, Wilder-smoothed by summing the last period values and
// dividing by the period, then DX. Java's calculateSmoothedADX is currently
// an identity, so no second smoothing happens.
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
	trueRanges := make([]float64, len(candles))
	for i := range candles {
		trueRanges[i] = trueRange(candles, i)
	}
	atrValue := wilderAverage(trueRanges, period)
	if atrValue == 0 {
		return nil, nil
	}
	smoothedPlus := lastPeriodSum(plusDM, period) / float64(period)
	smoothedMinus := lastPeriodSum(minusDM, period) / float64(period)
	plusDI := abs(smoothedPlus / atrValue * 100)
	minusDI := abs(smoothedMinus / atrValue * 100)
	sumDI := plusDI + minusDI
	if sumDI == 0 {
		return nil, nil
	}
	dx := abs(plusDI-minusDI) / sumDI * 100
	return &dx, nil
}

// emaValue is the shared TA4J EMA calculation used by CalculateEMA and MACD.
func emaValue(prices []float64, period int) (float64, error) {
	if len(prices) == 0 {
		return 0, fmt.Errorf("prices are required")
	}
	if period <= 0 {
		return 0, fmt.Errorf("period must be positive")
	}
	multiplier := ta4jEMAMultiplierDivisor / float64(period+1)
	value := prices[0]
	for _, price := range prices[1:] {
		value += (price - value) * multiplier
	}
	return value, nil
}

// wilderAverage is TA4J's AbstractEMAIndicator recursion with multiplier
// 1/period, seeded at the first value (MMA semantics used by RSI and ATR).
func wilderAverage(values []float64, period int) float64 {
	if len(values) == 0 {
		return 0
	}
	multiplier := 1 / float64(period)
	value := values[0]
	for _, current := range values[1:] {
		value += (current - value) * multiplier
	}
	return value
}

// trueRange mirrors TA4J's TRIndicator for bar index i.
func trueRange(candles []CandleWithPrices, index int) float64 {
	candle := candles[index]
	highLow := candle.High - candle.Low
	if index == 0 {
		return abs(highLow)
	}
	previousClose := candles[index-1].Close
	gapHigh := abs(candle.High - previousClose)
	gapLow := abs(candle.Low - previousClose)
	return max(highLow, max(gapHigh, gapLow))
}

func highestHigh(candles []CandleWithPrices, index, period int) float64 {
	start := index - period + 1
	if start < 0 {
		start = 0
	}
	highest := candles[start].High
	for _, candle := range candles[start : index+1] {
		if candle.High > highest {
			highest = candle.High
		}
	}
	return highest
}

func lowestLow(candles []CandleWithPrices, index, period int) float64 {
	start := index - period + 1
	if start < 0 {
		start = 0
	}
	lowest := candles[start].Low
	for _, candle := range candles[start : index+1] {
		if candle.Low < lowest {
			lowest = candle.Low
		}
	}
	return lowest
}

// lastPeriodSum sums the final period entries the way Java's
// calculateSmoothedValue does, including a partial window.
func lastPeriodSum(values []float64, period int) float64 {
	start := len(values) - period
	if start < 0 {
		start = 0
	}
	var sum float64
	for _, value := range values[start:] {
		sum += value
	}
	return sum
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
