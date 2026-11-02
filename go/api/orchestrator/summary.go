package orchestrator

import "time"

// StageStats mirrors Java JobRunSummary.StageStats.
type StageStats struct {
	Total           int   `json:"total"`
	Completed       int   `json:"completed"`
	Errors          int   `json:"errors"`
	TotalDurationMs int64 `json:"totalDurationMs"`
}

// DegradedStageBreakdown mirrors Java.
type DegradedStageBreakdown struct {
	Stage  StageName `json:"stage"`
	Reason string    `json:"reason"`
	Count  int       `json:"count"`
}

// SkippedStrategyBreakdown mirrors Java.
type SkippedStrategyBreakdown struct {
	VariantID string `json:"variantId"`
	Outcome   string `json:"outcome"`
	Reason    string `json:"reason"`
	Symbols   int    `json:"symbols"`
}

// SymbolDetail mirrors Java.
type SymbolDetail struct {
	Symbol         string  `json:"symbol"`
	OverallStatus  string  `json:"overallStatus"`
	CompletedCount int     `json:"completedCount"`
	TotalCount     int     `json:"totalCount"`
	Stages         []Stage `json:"stages"`
}

// RunSummary mirrors Java JobRunSummary.
type RunSummary struct {
	RunID             RunID                      `json:"runId"`
	Status            RunStatus                  `json:"status"`
	StartedAt         time.Time                  `json:"startedAt"`
	CompletedAt       *time.Time                 `json:"completedAt,omitempty"`
	DurationMs        int64                      `json:"durationMs"`
	SymbolsCount      int                        `json:"symbolsCount"`
	CompletedCount    int                        `json:"completedCount"`
	FailedCount       int                        `json:"failedCount"`
	CountsByStage     map[StageName]StageStats   `json:"countsByStage"`
	DegradedStages    []DegradedStageBreakdown   `json:"degradedStages,omitempty"`
	SkippedStrategies []SkippedStrategyBreakdown `json:"skippedStrategies,omitempty"`
	SymbolDetails     []SymbolDetail             `json:"symbolDetails,omitempty"`
}

// Progress mirrors Java JobRunProgress.
type Progress struct {
	RunID            RunID     `json:"runId"`
	Status           RunStatus `json:"status"`
	SymbolsTotal     int       `json:"symbolsTotal"`
	SymbolsCompleted int       `json:"symbolsCompleted"`
	SymbolsFailed    int       `json:"symbolsFailed"`
	StagesTotal      int       `json:"stagesTotal"`
	StagesCompleted  int       `json:"stagesCompleted"`
	StagesError      int       `json:"stagesError"`
}

// WorklistSymbols returns distinct symbols present in stages.
func WorklistSymbols(stages []Stage) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range stages {
		if _, ok := seen[s.Symbol]; !ok {
			seen[s.Symbol] = struct{}{}
			out = append(out, s.Symbol)
		}
	}
	return out
}

// AssembleSummary builds a RunSummary from a Run and its stages.
func AssembleSummary(run Run, stages []Stage) RunSummary {
	summary := RunSummary{
		RunID:          run.RunID,
		Status:         run.Status,
		StartedAt:      run.StartedAt,
		CompletedAt:    run.CompletedAt,
		SymbolsCount:   run.SymbolsCount,
		CompletedCount: run.CompletedCount,
		FailedCount:    run.FailedCount,
		CountsByStage:  map[StageName]StageStats{},
	}
	if run.CompletedAt != nil {
		summary.DurationMs = run.CompletedAt.Sub(run.StartedAt).Milliseconds()
	}
	for _, name := range AllStageNames() {
		summary.CountsByStage[name] = StageStats{}
	}
	symbolStages := map[string][]Stage{}
	for _, st := range stages {
		ss := summary.CountsByStage[st.StageName]
		ss.Total++
		switch st.Status {
		case StageStatusCompleted, StageStatusDegraded:
			ss.Completed++
		case StageStatusError:
			ss.Errors++
		}
		if st.DurationMs != nil {
			ss.TotalDurationMs += *st.DurationMs
		}
		summary.CountsByStage[st.StageName] = ss
		symbolStages[st.Symbol] = append(symbolStages[st.Symbol], st)
	}
	for _, sym := range WorklistSymbols(stages) {
		sts := symbolStages[sym]
		completed := 0
		for _, st := range sts {
			if st.Status == StageStatusCompleted || st.Status == StageStatusDegraded {
				completed++
			}
		}
		overall := "RUNNING"
		if len(sts) > 0 {
			allTerminal := true
			anyError := false
			for _, st := range sts {
				if !IsTerminalStage(st.Status) {
					allTerminal = false
				}
				if st.Status == StageStatusError || st.Status == StageStatusCancelled {
					anyError = true
				}
			}
			if allTerminal {
				if anyError {
					overall = "FAILED"
				} else {
					overall = "COMPLETED"
				}
			}
		}
		summary.SymbolDetails = append(summary.SymbolDetails, SymbolDetail{
			Symbol:         sym,
			OverallStatus:  overall,
			CompletedCount: completed,
			TotalCount:     len(sts),
			Stages:         sts,
		})
	}
	return summary
}

// AssembleProgress builds a Progress from a Run and stages.
func AssembleProgress(run Run, stages []Stage) Progress {
	p := Progress{
		RunID:            run.RunID,
		Status:           run.Status,
		SymbolsTotal:     run.SymbolsCount,
		SymbolsCompleted: run.CompletedCount,
		SymbolsFailed:    run.FailedCount,
	}
	for _, ss := range AssembleSummary(run, stages).CountsByStage {
		p.StagesTotal += ss.Total
		p.StagesCompleted += ss.Completed
		p.StagesError += ss.Errors
	}
	return p
}
