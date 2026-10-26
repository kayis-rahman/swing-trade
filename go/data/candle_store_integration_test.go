package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kayis-rahman/swing-trade/go/data/db"
)

func TestCandleWriterUsesIdempotentV75IdentityInIsolatedSchema(t *testing.T) {
	ctx, pool := newCandleIntegrationPool(t)
	defer pool.Close()
	schema := createCandleTestSchema(t, ctx, pool)
	defer dropCandleTestSchema(t, pool, schema)
	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()
	if _, err := connection.Exec(ctx, "SET search_path TO "+schema); err != nil {
		t.Fatal(err)
	}
	_, err = connection.Exec(ctx, `CREATE TABLE ohlcv_candles (
		symbol text NOT NULL, timeframe text NOT NULL, date date NOT NULL, bar_time time NOT NULL,
		open_price numeric(15,4), high_price numeric(15,4), low_price numeric(15,4), close_price numeric(15,4),
		volume bigint, adj_close_price numeric(15,4), UNIQUE (symbol, timeframe, date, bar_time)
	)`)
	if err != nil {
		t.Fatalf("create isolated candle table: %v", err)
	}
	writer, err := NewCandleWriter(connection)
	if err != nil {
		t.Fatal(err)
	}
	candle := validYahooCandle()
	inserted, err := writer.InsertIfAbsent(ctx, candle, "15m")
	if err != nil || !inserted {
		t.Fatalf("first insert = %v, %v", inserted, err)
	}
	inserted, err = writer.InsertIfAbsent(ctx, candle, "15m")
	if err != nil || inserted {
		t.Fatalf("duplicate insert = %v, %v", inserted, err)
	}
	var count int
	if err := connection.QueryRow(ctx, "SELECT count(*) FROM ohlcv_candles").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("candle count = %d, want 1", count)
	}
}

func newCandleIntegrationPool(t *testing.T) (context.Context, *pgxpool.Pool) {
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
		t.Fatal("create isolated candle test database pool")
	}
	return ctx, pool
}

func createCandleTestSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "go_port_test_" + hex.EncodeToString(random[:])
	if !isTestCandleSchema(schema) {
		t.Fatalf("generated unsafe schema name %q", schema)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	return schema
}

func dropCandleTestSchema(t *testing.T, pool *pgxpool.Pool, schema string) {
	t.Helper()
	if !isTestCandleSchema(schema) {
		t.Fatalf("refusing to drop non-test schema %q", schema)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
		t.Errorf("drop isolated schema: %v", err)
	}
}

func isTestCandleSchema(name string) bool {
	const prefix = "go_port_test_"
	if len(name) != len(prefix)+16 || name[:len(prefix)] != prefix {
		return false
	}
	_, err := hex.DecodeString(name[len(prefix):])
	return err == nil
}
