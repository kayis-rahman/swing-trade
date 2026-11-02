package orchestrator

import (
	"context"
	"time"
)

// Reaper marks runs that have been RUNNING longer than StaleThreshold as FAILED
// (orphaned runs whose process died or hung). It runs periodically and on startup.
type Reaper struct {
	Runs           RunRepository
	Stages         StageRepository
	Clock          Clock
	StaleThreshold time.Duration
	Logger         interface {
		Info(string, ...any)
		Warn(string, ...any)
		Error(string, ...any)
	}
}

// Reap scans for orphaned runs and marks them failed. Returns number reaped.
func (r *Reaper) Reap(ctx context.Context) (int, error) {
	if r.Runs == nil {
		return 0, nil
	}
	if r.StaleThreshold <= 0 {
		r.StaleThreshold = time.Hour
	}
	runs, err := r.Runs.List(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	if r.Clock != nil {
		now = r.Clock.Now()
	}
	reaped := 0
	for _, run := range runs {
		if run.Status != RunStatusRunning {
			continue
		}
		if now.Sub(run.StartedAt) < r.StaleThreshold {
			continue
		}
		msg := "reaped as orphaned (stale run)"
		if err := r.reapRun(ctx, run, now, msg); err != nil {
			return reaped, err
		}
		reaped++
		if r.Logger != nil {
			r.Logger.Warn("reaped orphaned run", "runId", run.RunID, "startedAt", run.StartedAt)
		}
	}
	return reaped, nil
}

// ReapAllOnStartup reaps every currently RUNNING run (used at application ready).
func (r *Reaper) ReapAllOnStartup(ctx context.Context) error {
	if r.Runs == nil {
		return nil
	}
	runs, err := r.Runs.List(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	if r.Clock != nil {
		now = r.Clock.Now()
	}
	for _, run := range runs {
		if run.Status != RunStatusRunning {
			continue
		}
		msg := "reaped on startup (orphaned run)"
		if err := r.reapRun(ctx, run, now, msg); err != nil {
			return err
		}
	}
	return nil
}

// reapRun marks a single run and its RUNNING stages as failed.
func (r *Reaper) reapRun(ctx context.Context, run Run, now time.Time, msg string) error {
	completed := now
	if err := r.Runs.UpdateStatus(ctx, run.RunID, RunStatusFailed, &completed, msg); err != nil {
		return err
	}
	if r.Stages != nil {
		stages, err := r.Stages.ListByRun(ctx, run.RunID)
		if err == nil {
			for _, st := range stages {
				if st.Status != StageStatusRunning {
					continue
				}
				st.Status = StageStatusError
				st.CompletedAt = &completed
				st.ErrorMessage = "Reaped as orphaned - run was force-failed while this stage was RUNNING"
				_ = r.Stages.Update(ctx, st)
			}
		}
	}
	return nil
}

// StaleThresholdForSeconds computes Java's generous threshold.
func StaleThresholdForSeconds(stageSumSeconds int64) time.Duration {
	mult := int64(6)
	return time.Duration(stageSumSeconds*mult) * time.Second
}
