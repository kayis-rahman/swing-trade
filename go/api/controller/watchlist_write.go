package controller

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

// WatchlistWriteStore supports watchlist mutations.
type WatchlistWriteStore interface {
	AddWatchlist(ctx context.Context, symbol, name, exchange string) (data.WatchlistEntry, error)
	RemoveWatchlist(ctx context.Context, symbol string) error
	ToggleWatchlist(ctx context.Context, symbol string, activate bool) (data.WatchlistEntry, error)
}

// RegisterWatchlistWriteRoutes registers watchlist mutation endpoints.
func RegisterWatchlistWriteRoutes(mux *http.ServeMux, writer WatchlistWriteStore, logger *slog.Logger, clk clock.Clock) {
	if writer == nil {
		return
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nilWriter{}, nil))
	}
	mux.HandleFunc("POST /api/watchlist", func(w http.ResponseWriter, r *http.Request) {
		symbol := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("symbol")))
		name := r.URL.Query().Get("name")
		exchange := r.URL.Query().Get("exchange")
		if exchange == "" {
			exchange = "NSE"
		}
		if symbol == "" || len(symbol) > 10 || len(name) > 255 || len(exchange) > 5 {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{Error: stringValuePtr("Symbol, name, or exchange exceeds its allowed length"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		entry, err := writer.AddWatchlist(r.Context(), symbol, name, exchange)
		if err != nil {
			logger.ErrorContext(r.Context(), "watchlist add failed", "module", "api", "error", err)
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{Error: stringValuePtr(err.Error()), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: entry, Timestamp: javaLocalDateTime(clk.Now())})
	})
	mux.HandleFunc("DELETE /api/watchlist/{symbol}", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.PathValue("symbol")
		if err := writer.RemoveWatchlist(r.Context(), symbol); err != nil {
			logger.ErrorContext(r.Context(), "watchlist remove failed", "module", "api", "error", err)
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{Error: stringValuePtr("Unable to remove watchlist entry"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: "Removed " + symbol + " from watchlist", Timestamp: javaLocalDateTime(clk.Now())})
	})
	mux.HandleFunc("PATCH /api/watchlist/{symbol}/toggle", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.PathValue("symbol")
		activate := r.URL.Query().Get("activate") == "true"
		entry, err := writer.ToggleWatchlist(r.Context(), symbol, activate)
		if err != nil {
			logger.ErrorContext(r.Context(), "watchlist toggle failed", "module", "api", "error", err)
			writeResponseJSON(w, http.StatusNotFound, apiEnvelope{Error: stringValuePtr("not found"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: entry, Timestamp: javaLocalDateTime(clk.Now())})
	})
}

type nilWriter struct{}

func (nilWriter) Write(p []byte) (int, error) { return len(p), nil }
