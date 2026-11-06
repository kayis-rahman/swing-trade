package domain

// BenchmarkComparison mirrors Java com.swingtrade.domain.BenchmarkComparison.
type BenchmarkComparison struct {
	BenchmarkName      string
	StrategyReturnPct  float64
	BenchmarkReturnPct float64
	ExcessReturnPct    float64
	Availability       BenchmarkAvailability
}

type BenchmarkAvailability string

const (
	BenchmarkAvailable   BenchmarkAvailability = "AVAILABLE"
	BenchmarkUnavailable BenchmarkAvailability = "UNAVAILABLE"
)

const (
	BenchmarkBuyAndHold = "BUY_AND_HOLD"
	BenchmarkNifty50    = "NIFTY50_PRICE"
)

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

type BenchmarkNameError struct{}

func (*BenchmarkNameError) Error() string { return "Benchmark name cannot be blank" }

func BuyAndHoldBenchmark(strategyReturnPct, benchmarkReturnPct float64) BenchmarkComparison {
	b, _ := NewBenchmarkComparison(BenchmarkBuyAndHold, strategyReturnPct, benchmarkReturnPct, BenchmarkAvailable)
	return b
}

func Nifty50PriceBenchmark(strategyReturnPct, benchmarkReturnPct float64) BenchmarkComparison {
	b, _ := NewBenchmarkComparison(BenchmarkNifty50, strategyReturnPct, benchmarkReturnPct, BenchmarkAvailable)
	return b
}

func UnavailableBenchmark(strategyReturnPct float64) BenchmarkComparison {
	return BenchmarkComparison{
		BenchmarkName:      BenchmarkBuyAndHold,
		StrategyReturnPct:  strategyReturnPct,
		BenchmarkReturnPct: 0.0,
		ExcessReturnPct:    0.0,
		Availability:       BenchmarkUnavailable,
	}
}
