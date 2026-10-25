package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func TestCandleReaderReturnsWindowAndEmptyWindow(t *testing.T) {
	date := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	pgDate := pgtype.Date{Time: date, Valid: true}
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("ORDER BY date ASC").WithArgs("ABC").WillReturnRows(pgxmock.NewRows([]string{"date"}).AddRow(pgDate))
	mock.ExpectQuery("ORDER BY date DESC").WithArgs("ABC").WillReturnRows(pgxmock.NewRows([]string{"date"}).AddRow(pgDate))
	mock.ExpectQuery("COUNT").WithArgs("ABC").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(1)))
	reader := NewCandleReader(mock)
	window, err := reader.GetDailyDateWindow(context.Background(), "ABC")
	if err != nil || window.Earliest == nil || window.Latest == nil || !window.Earliest.Equal(date) || !window.Latest.Equal(date) {
		t.Fatalf("window = %+v, %v", window, err)
	}
	if count, err := reader.CountDailyCandles(context.Background(), "ABC"); err != nil || count != 1 {
		t.Fatalf("count = %d, %v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	emptyMock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer emptyMock.Close()
	emptyMock.ExpectQuery("ORDER BY date ASC").WithArgs("NEW").WillReturnError(pgx.ErrNoRows)
	window, err = NewCandleReader(emptyMock).GetDailyDateWindow(context.Background(), "NEW")
	if err != nil || window.Earliest != nil || window.Latest != nil {
		t.Fatalf("empty window = %+v, %v", window, err)
	}
	if err := emptyMock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCandleReaderWrapsDatabaseErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("ORDER BY date ASC").WithArgs("ABC").WillReturnError(errors.New("offline"))
	if _, err := NewCandleReader(mock).GetDailyDateWindow(context.Background(), "ABC"); err == nil {
		t.Fatal("window error = nil")
	}
	mock.ExpectQuery("COUNT").WithArgs("ABC").WillReturnError(errors.New("offline"))
	if _, err := NewCandleReader(mock).CountDailyCandles(context.Background(), "ABC"); err == nil {
		t.Fatal("count error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
