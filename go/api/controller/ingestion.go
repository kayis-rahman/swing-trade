package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

type ingestionStatusResponse struct {
	EarliestCandleDate *string `json:"earliestCandleDate"`
	CandleCount        int64   `json:"candleCount"`
	Exchange           *string `json:"exchange"`
	HasData            bool    `json:"hasData"`
	IsActive           *bool   `json:"isActive"`
	LastCandleDate     *string `json:"lastCandleDate"`
	LastSyncedAt       *string `json:"lastSyncedAt"`
	DataQuality        string  `json:"dataQuality"`
	Name               *string `json:"name"`
	Symbol             string  `json:"symbol"`
}

// RegisterIngestionRoutes adds the read-only ingestion status endpoint with
// the Java ApiResponse envelope.
func RegisterIngestionRoutes(mux *http.ServeMux, status data.IngestionStatusLookup, logger *slog.Logger, clk clock.Clock) {
	mux.HandleFunc("GET /api/ingestion/status", func(w http.ResponseWriter, r *http.Request) {
		if status == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
				Data: nil, Error: stringValuePtr("ingestion status store is unavailable"),
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		entries, err := status.IngestionStatus(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "ingestion status read failed", "module", "api")
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
				Data: nil, Error: stringValuePtr("Failed to get status"),
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		response := make([]ingestionStatusResponse, 0, len(entries))
		for _, entry := range entries {
			response = append(response, ingestionStatusResponse{
				Symbol:             entry.Symbol,
				Name:               entry.Name,
				Exchange:           entry.Exchange,
				IsActive:           entry.IsActive,
				CandleCount:        entry.CandleCount,
				LastCandleDate:     formatDateOnly(entry.LastCandleDate),
				EarliestCandleDate: formatDateOnly(entry.EarliestCandleDate),
				LastSyncedAt:       formatOptionalTimestamp(entry.LastSyncedAt),
				HasData:            entry.HasData,
				DataQuality:        entry.DataQuality,
			})
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: response, Error: nil, Success: true,
			Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
}

// formatDateOnly renders a stored session date like Java's LocalDate JSON
// ("2006-01-02").
func formatDateOnly(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}

type validationErrorResponse struct {
	Status    int    `json:"status"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Path      string `json:"path"`
}

// RegisterIngestionReconcileRoute adds the read-only reconciliation preview
// endpoint, including Java's VALIDATION_ERROR envelope for bad ranges.
func RegisterIngestionReconcileRoute(
	mux *http.ServeMux,
	watchlist data.WatchlistReadStore,
	reconciler data.ReconciliationPreview,
	logger *slog.Logger,
	clk clock.Clock,
) {
	mux.HandleFunc("GET /api/ingestion/reconcile", func(w http.ResponseWriter, r *http.Request) {
		if reconciler == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
				Data: nil, Error: stringValuePtr("reconciliation preview is unavailable"),
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		query := r.URL.Query()
		from, fromErr := time.Parse("2006-01-02", query.Get("from"))
		to, toErr := time.Parse("2006-01-02", query.Get("to"))
		if fromErr != nil || toErr != nil {
			writeValidationError(w, r, clk, "invalid reconciliation date: from and to must be yyyy-mm-dd")
			return
		}
		if to.Before(from) || from.AddDate(0, 0, 10).Before(to) {
			writeValidationError(w, r, clk, data.ErrInvalidReconciliationRange.Error())
			return
		}
		symbol := strings.ToUpper(strings.TrimSpace(query.Get("symbol")))
		if symbol != "" {
			report, err := reconciler.PreviewReconciliation(r.Context(), symbol, from, to)
			if errors.Is(err, data.ErrInvalidReconciliationRange) {
				writeValidationError(w, r, clk, err.Error())
				return
			}
			if err != nil {
				logger.ErrorContext(r.Context(), "reconciliation preview failed", "module", "api")
				writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
					Data: nil, Error: stringValuePtr("Failed to reconcile"),
					Timestamp: javaLocalDateTime(clk.Now()),
				})
				return
			}
			writeResponseJSON(w, http.StatusOK, apiEnvelope{
				Data: reconciliationReportJSON(report), Error: nil, Success: true,
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		if watchlist == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
				Data: nil, Error: stringValuePtr("reconciliation preview is unavailable"),
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		entries, err := watchlist.ListActiveWatchlist(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "reconciliation watchlist read failed", "module", "api")
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
				Data: nil, Error: stringValuePtr("Failed to reconcile"),
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		reports := make([]map[string]any, 0, len(entries))
		overall := "COMPLETED"
		for _, entry := range entries {
			if entry.Symbol == "" {
				continue
			}
			report, err := reconciler.PreviewReconciliation(r.Context(), entry.Symbol, from, to)
			if err != nil {
				logger.ErrorContext(r.Context(), "reconciliation preview failed", "module", "api", "symbol", entry.Symbol)
				writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
					Data: nil, Error: stringValuePtr("Failed to reconcile"),
					Timestamp: javaLocalDateTime(clk.Now()),
				})
				return
			}
			reports = append(reports, reconciliationReportJSON(report))
			if overall == "COMPLETED" && report.Status != "COMPLETED" {
				overall = report.Status
			}
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{
			Data: map[string]any{"reports": reports, "status": overall}, Error: nil, Success: true,
			Timestamp: javaLocalDateTime(clk.Now()),
		})
	})
}

func writeValidationError(w http.ResponseWriter, r *http.Request, clk clock.Clock, message string) {
	writeResponseJSON(w, http.StatusBadRequest, validationErrorResponse{
		Status: http.StatusBadRequest, Error: "VALIDATION_ERROR", Message: message,
		Timestamp: javaLocalDateTime(clk.Now()), Path: r.URL.RequestURI(),
	})
}

// reconciliationReportJSON renders one report with Java's LocalDate date
// strings; CALENDAR_INCOMPLETE responses carry only the four keys Java's
// immutable Map.of allows.
func reconciliationReportJSON(report data.ReconciliationReport) map[string]any {
	from := report.From.Format(time.DateOnly)
	to := report.To.Format(time.DateOnly)
	if report.Status == "CALENDAR_INCOMPLETE" {
		return map[string]any{"symbol": report.Symbol, "from": from, "to": to, "status": report.Status}
	}
	nonTrading := make([]string, 0, len(report.NonTradingDayDates))
	for _, date := range report.NonTradingDayDates {
		nonTrading = append(nonTrading, date.Format(time.DateOnly))
	}
	duplicates := make([]map[string]any, 0, len(report.DuplicateGroups))
	for _, group := range report.DuplicateGroups {
		duplicates = append(duplicates, map[string]any{
			"date": group.Date.Format(time.DateOnly), "ids": group.IDs, "count": group.Count,
		})
	}
	return map[string]any{
		"symbol":               report.Symbol,
		"from":                 from,
		"to":                   to,
		"expected":             formatDateList(report.Expected),
		"storedDates":          formatDateList(report.StoredDates),
		"missing":              formatDateList(report.Missing),
		"nonTradingDayDates":   nonTrading,
		"invalidStored":        nonTrading,
		"duplicateGroups":      duplicates,
		"sourceIncomplete":     formatDateList(report.SourceIncomplete),
		"inserted":             report.Inserted,
		"invalidStoredRemoved": report.InvalidStoredRemoved,
		"qualityStatus":        report.QualityStatus,
		"earliestDate":         formatDateOnly(report.EarliestDate),
		"latestDate":           formatDateOnly(report.LatestDate),
		"count":                report.Count,
		"status":               report.Status,
	}
}

func formatDateList(dates []time.Time) []string {
	formatted := make([]string, 0, len(dates))
	for _, date := range dates {
		formatted = append(formatted, date.Format(time.DateOnly))
	}
	return formatted
}
