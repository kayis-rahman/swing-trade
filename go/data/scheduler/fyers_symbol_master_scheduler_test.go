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
)

type fakeSymbolMasterJob struct {
	mu           sync.Mutex
	refreshCalls int
	refreshErr   error
	countCalls   int
	count        int64
	countErr     error
	refreshed    chan struct{}
}

func (f *fakeSymbolMasterJob) Refresh(context.Context) (int, error) {
	f.mu.Lock()
	f.refreshCalls++
	f.mu.Unlock()
	if f.refreshed != nil {
		select {
		case f.refreshed <- struct{}{}:
		default:
		}
	}
	if f.refreshErr != nil {
		return 0, f.refreshErr
	}
	return 10, nil
}

func (f *fakeSymbolMasterJob) CountSymbols(context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.countCalls++
	return f.count, f.countErr
}

func (f *fakeSymbolMasterJob) refreshes() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshCalls
}

func (f *fakeSymbolMasterJob) counts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.countCalls
}

func brokerSourceReturning(broker string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return broker, nil }
}

func newSymbolMasterSchedulerForTest(t *testing.T, cfg FyersSymbolMasterConfig, brokerSource func(context.Context) (string, error)) (*FyersSymbolMasterScheduler, *fakeSymbolMasterJob, *fakeLocker) {
	t.Helper()
	job := &fakeSymbolMasterJob{}
	locker := &fakeLocker{acquired: true}
	scheduler, err := NewFyersSymbolMasterScheduler(cfg, discardLogger(), fixedNow, brokerSource, job, locker.lock)
	if err != nil {
		t.Fatalf("NewFyersSymbolMasterScheduler() error = %v", err)
	}
	return scheduler, job, locker
}

func TestScheduledRefreshSkipsWhenBrokerIsYahoo(t *testing.T) {
	scheduler, job, locker := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("yahoo"))

	scheduler.ScheduledRefresh(context.Background())

	if job.refreshes() != 0 || locker.calls != 0 {
		t.Fatalf("yahoo refresh touched dependencies: refresh=%d lock=%d", job.refreshes(), locker.calls)
	}
}

func TestRefreshIfEmptySkipsWhenBrokerIsYahoo(t *testing.T) {
	scheduler, job, locker := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("yahoo"))

	scheduler.RefreshIfEmpty(context.Background())

	if job.counts() != 0 || job.refreshes() != 0 || locker.calls != 0 {
		t.Fatalf("yahoo startup touched dependencies: count=%d refresh=%d lock=%d", job.counts(), job.refreshes(), locker.calls)
	}
}

func TestRefreshIfEmptyProceedsWhenBrokerIsFyers(t *testing.T) {
	scheduler, job, _ := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("fyers"))
	job.count = 5

	scheduler.RefreshIfEmpty(context.Background())

	if job.counts() != 1 {
		t.Fatalf("count calls = %d, want the emptiness check to run", job.counts())
	}
	if job.refreshes() != 0 {
		t.Fatalf("populated master was refreshed anyway: %d", job.refreshes())
	}
}

func TestRefreshIfEmptyRefreshesEmptyMaster(t *testing.T) {
	scheduler, job, locker := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("FYERS"))

	scheduler.RefreshIfEmpty(context.Background())

	if job.counts() != 1 || job.refreshes() != 1 || locker.calls != 1 {
		t.Fatalf("empty refresh = count %d, refresh %d, lock %d", job.counts(), job.refreshes(), locker.calls)
	}
}

func TestGateDefaultsActiveWithoutBrokerSource(t *testing.T) {
	scheduler, job, _ := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, nil)
	job.count = 5

	scheduler.RefreshIfEmpty(context.Background())

	if job.counts() != 1 {
		t.Fatalf("count calls = %d, want the no-provider gate to stay active", job.counts())
	}
}

func TestScheduledRefreshRunsWhenFyersActive(t *testing.T) {
	scheduler, job, locker := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("fyers"))

	scheduler.ScheduledRefresh(context.Background())

	if job.refreshes() != 1 || locker.calls != 1 {
		t.Fatalf("scheduled refresh = %d, lock = %d", job.refreshes(), locker.calls)
	}
}

func TestDisabledSymbolMasterJobsTouchNothing(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	job := &fakeSymbolMasterJob{}
	locker := &fakeLocker{acquired: true}
	scheduler, err := NewFyersSymbolMasterScheduler(
		FyersSymbolMasterConfig{Enabled: false}, logger, fixedNow, brokerSourceReturning("fyers"), job, locker.lock)
	if err != nil {
		t.Fatal(err)
	}

	scheduler.ScheduledRefresh(context.Background())
	scheduler.RefreshIfEmpty(context.Background())

	if job.refreshes() != 0 || job.counts() != 0 || locker.calls != 0 {
		t.Fatalf("disabled job touched dependencies: refresh=%d count=%d lock=%d", job.refreshes(), job.counts(), locker.calls)
	}
	if !strings.Contains(buffer.String(), "disabled") {
		t.Fatalf("disabled logs missing: %s", buffer.String())
	}
}

func TestBrokerSourceFailureSkipsWork(t *testing.T) {
	scheduler, job, locker := newSymbolMasterSchedulerForTest(t, FyersSymbolMasterConfig{Enabled: true},
		func(context.Context) (string, error) { return "", errors.New("settings down") })

	scheduler.ScheduledRefresh(context.Background())
	scheduler.RefreshIfEmpty(context.Background())

	if job.refreshes() != 0 || job.counts() != 0 || locker.calls != 0 {
		t.Fatalf("broker failure touched dependencies: refresh=%d count=%d lock=%d", job.refreshes(), job.counts(), locker.calls)
	}
}

func TestSymbolMasterLockBoundaries(t *testing.T) {
	t.Run("lock held elsewhere skips refresh", func(t *testing.T) {
		scheduler, job, locker := newSymbolMasterSchedulerForTest(t,
			FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("fyers"))
		locker.acquired = false

		scheduler.ScheduledRefresh(context.Background())

		if locker.calls != 1 || job.refreshes() != 0 {
			t.Fatalf("lock skip failed: lock=%d refresh=%d", locker.calls, job.refreshes())
		}
	})

	t.Run("lock failure is logged and skips refresh", func(t *testing.T) {
		scheduler, job, locker := newSymbolMasterSchedulerForTest(t,
			FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("fyers"))
		locker.err = errors.New("pool closed")

		scheduler.ScheduledRefresh(context.Background())

		if locker.calls != 1 || job.refreshes() != 0 {
			t.Fatalf("lock failure handling: lock=%d refresh=%d", locker.calls, job.refreshes())
		}
	})
}

func TestRefreshFailuresAreReportedNotFatal(t *testing.T) {
	scheduler, job, _ := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("fyers"))
	job.refreshErr = errors.New("download failed")

	scheduler.ScheduledRefresh(context.Background())

	if job.refreshes() != 1 {
		t.Fatalf("refresh attempts = %d, want the failure to be reported", job.refreshes())
	}
}

func TestStartupCountFailureIsReportedNotFatal(t *testing.T) {
	scheduler, job, _ := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("fyers"))
	job.countErr = errors.New("count failed")

	scheduler.RefreshIfEmpty(context.Background())

	if job.counts() != 1 || job.refreshes() != 0 {
		t.Fatalf("count failure handling: count=%d refresh=%d", job.counts(), job.refreshes())
	}
}

func TestNextFyersSymbolMasterRunIsNextWeekday0845IST(t *testing.T) {
	istZone := time.FixedZone("IST", 5*60*60+30*60)
	for name, tc := range map[string]struct {
		now  time.Time
		want time.Time
	}{
		"before fire same weekday": {
			now:  time.Date(2026, 9, 16, 7, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 16, 8, 45, 0, 0, istZone),
		},
		"after fire rolls to next weekday": {
			now:  time.Date(2026, 9, 16, 9, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 17, 8, 45, 0, 0, istZone),
		},
		"friday evening rolls to monday": {
			now:  time.Date(2026, 9, 18, 17, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 21, 8, 45, 0, 0, istZone),
		},
		"saturday morning rolls to monday": {
			now:  time.Date(2026, 9, 19, 7, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 21, 8, 45, 0, 0, istZone),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := NextFyersSymbolMasterRun(tc.now)
			if !got.Equal(tc.want) || !got.After(tc.now) {
				t.Fatalf("NextFyersSymbolMasterRun(%s) = %s, want %s", tc.now, got, tc.want)
			}
		})
	}
}

func TestSymbolMasterStartLoopRefreshesAtFireTimeAndStops(t *testing.T) {
	//08:44 IST on Wednesday: the next fire is the same weekday at 08:45 IST.
	wednesday := time.Date(2026, 9, 16, 3, 14, 0, 0, time.UTC)
	scheduler, job, _ := newSymbolMasterSchedulerForTest(t,
		FyersSymbolMasterConfig{Enabled: true}, brokerSourceReturning("fyers"))
	scheduler.now = func() time.Time { return wednesday }
	job.refreshed = make(chan struct{}, 1)

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
	want := time.Date(2026, 9, 16, 8, 45, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
	if !deadline.Equal(want) {
		t.Fatalf("loop deadline = %s, want %s", deadline, want)
	}
	<-job.refreshed
	<-ran
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StartLoop did not stop after cancel")
	}
	if job.refreshes() == 0 {
		t.Fatal("refresh never ran at the fire time")
	}
}

func TestNewFyersSymbolMasterSchedulerValidatesDependencies(t *testing.T) {
	if _, err := NewFyersSymbolMasterScheduler(
		FyersSymbolMasterConfig{Enabled: true}, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
}
