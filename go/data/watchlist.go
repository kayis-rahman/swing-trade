package data

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kayis-rahman/swing-trade/go/data/sqlc"
)

// WatchlistEntry is the read model serialized by the watchlist API.
type WatchlistEntry struct {
	ID           int64
	Symbol       string
	Name         *string
	Exchange     *string
	IsActive     *bool
	AddedAt      *time.Time
	LastSyncedAt *time.Time
	CandleCount  *int32
}

// WatchlistReadStore exposes read-only watchlist queries to the API layer.
type WatchlistReadStore interface {
	ListActiveWatchlist(context.Context) ([]WatchlistEntry, error)
	ListAllWatchlist(context.Context) ([]WatchlistEntry, error)
}

// WatchlistReader executes generated read-only watchlist queries.
type WatchlistReader struct {
	queries *sqlc.Queries
}

// NewWatchlistReader builds a watchlist reader on a pgx-compatible database handle.
func NewWatchlistReader(db sqlc.DBTX) *WatchlistReader {
	return &WatchlistReader{queries: sqlc.New(db)}
}

// ListActiveWatchlist returns active entries ordered by symbol.
func (r *WatchlistReader) ListActiveWatchlist(ctx context.Context) ([]WatchlistEntry, error) {
	rows, err := r.queries.ListActiveWatchlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active watchlist: %w", err)
	}
	return mapWatchlistRows(rows), nil
}

// ListAllWatchlist returns all entries ordered by id.
func (r *WatchlistReader) ListAllWatchlist(ctx context.Context) ([]WatchlistEntry, error) {
	rows, err := r.queries.ListAllWatchlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("list watchlist: %w", err)
	}
	return mapWatchlistRows(rows), nil
}

func mapWatchlistRows(rows []sqlc.Watchlist) []WatchlistEntry {
	entries := make([]WatchlistEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, WatchlistEntry{
			ID:           row.ID,
			Symbol:       row.Symbol,
			Name:         nullableText(row.Name),
			Exchange:     nullableText(row.Exchange),
			IsActive:     nullableBool(row.IsActive),
			AddedAt:      nullableTimestamp(row.AddedAt),
			LastSyncedAt: nullableTimestamp(row.LastSyncedAt),
			CandleCount:  nullableInt32(row.CandleCount),
		})
	}
	return entries
}

func nullableText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableTimestamp(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullableBool(value pgtype.Bool) *bool {
	if !value.Valid {
		return nil
	}
	return &value.Bool
}

func nullableInt32(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}
