package strategy

import (
	"bytes"
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// javaIndicatorVectors is emitted by the real Java TechnicalIndicators class via
// docs/plans/go-migration/evidence/scripts/IndicatorVectors.java.
type javaIndicatorVectors struct {
	Candles []struct {
		Open   json.Number `json:"open"`
		High   json.Number `json:"high"`
		Low    json.Number `json:"low"`
		Close  json.Number `json:"close"`
		Volume json.Number `json:"volume"`
	} `json:"candles"`
	Cases []struct {
		Bars      int        `json:"bars"`
		Indicator string     `json:"indicator"`
		Params    []float64  `json:"params"`
		Values    []*float64 `json:"values"`
	} `json:"cases"`
}

func TestIndicatorsMatchJavaVectorsBitForBit(t *testing.T) {
	raw, err := os.ReadFile("testdata/java_indicator_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var vectors javaIndicatorVectors
	if err := decoder.Decode(&vectors); err != nil {
		t.Fatal(err)
	}
	candles := make([]CandleWithPrices, len(vectors.Candles))
	for i, c := range vectors.Candles {
		candles[i] = CandleWithPrices{Open: number(t, c.Open), High: number(t, c.High), Low: number(t, c.Low),
			Close: number(t, c.Close), Volume: number(t, c.Volume)}
	}
	if len(vectors.Cases) == 0 {
		t.Fatal("no Java vectors recorded")
	}
	for _, vector := range vectors.Cases {
		got, err := evaluateJavaVector(t, candles[:vector.Bars], vector.Indicator, vector.Params)
		if err != nil {
			t.Fatalf("%s%v over %d bars: %v", vector.Indicator, vector.Params, vector.Bars, err)
		}
		if len(got) != len(vector.Values) {
			t.Fatalf("%s%v over %d bars: got %d values, Java emitted %d", vector.Indicator, vector.Params, vector.Bars, len(got), len(vector.Values))
		}
		for i, want := range vector.Values {
			if (want == nil) != (got[i] == nil) || (want != nil && *want != *got[i]) {
				t.Errorf("%s%v over %d bars value %d: Go %v, Java %v", vector.Indicator, vector.Params, vector.Bars, i, deref(got[i]), deref(want))
			}
		}
	}
}

func evaluateJavaVector(t *testing.T, window []CandleWithPrices, indicator string, params []float64) ([]*float64, error) {
	t.Helper()
	closes := make([]float64, len(window))
	volumes := make([]float64, len(window))
	for i, c := range window {
		closes[i], volumes[i] = c.Close, c.Volume
	}
	period := func(i int) int { return int(params[i]) }
	switch indicator {
	case "SMA":
		return one(CalculateSMA(closes, period(0)))
	case "EMA":
		return one(CalculateEMA(closes, period(0)))
	case "VolumeMA":
		return one(CalculateVolumeMA(volumes, period(0)))
	case "RSI":
		return one(CalculateRSI(closes, period(0)))
	case "MACD":
		return one(CalculateMACD(closes, period(0), period(1), period(2)))
	case "ATR":
		return one(CalculateATR(window, period(0)))
	case "ADX":
		return one(CalculateADX(window, period(0)))
	case "VWAP":
		return one(CalculateVWAP(window))
	case "Stochastic":
		s, err := CalculateStochastic(window, period(0), period(1))
		if err != nil || s == nil {
			return nil, err
		}
		return []*float64{&s.K, &s.D}, nil
	case "BollingerBands":
		b, err := CalculateBollingerBands(closes, period(0), params[1])
		if err != nil || b == nil {
			return nil, err
		}
		return []*float64{&b.Upper, &b.Middle, &b.Lower}, nil
	}
	t.Fatalf("unknown indicator %q", indicator)
	return nil, nil
}

func one(value *float64, err error) ([]*float64, error) { return []*float64{value}, err }

func number(t *testing.T, n json.Number) float64 {
	t.Helper()
	value, err := strconv.ParseFloat(string(n), 64)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func deref(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
