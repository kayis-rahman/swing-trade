package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

// Holiday is the read model shared by NSE calendar API and service consumers.
type Holiday struct {
	Date     time.Time
	Occasion string
	Type     string
}

// HolidayReadStore exposes read-only NSE holiday queries.
type HolidayReadStore interface {
	GetHolidayByDate(context.Context, time.Time) (*Holiday, error)
	ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]Holiday, error)
	ListUpcomingHolidays(context.Context, time.Time) ([]Holiday, error)
}

// HolidayReader executes generated read-only NSE holiday queries.
type HolidayReader struct {
	queries *sqlc.Queries
}

// NewHolidayReader constructs a holiday reader on a pgx-compatible database handle.
func NewHolidayReader(db sqlc.DBTX) *HolidayReader {
	return &HolidayReader{queries: sqlc.New(db)}
}

// GetHolidayByDate returns the holiday on date or nil when the date is not a holiday.
func (r *HolidayReader) GetHolidayByDate(ctx context.Context, date time.Time) (*Holiday, error) {
	row, err := r.queries.GetHolidayByDate(ctx, sqlDate(date))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get holiday: %w", err)
	}
	return &Holiday{Date: row.HolidayDate.Time, Occasion: row.Occasion, Type: row.HolidayType}, nil
}

// ListHolidaysByDateRange returns holidays in the inclusive date range, ordered by date.
func (r *HolidayReader) ListHolidaysByDateRange(ctx context.Context, from, to time.Time) ([]Holiday, error) {
	rows, err := r.queries.ListHolidaysByDateRange(ctx, sqlc.ListHolidaysByDateRangeParams{
		HolidayDate: sqlDate(from), HolidayDate_2: sqlDate(to),
	})
	if err != nil {
		return nil, fmt.Errorf("list holidays by date range: %w", err)
	}
	result := make([]Holiday, 0, len(rows))
	for _, row := range rows {
		result = append(result, Holiday{Date: row.HolidayDate.Time, Occasion: row.Occasion, Type: row.HolidayType})
	}
	return result, nil
}

// ListUpcomingHolidays returns holidays on or after from, ordered by date.
func (r *HolidayReader) ListUpcomingHolidays(ctx context.Context, from time.Time) ([]Holiday, error) {
	rows, err := r.queries.ListUpcomingHolidays(ctx, sqlDate(from))
	if err != nil {
		return nil, fmt.Errorf("list upcoming holidays: %w", err)
	}
	result := make([]Holiday, 0, len(rows))
	for _, row := range rows {
		result = append(result, Holiday{Date: row.HolidayDate.Time, Occasion: row.Occasion, Type: row.HolidayType})
	}
	return result, nil
}

func sqlDate(value time.Time) pgtype.Date {
	return pgtype.Date{
		Time:  time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}
