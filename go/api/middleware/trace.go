package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
)

type traceIDKey struct{}

// TraceID returns the request trace identifier stored in ctx, if present.
func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(traceIDKey{}).(string)
	return id
}

// Trace adds a trace identifier to each request and logs completion metadata.
func Trace(logger *slog.Logger, clock coreclock.Clock) func(http.Handler) http.Handler {
	if clock == nil {
		clock = coreclock.System{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var raw [4]byte
			if _, err := rand.Read(raw[:]); err != nil {
				http.Error(w, `{"error":"request initialization failed"}`, http.StatusInternalServerError)
				return
			}
			id := hex.EncodeToString(raw[:])
			ctx := context.WithValue(r.Context(), traceIDKey{}, id)
			started := clock.Now()
			wrapped := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(wrapped, r.WithContext(ctx))
			if logger != nil {
				attrs := []any{
					"traceId", id,
					"method", r.Method,
					"path", r.URL.Path,
					"status", wrapped.status,
					"durationMs", clock.Now().Sub(started).Milliseconds(),
				}
				if symbol := requestSymbol(r); symbol != "" {
					attrs = append(attrs, "symbol", symbol)
				}
				logger.InfoContext(ctx, "HTTP request", attrs...)
			}
		})
	}
}

func requestSymbol(r *http.Request) string {
	query := r.URL.Query()
	symbol, exists := query["symbol"]
	if !exists {
		symbol, exists = query["code"]
	}
	if !exists && (strings.Contains(r.URL.Path, "/signals/") || strings.Contains(r.URL.Path, "/positions/")) {
		return r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	}
	if !exists || len(symbol) == 0 || strings.TrimSpace(symbol[0]) == "" {
		return ""
	}
	return symbol[0]
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}
