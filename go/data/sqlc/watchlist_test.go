package sqlc

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestWatchlistQueriesUseGeneratedSQLAndReturnRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("FROM watchlist").WillReturnRows(pgxmock.NewRows([]string{
		"id", "symbol", "name", "exchange", "is_active", "added_at", "last_synced_at", "candle_count",
	}).AddRow(int64(1), "TCS", "TCS", "NSE", true, nil, nil, int64(0)))
	rows, err := New(mock).ListActiveWatchlist(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Symbol != "TCS" {
		t.Fatalf("ListActiveWatchlist() = %#v, %v", rows, err)
	}

	mock.ExpectQuery("FROM watchlist").WillReturnRows(pgxmock.NewRows([]string{
		"id", "symbol", "name", "exchange", "is_active", "added_at", "last_synced_at", "candle_count",
	}).AddRow(int64(1), "TCS", "TCS", "NSE", true, nil, nil, int64(0)))
	rows, err = New(mock).ListAllWatchlist(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Symbol != "TCS" {
		t.Fatalf("ListAllWatchlist() = %#v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestQueriesWithTransaction(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	transaction, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("FROM watchlist").WillReturnRows(pgxmock.NewRows([]string{
		"id", "symbol", "name", "exchange", "is_active", "added_at", "last_synced_at", "candle_count",
	}))
	if _, err := New(mock).WithTx(transaction).ListAllWatchlist(context.Background()); err != nil {
		t.Fatal(err)
	}
	mock.ExpectCommit()
	if err := transaction.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
