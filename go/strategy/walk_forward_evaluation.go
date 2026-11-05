package strategy

import "time"

// WalkForwardEvaluation mirrors Java's WalkForwardEvaluation record.
type WalkForwardEvaluation struct {
	Folds              []Fold
	AverageWinRate     float64
	AverageTotalReturn float64
	TotalTrades        int
}

// Fold mirrors Java's nested Fold record.
type Fold struct {
	StartDate time.Time
	EndDate   time.Time
	Result    any
}

// NewWalkForwardEvaluation copies folds defensively.
func NewWalkForwardEvaluation(folds []Fold, avgWin, avgReturn float64, totalTrades int) WalkForwardEvaluation {
	cp := make([]Fold, len(folds))
	copy(cp, folds)
	return WalkForwardEvaluation{
		Folds:              cp,
		AverageWinRate:     avgWin,
		AverageTotalReturn: avgReturn,
		TotalTrades:        totalTrades,
	}
}

func (w WalkForwardEvaluation) IsComplete(expectedFolds int) bool {
	return len(w.Folds) == expectedFolds
}
