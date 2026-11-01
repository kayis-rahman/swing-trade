package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestStrategyConfigReader(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	columns := []string{"id", "variant_id", "version", "strategy_type", "params", "overlays", "params_hash", "mode", "paper_capital", "is_current", "notes", "created_at"}
	mock.ExpectQuery("SELECT id, variant_id").WillReturnRows(pgxmock.NewRows(columns).AddRow(
		int64(1), "alpha", int32(2), "PULLBACK", []byte(`{"lookback":5}`), []byte(`{}`), "hash", "SHADOW", "500000.00", true, nil, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	))
	rows, err := NewStrategyConfigReader(mock).ListStrategyConfigs(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if rows[0].PaperCapital != "500000.00" || string(rows[0].Params) != `{"lookback":5}` || rows[0].CreatedAt != "2026-01-02T03:04:05.000000" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
	mock.ExpectQuery("SELECT id, variant_id").WillReturnError(errors.New("database failure"))
	if _, err := NewStrategyConfigReader(mock).ListStrategyConfigs(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
