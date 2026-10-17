package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
)

type window struct {
	started time.Time
	count   int
}

// RateLimiter applies a per-client fixed-window request limit.
type RateLimiter struct {
	limit int
	clock coreclock.Clock
	mu    sync.Mutex
	state map[string]window
	hits  prometheus.Counter
}

// NewRateLimiter creates a limiter without an optional Prometheus counter.
func NewRateLimiter(limit int, clock coreclock.Clock) *RateLimiter {
	return NewRateLimiterWithCounter(limit, clock, nil)
}

// NewRateLimiterWithCounter creates a limiter that counts rejected requests when hits is non-nil.
func NewRateLimiterWithCounter(limit int, clock coreclock.Clock, hits prometheus.Counter) *RateLimiter {
	if clock == nil {
		clock = coreclock.System{}
	}
	if limit < 1 {
		limit = 1
	}
	return &RateLimiter{limit: limit, clock: clock, state: make(map[string]window), hits: hits}
}

// Middleware wraps next with the fixed-window limiter.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || isAdminPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		client := remoteIP(r.RemoteAddr)
		now := l.clock.Now()
		l.mu.Lock()
		current, exists := l.state[client]
		if !exists || !now.Before(current.started.Add(time.Minute)) {
			current = window{started: now, count: 1}
		} else {
			current.count++
		}
		l.state[client] = current
		l.mu.Unlock()

		remaining := l.limit - current.count
		if remaining < 0 {
			remaining = 0
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(l.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if current.count > l.limit {
			if l.hits != nil {
				l.hits.Inc()
			}
			retryAfter := int64(current.started.Add(time.Minute).Sub(now).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func remoteIP(address string) string {
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return strings.TrimSpace(address)
}

func isAdminPath(path string) bool {
	return path == "/api/admin" || strings.HasPrefix(path, "/api/admin/")
}
