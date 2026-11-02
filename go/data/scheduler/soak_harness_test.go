package scheduler

import (
	"context"
	"testing"
	"time"
)

type fakeSoakScheduler struct {
	name  string
	calls int
}

func (f *fakeSoakScheduler) Name() string { return f.name }

func (f *fakeSoakScheduler) Run(context.Context) { f.calls++ }

func TestSoakHarnessRunsTwoCycles(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 9, 6, 0, 0, 0, 0, ist) }
	h := NewSoakHarness(now, 2)
	s := &fakeSoakScheduler{name: "digest"}
	if err := h.Run(context.Background(), []SoakScheduler{s}); err != nil {
		t.Fatal(err)
	}
	if s.calls != 2 {
		t.Fatalf("calls=%d want 2", s.calls)
	}
	if len(h.CycleTimes()) != 2 {
		t.Fatalf("cycle count=%d", len(h.CycleTimes()))
	}
}

func TestSoakHarnessDefaultsToTwoCycles(t *testing.T) {
	h := NewSoakHarness(func() time.Time { return time.Now() }, 0)
	if h.Cycles != 2 {
		t.Fatalf("cycles=%d", h.Cycles)
	}
}
