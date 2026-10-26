package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const insertCandleIfAbsentSQL = `
INSERT INTO ohlcv_candles (
    symbol, timeframe, date, bar_time, open_price, high_price, low_price,
    close_price, volume, adj_close_price
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (symbol, timeframe, date, bar_time) DO NOTHING`

func storageTimeframe(interval string) string {
	switch strings.ToLower(interval) {
	case "1d", "d":
		return "D"
	case "15m", "15":
		return "15m"
	default:
		return interval
	}
}

// CandleWriteDB is the PostgreSQL execution surface required by the candle writer.
type CandleWriteDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// CandleWriter persists candles using Java's insert-if-absent identity contract.
type CandleWriter struct {
	db CandleWriteDB
}

// NewCandleWriter creates a writer for an explicitly supplied database handle.
func NewCandleWriter(database CandleWriteDB) (*CandleWriter, error) {
	if database == nil {
		return nil, fmt.Errorf("candle database is required")
	}
	return &CandleWriter{db: database}, nil
}

// InsertIfAbsent persists a single candle and reports whether this call inserted it.
func (writer *CandleWriter) InsertIfAbsent(ctx context.Context, candle YahooCandle, timeframe string) (bool, error) {
	if strings.TrimSpace(candle.Symbol) == "" || candle.Date.IsZero() || candle.BarTime.IsZero() || strings.TrimSpace(timeframe) == "" {
		return false, fmt.Errorf("candle identity and timeframe are required")
	}
	open, high, low, closePrice := candle.Open, candle.High, candle.Low, candle.Close
	if !IsValidCandle(&CandleData{Symbol: candle.Symbol, Open: &open, High: &high, Low: &low, Close: &closePrice, Volume: candle.Volume}) {
		return false, fmt.Errorf("invalid OHLCV candle for %s on %s", candle.Symbol, candle.Date.Format(time.DateOnly))
	}
	adjusted := candle.AdjustedClose
	if adjusted.Cmp(closePrice) == 0 {
		adjusted = closePrice
	}
	result, err := writer.db.Exec(ctx, insertCandleIfAbsentSQL,
		candle.Symbol, timeframe, candle.Date.Format(time.DateOnly), candle.BarTime.Format("15:04:05"),
		open.String(), high.String(), low.String(), closePrice.String(), candle.Volume, adjusted.String())
	if err != nil {
		return false, fmt.Errorf("insert candle: %w", err)
	}
	return result.RowsAffected() == 1, nil
}
