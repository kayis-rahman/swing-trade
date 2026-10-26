package data

import (
	"context"
	"fmt"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

// CalendarCoverageStore reports whether every year spanned by a date range has
// a VERIFIED nse_calendar_coverage row, matching Java's
// MarketCalendar.isCoverageVerified.
type CalendarCoverageStore interface {
	IsCoverageVerified(ctx context.Context, from, to time.Time) (bool, error)
}

// CoverageReader reads the verified-year coverage table.
type CoverageReader struct{ queries *sqlc.Queries }

// NewCoverageReader creates a reader on a pgx-compatible database handle.
func NewCoverageReader(database sqlc.DBTX) *CoverageReader {
	return &CoverageReader{queries: sqlc.New(database)}
}

// IsCoverageVerified returns true only when every year in the inclusive range
// exists with status VERIFIED; a missing year counts as unverified.
func (reader *CoverageReader) IsCoverageVerified(ctx context.Context, from, to time.Time) (bool, error) {
	// #nosec G115 -- calendar years are four-digit values far inside int32.
	rows, err := reader.queries.ListNseCalendarCoverageYears(ctx, sqlc.ListNseCalendarCoverageYearsParams{
		CalendarYear:   int32(from.Year()),
		CalendarYear_2: int32(to.Year()),
	})
	if err != nil {
		return false, fmt.Errorf("read NSE calendar coverage: %w", err)
	}
	verifiedYears := make(map[int]bool, len(rows))
	for _, row := range rows {
		verifiedYears[int(row.CalendarYear)] = row.Status == "VERIFIED"
	}
	for year := from.Year(); year <= to.Year(); year++ {
		if !verifiedYears[year] {
			return false, nil
		}
	}
	return true, nil
}
