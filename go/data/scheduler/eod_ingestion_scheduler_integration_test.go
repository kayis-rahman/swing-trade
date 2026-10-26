package scheduler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
	"github.com/kayis-rahman/swing-trade/go/data"
	"github.com/kayis-rahman/swing-trade/go/data/db"
)

// recordingProvider serves one valid daily bar inside every requested window so
// the real writer and existence query carry the test.
type recordingProvider struct {
	mu    sync.Mutex
	calls int
}

func (p *recordingProvider) FetchCandles(_ context.Context, symbol string, from, _ time.Time, _ string) ([]data.YahooCandle, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	open, _ := coredecimal.Parse("10")
	high, _ := coredecimal.Parse("12")
	low, _ := coredecimal.Parse("9")
	closePrice, _ := coredecimal.Parse("11")
	return []data.YahooCandle{{
		Symbol: symbol, Timeframe: "1d", Date: from,
		BarTime:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Open:          open,
		High:          high,
		Low:           low,
		Close:         closePrice,
		AdjustedClose: closePrice,
		Volume:        1000,
	}}, nil
}

func (p *recordingProvider) fetches() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func TestEODSchedulerStoresCandlesThroughRealQueriesAndJobLock(t *testing.T) {
	ctx, databaseURL, adminPool := newSchedulerIntegrationPool(t)
	defer adminPool.Close()
	schema := createSchedulerTestSchema(t, ctx, adminPool)
	defer dropSchedulerTestSchema(t, adminPool, schema)

	config, err := db.ReadOnlyPoolConfig(databaseURL)
	if err != nil {
		t.Fatal("development database is not an authorized test target")
	}
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "off"
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("create schema-scoped test pool")
	}
	defer pool.Close()

	for _, statement := range []string{
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
		`INSERT INTO watchlist (symbol, is_active) VALUES ('TCS', TRUE), ('INACTIVE', FALSE)`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("prepare isolated schema: %v", err)
		}
	}

	writer, err := data.NewCandleWriter(pool)
	if err != nil {
		t.Fatal(err)
	}
	provider := &recordingProvider{}
	ingester, err := data.NewCandleIngester(provider, writer, data.NewHolidayReader(pool))
	if err != nil {
		t.Fatal(err)
	}
	scheduler, err := NewEodIngestionScheduler(
		EodConfig{Enabled: true, BenchmarkBackfillYears: 10},
		discardLogger(),
		fixedNow,
		data.NewWatchlistReader(pool),
		data.NewHolidayReader(pool),
		data.NewCandleReader(pool),
		data.NewCandleReader(pool),
		ingester,
		func(ctx context.Context, fn func(context.Context) error) (bool, error) {
			return db.WithAdvisoryLock(ctx, pool, JobLockEODIngestion, fn)
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	scheduler.IngestLatestForAll(ctx)

	// Wednesday's run stores yesterday for the active symbol and the benchmark.
	assertSchedulerCandleSymbols(t, ctx, pool, []string{NIFTY50Symbol, "TCS"})
	if provider.fetches() != 2 {
		t.Fatalf("provider fetches = %d, want 2", provider.fetches())
	}

	// A repeated run must rely on the real existence query: no new rows, no fetches.
	scheduler.IngestLatestForAll(ctx)
	assertSchedulerCandleSymbols(t, ctx, pool, []string{NIFTY50Symbol, "TCS"})
	if provider.fetches() != 2 {
		t.Fatalf("provider fetches after repeat = %d, want 2 (stored candles short-circuit)", provider.fetches())
	}

	var date string
	if err := pool.QueryRow(ctx, `SELECT date::text FROM ohlcv_candles WHERE symbol = 'TCS'`).Scan(&date); err != nil {
		t.Fatal(err)
	}
	if date != "2026-09-15" {
		t.Fatalf("stored date = %s, want 2026-09-15", date)
	}
}

func assertSchedulerCandleSymbols(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want []string) {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT symbol FROM ohlcv_candles ORDER BY symbol`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			t.Fatal(err)
		}
		got = append(got, symbol)
	}
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
	if len(got) != len(want) {
		t.Fatalf("stored symbols = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stored symbols = %v, want %v", got, want)
		}
	}
}

func newSchedulerIntegrationPool(t *testing.T) (context.Context, string, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("authorized development database URL is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	config, err := db.ReadOnlyPoolConfig(databaseURL)
	if err != nil {
		t.Fatal("development database is not an authorized test target")
	}
	config.ConnConfig.RuntimeParams["default_transaction_read_only"] = "off"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("create scheduler test pool")
	}
	return ctx, databaseURL, pool
}

func createSchedulerTestSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "go_port_test_" + hex.EncodeToString(random[:])
	if !isSchedulerTestSchema(schema) {
		t.Fatalf("generated unsafe schema name %q", schema)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	return schema
}

func dropSchedulerTestSchema(t *testing.T, pool *pgxpool.Pool, schema string) {
	t.Helper()
	if !isSchedulerTestSchema(schema) {
		t.Fatalf("refusing to drop non-test schema %q", schema)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
		t.Errorf("drop isolated schema: %v", err)
	}
}

func isSchedulerTestSchema(name string) bool {
	const prefix = "go_port_test_"
	if len(name) != len(prefix)+16 || name[:len(prefix)] != prefix {
		return false
	}
	_, err := hex.DecodeString(name[len(prefix):])
	return err == nil
}
