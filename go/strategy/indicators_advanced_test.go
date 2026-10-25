package strategy

import "testing"

func risingCandles(count int) []CandleWithPrices {
	candles := make([]CandleWithPrices, count)
	for i := range candles {
		candles[i] = CandleWithPrices{
			Open: 100 + float64(i), High: 105 + float64(i),
			Low: 95 + float64(i), Close: 102 + float64(i), Volume: 1_000_000,
		}
	}
	return candles
}

func TestCalculateMACDMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateMACD(seqPrices(20, 1), 12, 26, 9)
		if err != nil || value != nil {
			t.Fatalf("MACD = %v, %v, want nil", value, err)
		}
	})
	t.Run("valid data returns a value", func(t *testing.T) {
		value, err := CalculateMACD(seqPrices(50, 0.5), 12, 26, 9)
		if err != nil || value == nil {
			t.Fatalf("MACD = %v, %v, want a value", value, err)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateMACD([]float64{}, 12, 26, 9)
		if err != nil || value != nil {
			t.Fatalf("MACD = %v, %v, want nil", value, err)
		}
	})
	t.Run("constant prices return a value like Java", func(t *testing.T) {
		constant := make([]float64, 50)
		for i := range constant {
			constant[i] = 100
		}
		value, err := CalculateMACD(constant, 12, 26, 9)
		if err != nil || value == nil {
			t.Fatalf("MACD = %v, %v, want a value", value, err)
		}
	})
	t.Run("signal period is accepted and ignored like Java", func(t *testing.T) {
		prices := seqPrices(50, 0.1)
		first, err := CalculateMACD(prices, 12, 26, 9)
		if err != nil || first == nil {
			t.Fatalf("MACD = %v, %v", first, err)
		}
		second, err := CalculateMACD(prices, 12, 26, 30)
		if err != nil || second == nil || *second != *first {
			t.Fatalf("signal period changed the MACD line: %v vs %v (%v)", first, second, err)
		}
	})
	t.Run("zero period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateMACD(seqPrices(50, 1), 0, 26, 9); err == nil || err.Error() != "MACD periods must be positive" {
			t.Fatalf("error = %v, want MACD periods must be positive", err)
		}
	})
	t.Run("fast period must be below slow period", func(t *testing.T) {
		if _, err := CalculateMACD(seqPrices(50, 1), 26, 26, 9); err == nil || err.Error() != "Fast period must be less than slow period" {
			t.Fatalf("error = %v, want Fast period must be less than slow period", err)
		}
	})
}

func TestCalculateATRMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateATR(risingCandles(10), 14)
		if err != nil || value != nil {
			t.Fatalf("ATR = %v, %v, want nil", value, err)
		}
	})
	t.Run("valid data returns a value", func(t *testing.T) {
		value, err := CalculateATR(risingCandles(20), 14)
		if err != nil || value == nil {
			t.Fatalf("ATR = %v, %v, want a value", value, err)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateATR([]CandleWithPrices{}, 14)
		if err != nil || value != nil {
			t.Fatalf("ATR = %v, %v, want nil", value, err)
		}
	})
	t.Run("small ranges stay below two", func(t *testing.T) {
		candles := make([]CandleWithPrices, 20)
		for i := range candles {
			candles[i] = CandleWithPrices{Open: 100, High: 100.5, Low: 99.5, Close: 100, Volume: 1_000_000}
		}
		value, err := CalculateATR(candles, 14)
		if err != nil || value == nil || *value >= 2.0 {
			t.Fatalf("ATR = %v, %v, want a small value below 2", value, err)
		}
	})
	t.Run("large ranges stay above five", func(t *testing.T) {
		candles := make([]CandleWithPrices, 20)
		for i := range candles {
			candles[i] = CandleWithPrices{Open: 100, High: 110, Low: 90, Close: 105, Volume: 1_000_000}
		}
		value, err := CalculateATR(candles, 14)
		if err != nil || value == nil || *value <= 5.0 {
			t.Fatalf("ATR = %v, %v, want a large value above 5", value, err)
		}
	})
	t.Run("zero period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateATR(risingCandles(20), 0); err == nil || err.Error() != "ATR period must be positive" {
			t.Fatalf("error = %v, want ATR period must be positive", err)
		}
	})
	t.Run("BigDecimal and Double variants share this implementation", func(t *testing.T) {
		value, err := CalculateATR(risingCandles(20), 14)
		if err != nil || value == nil {
			t.Fatalf("shared variant result = %v, %v", value, err)
		}
	})
}

func TestCalculateStochasticMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateStochastic(risingCandles(10), 14, 3)
		if err != nil || value != nil {
			t.Fatalf("Stochastic = %v, %v, want nil", value, err)
		}
	})
	t.Run("valid data returns both lines", func(t *testing.T) {
		value, err := CalculateStochastic(risingCandles(30), 14, 3)
		if err != nil || value == nil {
			t.Fatalf("Stochastic = %v, %v", value, err)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateStochastic([]CandleWithPrices{}, 14, 3)
		if err != nil || value != nil {
			t.Fatalf("Stochastic = %v, %v, want nil", value, err)
		}
	})
	t.Run("zero periods are rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateStochastic(risingCandles(30), 0, 3); err == nil || err.Error() != "Stochastic periods must be positive" {
			t.Fatalf("error = %v, want Stochastic periods must be positive", err)
		}
	})
}

func TestCalculateBollingerBandsMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateBollingerBands(seqPrices(15, 1), 20, 2.0)
		if err != nil || value != nil {
			t.Fatalf("Bands = %v, %v, want nil", value, err)
		}
	})
	t.Run("valid data returns all bands", func(t *testing.T) {
		value, err := CalculateBollingerBands(seqPrices(30, 0.5), 20, 2.0)
		if err != nil || value == nil {
			t.Fatalf("Bands = %v, %v", value, err)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateBollingerBands([]float64{}, 20, 2.0)
		if err != nil || value != nil {
			t.Fatalf("Bands = %v, %v, want nil", value, err)
		}
	})
	t.Run("zero period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateBollingerBands(seqPrices(30, 1), 0, 2.0); err == nil || err.Error() != "Bollinger Bands period must be positive" {
			t.Fatalf("error = %v, want Bollinger Bands period must be positive", err)
		}
	})
	t.Run("upper band above middle above lower", func(t *testing.T) {
		value, err := CalculateBollingerBands(seqPrices(30, 0.5), 20, 2.0)
		if err != nil || value == nil {
			t.Fatalf("Bands = %v, %v", value, err)
		}
		if !(value.Upper > value.Middle && value.Middle > value.Lower) {
			t.Fatalf("band order = %+v", value)
		}
	})
	t.Run("constant prices collapse the bands", func(t *testing.T) {
		constant := make([]float64, 30)
		for i := range constant {
			constant[i] = 100
		}
		value, err := CalculateBollingerBands(constant, 20, 2.0)
		if err != nil || value == nil || value.Upper != 100 || value.Middle != 100 || value.Lower != 100 {
			t.Fatalf("constant bands = %+v, %v", value, err)
		}
	})
}

func TestCalculateVWAPMatchesJavaBehaviors(t *testing.T) {
	t.Run("valid data returns a value", func(t *testing.T) {
		value, err := CalculateVWAP(risingCandles(10))
		if err != nil || value == nil {
			t.Fatalf("VWAP = %v, %v, want a value", value, err)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateVWAP([]CandleWithPrices{})
		if err != nil || value != nil {
			t.Fatalf("VWAP = %v, %v, want nil", value, err)
		}
	})
	t.Run("nil list returns nil", func(t *testing.T) {
		value, err := CalculateVWAP(nil)
		if err != nil || value != nil {
			t.Fatalf("VWAP = %v, %v, want nil", value, err)
		}
	})
	t.Run("zero volume returns nil like Java", func(t *testing.T) {
		candles := risingCandles(5)
		for i := range candles {
			candles[i].Volume = 0
		}
		value, err := CalculateVWAP(candles)
		if err != nil || value != nil {
			t.Fatalf("VWAP = %v, %v, want nil for zero volume", value, err)
		}
	})
	t.Run("equal volumes average the typical prices", func(t *testing.T) {
		value, err := CalculateVWAP([]CandleWithPrices{
			{High: 12, Low: 9, Close: 11, Volume: 500},
			{High: 15, Low: 12, Close: 14, Volume: 500},
		})
		if err != nil || value == nil {
			t.Fatalf("VWAP = %v, %v", value, err)
		}
		// typical prices: (12+9+11)/3 = 10.666... and (15+12+14)/3 = 13.666...
		want := ((12.0 + 9 + 11) / 3 * 500) + ((15.0 + 12 + 14) / 3 * 500)
		want /= 1000
		diff := *value - want
		if diff > 1e-9 || diff < -1e-9 {
			t.Fatalf("VWAP = %v, want %v", *value, want)
		}
	})
}

func TestCalculateADXMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateADX(risingCandles(10), 14)
		if err != nil || value != nil {
			t.Fatalf("ADX = %v, %v, want nil", value, err)
		}
	})
	t.Run("valid data returns a value", func(t *testing.T) {
		value, err := CalculateADX(risingCandles(30), 14)
		if err != nil || value == nil {
			t.Fatalf("ADX = %v, %v, want a value", value, err)
		}
	})
	t.Run("exact period boundary returns nil like Java", func(t *testing.T) {
		value, err := CalculateADX(risingCandles(14), 14)
		if err != nil || value != nil {
			t.Fatalf("ADX = %v, %v, want nil at size == period", value, err)
		}
	})
	t.Run("zero period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateADX(risingCandles(30), 0); err == nil || err.Error() != "ADX period must be positive" {
			t.Fatalf("error = %v, want ADX period must be positive", err)
		}
	})
	t.Run("flat candles with no true range return nil", func(t *testing.T) {
		candles := make([]CandleWithPrices, 30)
		for i := range candles {
			candles[i] = CandleWithPrices{Open: 100, High: 100, Low: 100, Close: 100, Volume: 1_000_000}
		}
		value, err := CalculateADX(candles, 14)
		if err != nil || value != nil {
			t.Fatalf("ADX = %v, %v, want nil when ATR is zero", value, err)
		}
	})
}

func TestAdvancedIndicatorGuardOrderKeepsEmptyInputsNull(t *testing.T) {
	for name, run := range map[string]func() (bool, error){
		"macd":       func() (bool, error) { value, err := CalculateMACD(nil, 0, 0, 0); return value == nil, err },
		"atr":        func() (bool, error) { value, err := CalculateATR(nil, 0); return value == nil, err },
		"stochastic": func() (bool, error) { value, err := CalculateStochastic(nil, 0, 0); return value == nil, err },
		"bollinger":  func() (bool, error) { value, err := CalculateBollingerBands(nil, 0, 0); return value == nil, err },
		"adx":        func() (bool, error) { value, err := CalculateADX(nil, 0); return value == nil, err },
	} {
		t.Run(name, func(t *testing.T) {
			isNil, err := run()
			if !isNil || err != nil {
				t.Fatalf("%s nil-input result nil=%v err=%v, want Java's null", name, isNil, err)
			}
		})
	}
}
