package sqlc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func fyersSymbolRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "fy_token", "fyers_symbol", "trading_symbol", "name", "exchange",
		"segment", "lot_size", "tick_size", "isin", "updated_at",
	})
}

func TestGeneratedFyersSymbolQueriesReadAndWrite(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	updatedAt := pgtype.Timestamp{Time: time.Date(2026, 10, 25, 8, 45, 0, 0, time.UTC), Valid: true}
	row := []any{
		int64(7), "1000000000981", "NSE:ABC-EQ", "ABC", pgtype.Text{String: "Alpha", Valid: true}, "NSE",
		pgtype.Text{}, pgtype.Int4{Int32: 5, Valid: true}, mustNumeric(t, "0.0500"), pgtype.Text{String: "INE000A", Valid: true}, updatedAt,
	}

	mock.ExpectExec("DELETE FROM fyers_symbol_master").WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("INSERT INTO fyers_symbol_master").WithArgs(
		"1000000000981", "NSE:ABC-EQ", "ABC", pgtype.Text{String: "Alpha", Valid: true}, "NSE",
		pgtype.Text{}, pgtype.Int4{Int32: 5, Valid: true}, mustNumeric(t, "0.0500"),
		pgtype.Text{String: "INE000A", Valid: true}, updatedAt,
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery("COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(3)))
	mock.ExpectQuery("LIKE").WithArgs(pgtype.Text{String: "alp", Valid: true}, int32(10)).WillReturnRows(fyersSymbolRows().AddRow(row...))
	mock.ExpectQuery("UPPER").WithArgs("abc").WillReturnRows(fyersSymbolRows().AddRow(row...))
	mock.ExpectQuery("SELECT trading_symbol").WillReturnRows(pgxmock.NewRows([]string{"trading_symbol"}).AddRow("ABC"))
	mock.ExpectQuery("UPPER").WithArgs("NSE").WillReturnRows(fyersSymbolRows().AddRow(row...))

	queries := New(mock)
	if err := queries.DeleteAllFyersSymbols(context.Background()); err != nil {
		t.Fatalf("delete error = %v", err)
	}
	if err := queries.InsertFyersSymbol(context.Background(), InsertFyersSymbolParams{
		FyToken: "1000000000981", FyersSymbol: "NSE:ABC-EQ", TradingSymbol: "ABC",
		Name: pgtype.Text{String: "Alpha", Valid: true}, Exchange: "NSE",
		LotSize: pgtype.Int4{Int32: 5, Valid: true}, TickSize: mustNumeric(t, "0.0500"),
		Isin: pgtype.Text{String: "INE000A", Valid: true}, UpdatedAt: updatedAt,
	}); err != nil {
		t.Fatalf("insert error = %v", err)
	}
	if count, err := queries.CountFyersSymbols(context.Background()); err != nil || count != 3 {
		t.Fatalf("count = %d, %v", count, err)
	}
	if rows, err := queries.SearchFyersSymbols(context.Background(), SearchFyersSymbolsParams{
		Column1: pgtype.Text{String: "alp", Valid: true}, Limit: 10,
	}); err != nil || len(rows) != 1 || rows[0].TradingSymbol != "ABC" {
		t.Fatalf("search = %+v, %v", rows, err)
	}
	if rows, err := queries.FindFyersSymbolsByTradingSymbol(context.Background(), "abc"); err != nil || len(rows) != 1 {
		t.Fatalf("find = %+v, %v", rows, err)
	}
	if rows, err := queries.ListFyersTradingSymbols(context.Background()); err != nil || len(rows) != 1 || rows[0] != "ABC" {
		t.Fatalf("trading symbols = %v, %v", rows, err)
	}
	if rows, err := queries.ListFyersSymbolsByExchange(context.Background(), "NSE"); err != nil || len(rows) != 1 {
		t.Fatalf("exchange rows = %+v, %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedFyersSymbolQueriesReturnDatabaseErrors(t *testing.T) {
	for name, query := range map[string]func(*Queries) error{
		"delete": func(q *Queries) error { return q.DeleteAllFyersSymbols(context.Background()) },
		"insert": func(q *Queries) error {
			return q.InsertFyersSymbol(context.Background(), InsertFyersSymbolParams{FyToken: "1"})
		},
		"count": func(q *Queries) error {
			_, err := q.CountFyersSymbols(context.Background())
			return err
		},
		"search": func(q *Queries) error {
			_, err := q.SearchFyersSymbols(context.Background(), SearchFyersSymbolsParams{})
			return err
		},
		"find": func(q *Queries) error {
			_, err := q.FindFyersSymbolsByTradingSymbol(context.Background(), "ABC")
			return err
		},
		"list trading symbols": func(q *Queries) error {
			_, err := q.ListFyersTradingSymbols(context.Background())
			return err
		},
		"list by exchange": func(q *Queries) error {
			_, err := q.ListFyersSymbolsByExchange(context.Background(), "NSE")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			switch name {
			case "delete":
				mock.ExpectExec("DELETE").WillReturnError(errors.New("database unavailable"))
			case "insert":
				mock.ExpectExec("INSERT").WithArgs(
					pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
				).WillReturnError(errors.New("database unavailable"))
			default:
				mock.ExpectQuery("SELECT|COUNT").WillReturnError(errors.New("database unavailable"))
			}
			if err := query(New(mock)); err == nil {
				t.Error("query error = nil")
			}
		})
	}
}

func mustNumeric(t *testing.T, value string) pgtype.Numeric {
	t.Helper()
	var numeric pgtype.Numeric
	if err := numeric.Scan(value); err != nil {
		t.Fatal(err)
	}
	return numeric
}
