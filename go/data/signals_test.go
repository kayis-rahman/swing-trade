package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func signalColumns() []string {
	return []string{
		"id", "symbol", "date", "signal_type", "confidence_score", "reasoning", "entry_price",
		"stop_loss", "target", "risk_reward", "indicators", "warning_flag", "generated_at",
		"processed", "strategy", "sentiment_score", "sentiment_reasoning", "created_at",
		"updated_at", "version", "strategy_version", "params_hash",
	}
}

func TestSignalReaderQueriesAndMapsNullableFields(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	date := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	generatedAt := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 10, 27, 18, 30, 0, 0, time.UTC)
	mock.ExpectQuery("FROM signals").WillReturnRows(pgxmock.NewRows(signalColumns()).
		AddRow(int64(7), "INFY", date, "BUY", "0.75", "All rules passed", "123.4500", "118.0000",
			"134.3500", "2.00", "ema20,rsi", nil, generatedAt, true, "PULLBACK", "NEUTRAL",
			"No news", createdAt, createdAt, int64(4), int32(3), "hash").
		AddRow(int64(8), "TCS", date, "HOLD", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
			"DEFAULT", nil, nil, createdAt, createdAt, nil, int32(1), nil))

	rows, err := NewSignalReader(mock).ListLatestSignals(context.Background())
	if err != nil {
		t.Fatalf("ListLatestSignals() error = %v", err)
	}
	if len(rows) != 2 || rows[0].ID != 7 || rows[0].Symbol != "INFY" || rows[0].SignalType != "BUY" {
		t.Fatalf("ListLatestSignals() = %#v", rows)
	}
	if !rows[0].Date.Equal(date) || rows[0].GeneratedAt == nil || !rows[0].GeneratedAt.Equal(generatedAt) {
		t.Fatalf("date fields = %#v", rows[0])
	}
	if rows[0].Confidence == nil || *rows[0].Confidence != "0.75" {
		t.Fatalf("confidence = %v, want 0.75", rows[0].Confidence)
	}
	if rows[0].EntryPrice == nil || *rows[0].EntryPrice != "123.4500" {
		t.Fatalf("entryPrice = %v, want 123.4500", rows[0].EntryPrice)
	}
	if rows[0].Indicators == nil || *rows[0].Indicators != "ema20,rsi" {
		t.Fatalf("indicators = %v", rows[0].Indicators)
	}
	if rows[0].Strategy != "PULLBACK" || rows[0].StrategyVersion != 3 {
		t.Fatalf("strategy fields = %#v", rows[0])
	}
	if rows[1].Confidence != nil || rows[1].Reasoning != nil || rows[1].GeneratedAt != nil ||
		rows[1].SentimentScore != nil || rows[1].SentimentReasoning != nil {
		t.Fatalf("database nulls were not preserved: %#v", rows[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSignalReaderReturnsDatabaseErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("FROM signals").WillReturnError(errors.New("database unavailable"))
	if _, err := NewSignalReader(mock).ListLatestSignals(context.Background()); err == nil {
		t.Fatal("ListLatestSignals() error = nil, want wrapped database error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
