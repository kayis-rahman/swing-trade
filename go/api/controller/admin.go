package controller

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
)

// AdminSymbolMaster refreshes and counts the Fyers symbol master table.
type AdminSymbolMaster interface {
	Refresh(ctx context.Context) (int, error)
	Count(ctx context.Context) (int, error)
}

// DataQualityValidator validates persisted OHLCV data for a symbol.
type DataQualityValidator interface {
	ValidateDataQuality(ctx context.Context, symbol string, from, to time.Time) (any, error)
}

// RegisterAdminRoutes registers the slice-E admin endpoints for symbols and data validation.
func RegisterAdminRoutes(mux *http.ServeMux, symbols AdminSymbolMaster, validator DataQualityValidator, logger *slog.Logger, clk clock.Clock) {
	if mux == nil {
		return
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	mux.HandleFunc("POST /api/admin/symbols/refresh", func(w http.ResponseWriter, r *http.Request) {
		if symbols == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{Error: stringValuePtr("symbol master is unavailable"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		count, err := symbols.Refresh(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "symbol refresh failed", "module", "api", "error", err)
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{Error: stringValuePtr("Failed to refresh symbol master: " + err.Error()), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: map[string]any{"count": count}, Timestamp: javaLocalDateTime(clk.Now())})
	})
	mux.HandleFunc("GET /api/admin/symbols/status", func(w http.ResponseWriter, r *http.Request) {
		if symbols == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{Error: stringValuePtr("symbol master is unavailable"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		count, err := symbols.Count(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "symbol status failed", "module", "api", "error", err)
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{Error: stringValuePtr("Failed to get symbol master status: " + err.Error()), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: map[string]any{"count": count}, Timestamp: javaLocalDateTime(clk.Now())})
	})
	mux.HandleFunc("POST /api/admin/data/validate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Symbol   string `json:"symbol"`
			FromDate string `json:"fromDate"`
			ToDate   string `json:"toDate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{Error: stringValuePtr("invalid request body"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		symbol := strings.TrimSpace(strings.ToUpper(req.Symbol))
		if symbol == "" {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{Error: stringValuePtr("symbol, fromDate, and toDate are required and the date range must be ordered"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		from, errFrom := time.Parse("2006-01-02", req.FromDate)
		to, errTo := time.Parse("2006-01-02", req.ToDate)
		if errFrom != nil || errTo != nil || from.After(to) {
			writeResponseJSON(w, http.StatusBadRequest, apiEnvelope{Error: stringValuePtr("symbol, fromDate, and toDate are required and the date range must be ordered"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		if validator == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{Error: stringValuePtr("data validation is unavailable"), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		report, err := validator.ValidateDataQuality(r.Context(), symbol, from, to)
		if err != nil {
			logger.ErrorContext(r.Context(), "data validation failed", "module", "api", "error", err)
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{Error: stringValuePtr("Failed to validate data quality: " + err.Error()), Timestamp: javaLocalDateTime(clk.Now())})
			return
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Success: true, Data: report, Timestamp: javaLocalDateTime(clk.Now())})
	})
}
