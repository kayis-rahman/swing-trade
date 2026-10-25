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

func dateValue(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	date := value.Time
	return &date
}
