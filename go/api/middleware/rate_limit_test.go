package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

func TestRateLimiterUsesFixedWindowAndExcludesAdminAndPreflight(t *testing.T) {
	clock := &fixedClock{now: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	limit := NewRateLimiter(1, clock)
	handler := limit.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	request := func(path, method string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if got := request("/api/health", http.MethodGet); got.Code != http.StatusOK || got.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("first public request: %d %v", got.Code, got.Header())
	}
	if got := request("/api/health", http.MethodGet); got.Code != http.StatusTooManyRequests || got.Header().Get("Retry-After") == "" {
		t.Fatalf("limited request: %d %v", got.Code, got.Header())
	}
	if got := request("/api/admin/health", http.MethodGet); got.Code != http.StatusOK {
		t.Fatalf("admin request was limited: %d", got.Code)
	}
	if got := request("/api/health", http.MethodOptions); got.Code != http.StatusOK {
		t.Fatalf("preflight request was limited: %d", got.Code)
	}
	clock.now = clock.now.Add(time.Minute)
	if got := request("/api/health", http.MethodGet); got.Code != http.StatusOK {
		t.Fatalf("new fixed window was limited: %d", got.Code)
	}
}

func TestRateLimiterBoundedMemory(t *testing.T) {
	clock := &fixedClock{now: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	limit := NewRateLimiter(100, clock)
	handler := limit.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	httpCalls := 0
	for i := 0; i < 50; i++ {
		r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		// Use unique IPs that won't expire.
		r.RemoteAddr = net.ParseIP(t.Name()).String() + ":" + strconv.Itoa(i)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		httpCalls++
		_ = httpCalls
	}
	if len(limit.state) == 0 {
		t.Fatal("expected some entries after 50 requests")
	}
	// Advance time so old entries expire.
	clock.now = clock.now.Add(2 * time.Minute)
	// Additional requests trigger cleanup.
	for i := 50; i < 70; i++ {
		r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		r.RemoteAddr = net.ParseIP(t.Name()).String() + ":" + strconv.Itoa(i)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
	if len(limit.state) >= 50 {
		t.Fatalf("expected expired entries to be cleaned up, got %d entries", len(limit.state))
	}
}
