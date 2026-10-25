package data

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// HistoricalCandleProvider supplies candles for an inclusive requested window.
type HistoricalCandleProvider interface {
	FetchCandles(context.Context, string, time.Time, time.Time, string) ([]YahooCandle, error)
}

// IngestionOutcome summarizes one provider range fetch and its persistence result.
type IngestionOutcome struct {
	SourceOutcome string
	FetchedRows   int
	SavedRows     int
	InvalidRows   int
	ErrorMessage  string
}

// CandleIngester validates provider candles against the NSE calendar before storage.
const defaultIngestionChunkDays = 365

// CandleIngester coordinates historical provider reads, session validation, and storage.
type CandleIngester struct {
	provider HistoricalCandleProvider
	writer   *CandleWriter
	holidays HolidayReadStore
}

// NewCandleIngester constructs an ingestion service over explicit provider and storage ports.
func NewCandleIngester(provider HistoricalCandleProvider, writer *CandleWriter, holidays HolidayReadStore) (*CandleIngester, error) {
	if provider == nil || writer == nil || holidays == nil {
		return nil, fmt.Errorf("provider, candle writer, and holiday store are required")
	}
	return &CandleIngester{provider: provider, writer: writer, holidays: holidays}, nil
}

// IngestRange fetches, validates, and idempotently stores a single symbol's inclusive range.
func (ingester *CandleIngester) IngestRange(ctx context.Context, symbol string, from, to time.Time, providerInterval, storageTimeframe string) IngestionOutcome {
	if strings.TrimSpace(symbol) == "" || from.IsZero() || to.IsZero() || from.After(to) || strings.TrimSpace(providerInterval) == "" || strings.TrimSpace(storageTimeframe) == "" {
		return IngestionOutcome{SourceOutcome: "INVALID_REQUEST", ErrorMessage: "symbol, intervals, and ordered date range are required"}
	}
	outcome := IngestionOutcome{SourceOutcome: "NO_USABLE_DATA"}
	for chunkStart := from; !chunkStart.After(to); {
		chunkEnd := chunkStart.AddDate(0, 0, defaultIngestionChunkDays-1)
		if chunkEnd.After(to) {
			chunkEnd = to
		}
		candles, err := ingester.provider.FetchCandles(ctx, symbol, chunkStart, chunkEnd, providerInterval)
		if err != nil {
			outcome.SourceOutcome = "TRANSIENT_SOURCE_FAILURE"
			outcome.ErrorMessage = err.Error()
			return outcome
		}
		outcome.FetchedRows += len(candles)
		if err := ingester.storeChunk(ctx, &outcome, candles, chunkStart, chunkEnd, storageTimeframe); err != nil {
			outcome.SourceOutcome = "TRANSIENT_SOURCE_FAILURE"
			outcome.ErrorMessage = err.Error()
			return outcome
		}
		chunkStart = chunkEnd.AddDate(0, 0, 1)
	}
	switch {
	case outcome.FetchedRows == 0:
		outcome.SourceOutcome = "NO_USABLE_DATA"
	case outcome.InvalidRows > 0 && outcome.SavedRows == 0:
		outcome.SourceOutcome = "INVALID_ROWS_REJECTED"
	default:
		outcome.SourceOutcome = "DATA_RECEIVED"
	}
	return outcome
}

func (ingester *CandleIngester) storeChunk(ctx context.Context, outcome *IngestionOutcome, candles []YahooCandle, from, to time.Time, timeframe string) error {
	for _, candle := range candles {
		if candle.Date.Before(from) || candle.Date.After(to) || !isValidYahooCandle(&candle) {
			outcome.InvalidRows++
			continue
		}
		trading, err := IsNSETradingSession(ctx, candle.Date, ingester.holidays)
		if err != nil {
			return fmt.Errorf("check NSE session: %w", err)
		}
		if !trading {
			outcome.InvalidRows++
			continue
		}
		inserted, err := ingester.writer.InsertIfAbsent(ctx, candle, timeframe)
		if err != nil {
			return err
		}
		if inserted {
			outcome.SavedRows++
		}
	}
	return nil
}

func isValidYahooCandle(candle *YahooCandle) bool {
	if candle == nil || candle.Volume <= 0 || candle.Symbol == "" || candle.Date.IsZero() {
		return false
	}
	return IsValidCandle(&CandleData{Symbol: candle.Symbol, Open: &candle.Open, High: &candle.High, Low: &candle.Low, Close: &candle.Close, Volume: candle.Volume})
}
