package data

import (
	"context"
	"sync"
	"time"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
)

// RequestPacer serializes requests to enforce a provider-wide requests-per-minute quota.
type RequestPacer struct {
	mu        sync.Mutex
	clock     coreclock.Clock
	interval  time.Duration
	unlimited bool
	next      time.Time
}

// NewRequestPacer creates a shared pacer. Nonpositive quotas disable pacing.
func NewRequestPacer(requestsPerMinute int64, clk coreclock.Clock) *RequestPacer {
	if clk == nil {
		clk = coreclock.System{}
	}
	if requestsPerMinute <= 0 {
		return &RequestPacer{clock: clk, unlimited: true}
	}
	interval := time.Minute / time.Duration(requestsPerMinute)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	return &RequestPacer{clock: clk, interval: interval}
}

// Acquire waits for the next permit, or returns when the context is canceled.
func (p *RequestPacer) Acquire(ctx context.Context) error {
	if p.unlimited {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	wait := p.next.Sub(p.clock.Now())
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	p.next = p.clock.Now().Add(p.interval)
	return nil
}
