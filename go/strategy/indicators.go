package strategy

import "fmt"

// The indicator ports mirror Java's TechnicalIndicators, which wraps
// ta4j-core 0.16 and returns Double, so indicator values are float paths:
// Go returns (*float64, error) where a nil value is Java's null and the
// error carries Java's IllegalArgumentException with its exact message.
// The BigDecimal and Double Java variants feed the same TA4J math through
// doubleValue(), so one float64 implementation covers both.

// ta4jEMAMultiplier is TA4J's EMA factor: 2 / (period + 1).
const ta4jEMAMultiplierDivisor = 2.0

// CalculateSMA ports TechnicalIndicators.calculateSMA: the mean of the last
// period closes (TA4J SMAIndicator).
func CalculateSMA(closePrices []float64, period int) (*float64, error) {
	if len(closePrices) == 0 {
		return nil, nil
	}
	if period <= 0 {
		return nil, fmt.Errorf("SMA period must be positive")
	}
	if len(closePrices) < period {
		return nil, nil
	}
	var sum float64
	for _, price := range closePrices[len(closePrices)-period:] {
		sum += price
	}
	value := sum / float64(period)
	return &value, nil
}

// CalculateEMA ports TechnicalIndicators.calculateEMA with TA4J's seeding:
// the first close seeds the series and each step is
// previous + (price - previous) * 2/(period+1).
func CalculateEMA(closePrices []float64, period int) (*float64, error) {
	if len(closePrices) == 0 {
		return nil, nil
	}
	if period <= 0 {
		return nil, fmt.Errorf("EMA period must be positive")
	}
	if len(closePrices) < period {
		return nil, nil
	}
	multiplier := ta4jEMAMultiplierDivisor / float64(period+1)
	value := closePrices[0]
	for _, price := range closePrices[1:] {
		value += (price - value) * multiplier
	}
	return &value, nil
}

// CalculateRSI ports TechnicalIndicators.calculateRSI: Wilder averages of
// gains and losses through TA4J's MMA (multiplier 1/period seeded at zero)
// and the TA4J boundary cases when the average loss (or both averages) is
// zero. The result expression keeps Java's evaluation order
// 100 - 100/(1+rs) so the float path matches bit for bit.
func CalculateRSI(closePrices []float64, period int) (*float64, error) {
	if len(closePrices) == 0 {
		return nil, nil
	}
	if period <= 0 {
		return nil, fmt.Errorf("RSI period must be positive")
	}
	if len(closePrices) < period {
		return nil, nil
	}
	avgGain, avgLoss := 0.0, 0.0
	for i := 1; i < len(closePrices); i++ {
		change := closePrices[i] - closePrices[i-1]
		gain, loss := 0.0, 0.0
		if change > 0 {
			gain = change
		} else if change < 0 {
			loss = -change
		}
		avgGain += (gain - avgGain) / float64(period)
		avgLoss += (loss - avgLoss) / float64(period)
	}
	var value float64
	switch avgLoss {
	case 0:
		if avgGain == 0 {
			value = 0
		} else {
			value = 100
		}
	default:
		rs := avgGain / avgLoss
		value = 100 - 100/(1+rs)
	}
	return &value, nil
}

// CalculateVolumeMA ports TechnicalIndicators.calculateVolumeMA: an SMA over
// the volume series with its own guard message.
func CalculateVolumeMA(volumes []float64, period int) (*float64, error) {
	if len(volumes) == 0 {
		return nil, nil
	}
	if period <= 0 {
		return nil, fmt.Errorf("VolumeMA period must be positive")
	}
	if len(volumes) < period {
		return nil, nil
	}
	var sum float64
	for _, volume := range volumes[len(volumes)-period:] {
		sum += volume
	}
	value := sum / float64(period)
	return &value, nil
}
