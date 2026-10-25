package strategy

import "testing"

func seqPrices(count int, step float64) []float64 {
	prices := make([]float64, count)
	for i := range prices {
		prices[i] = 100 + float64(i)*step
	}
	return prices
}

func TestCalculateRSIMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateRSI(seqPrices(10, 1), 14)
		if err != nil || value != nil {
			t.Fatalf("RSI = %v, %v, want nil", value, err)
		}
	})
	t.Run("exact period returns a value", func(t *testing.T) {
		value, err := CalculateRSI(seqPrices(14, 1), 14)
		if err != nil || value == nil {
			t.Fatalf("RSI = %v, %v, want a value", value, err)
		}
	})
	t.Run("more data returns a value", func(t *testing.T) {
		value, err := CalculateRSI(seqPrices(20, 0.5), 14)
		if err != nil || value == nil {
			t.Fatalf("RSI = %v, %v, want a value", value, err)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateRSI([]float64{}, 14)
		if err != nil || value != nil {
			t.Fatalf("RSI = %v, %v, want nil", value, err)
		}
	})
	t.Run("nil list returns nil", func(t *testing.T) {
		value, err := CalculateRSI(nil, 14)
		if err != nil || value != nil {
			t.Fatalf("RSI = %v, %v, want nil", value, err)
		}
	})
	t.Run("constant prices return a value like Java", func(t *testing.T) {
		constant := make([]float64, 20)
		for i := range constant {
			constant[i] = 100
		}
		value, err := CalculateRSI(constant, 14)
		if err != nil || value == nil {
			t.Fatalf("RSI = %v, %v, want a value", value, err)
		}
	})
	t.Run("zero period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateRSI(seqPrices(20, 1), 0); err == nil || err.Error() != "RSI period must be positive" {
			t.Fatalf("error = %v, want RSI period must be positive", err)
		}
	})
	t.Run("empty list with a bad period stays null like Java", func(t *testing.T) {
		value, err := CalculateRSI(nil, -1)
		if err != nil || value != nil {
			t.Fatalf("RSI = %v, %v, want nil before period validation", value, err)
		}
	})
}

func TestCalculateSMAMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateSMA(seqPrices(15, 1), 20)
		if err != nil || value != nil {
			t.Fatalf("SMA = %v, %v, want nil", value, err)
		}
	})
	t.Run("average of last period", func(t *testing.T) {
		prices := make([]float64, 20)
		for i := range prices {
			prices[i] = 100 + float64(i)
		}
		value, err := CalculateSMA(prices, 5)
		if err != nil || value == nil {
			t.Fatalf("SMA = %v, %v", value, err)
		}
		if diff := *value - 117.0; diff > 0.01 || diff < -0.01 {
			t.Fatalf("SMA = %v, want 117 within 0.01", *value)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateSMA([]float64{}, 20)
		if err != nil || value != nil {
			t.Fatalf("SMA = %v, %v, want nil", value, err)
		}
	})
	t.Run("single value with longer period returns nil", func(t *testing.T) {
		value, err := CalculateSMA([]float64{100}, 20)
		if err != nil || value != nil {
			t.Fatalf("SMA = %v, %v, want nil", value, err)
		}
	})
	t.Run("period one equals the last close", func(t *testing.T) {
		value, err := CalculateSMA([]float64{100, 101, 102}, 1)
		if err != nil || value == nil || *value != 102.0 {
			t.Fatalf("SMA = %v, %v, want exactly 102", value, err)
		}
	})
	t.Run("zero period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateSMA(seqPrices(20, 1), 0); err == nil || err.Error() != "SMA period must be positive" {
			t.Fatalf("error = %v, want SMA period must be positive", err)
		}
	})
}

func TestCalculateEMAMatchesJavaBehaviors(t *testing.T) {
	t.Run("insufficient data returns nil", func(t *testing.T) {
		value, err := CalculateEMA(seqPrices(10, 1), 20)
		if err != nil || value != nil {
			t.Fatalf("EMA = %v, %v, want nil", value, err)
		}
	})
	t.Run("valid data returns a value", func(t *testing.T) {
		value, err := CalculateEMA(seqPrices(30, 0.5), 10)
		if err != nil || value == nil {
			t.Fatalf("EMA = %v, %v, want a value", value, err)
		}
	})
	t.Run("empty list returns nil", func(t *testing.T) {
		value, err := CalculateEMA([]float64{}, 10)
		if err != nil || value != nil {
			t.Fatalf("EMA = %v, %v, want nil", value, err)
		}
	})
	t.Run("constant prices stay at the constant", func(t *testing.T) {
		constant := make([]float64, 20)
		for i := range constant {
			constant[i] = 100
		}
		value, err := CalculateEMA(constant, 10)
		if err != nil || value == nil || *value != 100.0 {
			t.Fatalf("EMA = %v, %v, want exactly 100", value, err)
		}
	})
	t.Run("trending prices trail the last price", func(t *testing.T) {
		prices := seqPrices(20, 5)
		value, err := CalculateEMA(prices, 10)
		if err != nil || value == nil {
			t.Fatalf("EMA = %v, %v", value, err)
		}
		if *value >= prices[len(prices)-1] {
			t.Fatalf("EMA = %v, want below last price %v", *value, prices[len(prices)-1])
		}
	})
	t.Run("negative period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateEMA(seqPrices(20, 1), -3); err == nil || err.Error() != "EMA period must be positive" {
			t.Fatalf("error = %v, want EMA period must be positive", err)
		}
	})
}

func TestCalculateVolumeMAMatchesJavaBehaviors(t *testing.T) {
	constantVolumes := func() []float64 {
		volumes := make([]float64, 20)
		for i := range volumes {
			volumes[i] = 1_000_000
		}
		return volumes
	}
	t.Run("insufficient data returns nil", func(t *testing.T) {
		volumes := make([]float64, 15)
		for i := range volumes {
			volumes[i] = 1_000_000 + float64(i)*100_000
		}
		value, err := CalculateVolumeMA(volumes, 20)
		if err != nil || value != nil {
			t.Fatalf("VolumeMA = %v, %v, want nil", value, err)
		}
	})
	t.Run("null list returns nil", func(t *testing.T) {
		value, err := CalculateVolumeMA(nil, 20)
		if err != nil || value != nil {
			t.Fatalf("VolumeMA = %v, %v, want nil", value, err)
		}
	})
	t.Run("zero period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateVolumeMA(constantVolumes(), 0); err == nil || err.Error() != "VolumeMA period must be positive" {
			t.Fatalf("error = %v, want VolumeMA period must be positive", err)
		}
	})
	t.Run("negative period is rejected with Java's message", func(t *testing.T) {
		if _, err := CalculateVolumeMA(constantVolumes(), -5); err == nil || err.Error() != "VolumeMA period must be positive" {
			t.Fatalf("error = %v, want VolumeMA period must be positive", err)
		}
	})
	t.Run("constant volumes equal the constant exactly", func(t *testing.T) {
		value, err := CalculateVolumeMA(constantVolumes(), 5)
		if err != nil || value == nil || *value != 1_000_000.0 {
			t.Fatalf("VolumeMA = %v, %v, want exactly 1000000", value, err)
		}
	})
	t.Run("BigDecimal and Double variants share this implementation", func(t *testing.T) {
		// Java's two variants only differ in input type before doubleValue();
		// one call covers the CalculateVolumeMABigDecimalTests and
		// CalculateVolumeMADoubleTests expectations together.
		value, err := CalculateVolumeMA(constantVolumes(), 5)
		if err != nil || value == nil || *value != 1_000_000.0 {
			t.Fatalf("shared variant result = %v, %v", value, err)
		}
	})
}

func TestIndicatorNilValueAndErrorContract(t *testing.T) {
	for name, run := range map[string]func() (*float64, error){
		"sma":    func() (*float64, error) { return CalculateSMA(nil, 0) },
		"ema":    func() (*float64, error) { return CalculateEMA(nil, 0) },
		"rsi":    func() (*float64, error) { return CalculateRSI(nil, 0) },
		"volume": func() (*float64, error) { return CalculateVolumeMA(nil, 0) },
	} {
		t.Run(name, func(t *testing.T) {
			value, err := run()
			if value != nil || err != nil {
				t.Fatalf("%s nil-input result = %v, %v, want Java's null", name, value, err)
			}
		})
	}
}
