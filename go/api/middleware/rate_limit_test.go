package middleware

import (
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

func TestRateLimiterBoundsTrackedClientsAndPreservesWindows(t *testing.T) {
	clock := &fixedClock{now: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)}
	limiter := NewRateLimiter(2, clock)
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	request := func(client int) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		r.RemoteAddr = "client-" + strconv.Itoa(client) + ":1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}

	for i := 0; i < maxTrackedClients; i++ {
		if got := request(i); got.Code != http.StatusOK {
			t.Fatalf("initial request for client %d: %d", i, got.Code)
		}
	}
	if got := len(limiter.state); got != maxTrackedClients {
		t.Fatalf("tracked %d clients, want %d", got, maxTrackedClients)
	}
	if got := request(maxTrackedClients); got.Code != http.StatusTooManyRequests || got.Header().Get("Retry-After") == "" {
		t.Fatalf("untracked client at capacity: %d %v", got.Code, got.Header())
	}
	if got := len(limiter.state); got != maxTrackedClients {
		t.Fatalf("capacity rejection grew state to %d", got)
	}
	if got := request(0); got.Code != http.StatusOK {
		t.Fatalf("retained client second request: %d", got.Code)
	}
	if got := request(0); got.Code != http.StatusTooManyRequests {
		t.Fatalf("retained client exceeded limit: %d", got.Code)
	}

	clock.now = clock.now.Add(time.Minute)
	if got := request(maxTrackedClients); got.Code != http.StatusOK {
		t.Fatalf("client request after window expiry: %d", got.Code)
	}
	if got := len(limiter.state); got != 1 {
		t.Fatalf("expired entries were not reclaimed before insertion: %d remain", got)
	}
}
