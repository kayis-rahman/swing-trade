package middleware

import (
	"net/http"
	"net/http/httptest"
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
