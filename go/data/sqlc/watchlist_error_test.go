package sqlc

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestGeneratedWatchlistQueriesReturnQueryAndScanErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query func(*Queries) error
		want  string
	}{
		{name: "active-query", query: func(q *Queries) error { _, err := q.ListActiveWatchlist(context.Background()); return err }, want: "active"},
		{name: "all-query", query: func(q *Queries) error { _, err := q.ListAllWatchlist(context.Background()); return err }, want: "all"},
		{name: "active-scan", query: func(q *Queries) error { _, err := q.ListActiveWatchlist(context.Background()); return err }, want: "scan-active"},
		{name: "all-scan", query: func(q *Queries) error { _, err := q.ListAllWatchlist(context.Background()); return err }, want: "scan-all"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			switch tc.want {
			case "active", "all":
				mock.ExpectQuery("FROM watchlist").WillReturnError(errors.New("query failed"))
			case "scan-active", "scan-all":
				mock.ExpectQuery("FROM watchlist").WillReturnRows(pgxmock.NewRows([]string{
					"id", "symbol", "name", "exchange", "is_active", "added_at", "last_synced_at", "candle_count",
				}).AddRow("not-an-id", "TCS", "TCS", "NSE", true, nil, nil, int64(0)))
			}
			if err := tc.query(New(mock)); err == nil {
				t.Fatal("query error = nil, want error")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
