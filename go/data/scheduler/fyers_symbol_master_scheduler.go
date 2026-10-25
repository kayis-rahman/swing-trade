package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// JobLockFyersSymbolMaster is the per-job PostgreSQL advisory-lock key for the
// symbol master refresh, distinct from the EOD ingestion lock.
const JobLockFyersSymbolMaster int64 = 57060006

// SymbolMasterJob is the refresh surface the scheduled job needs, mirroring
// Java's FyersSymbolMasterService refresh and count methods.
type SymbolMasterJob interface {
	Refresh(ctx context.Context) (int, error)
	CountSymbols(ctx context.Context) (int64, error)
}

// FyersSymbolMasterConfig carries the Go-side job gate. Enabled defaults to
// false in configuration so Go jobs stay off until the G8 switch.
type FyersSymbolMasterConfig struct {
	Enabled bool
}

// FyersSymbolMasterScheduler ports Java's FyersSymbolMasterService scheduling:
// a weekday 08:45 IST refresh and the startup refresh-when-empty path, both
// gated on Fyers being the active broker.
type FyersSymbolMasterScheduler struct {
	enabled      bool
	brokerSource func(context.Context) (string, error)
	job          SymbolMasterJob
	lock         Locker
	now          func() time.Time
	logger       *slog.Logger
	waitFor      func(context.Context, time.Time) error
}

// NewFyersSymbolMasterScheduler wires the job over explicit ports. A nil
// brokerSource means "no provider wired", which Java treats as active so
// manual refresh paths are never silently disabled.
func NewFyersSymbolMasterScheduler(
	cfg FyersSymbolMasterConfig,
	logger *slog.Logger,
	now func() time.Time,
	brokerSource func(context.Context) (string, error),
	job SymbolMasterJob,
	lock Locker,
) (*FyersSymbolMasterScheduler, error) {
	if logger == nil || now == nil || job == nil || lock == nil {
		return nil, fmt.Errorf("logger, clock, symbol master, and locker are required")
	}
	return &FyersSymbolMasterScheduler{
		enabled:      cfg.Enabled,
		brokerSource: brokerSource,
		job:          job,
		lock:         lock,
		now:          now,
		logger:       logger,
		waitFor:      waitForDeadline,
	}, nil
}

// ScheduledRefresh mirrors Java's cron `0 45 8 * * MON-FRI` method: gate,
// active-broker check, then one locked refresh that reports failures instead
// of aborting the loop.
func (s *FyersSymbolMasterScheduler) ScheduledRefresh(ctx context.Context) {
	if !s.enabled {
		s.logger.Info("Fyers symbol master refresh scheduler disabled")
		return
	}
	if !s.fyersActive(ctx, "scheduled") {
		return
	}
	acquired, err := s.lock(ctx, func(ctx context.Context) error {
		_, err := s.job.Refresh(ctx)
		return err
	})
	if err != nil {
		s.logger.Error("scheduled Fyers symbol master refresh failed", "module", "data", "error", err)
		return
	}
	if !acquired {
		s.logger.Info("Fyers symbol master refresh skipped: another worker holds the job lock")
	}
}

// RefreshIfEmpty mirrors Java's ApplicationReadyEvent listener: refresh only
// when Fyers is active and the table has no rows yet.
func (s *FyersSymbolMasterScheduler) RefreshIfEmpty(ctx context.Context) {
	if !s.enabled {
		s.logger.Info("startup Fyers symbol master refresh disabled")
		return
	}
	if !s.fyersActive(ctx, "startup") {
		return
	}
	acquired, err := s.lock(ctx, func(ctx context.Context) error {
		count, err := s.job.CountSymbols(ctx)
		if err != nil {
			return fmt.Errorf("count stored Fyers symbols: %w", err)
		}
		if count > 0 {
			s.logger.Debug("Fyers symbol master already populated", "symbols", count)
			return nil
		}
		_, err = s.job.Refresh(ctx)
		return err
	})
	if err != nil {
		s.logger.Warn("startup Fyers symbol master refresh failed", "module", "data", "error", err)
		return
	}
	if !acquired {
		s.logger.Info("startup Fyers symbol master refresh skipped: another worker holds the job lock")
	}
}

// StartLoop sleeps until the next weekday 08:45 IST fire time until ctx is
// cancelled.
func (s *FyersSymbolMasterScheduler) StartLoop(ctx context.Context) {
	runOnSchedule(ctx, s.waitFor, s.now, NextFyersSymbolMasterRun, s.ScheduledRefresh)
}

// fyersActive mirrors Java's isFyersActive: no configured provider source is
// active, and otherwise only an explicit fyers selection refreshes.
func (s *FyersSymbolMasterScheduler) fyersActive(ctx context.Context, trigger string) bool {
	if s.brokerSource == nil {
		return true
	}
	broker, err := s.brokerSource(ctx)
	if err != nil {
		s.logger.Warn("Fyers symbol master refresh skipped: active broker unavailable",
			"module", "data", "trigger", trigger, "error", err)
		return false
	}
	if !strings.EqualFold(broker, "fyers") {
		s.logger.Debug("Fyers symbol master refresh skipped: active broker is not fyers",
			"module", "data", "trigger", trigger, "broker", broker)
		return false
	}
	return true
}
