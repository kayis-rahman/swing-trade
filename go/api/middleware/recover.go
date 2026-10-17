package middleware

import (
	"io"
	"log/slog"
	"net/http"
)

// Recover converts request-handler panics to sanitized HTTP 500 responses.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if logger != nil {
						logger.ErrorContext(r.Context(), "request panic recovered")
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = io.WriteString(w, `{"error":"internal server error"}`)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
