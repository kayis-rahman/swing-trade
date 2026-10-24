package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestWatchlistReaderQueriesAndMapsNullableFields(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	addedAt := time.Date(2026, 10, 15, 6, 49, 42, 140204000, time.UTC)
	mock.ExpectQuery("FROM watchlist").WillReturnRows(pgxmock.NewRows([]string{
		"id", "symbol", "name", "exchange", "is_active", "added_at", "last_synced_at", "candle_count",
	}).AddRow(int64(3), "INFY", "Infosys Limited", "NSE", true, addedAt, nil, int64(0)).AddRow(int64(4), "UNKNOWN", nil, nil, nil, nil, nil, nil))

	rows, err := NewWatchlistReader(mock).ListActiveWatchlist(context.Background())
	if err != nil {
		t.Fatalf("ListActiveWatchlist() error = %v", err)
	}
	if len(rows) != 2 || rows[0].Symbol != "INFY" || rows[0].ID != 3 {
		t.Fatalf("ListActiveWatchlist() = %#v", rows)
	}
	if rows[0].Name == nil || *rows[0].Name != "Infosys Limited" || rows[0].Exchange == nil || *rows[0].Exchange != "NSE" {
		t.Fatalf("nullable text fields = %#v", rows[0])
	}
	if rows[0].IsActive == nil || !*rows[0].IsActive || rows[0].AddedAt == nil || !rows[0].AddedAt.Equal(addedAt) {
		t.Fatalf("nullable flag or timestamp fields = %#v", rows[0])
	}
	if rows[0].LastSyncedAt != nil || rows[0].CandleCount == nil || *rows[0].CandleCount != 0 {
		t.Fatalf("nullable fields = %#v", rows[0])
	}
	if rows[1].Name != nil || rows[1].Exchange != nil || rows[1].IsActive != nil || rows[1].AddedAt != nil || rows[1].LastSyncedAt != nil || rows[1].CandleCount != nil {
		t.Fatalf("database nulls were not preserved: %#v", rows[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWatchlistReaderReturnsDatabaseErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("FROM watchlist").WillReturnError(errors.New("database unavailable"))
	if _, err := NewWatchlistReader(mock).ListAllWatchlist(context.Background()); err == nil {
		t.Fatal("ListAllWatchlist() error = nil, want wrapped database error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
