package sqlc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

var (
	holidayColumns  = []string{"holiday_date", "occasion", "holiday_type"}
	holidayTestDate = time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
)

func holidayTestPGDate() pgtype.Date { return pgtype.Date{Time: holidayTestDate, Valid: true} }

func TestGeneratedHolidayQueriesReturnRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	date := holidayTestDate
	pgDate := holidayTestPGDate()
	mock.ExpectQuery("WHERE holiday_date =").WithArgs(pgDate).WillReturnRows(pgxmock.NewRows(holidayColumns).AddRow(date, "Diwali", "FULL"))
	mock.ExpectQuery("BETWEEN").WithArgs(pgDate, pgDate).WillReturnRows(pgxmock.NewRows(holidayColumns).AddRow(date, "Diwali", "FULL"))
	mock.ExpectQuery("holiday_date >=").WithArgs(pgDate).WillReturnRows(pgxmock.NewRows(holidayColumns).AddRow(date, "Diwali", "FULL"))
	queries := New(mock)
	if row, err := queries.GetHolidayByDate(context.Background(), pgDate); err != nil || row.Occasion != "Diwali" {
		t.Fatalf("GetHolidayByDate() = %#v, %v", row, err)
	}
	if rows, err := queries.ListHolidaysByDateRange(context.Background(), ListHolidaysByDateRangeParams{HolidayDate: pgDate, HolidayDate_2: pgDate}); err != nil || len(rows) != 1 {
		t.Fatalf("ListHolidaysByDateRange() = %#v, %v", rows, err)
	}
	if rows, err := queries.ListUpcomingHolidays(context.Background(), pgDate); err != nil || len(rows) != 1 {
		t.Fatalf("ListUpcomingHolidays() = %#v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedHolidayQueriesReturnQueryAndScanErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query func(*Queries) error
		kind  string
	}{
		{name: "get-query", query: func(q *Queries) error {
			_, err := q.GetHolidayByDate(context.Background(), holidayTestPGDate())
			return err
		}, kind: "query"},
		{name: "get-scan", query: func(q *Queries) error {
			_, err := q.GetHolidayByDate(context.Background(), holidayTestPGDate())
			return err
		}, kind: "scan"},
		{name: "range-query", query: func(q *Queries) error {
			_, err := q.ListHolidaysByDateRange(context.Background(), ListHolidaysByDateRangeParams{HolidayDate: holidayTestPGDate(), HolidayDate_2: holidayTestPGDate()})
			return err
		}, kind: "query"},
		{name: "range-scan", query: func(q *Queries) error {
			_, err := q.ListHolidaysByDateRange(context.Background(), ListHolidaysByDateRangeParams{HolidayDate: holidayTestPGDate(), HolidayDate_2: holidayTestPGDate()})
			return err
		}, kind: "scan"},
		{name: "upcoming-query", query: func(q *Queries) error {
			_, err := q.ListUpcomingHolidays(context.Background(), holidayTestPGDate())
			return err
		}, kind: "query"},
		{name: "upcoming-scan", query: func(q *Queries) error {
			_, err := q.ListUpcomingHolidays(context.Background(), holidayTestPGDate())
			return err
		}, kind: "scan"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			expectation := mock.ExpectQuery("FROM nse_holidays")
			switch tc.name {
			case "get-query", "get-scan", "upcoming-query", "upcoming-scan":
				expectation.WithArgs(holidayTestPGDate())
			default:
				expectation.WithArgs(holidayTestPGDate(), holidayTestPGDate())
			}
			if tc.kind == "query" {
				expectation.WillReturnError(errors.New("query failed"))
			} else {
				expectation.WillReturnRows(pgxmock.NewRows(holidayColumns).AddRow("not-a-date", "Holiday", "FULL"))
			}
			if err := tc.query(New(mock)); err == nil {
				t.Fatal("query error = nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
