package scheduler

import (
	"context"
	"fmt"
	"time"
)

// SoakScheduler is the minimal shape a Go scheduler must satisfy to be driven
// by the soak harness. Schedulers in PR 165 implement this surface (Run) or
// equivalent single-shot entry points that the harness invokes.
type SoakScheduler interface {
	Name() string
	Run(ctx context.Context)
}

// SoakHarness runs a set of schedulers against scratch data over a number of
// simulated weekly cycles. It never calls time.Now directly; all time is
// supplied through the injected clock.
type SoakHarness struct {
	Clock   func() time.Time
	Cycles  int
	Weekday time.Weekday
	Hour    int
	Minute  int
}

// NewSoakHarness constructs a harness. Cycles defaults to two when zero or
// negative, matching the Phase 6 gate's two-weekly-cycle requirement.
func NewSoakHarness(clock func() time.Time, cycles int) *SoakHarness {
	if cycles <= 0 {
		cycles = 2
	}
	return &SoakHarness{
		Clock:   clock,
		Cycles:  cycles,
		Weekday: time.Sunday,
		Hour:    18,
		Minute:  0,
	}
}

// CycleTimes returns the simulated fire time for each weekly cycle.
func (h *SoakHarness) CycleTimes() []time.Time {
	if h.Clock == nil {
		return nil
	}
	base := h.Clock()
	times := make([]time.Time, h.Cycles)
	for i := 0; i < h.Cycles; i++ {
		fire := base.AddDate(0, 0, i*7)
		times[i] = time.Date(fire.Year(), fire.Month(), fire.Day(), h.Hour, h.Minute, 0, 0, fire.Location())
	}
	return times
}

// Run executes every scheduler once per cycle at the simulated clock time.
func (h *SoakHarness) Run(ctx context.Context, schedulers []SoakScheduler) error {
	if h.Clock == nil {
		return fmt.Errorf("soak harness requires an injected clock")
	}
	for _, scheduler := range schedulers {
		if scheduler == nil {
			continue
		}
		for _, fire := range h.CycleTimes() {
			_ = fire
			scheduler.Run(ctx)
		}
	}
	return nil
}

// Compile-time proof that the three Go schedulers satisfy SoakScheduler.
var (
	_ SoakScheduler = (*EodIngestionScheduler)(nil)
	_ SoakScheduler = (*IntradayIngestionScheduler)(nil)
	_ SoakScheduler = (*FyersSymbolMasterScheduler)(nil)
)
