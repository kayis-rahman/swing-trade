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

// CandleDateWindow is the earliest and latest daily session stored for one symbol.
type CandleDateWindow struct {
	Earliest *time.Time
	Latest   *time.Time
}

// CandleReadStore provides daily-candle metadata for ingestion and status workflows.
type CandleReadStore interface {
	GetDailyDateWindow(context.Context, string) (CandleDateWindow, error)
	CountDailyCandles(context.Context, string) (int64, error)
}

// CandleReader executes generated daily-candle metadata queries.
type CandleReader struct{ queries *sqlc.Queries }

// NewCandleReader creates a candle reader on a pgx-compatible database handle.
func NewCandleReader(database sqlc.DBTX) *CandleReader {
	return &CandleReader{queries: sqlc.New(database)}
}

// GetDailyDateWindow returns null endpoints when no daily candle exists.
func (reader *CandleReader) GetDailyDateWindow(ctx context.Context, symbol string) (CandleDateWindow, error) {
	first, err := reader.queries.GetEarliestDailyCandleDate(ctx, symbol)
	if errors.Is(err, pgx.ErrNoRows) {
		return CandleDateWindow{}, nil
	}
	if err != nil {
		return CandleDateWindow{}, fmt.Errorf("get earliest daily candle: %w", err)
	}
	last, err := reader.queries.GetLatestDailyCandleDate(ctx, symbol)
	if err != nil {
		return CandleDateWindow{}, fmt.Errorf("get latest daily candle: %w", err)
	}
	return CandleDateWindow{Earliest: dateValue(first), Latest: dateValue(last)}, nil
}

// CountDailyCandles returns the daily-only row count for a symbol.
func (reader *CandleReader) CountDailyCandles(ctx context.Context, symbol string) (int64, error) {
	count, err := reader.queries.CountDailyCandles(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("count daily candles: %w", err)
	}
	return count, nil
}

// CandleExistsOnDate reports whether a daily candle row already exists for a symbol and date.
func (reader *CandleReader) CandleExistsOnDate(ctx context.Context, symbol string, date time.Time) (bool, error) {
	exists, err := reader.queries.DailyCandleExists(ctx, sqlc.DailyCandleExistsParams{Symbol: symbol, Date: sqlDate(date)})
	if err != nil {
		return false, fmt.Errorf("check daily candle existence: %w", err)
	}
	return exists, nil
}

// LatestIntervalCandleDate returns the newest stored date for a symbol and
// timeframe, or nil when no row exists.
func (reader *CandleReader) LatestIntervalCandleDate(ctx context.Context, symbol, timeframe string) (*time.Time, error) {
	date, err := reader.queries.GetLatestIntervalCandleDate(ctx, sqlc.GetLatestIntervalCandleDateParams{
		Symbol: symbol, Timeframe: timeframe,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest interval candle date: %w", err)
	}
	return dateValue(date), nil
}

func dateValue(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	date := value.Time
	return &date
}

// DailyCandleRow is one stored daily candle identity in a reconciliation range.
type DailyCandleRow struct {
	ID   int64
	Date time.Time
}

// DailyCandlesInRange returns the stored daily rows for a symbol and inclusive
// date range, ordered newest first like Java's findBySymbolAndDateRange.
func (reader *CandleReader) DailyCandlesInRange(ctx context.Context, symbol string, from, to time.Time) ([]DailyCandleRow, error) {
	rows, err := reader.queries.ListDailyCandlesInDateRange(ctx, sqlc.ListDailyCandlesInDateRangeParams{
		Symbol: symbol,
		Date:   pgtype.Date{Time: from, Valid: true},
		Date_2: pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("list daily candles in range: %w", err)
	}
	result := make([]DailyCandleRow, 0, len(rows))
	for _, row := range rows {
		date := dateValue(row.Date)
		if date == nil {
			continue
		}
		result = append(result, DailyCandleRow{ID: row.ID, Date: *date})
	}
	return result, nil
}
