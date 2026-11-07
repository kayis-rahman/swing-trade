package domain

// BenchmarkComparison mirrors Java com.swingtrade.domain.BenchmarkComparison.
type BenchmarkComparison struct {
	BenchmarkName      string
	StrategyReturnPct  float64
	BenchmarkReturnPct float64
	ExcessReturnPct    float64
	Availability       BenchmarkAvailability
}

// BenchmarkAvailability describes whether a benchmark result is usable.
type BenchmarkAvailability string

// Benchmark availability values.
const (
	BenchmarkAvailable   BenchmarkAvailability = "AVAILABLE"
	BenchmarkUnavailable BenchmarkAvailability = "UNAVAILABLE"
)

// Benchmark name constants.
const (
	BenchmarkBuyAndHold = "BUY_AND_HOLD"
	BenchmarkNifty50    = "NIFTY50_PRICE"
)

// NewBenchmarkComparison constructs a BenchmarkComparison, validating its name.
func NewBenchmarkComparison(benchmarkName string, strategyReturnPct, benchmarkReturnPct float64, availability BenchmarkAvailability) (BenchmarkComparison, error) {
	if benchmarkName == "" {
		return BenchmarkComparison{}, &BenchmarkNameError{}
	}
	return BenchmarkComparison{
		BenchmarkName:      benchmarkName,
		StrategyReturnPct:  strategyReturnPct,
		BenchmarkReturnPct: benchmarkReturnPct,
		ExcessReturnPct:    strategyReturnPct - benchmarkReturnPct,
		Availability:       availability,
	}, nil
}

// BenchmarkNameError is returned when a benchmark name is blank.
type BenchmarkNameError struct{}

// Error implements error.
func (*BenchmarkNameError) Error() string { return "Benchmark name cannot be blank" }

// BuyAndHoldBenchmark builds an available buy-and-hold benchmark comparison.
func BuyAndHoldBenchmark(strategyReturnPct, benchmarkReturnPct float64) BenchmarkComparison {
	b, _ := NewBenchmarkComparison(BenchmarkBuyAndHold, strategyReturnPct, benchmarkReturnPct, BenchmarkAvailable)
	return b
}

// Nifty50PriceBenchmark builds an available NIFTY 50 price benchmark comparison.
func Nifty50PriceBenchmark(strategyReturnPct, benchmarkReturnPct float64) BenchmarkComparison {
	b, _ := NewBenchmarkComparison(BenchmarkNifty50, strategyReturnPct, benchmarkReturnPct, BenchmarkAvailable)
	return b
}

// UnavailableBenchmark builds an unavailable benchmark comparison.
func UnavailableBenchmark(strategyReturnPct float64) BenchmarkComparison {
	return BenchmarkComparison{
		BenchmarkName:      BenchmarkBuyAndHold,
		StrategyReturnPct:  strategyReturnPct,
		BenchmarkReturnPct: 0.0,
		ExcessReturnPct:    0.0,
		Availability:       BenchmarkUnavailable,
	}
}
