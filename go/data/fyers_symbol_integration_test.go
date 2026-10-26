package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kayis-rahman/swing-trade/go/data/db"
)

// newFyersSymbolTestStore creates an isolated symbol table behind a
// search-path-scoped pool and returns a cleanup function.
func newFyersSymbolTestStore(t *testing.T) (context.Context, *FyersSymbolStore, func()) {
	t.Helper()
	ctx, adminPool := newCandleIntegrationPool(t)
	schema := createCandleTestSchema(t, ctx, adminPool)

	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
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
	if _, err := pool.Exec(ctx, `CREATE TABLE fyers_symbol_master (
		id BIGSERIAL PRIMARY KEY,
		fy_token VARCHAR(32) NOT NULL UNIQUE,
		fyers_symbol VARCHAR(64) NOT NULL,
		trading_symbol VARCHAR(32) NOT NULL,
		name TEXT,
		exchange VARCHAR(16) NOT NULL DEFAULT 'NSE',
		segment VARCHAR(16),
		lot_size INTEGER,
		tick_size NUMERIC(10,4),
		isin VARCHAR(13),
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		pool.Close()
		dropCandleTestSchema(t, adminPool, schema)
		t.Fatalf("create isolated symbol table: %v", err)
	}
	cleanup := func() {
		pool.Close()
		dropCandleTestSchema(t, adminPool, schema)
		adminPool.Close()
	}
	return ctx, NewFyersSymbolStore(pool), cleanup
}

func testFyersSymbols(updatedAt time.Time) []FyersSymbol {
	name := "Alpha Industries"
	tick := "0.0500"
	lot := int32(5)
	isin := "INE000A01001"
	secondName := "Beta Ltd"
	return []FyersSymbol{{
		FyToken: "1", FyersSymbol: "NSE:ABC-EQ", TradingSymbol: "ABC",
		Name: &name, Exchange: "NSE", LotSize: &lot, TickSize: &tick, ISIN: &isin, UpdatedAt: updatedAt,
	}, {
		FyToken: "2", FyersSymbol: "NSE:BBB-EQ", TradingSymbol: "BBB",
		Name: &secondName, Exchange: "NSE", UpdatedAt: updatedAt,
	}}
}

func TestFyersSymbolStoreReplacesRowsThroughRealSQL(t *testing.T) {
	ctx, store, cleanup := newFyersSymbolTestStore(t)
	defer cleanup()
	updatedAt := time.Date(2026, 10, 25, 8, 45, 0, 0, time.UTC)
	rows := testFyersSymbols(updatedAt)

	if err := store.ReplaceFyersSymbols(ctx, rows[:1]); err != nil {
		t.Fatalf("first replacement: %v", err)
	}
	if count, err := store.CountFyersSymbols(ctx); err != nil || count != 1 {
		t.Fatalf("count after first replacement = %d, %v", count, err)
	}
	// The second replacement must wipe the first row, not append to it.
	if err := store.ReplaceFyersSymbols(ctx, rows); err != nil {
		t.Fatalf("second replacement: %v", err)
	}
	if count, err := store.CountFyersSymbols(ctx); err != nil || count != 2 {
		t.Fatalf("count after wipe-and-replace = %d, %v", count, err)
	}
}

func TestFyersSymbolStoreReadersThroughRealSQL(t *testing.T) {
	ctx, store, cleanup := newFyersSymbolTestStore(t)
	defer cleanup()
	updatedAt := time.Date(2026, 10, 25, 8, 45, 0, 0, time.UTC)
	if err := store.ReplaceFyersSymbols(ctx, testFyersSymbols(updatedAt)); err != nil {
		t.Fatalf("replacement: %v", err)
	}

	found, err := store.FindFyersSymbolsByTradingSymbol(ctx, "abc")
	if err != nil || len(found) != 1 {
		t.Fatalf("case-insensitive lookup = %+v, %v", found, err)
	}
	row := found[0]
	if row.TickSize == nil || *row.TickSize != "0.0500" ||
		row.LotSize == nil || *row.LotSize != 5 ||
		row.ISIN == nil || *row.ISIN != "INE000A01001" ||
		row.Name == nil || *row.Name != "Alpha Industries" {
		t.Fatalf("round-tripped row = %+v", row)
	}

	searched, err := store.SearchFyersSymbols(ctx, "alp", 10)
	if err != nil || len(searched) != 1 || searched[0].TradingSymbol != "ABC" {
		t.Fatalf("name-contains search = %+v, %v", searched, err)
	}
	symbols, err := store.ListFyersTradingSymbols(ctx)
	if err != nil || len(symbols) != 2 || symbols[0] != "ABC" || symbols[1] != "BBB" {
		t.Fatalf("trading symbols = %v, %v", symbols, err)
	}
	exchanged, err := store.ListFyersSymbolsByExchange(ctx, "nse")
	if err != nil || len(exchanged) != 2 {
		t.Fatalf("exchange lookup = %+v, %v", exchanged, err)
	}
}
