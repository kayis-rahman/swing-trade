//go:build integration

package data

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kayis-rahman/swing-trade/go/data/db"
)

// newIntradayTestService creates the candle, watchlist, and holiday tables in
// an isolated schema behind a search-path-scoped pool.
func newIntradayTestService(t *testing.T, provider HistoricalCandleProvider) (*IntradayIngestion, *pgxpool.Pool, func()) {
	t.Helper()
	ctx, adminPool := newCandleIntegrationPool(t)
	schema := createCandleTestSchema(t, ctx, adminPool)

	config, err := db.ReadOnlyPoolConfig(os.Getenv("SWINGTRADE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal("development database is not an authorized test target")
	}
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "off"
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("create schema-scoped test pool")
	}
	cleanup := func() {
		pool.Close()
		dropCandleTestSchema(t, adminPool, schema)
		adminPool.Close()
	}
	for _, statement := range []string{
		`CREATE TABLE ohlcv_candles (
			id BIGSERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			date DATE NOT NULL,
			open_price NUMERIC(15,4),
			high_price NUMERIC(15,4),
			low_price NUMERIC(15,4),
			close_price NUMERIC(15,4),
			volume BIGINT,
			adj_close_price NUMERIC(15,4),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			version INTEGER,
			timeframe VARCHAR(10) NOT NULL DEFAULT 'D',
			bar_time TIME NOT NULL DEFAULT '00:00:00',
			CONSTRAINT uq_ohlcv_candles_symbol_timeframe_date_bar_time UNIQUE (symbol, timeframe, date, bar_time)
		)`,
		`CREATE TABLE watchlist (
			id BIGSERIAL PRIMARY KEY,
			symbol VARCHAR(10) NOT NULL UNIQUE,
			name TEXT,
			exchange VARCHAR(5) DEFAULT 'NSE',
			is_active BOOLEAN DEFAULT TRUE,
			added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_synced_at TIMESTAMP,
			candle_count INTEGER DEFAULT 0
		)`,
		`CREATE TABLE nse_holidays (
			id BIGSERIAL PRIMARY KEY,
			holiday_date DATE NOT NULL UNIQUE,
			occasion VARCHAR(128) NOT NULL,
			holiday_type VARCHAR(32) NOT NULL DEFAULT 'FULL' CHECK (holiday_type IN ('FULL', 'PARTIAL')),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			version INTEGER
		)`,
		`INSERT INTO watchlist (symbol, is_active) VALUES ('TCS', TRUE)`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			cleanup()
			t.Fatalf("prepare isolated intraday schema: %v", err)
		}
	}
	writer, err := NewCandleWriter(pool)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	service, err := NewIntradayIngestion(
		NewWatchlistReader(pool),
		NewCandleReader(pool),
		NewHolidayReader(pool),
		provider,
		writer,
		func() time.Time { return intradayNow },
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	return service, pool, cleanup
}

type recordingIntradayProvider struct {
	candles []YahooCandle
	calls   int
}

func (p *recordingIntradayProvider) FetchCandles(context.Context, string, time.Time, time.Time, string) ([]YahooCandle, error) {
	p.calls++
	return p.candles, nil
}

func TestIntradayIngestionStoresAndResumesThroughRealSQL(t *testing.T) {
	provider := &recordingIntradayProvider{candles: []YahooCandle{
		intradayBar(t, "TCS", intradaySession, barClock(9, 15), "4000"),
		intradayBar(t, "TCS", intradaySession, barClock(15, 45), "4105"), // outside session: rejected
	}}
	service, pool, cleanup := newIntradayTestService(t, provider)
	defer cleanup()
	ctx := context.Background()

	first, err := service.IngestWatchlistInterval(ctx, "15m", intradaySession, intradaySession)
	if err != nil {
		t.Fatalf("first run error = %v", err)
	}
	if first.BarsSaved != 1 || first.BarsRejected != 1 {
		t.Fatalf("first run = %+v, want1 saved and1 rejected", first)
	}

	// The stored bar makes the second run resume at the latest date and the
	// real ON CONFLICT identity absorbs the duplicate fetch.
	second, err := service.IngestWatchlistInterval(ctx, "15m", intradaySession, intradaySession)
	if err != nil {
		t.Fatalf("second run error = %v", err)
	}
	if second.BarsSaved != 0 || second.BarsFetched != 2 || second.SymbolsSucceeded != 1 {
		t.Fatalf("second run = %+v, want an idempotent no-op repeat of the refetched latest date", second)
	}
	if provider.calls != 2 {
		t.Fatalf("provider calls = %d, want one fetch per run", provider.calls)
	}

	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM ohlcv_candles WHERE timeframe = '15m'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stored bars = %d, want exactly the one valid session bar", count)
	}
}
