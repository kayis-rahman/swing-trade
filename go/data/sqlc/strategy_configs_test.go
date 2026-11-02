package sqlc

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestListStrategyConfigs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	columns := []string{"id", "variant_id", "version", "strategy_type", "params", "overlays", "params_hash", "mode", "paper_capital", "is_current", "notes", "created_at"}
	mock.ExpectQuery("ORDER BY variant_id DESC, version DESC").WillReturnRows(pgxmock.NewRows(columns).AddRow(
		int64(1), "variant", int32(2), "PULLBACK", []byte(`{}`), []byte(`{}`), "hash", "SHADOW", "500000.00", true, nil, time.Now(),
	))
	rows, err := New(mock).ListStrategyConfigs(context.Background())
	if err != nil || len(rows) != 1 || rows[0].VariantID != "variant" {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
