package orchestrator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
)

type fakeNotifier struct {
	mu   sync.Mutex
	seen []RunSummary
	err  error
}

func (f *fakeNotifier) Notify(_ context.Context, s RunSummary) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.seen = append(f.seen, s)
	return nil
}

func TestExtra_TransitionFunctions(t *testing.T) {
	if _, err := TransitionRun(RunStatusRunning, RunStatusCompleted); err != nil {
		t.Fatal(err)
	}
	if _, err := TransitionRun(RunStatusCompleted, RunStatusRunning); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("expected illegal transition, got %v", err)
	}
	if _, err := TransitionStage(StageStatusPending, StageStatusRunning); err != nil {
		t.Fatal(err)
	}
	if _, err := TransitionStage(StageStatusRunning, StageStatusPending); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("expected illegal transition, got %v", err)
	}
	if !IsTerminalRun(RunStatusFailed) || IsTerminalRun(RunStatusRunning) {
		t.Fatal("terminal mismatch")
	}
	if !IsTerminalStage(StageStatusCancelled) || IsTerminalStage(StageStatusPending) {
		t.Fatal("stage terminal mismatch")
	}
}

func TestExtra_IsCancelledAndCleanup(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	stages := NewMemoryStageStore()
	o := New(store, stages, nil)
	run, _, cancel, err := o.StartRun(ctx, TriggerScheduled, RunRequest{Symbols: []string{"AAA"}}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if o.IsCancelled(run.RunID) {
		t.Fatal("should not be cancelled yet")
	}
	if err := o.CancelRun(ctx, run.RunID); err != nil {
		t.Fatal(err)
	}
	if !o.IsCancelled(run.RunID) {
		t.Fatal("expected cancelled")
	}
	cancel()
	o.Cleanup(run.RunID)
	if o.IsCancelled(run.RunID) {
		t.Fatal("cleanup did not clear cancelled")
	}
}

func TestExtra_GetSummaryAndListRuns(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	stages := NewMemoryStageStore()
	o := New(store, stages, nil)
	run, _, cancel, err := o.StartRun(ctx, TriggerManual, RunRequest{Symbols: []string{"AAA"}}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	o.Cleanup(run.RunID)
	sum, err := o.GetSummary(ctx, run.RunID)
	if err != nil || sum.RunID != run.RunID {
		t.Fatalf("summary err=%v", err)
	}
	runs, err := o.ListRuns(ctx)
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs=%d err=%v", len(runs), err)
	}
	if _, err := o.GetSummary(ctx, "missing"); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestExtra_ConcurrentRun(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	stages := NewMemoryStageStore()
	o := New(store, stages, nil)
	run, _, cancel, err := o.StartRun(ctx, TriggerManual, RunRequest{}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	_, _, _, err = o.StartRun(ctx, TriggerManual, RunRequest{}, context.Background())
	if err == nil {
		t.Fatal("expected concurrent run error")
	}
	var cre *ConcurrentRunError
	if !errors.As(err, &cre) {
		t.Fatalf("expected ConcurrentRunError, got %T", err)
	}
	if cre.ActiveRun.RunID != run.RunID {
		t.Fatalf("active run mismatch")
	}
}

func TestExtra_ReapOrphanedRuns(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	stages := NewMemoryStageStore()
	r := Run{RunID: "old", Status: RunStatusRunning, StartedAt: time.Now().Add(-24 * time.Hour)}
	if err := store.Create(ctx, r); err != nil {
		t.Fatal(err)
	}
	stage := Stage{RunID: "old", Symbol: "AAA", StageName: StageDataFetch, Status: StageStatusRunning}
	if err := stages.CreateBatch(ctx, []Stage{stage}); err != nil {
		t.Fatal(err)
	}
	o := New(store, stages, nil)
	n, err := o.ReapOrphanedRuns(ctx)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	got, _, _ := store.Get(ctx, "old")
	if got.Status != RunStatusFailed {
		t.Fatalf("status=%s", got.Status)
	}
	stlist, _ := stages.ListByRun(ctx, "old")
	if stlist[0].Status != StageStatusError {
		t.Fatalf("stage status=%s", stlist[0].Status)
	}
}

func TestExtra_ReapAllStartup(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	if err := store.Create(ctx, Run{RunID: "a", Status: RunStatusRunning, StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	o := New(store, NewMemoryStageStore(), nil)
	if err := o.ReapAllRunningRunsOnStartup(ctx); err != nil {
		t.Fatal(err)
	}
	got, _, _ := store.Get(ctx, "a")
	if got.Status != RunStatusFailed {
		t.Fatalf("status=%s", got.Status)
	}
	// nil runs path
	o2 := New(nil, nil, nil)
	if err := o2.ReapAllRunningRunsOnStartup(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := o2.ReapOrphanedRuns(ctx)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestExtra_ReaperDirectNilRuns(t *testing.T) {
	ctx := context.Background()
	reaper := &Reaper{Runs: nil, Clock: nil, StaleThreshold: time.Hour}
	n, err := reaper.Reap(ctx)
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if err := reaper.ReapAllOnStartup(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestExtra_ReaperClockAndThreshold(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	if err := store.Create(ctx, Run{RunID: "stale", Status: RunStatusRunning, StartedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	clk := clock.System{}
	reaper := &Reaper{Runs: store, Clock: clk, StaleThreshold: 0}
	n, err := reaper.Reap(ctx)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if StaleThresholdForSeconds(10) != 60*time.Second {
		t.Fatalf("threshold=%v", StaleThresholdForSeconds(10))
	}
}

func TestExtra_Notify(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	o := New(store, NewMemoryStageStore(), nil)
	fn := &fakeNotifier{}
	o.Notifier = fn
	if err := o.Notify(ctx, RunSummary{RunID: "r"}); err != nil {
		t.Fatal(err)
	}
	if len(fn.seen) != 1 {
		t.Fatalf("seen=%d", len(fn.seen))
	}
	fn.err = errors.New("boom")
	if err := o.Notify(ctx, RunSummary{RunID: "r"}); err == nil {
		t.Fatal("expected error")
	}
	// nil notifier
	o2 := New(store, NewMemoryStageStore(), nil)
	if err := o2.Notify(ctx, RunSummary{}); err != nil {
		t.Fatal(err)
	}
}

func TestExtra_MemoryStoreIncrementAndUpdate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	if err := store.Create(ctx, Run{RunID: "r", Status: RunStatusRunning}); err != nil {
		t.Fatal(err)
	}
	if err := store.IncrementCompleted(ctx, "r"); err != nil {
		t.Fatal(err)
	}
	got, _, _ := store.Get(ctx, "r")
	if got.CompletedCount != 1 {
		t.Fatalf("count=%d", got.CompletedCount)
	}
	if err := store.IncrementCompleted(ctx, "missing"); err != ErrRunNotFound {
		t.Fatalf("err=%v", err)
	}
	if err := store.UpdateStatus(ctx, "r", RunStatusCompleted, func() *time.Time { now := time.Now(); return &now }(), "done"); err != nil {
		t.Fatal(err)
	}
	got, _, _ = store.Get(ctx, "r")
	if got.Status != RunStatusCompleted || got.ErrorMessage != "done" || got.CompletedAt == nil {
		t.Fatalf("update failed")
	}
	// stage store update
	ss := NewMemoryStageStore()
	st := Stage{RunID: "r", Symbol: "AAA", StageName: StageDataFetch, Status: StageStatusPending}
	if err := ss.CreateBatch(ctx, []Stage{st}); err != nil {
		t.Fatal(err)
	}
	st.Status = StageStatusRunning
	if err := ss.Update(ctx, st); err != nil {
		t.Fatal(err)
	}
	list, _ := ss.ListByRun(ctx, "r")
	if list[0].Status != StageStatusRunning {
		t.Fatalf("status=%s", list[0].Status)
	}
	if err := ss.Update(ctx, Stage{RunID: "r", Symbol: "ZZZ", StageName: StageDataFetch, Status: StageStatusRunning}); err == nil {
		t.Fatal("expected not found")
	}
}

func TestExtra_CancelRunWithRunningStage(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()
	stages := NewMemoryStageStore()
	o := New(store, stages, nil)
	run, _, cancel, err := o.StartRun(ctx, TriggerManual, RunRequest{Symbols: []string{"AAA"}}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	stlist, _ := stages.ListByRun(ctx, run.RunID)
	for _, st := range stlist {
		st.Status = StageStatusRunning
		_ = stages.Update(ctx, st)
	}
	if err := o.CancelRun(ctx, run.RunID); err != nil {
		t.Fatal(err)
	}
	got, _, _ := store.Get(ctx, run.RunID)
	if got.Status != RunStatusCancelled {
		t.Fatalf("status=%s", got.Status)
	}
	list, _ := stages.ListByRun(ctx, run.RunID)
	for _, st := range list {
		if st.Status != StageStatusCancelled {
			t.Fatalf("stage status=%s", st.Status)
		}
	}
}

func TestExtra_WorklistAndProgress(t *testing.T) {
	run := Run{RunID: "r", Status: RunStatusRunning, StartedAt: time.Now(), SymbolsCount: 1, CompletedCount: 0}
	stages := []Stage{{RunID: "r", Symbol: "AAA", StageName: StageDataFetch, Status: StageStatusCompleted}}
	s := AssembleSummary(run, stages)
	if len(WorklistSymbols(stages)) != 1 {
		t.Fatal("worklist")
	}
	if s.CountsByStage[StageDataFetch].Completed != 1 {
		t.Fatal("completed count")
	}
	p := AssembleProgress(run, stages)
	if p.StagesCompleted != 1 || p.StagesTotal != 1 {
		t.Fatalf("progress=%+v", p)
	}
	// ConcurrentRunError Error()
	if (&ConcurrentRunError{}).Error() == "" {
		t.Fatal("empty error")
	}
	// RunRepository missing returns nil for ListRuns when nil
	o := New(nil, nil, nil)
	if runs, err := o.ListRuns(context.Background()); err != nil || runs != nil {
		t.Fatalf("nil runs behavior changed: %v %v", runs, err)
	}
	// nil runs GetProgress
	if _, err := o.GetProgress(context.Background(), "x"); err != ErrRunNotFound {
		t.Fatalf("err=%v", err)
	}
}
