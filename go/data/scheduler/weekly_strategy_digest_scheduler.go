package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// StrategyReportTotals is part of the Go port of the swing-trade schedulers.
type StrategyReportTotals struct {
	Tournaments int
	Executed    int
	Blocked     int
	SelectedPnL string
}

// StrategyReportVariant is part of the Go port of the swing-trade schedulers.
type StrategyReportVariant struct {
	VariantID     string
	StrategyType  string
	TimesSelected int
	BuySignals    int
	ShadowTrades  int
	ShadowWinRate *float64
	ShadowPnL     string
}

// StrategyReport is part of the Go port of the swing-trade schedulers.
type StrategyReport struct {
	From     time.Time
	To       time.Time
	Totals   StrategyReportTotals
	Variants []StrategyReportVariant
}

// StrategyAttribution is part of the Go port of the swing-trade schedulers.
type StrategyAttribution interface {
	Report(ctx context.Context, from, to time.Time) (StrategyReport, error)
}

// EmbedSender is part of the Go port of the swing-trade schedulers.
type EmbedSender interface {
	SendEmbed(title, description string, color int) bool
}

// WeeklyStrategyDigestScheduler is part of the Go port of the swing-trade schedulers.
type WeeklyStrategyDigestScheduler struct {
	attribution StrategyAttribution
	discord     EmbedSender
	enabled     bool
	now         func() time.Time
	logger      *slog.Logger
}

// NewWeeklyStrategyDigestScheduler is part of the Go port of the swing-trade schedulers.
func NewWeeklyStrategyDigestScheduler(
	attribution StrategyAttribution,
	discord EmbedSender,
	enabled bool,
	logger *slog.Logger,
	now func() time.Time,
) (*WeeklyStrategyDigestScheduler, error) {
	if logger == nil || now == nil || attribution == nil || discord == nil {
		return nil, fmt.Errorf("attribution, discord, logger, and clock are required")
	}
	return &WeeklyStrategyDigestScheduler{attribution: attribution, discord: discord, enabled: enabled, logger: logger, now: now}, nil
}

// Enabled is part of the Go port of the swing-trade schedulers.
func (s *WeeklyStrategyDigestScheduler) Enabled() bool { return s.enabled }

// SendWeeklyStrategyDigest is part of the Go port of the swing-trade schedulers.
func (s *WeeklyStrategyDigestScheduler) SendWeeklyStrategyDigest(ctx context.Context) {
	s.sendWeeklyStrategyDigest(ctx)
}

func (s *WeeklyStrategyDigestScheduler) sendWeeklyStrategyDigest(ctx context.Context) {
	if !s.enabled {
		if s.logger != nil {
			s.logger.Debug("Weekly strategy digest disabled")
		}
		return
	}
	now := s.now()
	to := dateOnlyIST(now)
	report, err := s.attribution.Report(ctx, to.AddDate(0, 0, -7), to)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("weekly strategy digest failed", "error", err)
		}
		return
	}
	sent := s.discord.SendEmbed("Weekly Strategy Report", FormatStrategyDigest(report), 0x5865F2)
	if !sent && s.logger != nil {
		s.logger.Warn("Weekly strategy digest not delivered")
	}
}

// Send is part of the Go port of the swing-trade schedulers.
func (s *WeeklyStrategyDigestScheduler) Send(ctx context.Context) { s.SendWeeklyStrategyDigest(ctx) }

// FormatStrategyDigest is part of the Go port of the swing-trade schedulers.
func FormatStrategyDigest(report StrategyReport) string {
	sb := report.From.Format("2006-01-02") + " to " + report.To.Format("2006-01-02") + "\n"
	sb += strconv.Itoa(report.Totals.Tournaments) + " tournaments, " + strconv.Itoa(report.Totals.Executed) + " executed, " + strconv.Itoa(report.Totals.Blocked) + " blocked, selected-book P&L " + report.Totals.SelectedPnL + "\n"
	if len(report.Variants) == 0 {
		return sb + "No active variants."
	}
	for _, v := range report.Variants {
		line := "\n**" + v.VariantID + "** (" + v.StrategyType + "): won " + strconv.Itoa(v.TimesSelected) + ", " + strconv.Itoa(v.BuySignals) + " BUYs, shadow " + strconv.Itoa(v.ShadowTrades) + " trades"
		if v.ShadowWinRate != nil {
			line += " / " + formatFloat(*v.ShadowWinRate) + "% win"
		}
		line += ", P&L " + v.ShadowPnL
		sb += line
	}
	return sb
}

// StartLoop is part of the Go port of the swing-trade schedulers.
func (s *WeeklyStrategyDigestScheduler) StartLoop(ctx context.Context) {
	runOnSchedule(ctx, waitForDeadline, s.now, NextWeeklyStrategyDigestRun, func(ctx context.Context) { s.SendWeeklyStrategyDigest(ctx) })
}

// NextWeeklyStrategyDigestRun is part of the Go port of the swing-trade schedulers.
func NextWeeklyStrategyDigestRun(now time.Time) time.Time {
	return nextSundayFire(now, 18, 30)
}

func dateOnlyIST(t time.Time) time.Time {
	local := t.In(ist)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, ist)
}

func nextSundayFire(now time.Time, hour, minute int) time.Time {
	local := now.In(ist)
	next := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, ist)
	for !next.After(now) || next.Weekday() != time.Sunday {
		next = next.AddDate(0, 0, 1)
		next = time.Date(next.Year(), next.Month(), next.Day(), hour, minute, 0, 0, ist)
	}
	return next
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}
