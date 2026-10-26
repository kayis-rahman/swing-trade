package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kayis-rahman/swing-trade/go/data/db"
)

func TestWatchlistReaderUsesAuthorizedDevelopmentDatabaseReadOnly(t *testing.T) {
	if os.Getenv("SWINGTRADE_TEST_ALLOW_LOCAL_DATABASE") == "1" {
		t.Skip("local throwaway database has no migrated public watchlist table")
	}
	databaseURL := os.Getenv("SWINGTRADE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("authorized read-only development database URL is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.NewReadOnlyPool(ctx, databaseURL)
	if err != nil {
		t.Fatal("authorized development database connection failed")
	}
	defer pool.Close()
	var readOnly string
	if err := pool.QueryRow(ctx, "SHOW default_transaction_read_only").Scan(&readOnly); err != nil {
		t.Fatal("could not verify database read-only mode")
	}
	if readOnly != "on" {
		t.Fatalf("default_transaction_read_only = %q, want on", readOnly)
	}
	reader := NewWatchlistReader(pool)
	active, err := reader.ListActiveWatchlist(ctx)
	if err != nil {
		t.Fatal("active watchlist query failed")
	}
	for i, row := range active {
		if row.IsActive == nil || !*row.IsActive {
			t.Fatalf("active query returned an inactive or null row: %q", row.Symbol)
		}
		if i > 0 && active[i-1].Symbol > row.Symbol {
			t.Fatalf("active watchlist is not ordered by symbol: %q before %q", active[i-1].Symbol, row.Symbol)
		}
	}
	all, err := reader.ListAllWatchlist(ctx)
	if err != nil {
		t.Fatal("full watchlist query failed")
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID > all[i].ID {
			t.Fatalf("full watchlist is not ordered by id: %d before %d", all[i-1].ID, all[i].ID)
		}
	}
}
