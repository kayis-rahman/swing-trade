package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// SectorDigestProvider is part of the Go port of the swing-trade schedulers.
type SectorDigestProvider interface {
	GenerateSectorDigestForLastWeek() string
}

// WeeklySectorDigestScheduler is part of the Go port of the swing-trade schedulers.
type WeeklySectorDigestScheduler struct {
	sentiment SectorDigestProvider
	discord   EmbedSender
	enabled   bool
	now       func() time.Time
	logger    *slog.Logger
}

// NewWeeklySectorDigestScheduler is part of the Go port of the swing-trade schedulers.
func NewWeeklySectorDigestScheduler(
	sentiment SectorDigestProvider,
	discord EmbedSender,
	enabled bool,
	logger *slog.Logger,
	now func() time.Time,
) (*WeeklySectorDigestScheduler, error) {
	if logger == nil || now == nil || sentiment == nil || discord == nil {
		return nil, fmt.Errorf("sentiment, discord, logger, and clock are required")
	}
	return &WeeklySectorDigestScheduler{sentiment: sentiment, discord: discord, enabled: enabled, logger: logger, now: now}, nil
}

// Enabled is part of the Go port of the swing-trade schedulers.
func (s *WeeklySectorDigestScheduler) Enabled() bool { return s.enabled }

// SendWeeklySectorDigest is part of the Go port of the swing-trade schedulers.
func (s *WeeklySectorDigestScheduler) SendWeeklySectorDigest(ctx context.Context) {
	s.sendWeeklySectorDigest()
}

func (s *WeeklySectorDigestScheduler) sendWeeklySectorDigest() {
	if !s.enabled {
		if s.logger != nil {
			s.logger.Debug("weekly sector digest disabled")
		}
		return
	}
	if s.logger != nil {
		s.logger.Info("Starting weekly sector digest delivery to Discord")
	}
	digest := s.sentiment.GenerateSectorDigestForLastWeek()
	sent := s.discord.SendEmbed("Weekly Sector Digest", digest, 0x5865F2)
	if sent {
		if s.logger != nil {
			s.logger.Info("weekly sector digest sent successfully")
		}
	} else if s.logger != nil {
		s.logger.Warn("weekly sector digest delivery returned false")
	}
}

// Send is part of the Go port of the swing-trade schedulers.
func (s *WeeklySectorDigestScheduler) Send(ctx context.Context) { s.SendWeeklySectorDigest(ctx) }

// StartLoop is part of the Go port of the swing-trade schedulers.
func (s *WeeklySectorDigestScheduler) StartLoop(ctx context.Context) {
	runOnSchedule(ctx, waitForDeadline, s.now, NextWeeklySectorDigestRun, func(ctx context.Context) { s.SendWeeklySectorDigest(ctx) })
}

// NextWeeklySectorDigestRun is part of the Go port of the swing-trade schedulers.
func NextWeeklySectorDigestRun(now time.Time) time.Time {
	return nextSundayFire(now, 18, 0)
}
