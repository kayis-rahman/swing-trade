package sqlc

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func signalRowColumns() []string {
	return []string{
		"id", "symbol", "date", "signal_type", "confidence_score", "reasoning", "entry_price",
		"stop_loss", "target", "risk_reward", "indicators", "warning_flag", "generated_at",
		"processed", "strategy", "sentiment_score", "sentiment_reasoning", "created_at",
		"updated_at", "version", "strategy_version", "params_hash",
	}
}

func TestListLatestSignalsUsesRankedWindowQuery(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	date := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 10, 27, 18, 30, 0, 0, time.UTC)
	mock.ExpectQuery("ROW_NUMBER").WillReturnRows(pgxmock.NewRows(signalRowColumns()).
		AddRow(int64(7), "INFY", date, "BUY", "0.75", "All rules passed", "123.4500", "118.0000",
			"134.3500", "2.00", "ema20,rsi", nil, date, true, "PULLBACK", "NEUTRAL",
			"No news", createdAt, createdAt, int64(4), int32(3), "hash"))

	rows, err := New(mock).ListLatestSignals(context.Background())
	if err != nil {
		t.Fatalf("ListLatestSignals() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ID != 7 || rows[0].Symbol != "INFY" {
		t.Fatalf("ListLatestSignals() = %#v", rows)
	}
	if rows[0].StrategyVersion != 3 || !rows[0].Version.Valid || rows[0].Version.Int32 != 4 {
		t.Fatalf("version fields = %#v", rows[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
