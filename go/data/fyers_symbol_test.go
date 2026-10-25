package data

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
)

func eqCSVRow(fyToken, fyersSymbol, tradingSymbol, name, lotSize, tickSize, isin string) string {
	columns := make([]string, fyersMinimumColumns)
	for i := range columns {
		columns[i] = "x"
	}
	columns[fyersColumnFyToken] = fyToken
	columns[fyersColumnName] = name
	columns[fyersColumnLotSize] = lotSize
	columns[fyersColumnTickSize] = tickSize
	columns[fyersColumnISIN] = isin
	columns[fyersColumnFyersSymbol] = fyersSymbol
	columns[fyersColumnTradingSymbol] = tradingSymbol
	return strings.Join(columns, ",")
}

func TestParseFyersSymbolMasterCSVMapsJavaColumns(t *testing.T) {
	now := time.Date(2026, 10, 25, 8, 45, 0, 0, time.UTC)
	contents := strings.Join([]string{
		eqCSVRow("1000000000981", "NSE:RELIANCE-EQ", "RELIANCE", "Reliance Industries", "75", "0.05", "INE002A01018"),
		eqCSVRow("1000000000123", "NSE:ZZZ-EQ", "ZZZ", "", "abc", "not-a-number", ""),
		"too,short",
		strings.Join([]string{"NSE:ABC-EQ", "derivatives", "x", "x", "x", "x", "x", "x", "x", "NSE:ABC-FUT", "x", "x", "x", "ABC"}, ","),
	}, "\n")

	rows, skipped := ParseFyersSymbolMasterCSV(contents, now)

	if len(rows) != 2 || skipped != 1 {
		t.Fatalf("rows = %d, skipped = %d, want 2 rows and 1 skipped", len(rows), skipped)
	}
	reliance := rows[0]
	if reliance.FyToken != "1000000000981" || reliance.FyersSymbol != "NSE:RELIANCE-EQ" ||
		reliance.TradingSymbol != "RELIANCE" || reliance.Exchange != "NSE" ||
		reliance.Name == nil || *reliance.Name != "Reliance Industries" ||
		reliance.LotSize == nil || *reliance.LotSize != 75 ||
		reliance.TickSize == nil || *reliance.TickSize != "0.05" ||
		reliance.ISIN == nil || *reliance.ISIN != "INE002A01018" ||
		!reliance.UpdatedAt.Equal(now) {
		t.Fatalf("mapped row = %+v", reliance)
	}
	optional := rows[1]
	if optional.LotSize != nil || optional.TickSize != nil || optional.ISIN != nil ||
		optional.Name == nil || *optional.Name != "" {
		t.Fatalf("invalid optional fields = %+v", optional)
	}
}

func TestParseFyersSymbolMasterCSVDropsNonEquityAndEmptyIdentity(t *testing.T) {
	contents := strings.Join([]string{
		eqCSVRow("1", "NSE:ABC-EQ", "ABC", "Alpha", "1", "1", ""),
		eqCSVRow("2", "NSE:ABC-FUT", "ABC", "Alpha", "1", "1", ""),
		eqCSVRow("", "NSE:ABC-EQ", "ABC", "Alpha", "1", "1", ""),
		eqCSVRow("3", "NSE:ABC-EQ", "", "Alpha", "1", "1", ""),
	}, "\n")

	rows, skipped := ParseFyersSymbolMasterCSV(contents, time.Now())

	if len(rows) != 1 || rows[0].FyToken != "1" {
		t.Fatalf("rows = %+v, want only token 1", rows)
	}
	if skipped != 2 {
		t.Fatalf("skipped = %d, want 2 identity-less rows", skipped)
	}
}

func TestParseFyersSymbolMasterCSVStopsOnMalformedQuoting(t *testing.T) {
	contents := eqCSVRow("1", "NSE:ABC-EQ", "ABC", "Alpha", "1", "1", "") +
		"\n\"unterminated-quote,rest\n"
	rows, _ := ParseFyersSymbolMasterCSV(contents, time.Now())
	if len(rows) != 1 {
		t.Fatalf("rows before parse failure = %d, want the first valid row", len(rows))
	}
}

type fakeSymbolDownloader struct {
	body string
	err  error
}

func (f fakeSymbolDownloader) Download(context.Context) (string, error) {
	return f.body, f.err
}

type fakeSymbolReplacer struct {
	replaceErr error
	replace    []FyersSymbol
	count      int64
	countErr   error
}

func (f *fakeSymbolReplacer) ReplaceFyersSymbols(_ context.Context, rows []FyersSymbol) error {
	f.replace = rows
	return f.replaceErr
}

func (f *fakeSymbolReplacer) CountFyersSymbols(context.Context) (int64, error) {
	return f.count, f.countErr
}

func newTestSymbolMaster(t *testing.T, downloader FyersSymbolCSVDownloader, store FyersSymbolReplacer) *FyersSymbolMaster {
	t.Helper()
	master, err := NewFyersSymbolMaster(downloader, store, func() time.Time {
		return time.Date(2026, 10, 25, 8, 45, 0, 0, time.UTC)
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return master
}

func TestRefreshReplacesStoredSymbols(t *testing.T) {
	store := &fakeSymbolReplacer{}
	master := newTestSymbolMaster(t, fakeSymbolDownloader{body: eqCSVRow("1", "NSE:ABC-EQ", "ABC", "Alpha", "5", "0.005", "") + "\n"}, store)

	count, err := master.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if count != 1 || len(store.replace) != 1 || store.replace[0].TradingSymbol != "ABC" {
		t.Fatalf("refresh count = %d, replaced = %+v", count, store.replace)
	}
}

func TestRefreshNeverWipesOnDownloadProblems(t *testing.T) {
	for name, download := range map[string]FyersSymbolCSVDownloader{
		"download error": fakeSymbolDownloader{err: errors.New("network down")},
		"empty body":     fakeSymbolDownloader{body: "  \n"},
	} {
		t.Run(name, func(t *testing.T) {
			store := &fakeSymbolReplacer{}
			master := newTestSymbolMaster(t, download, store)
			if _, err := master.Refresh(context.Background()); err == nil {
				t.Fatal("Refresh() error = nil")
			}
			if store.replace != nil {
				t.Fatal("stored symbols were replaced despite a failed download")
			}
		})
	}
}

func TestRefreshPropagatesReplacementFailures(t *testing.T) {
	store := &fakeSymbolReplacer{replaceErr: errors.New("insert failed")}
	master := newTestSymbolMaster(t, fakeSymbolDownloader{body: eqCSVRow("1", "NSE:ABC-EQ", "ABC", "Alpha", "5", "0.005", "") + "\n"}, store)
	if count, err := master.Refresh(context.Background()); err == nil || count != 0 {
		t.Fatalf("Refresh() = %d, %v, want a failure", count, err)
	}
}

func TestNewFyersSymbolMasterValidatesDependencies(t *testing.T) {
	if _, err := NewFyersSymbolMaster(nil, nil, nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
}

func TestFyersSymbolMasterClientHandlesResponses(t *testing.T) {
	t.Run("returns body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "csv-body")
		}))
		defer server.Close()
		body, err := testSymbolMasterClient(t, server.URL).Download(context.Background())
		if err != nil || body != "csv-body" {
			t.Fatalf("Download() = %q, %v", body, err)
		}
	})

	t.Run("rejects non-200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()
		if _, err := testSymbolMasterClient(t, server.URL).Download(context.Background()); err == nil {
			t.Fatal("Download() error = nil")
		}
	})

	t.Run("refuses oversized bodies", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, fyersSymbolMasterDownloadLimit+1))
		}))
		defer server.Close()
		if _, err := testSymbolMasterClient(t, server.URL).Download(context.Background()); err == nil {
			t.Fatal("oversized Download() error = nil")
		}
	})
}

func testSymbolMasterClient(t *testing.T, url string) *FyersSymbolMasterClient {
	t.Helper()
	client := NewFyersSymbolMasterClient(nil)
	client.url = url
	return client
}

func TestFyersSymbolStoreReadersUseGeneratedQueries(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("COUNT").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(2)))
	mock.ExpectQuery("LIKE").WithArgs(pgtype.Text{String: "rel", Valid: true}, int32(10)).WillReturnRows(
		pgxmock.NewRows([]string{"id", "fy_token", "fyers_symbol", "trading_symbol", "name", "exchange", "segment", "lot_size", "tick_size", "isin", "updated_at"}))
	mock.ExpectQuery("UPPER").WithArgs("RELIANCE").WillReturnRows(
		pgxmock.NewRows([]string{"id", "fy_token", "fyers_symbol", "trading_symbol", "name", "exchange", "segment", "lot_size", "tick_size", "isin", "updated_at"}))
	mock.ExpectQuery("trading_symbol").WillReturnRows(pgxmock.NewRows([]string{"trading_symbol"}).AddRow("INFY").AddRow("TCS"))
	mock.ExpectQuery("exchange").WithArgs("NSE").WillReturnRows(
		pgxmock.NewRows([]string{"id", "fy_token", "fyers_symbol", "trading_symbol", "name", "exchange", "segment", "lot_size", "tick_size", "isin", "updated_at"}))

	store := NewFyersSymbolStore(mock)
	count, err := store.CountFyersSymbols(context.Background())
	if err != nil || count != 2 {
		t.Fatalf("count = %d, %v", count, err)
	}
	if _, err := store.SearchFyersSymbols(context.Background(), "rel", 10); err != nil {
		t.Fatalf("search error = %v", err)
	}
	if _, err := store.FindFyersSymbolsByTradingSymbol(context.Background(), "RELIANCE"); err != nil {
		t.Fatalf("find error = %v", err)
	}
	symbols, err := store.ListFyersTradingSymbols(context.Background())
	if err != nil || len(symbols) != 2 || symbols[0] != "INFY" || symbols[1] != "TCS" {
		t.Fatalf("trading symbols = %v, %v", symbols, err)
	}
	if _, err := store.ListFyersSymbolsByExchange(context.Background(), "NSE"); err != nil {
		t.Fatalf("exchange list error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFyersSymbolStoreReplaceCommitsAllRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	tick := "0.0500"
	name := "Alpha"
	lot := int32(5)
	isin := "INE000A"
	updatedAt := time.Date(2026, 10, 25, 8, 45, 0, 0, time.UTC)
	rows := []FyersSymbol{{
		FyToken: "1", FyersSymbol: "NSE:ABC-EQ", TradingSymbol: "ABC",
		Name: &name, Exchange: "NSE", LotSize: &lot, TickSize: &tick, ISIN: &isin, UpdatedAt: updatedAt,
	}, {
		FyToken: "2", FyersSymbol: "NSE:DEF-EQ", TradingSymbol: "DEF",
		Name: &name, Exchange: "NSE", UpdatedAt: updatedAt,
	}}
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM fyers_symbol_master").WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("INSERT INTO fyers_symbol_master").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec("INSERT INTO fyers_symbol_master").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	if err := NewFyersSymbolStore(mock).ReplaceFyersSymbols(context.Background(), rows); err != nil {
		t.Fatalf("ReplaceFyersSymbols() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFyersSymbolStoreReplaceRollsBackOnInsertFailure(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	name := "Alpha"
	updatedAt := time.Date(2026, 10, 25, 8, 45, 0, 0, time.UTC)
	rows := []FyersSymbol{{
		FyToken: "1", FyersSymbol: "NSE:ABC-EQ", TradingSymbol: "ABC",
		Name: &name, Exchange: "NSE", UpdatedAt: updatedAt,
	}}
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM fyers_symbol_master").WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec("INSERT INTO fyers_symbol_master").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnError(errors.New("unique violation"))
	mock.ExpectRollback()

	if err := NewFyersSymbolStore(mock).ReplaceFyersSymbols(context.Background(), rows); err == nil {
		t.Fatal("ReplaceFyersSymbols() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
