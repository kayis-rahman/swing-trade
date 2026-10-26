package sqlc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func TestGeneratedReconciliationQueriesReadRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	from := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("calendar_year").WithArgs(int32(2026), int32(2027)).
		WillReturnRows(pgxmock.NewRows([]string{"calendar_year", "status"}).AddRow(int32(2026), "VERIFIED"))
	mock.ExpectQuery("date BETWEEN").WithArgs("TCS",
		pgtype.Date{Time: from, Valid: true}, pgtype.Date{Time: to, Valid: true}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "date"}).
			AddRow(int64(3), pgtype.Date{Time: to, Valid: true}).
			AddRow(int64(2), pgtype.Date{Time: from, Valid: true}))

	queries := New(mock)
	years, err := queries.ListNseCalendarCoverageYears(context.Background(), ListNseCalendarCoverageYearsParams{
		CalendarYear: 2026, CalendarYear_2: 2027,
	})
	if err != nil || len(years) != 1 || years[0].Status != "VERIFIED" {
		t.Fatalf("coverage rows = %+v, %v", years, err)
	}
	rows, err := queries.ListDailyCandlesInDateRange(context.Background(), ListDailyCandlesInDateRangeParams{
		Symbol: "TCS",
		Date:   pgtype.Date{Time: from, Valid: true},
		Date_2: pgtype.Date{Time: to, Valid: true},
	})
	if err != nil || len(rows) != 2 || rows[0].ID != 3 || rows[1].ID != 2 {
		t.Fatalf("range rows = %+v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedReconciliationQueriesReturnDatabaseErrors(t *testing.T) {
	for name, run := range map[string]func(*Queries) error{
		"coverage": func(q *Queries) error {
			_, err := q.ListNseCalendarCoverageYears(context.Background(), ListNseCalendarCoverageYearsParams{})
			return err
		},
		"range": func(q *Queries) error {
			_, err := q.ListDailyCandlesInDateRange(context.Background(), ListDailyCandlesInDateRangeParams{Symbol: "TCS"})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			mock.ExpectQuery("SELECT").WillReturnError(errors.New("database unavailable"))
			if err := run(New(mock)); err == nil {
				t.Error("query error = nil")
			}
		})
	}
}
