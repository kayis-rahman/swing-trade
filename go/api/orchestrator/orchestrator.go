package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
)

// Orchestrator coordinates run lifecycle, cancellation/interrupt and reaping hooks.
// It does not execute stages itself; sibling slices drive StageRunner values and report
// stage transitions through the repository ports.
type Orchestrator struct {
	Runs           RunRepository
	Stages         StageRepository
	Clock          clock.Clock
	MaxConcurrent  int
	StaleThreshold time.Duration
	Notifier       RunSummaryNotifier

	mu            sync.Mutex
	startLock     sync.Mutex
	cancelFuncs   map[RunID]context.CancelFunc
	cancelledRuns map[RunID]struct{}
}

// New constructs an Orchestrator with sensible defaults.
func New(runs RunRepository, stages StageRepository, clk clock.Clock) *Orchestrator {
	if clk == nil {
		clk = clock.System{}
	}
	return &Orchestrator{
		Runs:          runs,
		Stages:        stages,
		Clock:         clk,
		MaxConcurrent: 1,
		cancelFuncs:   map[RunID]context.CancelFunc{},
		cancelledRuns: map[RunID]struct{}{},
	}
}

func (o *Orchestrator) now() time.Time {
	if o.Clock != nil {
		return o.Clock.Now()
	}
	return time.Now()
}

// StartRun creates a new RUNNING run. Returns ConcurrentRunError if another run is active.
func (o *Orchestrator) StartRun(ctx context.Context, trigger TriggerType, req RunRequest, runCtx context.Context) (Run, context.Context, context.CancelFunc, error) {
	o.startLock.Lock()
	defer o.startLock.Unlock()
	if o.Runs != nil {
		active, ok, err := o.Runs.FindActive(ctx)
		if err != nil {
			return Run{}, nil, nil, err
		}
		if ok {
			return Run{}, nil, nil, &ConcurrentRunError{ActiveRun: active}
		}
	}
	symbols := req.Symbols
	run := Run{
		RunID:        RunID(fmt.Sprintf("run-%d", o.now().UnixNano())),
		TriggerType:  trigger,
		Status:       RunStatusRunning,
		StartedAt:    o.now(),
		SymbolsCount: len(symbols),
	}
	if o.Runs != nil {
		if err := o.Runs.Create(ctx, run); err != nil {
			return Run{}, nil, nil, err
		}
		// create PENDING stage rows for every symbol x stage
		var stages []Stage
		for _, sym := range symbols {
			for _, name := range AllStageNames() {
				st := Stage{RunID: run.RunID, Symbol: sym, StageName: name, Status: StageStatusPending}
				stages = append(stages, st)
			}
		}
		if o.Stages != nil && len(stages) > 0 {
			_ = o.Stages.CreateBatch(ctx, stages)
		}
	}
	cctx, cancel := context.WithCancel(runCtx)
	o.mu.Lock()
	o.cancelFuncs[run.RunID] = cancel
	o.mu.Unlock()
	return run, cctx, cancel, nil
}

// CancelRun interrupts an active run.
func (o *Orchestrator) CancelRun(ctx context.Context, runID RunID) error {
	if o.Runs != nil {
		run, ok, err := o.Runs.Get(ctx, runID)
		if err != nil {
			return err
		}
		if !ok || run.Status != RunStatusRunning {
			return nil
		}
	}
	o.mu.Lock()
	cancel, ok := o.cancelFuncs[runID]
	o.cancelledRuns[runID] = struct{}{}
	o.mu.Unlock()
	if ok {
		cancel()
	}
	now := o.now()
	// mark running stages cancelled
	if o.Stages != nil {
		stages, _ := o.Stages.ListByRun(ctx, runID)
		for _, st := range stages {
			if st.Status != StageStatusRunning {
				continue
			}
			st.Status = StageStatusCancelled
			st.CompletedAt = &now
			st.ErrorMessage = "Cancelled by user request"
			_ = o.Stages.Update(ctx, st)
		}
	}
	if o.Runs != nil {
		if err := o.Runs.UpdateStatus(ctx, runID, RunStatusCancelled, &now, "cancelled by request"); err != nil {
			return err
		}
	}
	return nil
}

// IsCancelled reports whether runID has been cancelled.
func (o *Orchestrator) IsCancelled(runID RunID) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	_, ok := o.cancelledRuns[runID]
	return ok
}

// Cleanup cancels tracking for a finished run.
func (o *Orchestrator) Cleanup(runID RunID) {
	o.mu.Lock()
	delete(o.cancelFuncs, runID)
	delete(o.cancelledRuns, runID)
	o.mu.Unlock()
}

// GetProgress delegates to repository + AssembleProgress.
func (o *Orchestrator) GetProgress(ctx context.Context, runID RunID) (Progress, error) {
	if o.Runs == nil {
		return Progress{}, ErrRunNotFound
	}
	run, ok, err := o.Runs.Get(ctx, runID)
	if err != nil {
		return Progress{}, err
	}
	if !ok {
		return Progress{}, ErrRunNotFound
	}
	stages := []Stage{}
	if o.Stages != nil {
		stages, _ = o.Stages.ListByRun(ctx, runID)
	}
	return AssembleProgress(run, stages), nil
}

// GetSummary delegates to repository + AssembleSummary.
func (o *Orchestrator) GetSummary(ctx context.Context, runID RunID) (RunSummary, error) {
	if o.Runs == nil {
		return RunSummary{}, ErrRunNotFound
	}
	run, ok, err := o.Runs.Get(ctx, runID)
	if err != nil {
		return RunSummary{}, err
	}
	if !ok {
		return RunSummary{}, ErrRunNotFound
	}
	stages := []Stage{}
	if o.Stages != nil {
		stages, _ = o.Stages.ListByRun(ctx, runID)
	}
	return AssembleSummary(run, stages), nil
}

// ListRuns returns all runs.
func (o *Orchestrator) ListRuns(ctx context.Context) ([]Run, error) {
	if o.Runs == nil {
		return nil, nil
	}
	return o.Runs.List(ctx)
}

// ReapOrphanedRuns scans for stale RUNNING runs and marks them failed.
func (o *Orchestrator) ReapOrphanedRuns(ctx context.Context) (int, error) {
	if o.Runs == nil {
		return 0, nil
	}
	reaper := &Reaper{Runs: o.Runs, Stages: o.Stages, Clock: o.Clock, StaleThreshold: o.StaleThreshold}
	return reaper.Reap(ctx)
}

// ReapAllRunningRunsOnStartup marks all RUNNING runs failed (startup reaper).
func (o *Orchestrator) ReapAllRunningRunsOnStartup(ctx context.Context) error {
	if o.Runs == nil {
		return nil
	}
	reaper := &Reaper{Runs: o.Runs, Stages: o.Stages, Clock: o.Clock, StaleThreshold: o.StaleThreshold}
	return reaper.ReapAllOnStartup(ctx)
}

// Notify delegates to notifier if configured.
func (o *Orchestrator) Notify(ctx context.Context, summary RunSummary) error {
	if o.Notifier == nil {
		return nil
	}
	return o.Notifier.Notify(ctx, summary)
}
