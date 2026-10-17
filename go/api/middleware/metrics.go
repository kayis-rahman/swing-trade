package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
)

// RequestMetrics records the HTTP server histogram and API filter summary.
func RequestMetrics(clock coreclock.Clock, requests *prometheus.HistogramVec, apiRequests *prometheus.SummaryVec) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := clock.Now()
			wrapped := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(wrapped, r)
			status := wrapped.status
			if status == 0 {
				status = http.StatusOK
			}
			uri := r.Pattern
			if _, pattern, found := strings.Cut(uri, " "); found {
				uri = pattern
			}
			if uri == "" {
				uri = knownRoute(r.URL.Path)
			}
			outcome := "SUCCESS"
			if status >= http.StatusInternalServerError {
				outcome = "SERVER_ERROR"
			} else if status >= http.StatusBadRequest {
				outcome = "CLIENT_ERROR"
			}
			duration := clock.Now().Sub(started).Seconds()
			requests.WithLabelValues(r.Method, uri, strconv.Itoa(status), outcome, "").Observe(duration)
			apiRequests.WithLabelValues(r.Method, strconv.Itoa(status)).Observe(duration)
		})
	}
}

func knownRoute(path string) string {
	switch path {
	case "/api/health", "/api/admin/health", "/actuator/health", "/actuator/health/liveness", "/actuator/health/readiness", "/actuator/prometheus":
		return path
	default:
		return "unmatched"
	}
}
