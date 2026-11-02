package orchestrator

import (
	"context"
	"testing"
	"time"
)

func TestRunStateTransitions(t *testing.T) {
	if !CanTransitionRun(RunStatusRunning, RunStatusCompleted) {
		t.Fatal("expected running->completed allowed")
	}
	if CanTransitionRun(RunStatusCompleted, RunStatusRunning) {
		t.Fatal("terminal cannot transition")
	}
}

func TestStageStateTransitions(t *testing.T) {
	if !CanTransitionStage(StageStatusPending, StageStatusRunning) {
		t.Fatal("pending->running allowed")
	}
	if CanTransitionStage(StageStatusRunning, StageStatusPending) {
		t.Fatal("not allowed")
	}
}

func TestOrchestratorStartCancel(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	stageStore := NewMemoryStageStore()
	o := New(store, stageStore, nil)
	run, cctx, cancel, err := o.StartRun(ctx, TriggerManual, RunRequest{Symbols: []string{"AAA"}}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunStatusRunning {
		t.Fatalf("status %s", run.Status)
	}
	if err := o.CancelRun(ctx, run.RunID); err != nil {
		t.Fatal(err)
	}
	cancel()
	_ = cctx
	prog, _ := o.GetProgress(ctx, run.RunID)
	if prog.RunID != run.RunID {
		t.Fatal("progress missing")
	}
	o.Cleanup(run.RunID)
}

func TestReaperReaps(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	r := Run{RunID: "r1", Status: RunStatusRunning, StartedAt: time.Now().Add(-24 * time.Hour)}
	_ = store.Create(ctx, r)
	reaper := &Reaper{Runs: store, Clock: nil, StaleThreshold: time.Hour}
	n, err := reaper.Reap(ctx)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	got, _, _ := store.Get(ctx, "r1")
	if got.Status != RunStatusFailed {
		t.Fatalf("status %s", got.Status)
	}
}

func TestSummaryAssembly(t *testing.T) {
	run := Run{RunID: "r", Status: RunStatusCompleted, StartedAt: time.Now(), SymbolsCount: 1, CompletedCount: 1}
	stages := []Stage{{RunID: "r", Symbol: "AAA", StageName: StageDataFetch, Status: StageStatusCompleted}}
	s := AssembleSummary(run, stages)
	if s.RunID != "r" {
		t.Fatal("bad summary")
	}
}
