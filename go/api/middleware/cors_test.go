package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsConfiguredOriginAndRejectsOtherOrigins(t *testing.T) {
	handler := CORS([]string{"https://dashboard.example"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, test := range []struct {
		origin string
		want   string
	}{
		{origin: "https://dashboard.example", want: "https://dashboard.example"},
		{origin: "https://other.example", want: ""},
	} {
		r := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != test.want {
			t.Errorf("origin %q allowed as %q, want %q", test.origin, got, test.want)
		}
	}
}

func TestCORSPreflightDoesNotCallHandler(t *testing.T) {
	called := false
	handler := CORS([]string{"https://dashboard.example"})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	r := httptest.NewRequest(http.MethodOptions, "/api/admin/health", nil)
	r.Header.Set("Origin", "https://dashboard.example")
	r.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if called || w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("preflight: called=%t status=%d headers=%v", called, w.Code, w.Header())
	}
}
