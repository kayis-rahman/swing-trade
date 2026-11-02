package llm

import (
	"errors"
	"sync"
	"time"
)

// NewsCircuitBreaker provides a minimal closed/open/half-open circuit breaker.
type NewsCircuitBreaker struct {
	failures int
	threshold int
	openUntil time.Time
	mu       sync.Mutex
}

func NewNewsCircuitBreaker(threshold int) *NewsCircuitBreaker {
	if threshold <= 0 {
		threshold = 3
	}
	return &NewsCircuitBreaker{threshold: threshold}
}

var ErrCircuitOpen = errors.New("news circuit breaker open")

func (cb *NewsCircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.openUntil.IsZero() {
		return nil
	}
	if time.Now().Before(cb.openUntil) {
		return ErrCircuitOpen
	}
	cb.openUntil = time.Time{}
	cb.failures = 0
	return nil
}

func (cb *NewsCircuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.openUntil = time.Time{}
}

func (cb *NewsCircuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures >= cb.threshold {
		cb.openUntil = time.Now().Add(30 * time.Second)
	}
}
