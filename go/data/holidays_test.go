package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func TestHolidayReaderQueriesAndMapsDateRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	date := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	pgDate := pgtype.Date{Time: date, Valid: true}
	columns := []string{"holiday_date", "occasion", "holiday_type"}
	mock.ExpectQuery("WHERE holiday_date = \\$1").WithArgs(pgDate).WillReturnRows(
		pgxmock.NewRows(columns).AddRow(date, "Diwali", "FULL"),
	)
	mock.ExpectQuery("BETWEEN \\$1 AND \\$2").WithArgs(pgDate, pgDate).WillReturnRows(
		pgxmock.NewRows(columns).AddRow(date, "Diwali", "FULL"),
	)
	mock.ExpectQuery("holiday_date >= \\$1").WithArgs(pgDate).WillReturnRows(
		pgxmock.NewRows(columns).AddRow(date, "Diwali", "FULL"),
	)

	reader := NewHolidayReader(mock)
	holiday, err := reader.GetHolidayByDate(context.Background(), date)
	if err != nil || holiday == nil || holiday.Occasion != "Diwali" || holiday.Type != "FULL" || !holiday.Date.Equal(date) {
		t.Fatalf("GetHolidayByDate() = %#v, %v", holiday, err)
	}
	rows, err := reader.ListHolidaysByDateRange(context.Background(), date, date)
	if err != nil || len(rows) != 1 || rows[0].Occasion != "Diwali" {
		t.Fatalf("ListHolidaysByDateRange() = %#v, %v", rows, err)
	}
	rows, err = reader.ListUpcomingHolidays(context.Background(), date)
	if err != nil || len(rows) != 1 || rows[0].Date.Format("2006-01-02") != "2026-10-26" {
		t.Fatalf("ListUpcomingHolidays() = %#v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHolidayReaderReturnsNilWhenDateIsNotAHoliday(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	date := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM nse_holidays").WithArgs(pgtype.Date{Time: date, Valid: true}).WillReturnRows(pgxmock.NewRows([]string{"holiday_date", "occasion", "holiday_type"}))
	holiday, err := NewHolidayReader(mock).GetHolidayByDate(context.Background(), date)
	if err != nil || holiday != nil {
		t.Fatalf("GetHolidayByDate(missing) = %#v, %v", holiday, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHolidayReaderWrapsQueryErrors(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	date := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM nse_holidays").WithArgs(pgtype.Date{Time: date, Valid: true}).WillReturnError(errors.New("query failed"))
	if _, err := NewHolidayReader(mock).ListUpcomingHolidays(context.Background(), date); err == nil {
		t.Fatal("ListUpcomingHolidays() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
