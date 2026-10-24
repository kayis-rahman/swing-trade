package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kayis-rahman/swing-trade/go/api/controller"
	"github.com/kayis-rahman/swing-trade/go/api/middleware"
	coreclock "github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/core/config"
	corelog "github.com/kayis-rahman/swing-trade/go/core/log"
	"github.com/kayis-rahman/swing-trade/go/data"
)

type healthResponse struct {
	Status     string                     `json:"status"`
	Components map[string]healthComponent `json:"components,omitempty"`
}

type healthComponent struct {
	Status string `json:"status"`
}

// NewHandler builds an HTTP handler without a configured data store or logger.
func NewHandler(cfg config.Config) (http.Handler, error) {
	return NewHandlerWithLogger(cfg, corelog.New(io.Discard, slog.LevelInfo, "api"), coreclock.System{})
}

// NewHandlerWithLogger builds an HTTP handler with injected logging and time but no data store.
func NewHandlerWithLogger(cfg config.Config, logger *slog.Logger, clock coreclock.Clock) (http.Handler, error) {
	return NewHandlerWithWatchlist(cfg, logger, clock, nil)
}

// NewHandlerWithWatchlist builds the HTTP handler with an optional read-only watchlist store.
func NewHandlerWithWatchlist(cfg config.Config, logger *slog.Logger, clock coreclock.Clock, watchlist data.WatchlistReadStore) (http.Handler, error) {
	if clock == nil {
		clock = coreclock.System{}
	}
	if logger == nil {
		logger = corelog.New(io.Discard, slog.LevelInfo, "api")
	}
	registry := prometheus.NewRegistry()
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{"application": "swing-trade-api"}, registry)
	requests := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_server_requests_seconds",
		Help:    "HTTP server request duration in seconds.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "uri", "status", "outcome", "exception"})
	if err := registerer.Register(requests); err != nil {
		return nil, err
	}
	rateLimitHits := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "api_rate_limit_hits_total",
		Help: "Number of public API requests rejected by the per-IP rate limit.",
	})
	if err := registerer.Register(rateLimitHits); err != nil {
		return nil, err
	}
	apiRequests := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "api_request_duration_seconds",
		Help: "API filter request duration in seconds.",
	}, []string{"method", "status"})
	if err := registerer.Register(apiRequests); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	controller.RegisterHealthRoutes(mux, watchlist != nil, clock, clock.Now())
	controller.RegisterWatchlistRoutes(mux, watchlist, logger, clock)
	mux.HandleFunc("GET /api/admin/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "UP"})
	})
	mux.HandleFunc("GET /actuator/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status: "UP",
			Components: map[string]healthComponent{
				"livenessState":  {Status: "UP"},
				"readinessState": {Status: "UP"},
			},
		})
	})
	mux.HandleFunc("GET /actuator/health/liveness", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "UP"})
	})
	mux.HandleFunc("GET /actuator/health/readiness", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "UP"})
	})
	mux.Handle("GET /actuator/prometheus", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	var handler http.Handler = mux
	handler = middleware.AdminAPIKey(cfg.APIKeyEnabled, cfg.APIKey)(handler)
	handler = middleware.NewRateLimiterWithCounter(cfg.PublicRateLimitPerMinute, clock, rateLimitHits).Middleware(handler)
	handler = middleware.Recover(logger)(handler)
	handler = middleware.RequestMetrics(clock, requests, apiRequests)(handler)
	handler = middleware.Trace(logger, clock)(handler)
	handler = middleware.CORS(cfg.CORSAllowedOrigins)(handler)
	return handler, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
