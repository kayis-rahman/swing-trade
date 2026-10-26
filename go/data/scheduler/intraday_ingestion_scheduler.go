package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data"
)

// JobLockIntradayIngestion is the per-job PostgreSQL advisory-lock key for the
// scheduled intraday session-bar run, distinct from the other job locks.
const JobLockIntradayIngestion int64 = 57041545

// intradayInterval is the stage-0 storage and Yahoo interval code for the
// captain-selected intraday timeframe.
const intradayInterval = "15m"

// IntradayIngestionJob is the ingest surface the scheduled job needs.
type IntradayIngestionJob interface {
	IngestWatchlistInterval(ctx context.Context, interval string, from, to time.Time) (data.IntradayIngestionResult, error)
}

// IntradayIngestionConfig carries the Go-side job gates. Enabled defaults to
// false in configuration so Go jobs stay off until the G8 switch; Java's
// intraday.ingestion.enabled defaults to true.
type IntradayIngestionConfig struct {
	Enabled      bool
	LookbackDays int
}

// IntradayIngestionScheduler ports Java's IntradayIngestionScheduler: a
// weekday 15:45 IST run after the session close, skipping non-trading days and
// feeding the bounded service window.
type IntradayIngestionScheduler struct {
	enabled      bool
	lookbackDays int
	holidays     data.HolidayReadStore
	job          IntradayIngestionJob
	lock         Locker
	now          func() time.Time
	logger       *slog.Logger
	waitFor      func(context.Context, time.Time) error
}

// NewIntradayIngestionScheduler wires the job over explicit ports.
func NewIntradayIngestionScheduler(
	cfg IntradayIngestionConfig,
	logger *slog.Logger,
	now func() time.Time,
	holidays data.HolidayReadStore,
	job IntradayIngestionJob,
	lock Locker,
) (*IntradayIngestionScheduler, error) {
	if logger == nil || now == nil || holidays == nil || job == nil || lock == nil {
		return nil, fmt.Errorf("logger, clock, holiday store, ingest job, and locker are required")
	}
	return &IntradayIngestionScheduler{
		enabled:      cfg.Enabled,
		lookbackDays: cfg.LookbackDays,
		holidays:     holidays,
		job:          job,
		lock:         lock,
		now:          now,
		logger:       logger,
		waitFor:      waitForDeadline,
	}, nil
}

// IngestSessionBars mirrors Java's cron `0 45 15 * * MON-FRI` method: gate,
// trading-session check, bounded lookback window, then one locked run whose
// failures never abort the schedule loop.
func (s *IntradayIngestionScheduler) IngestSessionBars(ctx context.Context) {
	if !s.enabled {
		s.logger.Debug("intraday ingestion scheduler disabled")
		return
	}
	today := data.DateOnlyIST(s.now())
	trading, err := data.IsNSETradingSession(ctx, today, s.holidays)
	if err != nil {
		s.logger.Error("intraday ingestion session check failed", "module", "data", "error", err)
		return
	}
	if !trading {
		s.logger.Info("intraday ingestion skipped: not an NSE trading session",
			"module", "data", "date", today.Format(time.DateOnly))
		return
	}
	lookback := s.lookbackDays
	if lookback < 0 {
		lookback = 0
	}
	from := today.AddDate(0, 0, -lookback)
	s.logger.Info("intraday ingestion scheduled run",
		"module", "data", "from", from.Format(time.DateOnly), "to", today.Format(time.DateOnly))
	acquired, err := s.lock(ctx, func(ctx context.Context) error {
		result, err := s.job.IngestWatchlistInterval(ctx, intradayInterval, from, today)
		if err != nil {
			return err
		}
		s.logger.Info("intraday ingestion scheduled run completed",
			"module", "data",
			"succeeded", result.SymbolsSucceeded, "failed", result.SymbolsFailed, "saved", result.BarsSaved)
		return nil
	})
	if err != nil {
		s.logger.Warn("intraday ingestion scheduled run failed", "module", "data", "error", err)
		return
	}
	if !acquired {
		s.logger.Info("intraday ingestion skipped: another worker holds the job lock")
	}
}

// StartLoop sleeps until the next weekday 15:45 IST fire time and runs the
// job until ctx is cancelled.
func (s *IntradayIngestionScheduler) StartLoop(ctx context.Context) {
	runOnSchedule(ctx, s.waitFor, s.now, NextIntradayIngestionRun, s.IngestSessionBars)
}

// NextIntradayIngestionRun returns the next weekday 15:45 IST time strictly after now.
func NextIntradayIngestionRun(now time.Time) time.Time {
	return nextWeekdayFire(now, 15, 45)
}
