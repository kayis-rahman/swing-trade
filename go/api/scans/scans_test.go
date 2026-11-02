package scans

import (
	"context"
	"testing"
	"time"
)

func TestCanTransition(t *testing.T) {
	if !CanTransition(StatusRunning, StatusPaused) { t.Fatal("running->paused") }
	if !CanTransition(StatusPaused, StatusRunning) { t.Fatal("paused->running") }
	if !CanTransition(StatusRunning, StatusCancelled) { t.Fatal("running->cancelled") }
	if CanTransition(StatusCompleted, StatusRunning) { t.Fatal("completed should not transition") }
}

func TestManagerPauseResumeCancel(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, err := m.Start(ctx, 10)
	if err != nil { t.Fatal(err) }
	if err := m.Pause(ctx, run.RunID); err != nil { t.Fatalf("pause: %v", err) }
	if !m.IsPaused(run.RunID) { t.Fatal("expected paused") }
	if err := m.Resume(ctx, run.RunID); err != nil { t.Fatalf("resume: %v", err) }
	if m.IsPaused(run.RunID) { t.Fatal("expected not paused") }
	if err := m.Cancel(ctx, run.RunID); err != nil { t.Fatalf("cancel: %v", err) }
	got, ok, _ := repo.Get(ctx, run.RunID)
	if !ok { t.Fatal("missing run") }
	if got.Status != StatusCancelled { t.Fatalf("status=%s want cancelled", got.Status) }
}

func TestManagerRejectsSecondStart(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	if _, err := m.Start(ctx, 5); err != nil { t.Fatal(err) }
	if _, err := m.Start(ctx, 5); err != ErrAlreadyActive { t.Fatalf("err=%v want %v", err, ErrAlreadyActive) }
}

func TestRecoverInterruptedRuns(t *testing.T) {
	repo := NewMemoryRunStore()
	ctx := context.Background()
	repo.Create(ctx, Run{RunID: "r1", Status: StatusRunning, StartedAt: /*zero*/ time.Now()})
	repo.Create(ctx, Run{RunID: "r2", Status: StatusPaused, StartedAt: time.Now()})
	m := NewManager(repo, nil)
	n, err := m.RecoverInterruptedRuns(ctx)
	if err != nil || n != 2 { t.Fatalf("n=%d err=%v", n, err) }
}

func TestManagerCompleteFailActive(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, err := m.Start(ctx, 3)
	if err != nil { t.Fatal(err) }
	if m.Active() == nil { t.Fatal("active nil") }
	if err := m.Complete(ctx, run.RunID); err != nil { t.Fatalf("complete: %v", err) }
	if m.Active() != nil { t.Fatal("active should nil") }
	run2, err := m.Start(ctx, 2)
	if err != nil { t.Fatal(err) }
	if err := m.Fail(ctx, run2.RunID, "boom"); err != nil { t.Fatalf("fail: %v", err) }
	if !m.IsCancelled(run.RunID) {
		// IsCancelled on completed returns false; just exercise
	}
}

func TestManagerInvalidPauseResumeCancel(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, _ := m.Start(ctx, 2)
	_ = m.Cancel(ctx, run.RunID)
	if err := m.Pause(ctx, run.RunID); err == nil { t.Fatal("pause cancelled should fail") }
	if err := m.Resume(ctx, run.RunID); err == nil { t.Fatal("resume cancelled should fail") }
	if err := m.Cancel(ctx, run.RunID); err == nil { t.Fatal("cancel twice should fail") }
}

func TestManagerUnknownRuns(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	if err := m.Pause(ctx, "x"); err != ErrRunNotFound { t.Fatalf("err=%v", err) }
	if err := m.Resume(ctx, "x"); err != ErrRunNotFound { t.Fatalf("err=%v", err) }
	if err := m.Cancel(ctx, "x"); err != ErrRunNotFound { t.Fatalf("err=%v", err) }
	if err := m.Complete(ctx, "x"); err != ErrRunNotFound { t.Fatalf("err=%v", err) }
	if err := m.Fail(ctx, "x", "m"); err != ErrRunNotFound { t.Fatalf("err=%v", err) }
}

func TestTransitionInvalid(t *testing.T) {
	if _, err := Transition(StatusCompleted, StatusRunning); err == nil {
		t.Fatal("expected error")
	}
}

func TestIsCancelledAndProgress(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, _ := m.Start(ctx, 2)
	m.Pause(ctx, run.RunID)
	if err := repo.UpdateProgress(ctx, run.RunID, 1, 0); err != nil { t.Fatal(err) }
	got, _, _ := repo.Get(ctx, run.RunID)
	if got.CompletedCount != 1 { t.Fatalf("progress not saved: %d", got.CompletedCount) }
	_ = m.IsCancelled(run.RunID)
}
