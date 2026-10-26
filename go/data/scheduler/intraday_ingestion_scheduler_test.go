package scheduler

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

type fakeIntradayJob struct {
	mu           sync.Mutex
	calls        int
	lastFrom     time.Time
	lastTo       time.Time
	lastInterval string
	err          error
	result       data.IntradayIngestionResult
}

func (f *fakeIntradayJob) IngestWatchlistInterval(_ context.Context, interval string, from, to time.Time) (data.IntradayIngestionResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastInterval = interval
	f.lastFrom = from
	f.lastTo = to
	if f.err != nil {
		return data.IntradayIngestionResult{}, f.err
	}
	return f.result, nil
}

func (f *fakeIntradayJob) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type intradayHolidayStore struct {
	holidays map[string]data.Holiday
}

func (h *intradayHolidayStore) GetHolidayByDate(_ context.Context, date time.Time) (*data.Holiday, error) {
	if holiday, ok := h.holidays[date.Format("2006-01-02")]; ok {
		return &holiday, nil
	}
	return nil, nil
}

func (h *intradayHolidayStore) ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]data.Holiday, error) {
	return nil, nil
}

func (h *intradayHolidayStore) ListUpcomingHolidays(context.Context, time.Time) ([]data.Holiday, error) {
	return nil, nil
}

// wednesdayNow pins the scheduled run to Wednesday2026-09-1610:00 IST.
func wednesdayNow() time.Time {
	return time.Date(2026, 9, 16, 4, 30, 0, 0, time.UTC)
}

func newIntradaySchedulerForTest(t *testing.T, cfg IntradayIngestionConfig) (*IntradayIngestionScheduler, *fakeIntradayJob, *fakeLocker) {
	t.Helper()
	job := &fakeIntradayJob{result: data.IntradayIngestionResult{SymbolsSucceeded: 2, BarsSaved: 3}}
	locker := &fakeLocker{acquired: true}
	scheduler, err := NewIntradayIngestionScheduler(cfg, discardLogger(), wednesdayNow,
		&intradayHolidayStore{holidays: map[string]data.Holiday{}}, job, locker.lock)
	if err != nil {
		t.Fatalf("NewIntradayIngestionScheduler() error = %v", err)
	}
	return scheduler, job, locker
}

func TestIntradaySchedulerRunsBoundedWindowOnTradingDays(t *testing.T) {
	scheduler, job, locker := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true, LookbackDays: 10})

	scheduler.IngestSessionBars(context.Background())

	if job.callCount() != 1 || locker.calls != 1 {
		t.Fatalf("job calls = %d, lock calls = %d", job.callCount(), locker.calls)
	}
	if job.lastInterval != "15m" {
		t.Fatalf("interval = %q, want 15m", job.lastInterval)
	}
	wantFrom := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	if !job.lastFrom.Equal(wantFrom) || !job.lastTo.Equal(wantTo) {
		t.Fatalf("window = %s..%s, want %s..%s", job.lastFrom, job.lastTo, wantFrom, wantTo)
	}
}

func TestIntradaySchedulerDisabledTouchesNothing(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	job := &fakeIntradayJob{}
	locker := &fakeLocker{acquired: true}
	scheduler, err := NewIntradayIngestionScheduler(IntradayIngestionConfig{Enabled: false}, logger, wednesdayNow,
		&intradayHolidayStore{}, job, locker.lock)
	if err != nil {
		t.Fatal(err)
	}

	scheduler.IngestSessionBars(context.Background())

	if job.callCount() != 0 || locker.calls != 0 {
		t.Fatalf("disabled run touched dependencies: job=%d lock=%d", job.callCount(), locker.calls)
	}
	if !strings.Contains(buffer.String(), "scheduler disabled") {
		t.Fatalf("missing disabled log: %s", buffer.String())
	}
}

func TestIntradaySchedulerSkipsNonTradingDays(t *testing.T) {
	scheduler, job, locker := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true, LookbackDays: 10})
	scheduler.holidays = &intradayHolidayStore{holidays: map[string]data.Holiday{
		"2026-09-16": {Date: time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), Type: "FULL"},
	}}

	scheduler.IngestSessionBars(context.Background())

	if job.callCount() != 0 || locker.calls != 0 {
		t.Fatalf("holiday run touched dependencies: job=%d lock=%d", job.callCount(), locker.calls)
	}
}

func TestIntradaySchedulerSessionCheckFailureSkipsRun(t *testing.T) {
	scheduler, job, locker := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true, LookbackDays: 10})
	scheduler.holidays = failingHolidayStore{err: errors.New("db down")}

	scheduler.IngestSessionBars(context.Background())

	if job.callCount() != 0 || locker.calls != 0 {
		t.Fatalf("session failure touched dependencies: job=%d lock=%d", job.callCount(), locker.calls)
	}
}

type failingHolidayStore struct{ err error }

func (f failingHolidayStore) GetHolidayByDate(context.Context, time.Time) (*data.Holiday, error) {
	return nil, f.err
}

func (f failingHolidayStore) ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]data.Holiday, error) {
	return nil, f.err
}

func (f failingHolidayStore) ListUpcomingHolidays(context.Context, time.Time) ([]data.Holiday, error) {
	return nil, f.err
}

func TestIntradaySchedulerClampsNegativeLookback(t *testing.T) {
	scheduler, job, _ := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true, LookbackDays: -5})

	scheduler.IngestSessionBars(context.Background())

	if job.callCount() != 1 {
		t.Fatalf("job calls = %d", job.callCount())
	}
	wantTo := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	if !job.lastFrom.Equal(wantTo) || !job.lastTo.Equal(wantTo) {
		t.Fatalf("clamped window = %s..%s, want %s..%s", job.lastFrom, job.lastTo, wantTo, wantTo)
	}
}

func TestIntradaySchedulerLockBoundaries(t *testing.T) {
	t.Run("lock held elsewhere skips the run", func(t *testing.T) {
		scheduler, job, locker := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true})
		locker.acquired = false

		scheduler.IngestSessionBars(context.Background())

		if locker.calls != 1 || job.callCount() != 0 {
			t.Fatalf("lock skip failed: lock=%d job=%d", locker.calls, job.callCount())
		}
	})

	t.Run("lock failure is logged and skips the run", func(t *testing.T) {
		scheduler, job, locker := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true})
		locker.err = errors.New("pool closed")

		scheduler.IngestSessionBars(context.Background())

		if locker.calls != 1 || job.callCount() != 0 {
			t.Fatalf("lock failure handling: lock=%d job=%d", locker.calls, job.callCount())
		}
	})
}

func TestIntradaySchedulerJobFailureIsReportedNotFatal(t *testing.T) {
	scheduler, job, _ := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true})
	job.err = errors.New("provider down")

	scheduler.IngestSessionBars(context.Background())

	if job.callCount() != 1 {
		t.Fatalf("job calls = %d, want the failure to be reported", job.callCount())
	}
}

func TestNextIntradayIngestionRunIsNextWeekday1545IST(t *testing.T) {
	istZone := time.FixedZone("IST", 5*60*60+30*60)
	for name, tc := range map[string]struct {
		now  time.Time
		want time.Time
	}{
		"before fire same weekday": {
			now:  time.Date(2026, 9, 16, 14, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 16, 15, 45, 0, 0, istZone),
		},
		"after fire rolls to next weekday": {
			now:  time.Date(2026, 9, 16, 16, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 17, 15, 45, 0, 0, istZone),
		},
		"friday evening rolls to monday": {
			now:  time.Date(2026, 9, 18, 17, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 21, 15, 45, 0, 0, istZone),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := NextIntradayIngestionRun(tc.now)
			if !got.Equal(tc.want) || !got.After(tc.now) {
				t.Fatalf("NextIntradayIngestionRun(%s) = %s, want %s", tc.now, got, tc.want)
			}
		})
	}
}

func TestIntradaySchedulerStartLoopRunsAtFireTimeAndStops(t *testing.T) {
	//15:44 IST on Wednesday: the next fire is the same weekday at15:45 IST.
	wednesday := time.Date(2026, 9, 16, 10, 14, 0, 0, time.UTC)
	scheduler, job, _ := newIntradaySchedulerForTest(t, IntradayIngestionConfig{Enabled: true, LookbackDays: 10})
	scheduler.now = func() time.Time { return wednesday }

	fired := make(chan time.Time, 1)
	ran := make(chan struct{}, 1)
	var calls int
	scheduler.waitFor = func(ctx context.Context, deadline time.Time) error {
		calls++
		if calls == 1 {
			select {
			case fired <- deadline:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		select {
		case ran <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		scheduler.StartLoop(ctx)
		close(done)
	}()
	deadline := <-fired
	want := time.Date(2026, 9, 16, 15, 45, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
	if !deadline.Equal(want) {
		t.Fatalf("loop deadline = %s, want %s", deadline, want)
	}
	<-ran
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartLoop did not stop after cancel")
	}
	if job.callCount() == 0 {
		t.Fatal("scheduled run never happened at the fire time")
	}
}

func TestNewIntradayIngestionSchedulerValidatesDependencies(t *testing.T) {
	if _, err := NewIntradayIngestionScheduler(
		IntradayIngestionConfig{Enabled: true}, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
}
