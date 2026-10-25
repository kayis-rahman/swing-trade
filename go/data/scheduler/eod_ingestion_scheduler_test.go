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

// fixed Wednesday 2026-09-1611:00 UTC, matching Java's EodIngestionSchedulerTest pin.
func fixedNow() time.Time {
	return time.Date(2026, 9, 16, 11, 0, 0, 0, time.UTC)
}

type fakeWatchlist struct {
	rows  []data.WatchlistEntry
	err   error
	calls int
}

func (f *fakeWatchlist) ListActiveWatchlist(context.Context) ([]data.WatchlistEntry, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func (f *fakeWatchlist) ListAllWatchlist(context.Context) ([]data.WatchlistEntry, error) {
	return f.rows, f.err
}

type fakeHolidays struct {
	holidays map[string]data.Holiday
	err      error
	calls    int
}

func (f *fakeHolidays) GetHolidayByDate(_ context.Context, date time.Time) (*data.Holiday, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if holiday, ok := f.holidays[date.Format("2006-01-02")]; ok {
		return &holiday, nil
	}
	return nil, nil
}

func (f *fakeHolidays) ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]data.Holiday, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeHolidays) ListUpcomingHolidays(context.Context, time.Time) ([]data.Holiday, error) {
	return nil, errors.New("not implemented")
}

type fakeDays struct {
	exists map[string]bool
	err    error
	calls  int
}

func (f *fakeDays) CandleExistsOnDate(_ context.Context, symbol string, date time.Time) (bool, error) {
	f.calls++
	if f.err != nil {
		return false, f.err
	}
	return f.exists[symbol+"|"+date.Format("2006-01-02")], nil
}

type fakeWindows struct {
	window data.CandleDateWindow
	err    error
	calls  int
}

func (f *fakeWindows) GetDailyDateWindow(context.Context, string) (data.CandleDateWindow, error) {
	f.calls++
	return f.window, f.err
}

func (f *fakeWindows) CountDailyCandles(context.Context, string) (int64, error) { return 0, nil }

type rangeCall struct {
	symbol string
	from   time.Time
	to     time.Time
}

type fakeIngester struct {
	mu               sync.Mutex
	rangeCalls       []rangeCall
	incrementalCalls []rangeCall
	rangeOutcome     data.IngestionOutcome
	incremental      data.IngestionOutcome
	errOutcomeFor    string
}

func (f *fakeIngester) IngestRange(_ context.Context, symbol string, from, to time.Time, _, _ string) data.IngestionOutcome {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rangeCalls = append(f.rangeCalls, rangeCall{symbol: symbol, from: from, to: to})
	if symbol == f.errOutcomeFor {
		return data.IngestionOutcome{SourceOutcome: data.OutcomeTransientSourceFailure, ErrorMessage: "provider down"}
	}
	if f.rangeOutcome.SourceOutcome != "" {
		return f.rangeOutcome
	}
	return data.IngestionOutcome{SourceOutcome: data.OutcomeDataReceived, SavedRows: 1}
}

func (f *fakeIngester) IngestIncremental(_ context.Context, _ data.CandleReadStore, symbol string, from, to time.Time, _, _ string) data.IngestionOutcome {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.incrementalCalls = append(f.incrementalCalls, rangeCall{symbol: symbol, from: from, to: to})
	return f.incremental
}

func (f *fakeIngester) ranges() []rangeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]rangeCall(nil), f.rangeCalls...)
}

func (f *fakeIngester) incrementals() []rangeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]rangeCall(nil), f.incrementalCalls...)
}

type fakeLocker struct {
	mu       sync.Mutex
	calls    int
	acquired bool
	err      error
}

func (f *fakeLocker) lock(ctx context.Context, fn func(context.Context) error) (bool, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.err != nil {
		return false, f.err
	}
	if !f.acquired {
		return false, nil
	}
	return true, fn(ctx)
}

func testScheduler(t *testing.T, cfg EodConfig, logger *slog.Logger) (*EodIngestionScheduler, *fakeWatchlist, *fakeHolidays, *fakeDays, *fakeWindows, *fakeIngester, *fakeLocker) {
	t.Helper()
	watchlist := &fakeWatchlist{rows: []data.WatchlistEntry{{Symbol: "TCS"}}}
	holidays := &fakeHolidays{holidays: map[string]data.Holiday{}}
	days := &fakeDays{exists: map[string]bool{}}
	windows := &fakeWindows{}
	ingester := &fakeIngester{}
	locker := &fakeLocker{acquired: true}
	scheduler, err := NewEodIngestionScheduler(cfg, logger, fixedNow, watchlist, holidays, days, windows, ingester, locker.lock)
	if err != nil {
		t.Fatalf("NewEodIngestionScheduler() error = %v", err)
	}
	return scheduler, watchlist, holidays, days, windows, ingester, locker
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

func TestIngestsWatchlistSymbolsPlusBenchmark(t *testing.T) {
	scheduler, watchlist, holidays, _, _, ingester, locker := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())

	scheduler.IngestLatestForAll(context.Background())

	yesterday := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	calls := ingester.ranges()
	if len(calls) != 2 || calls[0].symbol != "TCS" || calls[1].symbol != NIFTY50Symbol {
		t.Fatalf("range calls = %+v, want TCS then NIFTY50", calls)
	}
	for _, call := range calls {
		if !call.from.Equal(yesterday) || !call.to.Equal(yesterday) {
			t.Fatalf("call window = %s..%s, want %s", call.from, call.to, yesterday)
		}
	}
	if watchlist.calls != 1 || holidays.calls != 3 || locker.calls != 1 {
		t.Fatalf("calls watchlist=%d holidays=%d lock=%d, want 1/3/1 (run check plus one session check per symbol)",
			watchlist.calls, holidays.calls, locker.calls)
	}
}

func TestDisabledRunTouchesNothing(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	scheduler, watchlist, holidays, days, windows, ingester, locker := testScheduler(t,
		EodConfig{Enabled: false, BenchmarkBackfillYears: 10}, logger)

	scheduler.IngestLatestForAll(context.Background())

	if watchlist.calls != 0 || holidays.calls != 0 || days.calls != 0 || windows.calls != 0 || locker.calls != 0 || len(ingester.ranges()) != 0 {
		t.Fatalf("disabled run touched dependencies: watchlist=%d holidays=%d days=%d windows=%d lock=%d",
			watchlist.calls, holidays.calls, days.calls, windows.calls, locker.calls)
	}
	if !strings.Contains(buffer.String(), "EOD ingestion scheduler disabled") {
		t.Fatalf("disabled log missing: %s", buffer.String())
	}
}

func TestSkipsWeekendRun(t *testing.T) {
	scheduler, watchlist, holidays, _, _, ingester, locker := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	scheduler.now = func() time.Time {
		return time.Date(2026, 9, 19, 11, 0, 0, 0, time.UTC) // Saturday16:30 IST
	}

	scheduler.IngestLatestForAll(context.Background())

	if watchlist.calls != 0 || holidays.calls != 0 || locker.calls != 0 || len(ingester.ranges()) != 0 {
		t.Fatalf("weekend run was not skipped: watchlist=%d holidays=%d lock=%d", watchlist.calls, holidays.calls, locker.calls)
	}
}

func TestSkipsWhenYesterdayIsHoliday(t *testing.T) {
	scheduler, watchlist, _, _, _, ingester, locker := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	scheduler.holidays = &fakeHolidays{holidays: map[string]data.Holiday{
		"2026-09-15": {Date: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), Type: "FULL"},
	}}

	scheduler.IngestLatestForAll(context.Background())

	if watchlist.calls != 0 || locker.calls != 0 || len(ingester.ranges()) != 0 {
		t.Fatalf("holiday run was not skipped: watchlist=%d lock=%d ranges=%d", watchlist.calls, locker.calls, len(ingester.ranges()))
	}
}

func TestSkipsMondayWhenYesterdayWasSundayLikeJava(t *testing.T) {
	scheduler, watchlist, _, _, _, ingester, locker := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	scheduler.now = func() time.Time {
		return time.Date(2026, 9, 21, 11, 0, 0, 0, time.UTC) // Monday16:30 IST, yesterday Sunday
	}

	scheduler.IngestLatestForAll(context.Background())

	// Java's isMarketClosed treats weekends as closed, so Monday's run skips.
	if watchlist.calls != 0 || locker.calls != 0 || len(ingester.ranges()) != 0 {
		t.Fatalf("Monday-after-Sunday run was not skipped: watchlist=%d lock=%d ranges=%d", watchlist.calls, locker.calls, len(ingester.ranges()))
	}
}

func TestHolidayStoreErrorStopsBeforeWatchlist(t *testing.T) {
	scheduler, watchlist, _, _, _, ingester, locker := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	scheduler.holidays = &fakeHolidays{err: errors.New("db down")}

	scheduler.IngestLatestForAll(context.Background())

	if watchlist.calls != 0 || locker.calls != 0 || len(ingester.ranges()) != 0 {
		t.Fatalf("holiday error did not stop the run: watchlist=%d lock=%d", watchlist.calls, locker.calls)
	}
}

func TestSkipsWorkWhenAnotherWorkerHoldsJobLock(t *testing.T) {
	scheduler, _, _, _, _, ingester, locker := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	locker.acquired = false

	scheduler.IngestLatestForAll(context.Background())

	if locker.calls != 1 || len(ingester.ranges()) != 0 {
		t.Fatalf("lock skip failed: lock calls=%d ranges=%d", locker.calls, len(ingester.ranges()))
	}
}

func TestLockFailureIsLoggedAndSkipsWork(t *testing.T) {
	scheduler, _, _, _, _, ingester, locker := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	locker.err = errors.New("pool closed")

	scheduler.IngestLatestForAll(context.Background())

	if locker.calls != 1 || len(ingester.ranges()) != 0 {
		t.Fatalf("lock failure handling: lock calls=%d ranges=%d", locker.calls, len(ingester.ranges()))
	}
}

func TestSymbolFailureDoesNotStopRemainingSymbols(t *testing.T) {
	scheduler, _, _, _, _, ingester, _ := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	ingester.errOutcomeFor = "TCS"

	scheduler.IngestLatestForAll(context.Background())

	calls := ingester.ranges()
	if len(calls) != 2 || calls[0].symbol != "TCS" || calls[1].symbol != NIFTY50Symbol {
		t.Fatalf("later symbols were abandoned: %+v", calls)
	}
}

func TestIngestSymbolSkipsNonSessionsAndStoredCandles(t *testing.T) {
	scheduler, _, _, days, _, ingester, _ := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	scheduler.holidays = &fakeHolidays{holidays: map[string]data.Holiday{
		"2026-09-15": {Date: time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), Type: "FULL"},
	}}
	yesterday := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	if err := scheduler.ingestSymbol(context.Background(), "TCS", yesterday); err != nil {
		t.Fatalf("non-session ingest error = %v", err)
	}
	if days.calls != 0 || len(ingester.ranges()) != 0 {
		t.Fatalf("non-session still consulted storage: days=%d ranges=%d", days.calls, len(ingester.ranges()))
	}

	scheduler.holidays = &fakeHolidays{holidays: map[string]data.Holiday{}}
	days.exists["TCS|2026-09-15"] = true
	if err := scheduler.ingestSymbol(context.Background(), "TCS", yesterday); err != nil {
		t.Fatalf("stored-candle ingest error = %v", err)
	}
	if len(ingester.ranges()) != 0 {
		t.Fatalf("stored candle was fetched again: %+v", ingester.ranges())
	}
}

func TestIngestSymbolOutcomeHandling(t *testing.T) {
	for name, tc := range map[string]struct {
		outcome data.IngestionOutcome
		wantErr bool
	}{
		"no usable data is success": {outcome: data.IngestionOutcome{SourceOutcome: data.OutcomeNoUsableData}},
		"invalid rows rejected is success": {
			outcome: data.IngestionOutcome{SourceOutcome: data.OutcomeInvalidRowsRejected, InvalidRows: 1},
		},
		"transient failure is an error": {
			outcome: data.IngestionOutcome{SourceOutcome: data.OutcomeTransientSourceFailure, ErrorMessage: "timeout"},
			wantErr: true,
		},
		"invalid request is an error": {
			outcome: data.IngestionOutcome{SourceOutcome: data.OutcomeInvalidRequest, ErrorMessage: "bad range"},
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			scheduler, _, _, _, _, ingester, _ := testScheduler(t,
				EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
			ingester.rangeOutcome = tc.outcome
			err := scheduler.ingestSymbol(context.Background(), "TCS", time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC))
			if (err != nil) != tc.wantErr {
				t.Fatalf("ingestSymbol() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestIngestSymbolPropagatesExistenceErrors(t *testing.T) {
	scheduler, _, _, days, _, _, _ := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	days.err = errors.New("db down")
	if err := scheduler.ingestSymbol(context.Background(), "TCS", time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("existence error = nil")
	}
}

func TestStartupBackfillGateAndWindowDecisions(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		scheduler, _, _, _, windows, ingester, locker := testScheduler(t,
			EodConfig{BenchmarkBackfillOnStartup: false, BenchmarkBackfillYears: 10}, discardLogger())
		scheduler.EnsureNiftyBenchmarkHistory(context.Background())
		if windows.calls != 0 || len(ingester.incrementals()) != 0 || locker.calls != 0 {
			t.Fatalf("disabled backfill touched dependencies: windows=%d lock=%d", windows.calls, locker.calls)
		}
	})

	t.Run("absent history backfills ten years", func(t *testing.T) {
		scheduler, _, _, _, windows, ingester, _ := testScheduler(t,
			EodConfig{BenchmarkBackfillOnStartup: true, BenchmarkBackfillYears: 10}, discardLogger())
		scheduler.EnsureNiftyBenchmarkHistory(context.Background())
		calls := ingester.incrementals()
		if len(calls) != 1 || calls[0].symbol != NIFTY50Symbol {
			t.Fatalf("incremental calls = %+v", calls)
		}
		wantFrom := time.Date(2016, 9, 16, 0, 0, 0, 0, time.UTC)
		wantTo := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
		if !calls[0].from.Equal(wantFrom) || !calls[0].to.Equal(wantTo) {
			t.Fatalf("backfill window = %s..%s, want %s..%s", calls[0].from, calls[0].to, wantFrom, wantTo)
		}
		if windows.calls == 0 {
			t.Fatal("window was not read")
		}
	})

	t.Run("shallow history backfills", func(t *testing.T) {
		scheduler, _, _, _, windows, ingester, _ := testScheduler(t,
			EodConfig{BenchmarkBackfillOnStartup: true, BenchmarkBackfillYears: 10}, discardLogger())
		recent := time.Date(2025, 9, 16, 0, 0, 0, 0, time.UTC)
		windows.window = data.CandleDateWindow{Earliest: &recent}
		scheduler.EnsureNiftyBenchmarkHistory(context.Background())
		if len(ingester.incrementals()) != 1 {
			t.Fatalf("shallow history was not backfilled: %+v", ingester.incrementals())
		}
	})

	t.Run("deep history skipped", func(t *testing.T) {
		scheduler, _, _, _, windows, ingester, _ := testScheduler(t,
			EodConfig{BenchmarkBackfillOnStartup: true, BenchmarkBackfillYears: 10}, discardLogger())
		deep := time.Date(2015, 9, 16, 0, 0, 0, 0, time.UTC)
		windows.window = data.CandleDateWindow{Earliest: &deep}
		scheduler.EnsureNiftyBenchmarkHistory(context.Background())
		if len(ingester.incrementals()) != 0 {
			t.Fatalf("deep history was backfilled anyway: %+v", ingester.incrementals())
		}
	})

	t.Run("backfill failures are logged not fatal", func(t *testing.T) {
		scheduler, _, _, _, _, ingester, _ := testScheduler(t,
			EodConfig{BenchmarkBackfillOnStartup: true, BenchmarkBackfillYears: 10}, discardLogger())
		ingester.incremental = data.IngestionOutcome{SourceOutcome: data.OutcomeTransientSourceFailure, ErrorMessage: "timeout"}
		scheduler.EnsureNiftyBenchmarkHistory(context.Background())
		if len(ingester.incrementals()) != 1 {
			t.Fatalf("backfill attempt missing: %+v", ingester.incrementals())
		}
	})
}

func TestNextEodRunIsNextWeekday1630IST(t *testing.T) {
	istZone := time.FixedZone("IST", 5*60*60+30*60)
	for name, tc := range map[string]struct {
		now  time.Time
		want time.Time
	}{
		"before fire same weekday": {
			now:  time.Date(2026, 9, 16, 10, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 16, 16, 30, 0, 0, istZone),
		},
		"after fire rolls to next weekday": {
			now:  time.Date(2026, 9, 16, 17, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 17, 16, 30, 0, 0, istZone),
		},
		"friday evening rolls to monday": {
			now:  time.Date(2026, 9, 18, 17, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 21, 16, 30, 0, 0, istZone),
		},
		"saturday rolls to monday": {
			now:  time.Date(2026, 9, 19, 12, 0, 0, 0, istZone),
			want: time.Date(2026, 9, 21, 16, 30, 0, 0, istZone),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := NextEodRun(tc.now)
			if !got.Equal(tc.want) {
				t.Fatalf("NextEodRun(%s) = %s, want %s", tc.now, got, tc.want)
			}
			if !got.After(tc.now) {
				t.Fatalf("NextEodRun(%s) = %s, not in the future", tc.now, got)
			}
		})
	}
}

func TestStartLoopRunsJobAtFireTimeAndStops(t *testing.T) {
	//16:29 IST on Wednesday: the next fire is the same weekday at 16:30 IST.
	wednesday := time.Date(2026, 9, 16, 10, 59, 0, 0, time.UTC)
	scheduler, watchlist, _, _, _, _, _ := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
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
		// Iteration two proves the first job run finished; then park until cancel.
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
	want := time.Date(2026, 9, 16, 16, 30, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
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
	if watchlist.calls == 0 {
		t.Fatal("job never ran at the fire time")
	}
}

func TestWaitForReturnsOnDeadlineAndCancellation(t *testing.T) {
	scheduler, _, _, _, _, _, _ := testScheduler(t,
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())

	if err := scheduler.waitFor(context.Background(), time.Now().Add(20*time.Millisecond)); err != nil {
		t.Fatalf("waitFor deadline error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go cancel()
	if err := scheduler.waitFor(ctx, time.Now().Add(time.Hour)); !errors.Is(err, context.Canceled) {
		t.Fatalf("waitFor cancel error = %v, want context.Canceled", err)
	}
}

func TestNewEodIngestionSchedulerValidatesDependencies(t *testing.T) {
	if _, err := NewEodIngestionScheduler(EodConfig{BenchmarkBackfillYears: 10}, nil, nil, nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
	scheduler, _, _, _, _, _, _ := testScheduler(t, EodConfig{Enabled: true, BenchmarkBackfillYears: 10}, discardLogger())
	if _, err := NewEodIngestionScheduler(EodConfig{BenchmarkBackfillYears: 0}, scheduler.logger, fixedNow,
		scheduler.watchlist, scheduler.holidays, scheduler.days, scheduler.windows, scheduler.ingester, scheduler.lock); err == nil {
		t.Fatal("zero benchmark years accepted")
	}
}
