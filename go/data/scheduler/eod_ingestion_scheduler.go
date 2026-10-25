// Package scheduler ports Java's @Scheduled ingestion jobs with Go's per-job
// advisory-lock safety; jobs stay off until the G8 switch.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

// NIFTY50Symbol matches Java's persisted benchmark symbol required for
// portfolio and excess-return metrics.
const NIFTY50Symbol = "NIFTY50"

// JobLockEODIngestion is the per-job PostgreSQL advisory-lock key so a scheduled
// EOD run can never overlap another worker (Java has no equivalent lock).
const JobLockEODIngestion int64 = 57070007

var ist = time.FixedZone("IST", 5*60*60+30*60)

const (
	providerInterval1d  = "1d"
	storageTimeframeDay = "D"
)

// CandleDayStore reports whether a daily candle already exists for a symbol and date.
type CandleDayStore interface {
	CandleExistsOnDate(ctx context.Context, symbol string, date time.Time) (bool, error)
}

// RangeIngester fetches and stores candle ranges incrementally or in full.
type RangeIngester interface {
	IngestRange(ctx context.Context, symbol string, from, to time.Time, providerInterval, storageTimeframe string) data.IngestionOutcome
	IngestIncremental(ctx context.Context, reader data.CandleReadStore, symbol string, from, to time.Time, providerInterval, storageTimeframe string) data.IngestionOutcome
}

// Locker runs a job function under the per-job advisory lock. A false acquired
// result means another worker owns the lock and the job must be skipped.
type Locker func(ctx context.Context, fn func(context.Context) error) (bool, error)

// EodConfig carries the Go-side ingestion job gates. Enabled and
// BenchmarkBackfillOnStartup default to false in configuration: Go jobs write
// only where authorized until G8 even though Java defaults them on.
type EodConfig struct {
	Enabled                    bool
	BenchmarkBackfillOnStartup bool
	BenchmarkBackfillYears     int
}

// EodIngestionScheduler ports Java's EodIngestionScheduler: weekday 16:30 IST
// latest-candle ingestion for the active watchlist plus the NIFTY50 benchmark,
// and the startup benchmark-history backfill.
type EodIngestionScheduler struct {
	enabled                    bool
	benchmarkBackfillOnStartup bool
	benchmarkBackfillYears     int
	now                        func() time.Time
	logger                     *slog.Logger
	watchlist                  data.WatchlistReadStore
	holidays                   data.HolidayReadStore
	days                       CandleDayStore
	windows                    data.CandleReadStore
	ingester                   RangeIngester
	lock                       Locker
	waitFor                    func(ctx context.Context, deadline time.Time) error
}

// NewEodIngestionScheduler wires the scheduler over explicit ports.
func NewEodIngestionScheduler(
	cfg EodConfig,
	logger *slog.Logger,
	now func() time.Time,
	watchlist data.WatchlistReadStore,
	holidays data.HolidayReadStore,
	days CandleDayStore,
	windows data.CandleReadStore,
	ingester RangeIngester,
	lock Locker,
) (*EodIngestionScheduler, error) {
	if cfg.BenchmarkBackfillYears < 1 {
		return nil, fmt.Errorf("benchmark backfill years must be positive")
	}
	if logger == nil || now == nil || watchlist == nil || holidays == nil || days == nil || windows == nil || ingester == nil || lock == nil {
		return nil, fmt.Errorf("logger, clock, watchlist, holiday store, candle store, ingester, and locker are required")
	}
	return &EodIngestionScheduler{
		enabled:                    cfg.Enabled,
		benchmarkBackfillOnStartup: cfg.BenchmarkBackfillOnStartup,
		benchmarkBackfillYears:     cfg.BenchmarkBackfillYears,
		now:                        now,
		logger:                     logger,
		watchlist:                  watchlist,
		holidays:                   holidays,
		days:                       days,
		windows:                    windows,
		ingester:                   ingester,
		lock:                       lock,
		waitFor:                    waitForDeadline,
	}, nil
}

// IngestLatestForAll mirrors Java's scheduled method: gate, weekend safety net,
// yesterday's holiday check, then one latest-candle ingest per active watchlist
// symbol plus the benchmark.
func (s *EodIngestionScheduler) IngestLatestForAll(ctx context.Context) {
	if !s.enabled {
		s.logger.Info("EOD ingestion scheduler disabled")
		return
	}
	today := dateOnly(s.now())
	if today.Weekday() == time.Saturday || today.Weekday() == time.Sunday {
		s.logger.Debug("today is a weekend; skipping scheduled EOD ingestion", "date", today.Format("2006-01-02"))
		return
	}
	yesterday := today.AddDate(0, 0, -1)
	closed, err := s.marketClosed(ctx, yesterday)
	if err != nil {
		s.logger.Error("EOD ingestion holiday check failed", "module", "data", "error", err)
		return
	}
	if closed {
		s.logger.Info("EOD ingestion skipped: previous day is an NSE market holiday", "date", yesterday.Format("2006-01-02"))
		return
	}

	entries, err := s.watchlist.ListActiveWatchlist(ctx)
	if err != nil {
		s.logger.Error("EOD ingestion watchlist read failed", "module", "data", "error", err)
		return
	}
	symbols := make([]string, 0, len(entries)+1)
	for _, entry := range entries {
		if entry.Symbol == "" {
			continue
		}
		symbols = append(symbols, entry.Symbol)
	}
	containsBenchmark := false
	for _, symbol := range symbols {
		if symbol == NIFTY50Symbol {
			containsBenchmark = true
			break
		}
	}
	if !containsBenchmark {
		symbols = append(symbols, NIFTY50Symbol)
	}
	s.logger.Info("EOD ingestion scheduled for watchlist symbols plus benchmark",
		"count", len(symbols), "date", yesterday.Format("2006-01-02"))

	start := s.now()
	acquired, err := s.lock(ctx, func(ctx context.Context) error {
		success, failures := 0, 0
		for _, symbol := range symbols {
			if err := s.ingestSymbol(ctx, symbol, yesterday); err != nil {
				failures++
				s.logger.Warn("EOD ingestion failed", "symbol", symbol, "error", err)
				continue
			}
			success++
		}
		s.logger.Info("EOD ingestion completed",
			"succeeded", success, "failed", failures, "elapsed", s.now().Sub(start))
		return nil
	})
	if err != nil {
		s.logger.Error("EOD ingestion run failed", "module", "data", "error", err)
		return
	}
	if !acquired {
		s.logger.Info("EOD ingestion skipped: another worker holds the job lock")
	}
}

// ingestSymbol mirrors Java's DataIngestionService.processSingleStock for one
// date: non-sessions and stored candles are no-ops; only provider/storage
// failures count as failures. Java's price-band persistence (phase 7) is not
// part of this job yet.
func (s *EodIngestionScheduler) ingestSymbol(ctx context.Context, symbol string, date time.Time) error {
	trading, err := data.IsNSETradingSession(ctx, date, s.holidays)
	if err != nil {
		return err
	}
	if !trading {
		return nil
	}
	exists, err := s.days.CandleExistsOnDate(ctx, symbol, date)
	if err != nil {
		return fmt.Errorf("check stored candle: %w", err)
	}
	if exists {
		return nil
	}
	// Provider clients pace themselves (60 requests/minute), matching Java's
	// per-symbol rate-limit sleep without a second scheduler-side delay.
	outcome := s.ingester.IngestRange(ctx, symbol, date, date, providerInterval1d, storageTimeframeDay)
	switch outcome.SourceOutcome {
	case data.OutcomeInvalidRequest, data.OutcomeTransientSourceFailure:
		return errors.New(outcome.ErrorMessage)
	default:
		return nil
	}
}

// EnsureNiftyBenchmarkHistory mirrors Java's startup backfill: fetch ten years
// (configurable) of benchmark history when the stored window does not reach it.
func (s *EodIngestionScheduler) EnsureNiftyBenchmarkHistory(ctx context.Context) {
	if !s.benchmarkBackfillOnStartup {
		s.logger.Info("startup NIFTY50 benchmark backfill disabled")
		return
	}
	acquired, err := s.lock(ctx, func(ctx context.Context) error {
		window, err := s.windows.GetDailyDateWindow(ctx, NIFTY50Symbol)
		if err != nil {
			return fmt.Errorf("read existing benchmark window: %w", err)
		}
		today := dateOnly(s.now())
		cutoff := today.AddDate(-s.benchmarkBackfillYears, 0, 0)
		if window.Earliest != nil && !window.Earliest.After(cutoff) {
			s.logger.Debug("NIFTY50 benchmark history already covers the requested window",
				"earliest", window.Earliest.Format("2006-01-02"))
			return nil
		}
		s.logger.Info("backfilling NIFTY50 benchmark history",
			"years", s.benchmarkBackfillYears, "earliest", window.Earliest)
		outcome := s.ingester.IngestIncremental(ctx, s.windows, NIFTY50Symbol, cutoff, today, providerInterval1d, storageTimeframeDay)
		if outcome.SourceOutcome == data.OutcomeTransientSourceFailure || outcome.SourceOutcome == data.OutcomeInvalidRequest {
			return errors.New(outcome.ErrorMessage)
		}
		return nil
	})
	if err != nil {
		s.logger.Warn("startup NIFTY50 benchmark historical backfill failed", "error", err)
		return
	}
	if !acquired {
		s.logger.Info("startup NIFTY50 benchmark backfill skipped: another worker holds the job lock")
	}
}

// StartLoop sleeps until the next weekday 16:30 IST fire time and runs the job
// until ctx is cancelled.
func (s *EodIngestionScheduler) StartLoop(ctx context.Context) {
	runOnSchedule(ctx, s.waitFor, s.now, NextEodRun, s.IngestLatestForAll)
}

// runOnSchedule waits for each computed fire time and runs the job until ctx ends.
func runOnSchedule(
	ctx context.Context,
	waitFor func(context.Context, time.Time) error,
	now func() time.Time,
	nextRun func(time.Time) time.Time,
	run func(context.Context),
) {
	for ctx.Err() == nil {
		if err := waitFor(ctx, nextRun(now())); err != nil {
			return
		}
		run(ctx)
	}
}

// waitForDeadline sleeps until the fire time or until ctx is cancelled.
func waitForDeadline(ctx context.Context, deadline time.Time) error {
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// NextEodRun returns the next weekday 16:30 IST time strictly after now.
func NextEodRun(now time.Time) time.Time {
	return nextWeekdayFire(now, 16, 30)
}

// NextFyersSymbolMasterRun returns the next weekday 08:45 IST time strictly after now.
func NextFyersSymbolMasterRun(now time.Time) time.Time {
	return nextWeekdayFire(now, 8, 45)
}

// nextWeekdayFire returns the next weekday hour:minute IST strictly after now.
func nextWeekdayFire(now time.Time, hour, minute int) time.Time {
	local := now.In(ist)
	next := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, ist)
	for !next.After(now) || next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
		next = next.AddDate(0, 0, 1)
		next = time.Date(next.Year(), next.Month(), next.Day(), hour, minute, 0, 0, ist)
	}
	return next
}

// dateOnly maps an instant to its IST calendar date at UTC midnight, the date
// convention the Go candle stores and provider fetches already use.
func dateOnly(t time.Time) time.Time {
	local := t.In(ist)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// marketClosed mirrors Java's NseHolidayService.isMarketClosed: weekends close
// the market and any stored holiday row counts as closed regardless of type.
func (s *EodIngestionScheduler) marketClosed(ctx context.Context, date time.Time) (bool, error) {
	local := date.In(ist)
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return true, nil
	}
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	holiday, err := s.holidays.GetHolidayByDate(ctx, day)
	if err != nil {
		return false, fmt.Errorf("look up NSE holiday: %w", err)
	}
	return holiday != nil, nil
}
