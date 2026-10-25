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

func TestCandleReaderChecksDailyCandleExistence(t *testing.T) {
	date := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("EXISTS").WithArgs("ABC", pgtype.Date{Time: date, Valid: true}).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))
	exists, err := NewCandleReader(mock).CandleExistsOnDate(context.Background(), "ABC", date)
	if err != nil || !exists {
		t.Fatalf("exists = %v, %v", exists, err)
	}
	mock.ExpectQuery("EXISTS").WithArgs("ABC", pgtype.Date{Time: date, Valid: true}).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	exists, err = NewCandleReader(mock).CandleExistsOnDate(context.Background(), "ABC", date)
	if err != nil || exists {
		t.Fatalf("exists = %v, %v", exists, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCandleReaderWrapsExistenceErrors(t *testing.T) {
	date := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("EXISTS").WithArgs("ABC", pgtype.Date{Time: date, Valid: true}).
		WillReturnError(errors.New("offline"))
	if _, err := NewCandleReader(mock).CandleExistsOnDate(context.Background(), "ABC", date); err == nil {
		t.Fatal("existence error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCandleReaderLatestIntervalDateAndEmptyResult(t *testing.T) {
	date := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("ORDER BY date DESC").WithArgs("TCS", "15m").
		WillReturnRows(pgxmock.NewRows([]string{"date"}).AddRow(pgtype.Date{Time: date, Valid: true}))
	got, err := NewCandleReader(mock).LatestIntervalCandleDate(context.Background(), "TCS", "15m")
	if err != nil || got == nil || !got.Equal(date) {
		t.Fatalf("latest = %v, %v", got, err)
	}

	emptyMock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer emptyMock.Close()
	emptyMock.ExpectQuery("ORDER BY date DESC").WithArgs("NEW", "15m").WillReturnError(pgx.ErrNoRows)
	got, err = NewCandleReader(emptyMock).LatestIntervalCandleDate(context.Background(), "NEW", "15m")
	if err != nil || got != nil {
		t.Fatalf("empty latest = %v, %v", got, err)
	}
}

func TestCandleReaderListsDailyCandlesInRange(t *testing.T) {
	from := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("date BETWEEN").WithArgs("TCS", pgtype.Date{Time: from, Valid: true}, pgtype.Date{Time: to, Valid: true}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "date"}).
			AddRow(int64(2), pgtype.Date{Time: to, Valid: true}).
			AddRow(int64(1), pgtype.Date{Time: from, Valid: true}))
	rows, err := NewCandleReader(mock).DailyCandlesInRange(context.Background(), "TCS", from, to)
	if err != nil || len(rows) != 2 || rows[0].ID != 2 || !rows[0].Date.Equal(to) || rows[1].ID != 1 {
		t.Fatalf("range rows = %+v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	failMock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer failMock.Close()
	failMock.ExpectQuery("date BETWEEN").WillReturnError(errors.New("db down"))
	if _, err := NewCandleReader(failMock).DailyCandlesInRange(context.Background(), "TCS", from, to); err == nil {
		t.Fatal("range error = nil")
	}
}
