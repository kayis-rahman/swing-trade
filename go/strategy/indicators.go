package strategy

import (
	"fmt"
	"strconv"
)

// The indicator ports mirror Java's TechnicalIndicators, which wraps
// ta4j-core 0.16 and returns Double: Go returns (*float64, error) where a nil
// value is Java's null and the error carries Java's IllegalArgumentException
// with its exact message. The wrapper feeds every price into TA4J as a double,
// and TA4J's default BaseBarSeries turns each double into a DecimalNum
// (precision 32, HALF_UP), so the TA4J-backed indicators compute on decimalNum
// and convert the last value back with doubleValue(). The BigDecimal and Double
// Java variants share that path, so one implementation covers both.

// CalculateSMA ports TechnicalIndicators.calculateSMA (TA4J SMAIndicator).
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
	return lastAsFloat(computeSMA(ta4jSeries(closePrices), period)), nil
}

// CalculateEMA ports TechnicalIndicators.calculateEMA (TA4J EMAIndicator,
// seeded at the first close with multiplier numOf(2.0/(period+1))).
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
	return lastAsFloat(ta4jEMA(ta4jSeries(closePrices), period)), nil
}

// CalculateRSI ports TechnicalIndicators.calculateRSI (TA4J RSIIndicator):
// MMA averages of gains and losses and the zero-average-loss boundaries.
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
	closes := ta4jSeries(closePrices)
	gains := make([]decimalNum, len(closes))
	losses := make([]decimalNum, len(closes))
	gains[0], losses[0] = decimalNumFromInt(0), decimalNumFromInt(0)
	for i := 1; i < len(closes); i++ {
		change := closes[i].sub(closes[i-1])
		gains[i], losses[i] = decimalNumFromInt(0), decimalNumFromInt(0)
		switch change.coefficient.Sign() {
		case 1:
			gains[i] = change
		case -1:
			losses[i] = decimalNumFromInt(0).sub(change)
		}
	}
	averageGain := last(ta4jMMA(gains, period))
	averageLoss := last(ta4jMMA(losses, period))
	var value decimalNum
	switch {
	case averageLoss.coefficient.Sign() == 0 && averageGain.coefficient.Sign() == 0:
		value = decimalNumFromInt(0)
	case averageLoss.coefficient.Sign() == 0:
		value = decimalNumFromInt(100)
	default:
		relativeStrength, err := averageGain.divide(averageLoss)
		if err != nil {
			return nil, err
		}
		value = sub100(relativeStrength)
	}
	return asFloat(value), nil
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
	return lastAsFloat(computeSMA(ta4jSeries(volumes), period)), nil
}

// ta4jSeries reproduces BaseBar's numOf(double): DecimalNum.valueOf(Double)
// parses Double.toString, the shortest decimal that round-trips the double.
func ta4jSeries(values []float64) []decimalNum {
	series := make([]decimalNum, len(values))
	for i, value := range values {
		series[i] = ta4jNum(value)
	}
	return series
}

func ta4jNum(value float64) decimalNum {
	num, err := newDecimalNum(strconv.FormatFloat(value, 'f', -1, 64))
	if err != nil {
		panic(fmt.Sprintf("format float %v as decimal: %v", value, err))
	}
	return num
}

// ta4jEMA is TA4J's EMAIndicator: AbstractEMAIndicator with numOf(2.0/(n+1)).
func ta4jEMA(series []decimalNum, period int) []decimalNum {
	return ta4jAbstractEMA(series, ta4jNum(2.0/float64(period+1)))
}

// ta4jMMA is TA4J's MMAIndicator: AbstractEMAIndicator with numOf(1.0/n).
func ta4jMMA(series []decimalNum, period int) []decimalNum {
	return ta4jAbstractEMA(series, ta4jNum(1.0/float64(period)))
}

// ta4jAbstractEMA is AbstractEMAIndicator.calculate: index 0 is the input and
// later values are (value - previous) * multiplier + previous.
func ta4jAbstractEMA(series []decimalNum, multiplier decimalNum) []decimalNum {
	out := make([]decimalNum, len(series))
	for i, value := range series {
		if i == 0 {
			out[i] = value
			continue
		}
		out[i] = value.sub(out[i-1]).multiply(multiplier).add(out[i-1])
	}
	return out
}

func last(series []decimalNum) decimalNum { return series[len(series)-1] }

func lastAsFloat(series []decimalNum) *float64 { return asFloat(last(series)) }

// asFloat is Num.doubleValue(): BigDecimal.doubleValue rounds to nearest.
func asFloat(value decimalNum) *float64 {
	result, err := strconv.ParseFloat(value.String(), 64)
	if err != nil {
		panic(fmt.Sprintf("parse decimal %s: %v", value.String(), err))
	}
	return &result
}
