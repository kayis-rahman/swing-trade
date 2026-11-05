package strategy

// WalkForwardResult mirrors Java record.
type WalkForwardResult struct {
	Folds                   []WalkForwardFold
	ConcatenatedEquityCurve []DailyEquityPoint
	SharpeStdDev            float64
	Unstable                bool
	HoldoutUnlocked         bool
}
