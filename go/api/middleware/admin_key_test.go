package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAPIKeyFailsClosedAndDoesNotLeakKey(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := AdminAPIKey(true, "test-only-key")(next)
	for _, key := range []string{"", "wrong-key"} {
		r := httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
		r.Header.Set("X-API-Key", key)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != "X-API-Key" {
			t.Fatalf("unauthorized response for key %q: %d %v", key, w.Code, w.Header())
		}
		if w.Body.Len() != 0 {
			t.Fatalf("unauthorized response leaked body: %q", w.Body.String())
		}
	}
	r := httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
	r.Header.Set("X-API-Key", "test-only-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("authorized status = %d", w.Code)
	}
}

func TestAdminAPIKeyFailsClosedWhenNoConfiguredKey(t *testing.T) {
	handler := AdminAPIKey(true, "")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	r := httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
	r.Header.Set("X-API-Key", "anything")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("empty configured key returned %d", w.Code)
	}
}
