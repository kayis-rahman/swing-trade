package strategy

// PortfolioMetrics mirrors Java record.
type PortfolioMetrics struct {
	CAGRPct                 float64
	TotalReturnPct          float64
	AnnualizedVolatilityPct float64
	SharpeRatio             float64
	SortinoRatio            float64
	MaxDrawdownPct          float64
	MaxDrawdownDurationDays int
	CalmarRatio             float64
	TotalTrades             int
	WinRatePct              float64
	WinRateCiLowPct         float64
	WinRateCiHighPct        float64
	AvgWinPct               float64
	AvgLossPct              float64
	PayoffRatio             float64
	ExpectancyR             float64
	ExpectancyRCiLow        float64
	ExpectancyRCiHigh       float64
	ExpectancyRupees        float64
	ProfitFactor            float64
	ExposurePct             float64
	AvgHoldingDays          float64
	Turnover                float64
	CostDragPct             float64
	BenchmarkComparison     BenchmarkComparison
}

// BenchmarkComparison mirrors Java nested record.
type BenchmarkComparison struct {
	Available bool
	Alpha     *float64
	Beta      *float64
	Note      string
}

func BenchmarkComparisonStub() BenchmarkComparison {
	return BenchmarkComparison{
		Available: false,
		Alpha:     nil,
		Beta:      nil,
		Note:      "Nifty buy&hold benchmark not wired up (plan A3/F16) - deferred to RS_NIFTY (§5.4) when a benchmark index data source is ingested anyway.",
	}
}
