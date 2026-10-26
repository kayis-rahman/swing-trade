package sqlc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func TestGeneratedCandleQueriesReadWindowAndDailyCount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	date := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	pgDate := pgtype.Date{Time: date, Valid: true}
	mock.ExpectQuery("ORDER BY date ASC").WithArgs("ABC").WillReturnRows(pgxmock.NewRows([]string{"date"}).AddRow(pgDate))
	mock.ExpectQuery("ORDER BY date DESC").WithArgs("ABC").WillReturnRows(pgxmock.NewRows([]string{"date"}).AddRow(pgDate))
	mock.ExpectQuery("COUNT\\(\\*\\)").WithArgs("ABC").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(1)))
	queries := New(mock)
	if first, err := queries.GetEarliestDailyCandleDate(context.Background(), "ABC"); err != nil || !first.Time.Equal(date) {
		t.Fatalf("earliest = %+v, %v", first, err)
	}
	if last, err := queries.GetLatestDailyCandleDate(context.Background(), "ABC"); err != nil || !last.Time.Equal(date) {
		t.Fatalf("latest = %+v, %v", last, err)
	}
	if count, err := queries.CountDailyCandles(context.Background(), "ABC"); err != nil || count != 1 {
		t.Fatalf("count = %d, %v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedCandleQueriesReturnDatabaseErrors(t *testing.T) {
	for _, query := range []func(*Queries) error{
		func(q *Queries) error {
			_, err := q.GetEarliestDailyCandleDate(context.Background(), "ABC")
			return err
		},
		func(q *Queries) error { _, err := q.GetLatestDailyCandleDate(context.Background(), "ABC"); return err },
		func(q *Queries) error { _, err := q.CountDailyCandles(context.Background(), "ABC"); return err },
	} {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		mock.ExpectQuery("FROM ohlcv_candles").WithArgs("ABC").WillReturnError(errors.New("database unavailable"))
		if err := query(New(mock)); err == nil {
			t.Error("query error = nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		mock.Close()
	}
}
