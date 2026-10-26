package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoverSanitizesPanics(t *testing.T) {
	handler := Recover(slog.New(slog.NewTextHandler(nilWriter{}, nil)))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("internal secret")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("response leaked panic or had wrong status: %d %q", response.Code, response.Body.String())
	}
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) { return len(p), nil }
