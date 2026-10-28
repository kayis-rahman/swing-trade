package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	"github.com/kayis-rahman/swing-trade/go/data"
)

// signalResponse mirrors Java's SignalResponse DTO field-for-field. The
// dashboard consumes GET /api/signals/latest as a direct array, so this route
// deliberately does not wrap the payload in the watchlist API envelope.
type signalResponse struct {
	ID                 int64        `json:"id"`
	Symbol             string       `json:"symbol"`
	Date               string       `json:"date"`
	SignalType         string       `json:"signalType"`
	Confidence         *json.Number `json:"confidence"`
	Reasoning          *string      `json:"reasoning"`
	EntryPrice         *json.Number `json:"entryPrice"`
	StopLoss           *json.Number `json:"stopLoss"`
	Target             *json.Number `json:"target"`
	RiskRewardRatio    *json.Number `json:"riskRewardRatio"`
	Indicators         []string     `json:"indicators"`
	GeneratedAt        *string      `json:"generatedAt"`
	Strategy           *string      `json:"strategy"`
	StrategyVersion    *int32       `json:"strategyVersion"`
	SentimentScore     *string      `json:"sentimentScore"`
	SentimentReasoning *string      `json:"sentimentReasoning"`
}

// RegisterSignalRoutes adds the Java-compatible read-only signal endpoints.
func RegisterSignalRoutes(mux *http.ServeMux, store data.SignalReadStore, logger *slog.Logger, clk clock.Clock) {
	mux.HandleFunc("GET /api/signals/latest", func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			writeResponseJSON(w, http.StatusServiceUnavailable, apiEnvelope{
				Data: nil, Error: stringValuePtr("signal data store is unavailable"),
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		rows, err := store.ListLatestSignals(r.Context())
		if err != nil {
			logger.ErrorContext(r.Context(), "signal read failed", "module", "api")
			writeResponseJSON(w, http.StatusInternalServerError, apiEnvelope{
				Data: nil, Error: stringValuePtr("Unable to read signals"),
				Timestamp: javaLocalDateTime(clk.Now()),
			})
			return
		}
		response := make([]signalResponse, 0, len(rows))
		for _, row := range rows {
			response = append(response, toSignalResponse(row))
		}
		writeResponseJSON(w, http.StatusOK, response)
	})
}

func toSignalResponse(row data.SignalEntry) signalResponse {
	strategy := row.Strategy
	strategyVersion := row.StrategyVersion
	return signalResponse{
		ID:                 row.ID,
		Symbol:             row.Symbol,
		Date:               javaLocalDate(row.Date),
		SignalType:         row.SignalType,
		Confidence:         optionalJSONNumber(row.Confidence),
		Reasoning:          row.Reasoning,
		EntryPrice:         optionalJSONNumber(row.EntryPrice),
		StopLoss:           optionalJSONNumber(row.StopLoss),
		Target:             optionalJSONNumber(row.Target),
		RiskRewardRatio:    optionalJSONNumber(row.RiskRewardRatio),
		Indicators:         parseIndicators(row.Indicators),
		GeneratedAt:        formatOptionalDate(row.GeneratedAt),
		Strategy:           &strategy,
		StrategyVersion:    &strategyVersion,
		SentimentScore:     row.SentimentScore,
		SentimentReasoning: row.SentimentReasoning,
	}
}

// parseIndicators mirrors Java's SignalResponse.parseIndicators: null or empty
// becomes an empty list, otherwise the comma-separated value is split as-is.
func parseIndicators(indicators *string) []string {
	if indicators == nil || *indicators == "" {
		return []string{}
	}
	return strings.Split(*indicators, ",")
}

func optionalJSONNumber(value *string) *json.Number {
	if value == nil {
		return nil
	}
	number := json.Number(*value)
	return &number
}

func formatOptionalDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := javaLocalDate(*value)
	return &formatted
}

func javaLocalDate(value time.Time) string {
	return value.Format("2006-01-02")
}
