package controller

import (
	"log/slog"
	"net/http"
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
