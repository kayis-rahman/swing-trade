package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

// SignalEntry is the read model for the latest signal per symbol, serialized
// by the signals API. Numeric values keep their database text form so the
// response preserves Java's BigDecimal scale instead of round-tripping through
// float64.
type SignalEntry struct {
	ID                 int64
	Symbol             string
	Date               time.Time
	SignalType         string
	Confidence         *string
	Reasoning          *string
	EntryPrice         *string
	StopLoss           *string
	Target             *string
	RiskRewardRatio    *string
	Indicators         *string
	GeneratedAt        *time.Time
	Strategy           string
	StrategyVersion    int32
	SentimentScore     *string
	SentimentReasoning *string
}

// SignalReadStore exposes read-only signal queries to the API layer.
type SignalReadStore interface {
	ListLatestSignals(context.Context) ([]SignalEntry, error)
}

// SignalReader executes generated read-only signal queries.
type SignalReader struct {
	queries *sqlc.Queries
}

// NewSignalReader builds a signal reader on a pgx-compatible database handle.
func NewSignalReader(db sqlc.DBTX) *SignalReader {
	return &SignalReader{queries: sqlc.New(db)}
}

// ListLatestSignals returns one signal per symbol: the newest by date, then
// creation time, then id. This mirrors Java's findLatestSignalPerSymbol window
// query so the dashboard sees the same rows.
func (r *SignalReader) ListLatestSignals(ctx context.Context) ([]SignalEntry, error) {
	rows, err := r.queries.ListLatestSignals(ctx)
	if err != nil {
		return nil, fmt.Errorf("list latest signals: %w", err)
	}
	return mapSignalRows(rows), nil
}

func mapSignalRows(rows []sqlc.Signal) []SignalEntry {
	entries := make([]SignalEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, SignalEntry{
			ID:                 row.ID,
			Symbol:             row.Symbol,
			Date:               row.Date.Time,
			SignalType:         row.SignalType,
			Confidence:         fromNumeric(row.ConfidenceScore),
			Reasoning:          nullableText(row.Reasoning),
			EntryPrice:         fromNumeric(row.EntryPrice),
			StopLoss:           fromNumeric(row.StopLoss),
			Target:             fromNumeric(row.Target),
			RiskRewardRatio:    fromNumeric(row.RiskReward),
			Indicators:         nullableText(row.Indicators),
			GeneratedAt:        nullableDate(row.GeneratedAt),
			Strategy:           row.Strategy,
			StrategyVersion:    row.StrategyVersion,
			SentimentScore:     nullableText(row.SentimentScore),
			SentimentReasoning: nullableText(row.SentimentReasoning),
		})
	}
	return entries
}

func nullableDate(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
