package scans

import (
	"context"
	"testing"
	"time"
)

func TestCanTransition(t *testing.T) {
	if !CanTransition(StatusRunning, StatusPaused) {
		t.Fatal("running->paused")
	}
	if !CanTransition(StatusPaused, StatusRunning) {
		t.Fatal("paused->running")
	}
	if !CanTransition(StatusRunning, StatusCancelled) {
		t.Fatal("running->cancelled")
	}
	if CanTransition(StatusCompleted, StatusRunning) {
		t.Fatal("completed should not transition")
	}
}

func TestManagerPauseResumeCancel(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, err := m.Start(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Pause(ctx, run.RunID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if !m.IsPaused(run.RunID) {
		t.Fatal("expected paused")
	}
	if err := m.Resume(ctx, run.RunID); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if m.IsPaused(run.RunID) {
		t.Fatal("expected not paused")
	}
	if err := m.Cancel(ctx, run.RunID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	got, ok, _ := repo.Get(ctx, run.RunID)
	if !ok {
		t.Fatal("missing run")
	}
	if got.Status != StatusCancelled {
		t.Fatalf("status=%s want cancelled", got.Status)
	}
}

func TestManagerRejectsSecondStart(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	if _, err := m.Start(ctx, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start(ctx, 5); err != ErrAlreadyActive {
		t.Fatalf("err=%v want %v", err, ErrAlreadyActive)
	}
}

func TestRecoverInterruptedRuns(t *testing.T) {
	repo := NewMemoryRunStore()
	ctx := context.Background()
	_ = repo.Create(ctx, Run{RunID: "r1", Status: StatusRunning, StartedAt: /*zero*/ time.Now()})
	_ = repo.Create(ctx, Run{RunID: "r2", Status: StatusPaused, StartedAt: time.Now()})
	m := NewManager(repo, nil)
	n, err := m.RecoverInterruptedRuns(ctx)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestManagerCompleteFailActive(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, err := m.Start(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	if m.Active() == nil {
		t.Fatal("active nil")
	}
	if err := m.Complete(ctx, run.RunID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if m.Active() != nil {
		t.Fatal("active should nil")
	}
	run2, err := m.Start(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Fail(ctx, run2.RunID, "boom"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	_ = m.IsCancelled(run.RunID) // IsCancelled on completed returns false; just exercise
}

func TestManagerInvalidPauseResumeCancel(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, _ := m.Start(ctx, 2)
	_ = m.Cancel(ctx, run.RunID)
	if err := m.Pause(ctx, run.RunID); err == nil {
		t.Fatal("pause cancelled should fail")
	}
	if err := m.Resume(ctx, run.RunID); err == nil {
		t.Fatal("resume cancelled should fail")
	}
	if err := m.Cancel(ctx, run.RunID); err == nil {
		t.Fatal("cancel twice should fail")
	}
}

func TestManagerUnknownRuns(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	if err := m.Pause(ctx, "x"); err != ErrRunNotFound {
		t.Fatalf("err=%v", err)
	}
	if err := m.Resume(ctx, "x"); err != ErrRunNotFound {
		t.Fatalf("err=%v", err)
	}
	if err := m.Cancel(ctx, "x"); err != ErrRunNotFound {
		t.Fatalf("err=%v", err)
	}
	if err := m.Complete(ctx, "x"); err != ErrRunNotFound {
		t.Fatalf("err=%v", err)
	}
	if err := m.Fail(ctx, "x", "m"); err != ErrRunNotFound {
		t.Fatalf("err=%v", err)
	}
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
	_ = m.Pause(ctx, run.RunID)
	if err := repo.UpdateProgress(ctx, run.RunID, 1, 0); err != nil {
		t.Fatal(err)
	}
	got, _, _ := repo.Get(ctx, run.RunID)
	if got.CompletedCount != 1 {
		t.Fatalf("progress not saved: %d", got.CompletedCount)
	}
	_ = m.IsCancelled(run.RunID)
}

func TestCancelClearsPausedFlag(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, err := m.Start(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Pause(ctx, run.RunID); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if !m.IsPaused(run.RunID) {
		t.Fatal("expected paused before cancel")
	}
	if err := m.Cancel(ctx, run.RunID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if m.IsPaused(run.RunID) {
		t.Fatal("expected pause cleared after cancel")
	}
	if !m.IsCancelled(run.RunID) {
		t.Fatal("expected cancelled")
	}
	got, ok, _ := repo.Get(ctx, run.RunID)
	if !ok || got.Status != StatusCancelled {
		t.Fatalf("status=%v want cancelled", got.Status)
	}
}

func TestCompleteAndFailClearPausedFlag(t *testing.T) {
	repo := NewMemoryRunStore()
	m := NewManager(repo, nil)
	ctx := context.Background()
	run, err := m.Start(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Pause(ctx, run.RunID); err != nil {
		t.Fatal(err)
	}
	if err := m.Resume(ctx, run.RunID); err != nil {
		t.Fatal(err)
	}
	if err := m.Pause(ctx, run.RunID); err != nil {
		t.Fatal(err)
	}
	if err := m.Complete(ctx, run.RunID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if m.IsPaused(run.RunID) {
		t.Fatal("complete should clear pause")
	}
	run2, err := m.Start(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	_ = m.Pause(ctx, run2.RunID)
	if err := m.Fail(ctx, run2.RunID, "boom"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if m.IsPaused(run2.RunID) {
		t.Fatal("fail should clear pause")
	}
}

func TestManagerReconstructsFlagsFromRepository(t *testing.T) { //nolint:funlen

	repo := NewMemoryRunStore()
	ctx := context.Background()
	_ = repo.Create(ctx, Run{RunID: "r-paused", Status: StatusPaused, StartedAt: time.Now()})
	m := NewManager(repo, nil)

	// Start should see the active run and reconstruct an in-memory pause flag.
	if _, err := m.Start(ctx, 2); err != ErrAlreadyActive {
		t.Fatalf("err=%v want %v", err, ErrAlreadyActive)
	}
	if !m.IsPaused("r-paused") {
		t.Fatal("reconstructed paused run should report paused")
	}

	// Pause path when no prior in-memory flag exists creates one.
	repo2 := NewMemoryRunStore()
	_ = repo2.Create(ctx, Run{RunID: "r-run", Status: StatusRunning, StartedAt: time.Now()})
	m2 := NewManager(repo2, nil)
	if err := m2.Pause(ctx, "r-run"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if !m2.IsPaused("r-run") {
		t.Fatal("pause should set flag")
	}

	// Resume path when no prior in-memory flag exists creates an unpaused flag.
	repo3 := NewMemoryRunStore()
	_ = repo3.Create(ctx, Run{RunID: "r-paused2", Status: StatusPaused, StartedAt: time.Now()})
	m3 := NewManager(repo3, nil)
	if err := m3.Resume(ctx, "r-paused2"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if m3.IsPaused("r-paused2") {
		t.Fatal("resume should clear pause")
	}

	// Cancel path when no prior in-memory flag creates cancelled and unpaused flags.
	repo4 := NewMemoryRunStore()
	_ = repo4.Create(ctx, Run{RunID: "r-cancel", Status: StatusRunning, StartedAt: time.Now()})
	m4 := NewManager(repo4, nil)
	if err := m4.Cancel(ctx, "r-cancel"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if !m4.IsCancelled("r-cancel") {
		t.Fatal("cancel should set cancelled")
	}
	if m4.IsPaused("r-cancel") {
		t.Fatal("cancel should clear pause")
	}

	// Complete and Fail paths with missing flags.
	repo5 := NewMemoryRunStore()
	_ = repo5.Create(ctx, Run{RunID: "r-complete", Status: StatusRunning, StartedAt: time.Now()})
	m5 := NewManager(repo5, nil)
	if err := m5.Complete(ctx, "r-complete"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if m5.IsPaused("r-complete") {
		t.Fatal("complete should clear pause")
	}
	repo6 := NewMemoryRunStore()
	_ = repo6.Create(ctx, Run{RunID: "r-fail", Status: StatusRunning, StartedAt: time.Now()})
	m6 := NewManager(repo6, nil)
	if err := m6.Fail(ctx, "r-fail", "boom"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if m6.IsPaused("r-fail") {
		t.Fatal("fail should clear pause")
	}
}
