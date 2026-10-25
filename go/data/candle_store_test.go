package data

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

type candleExecStub struct {
	query  string
	args   []any
	result pgconn.CommandTag
	err    error
}

func (stub *candleExecStub) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	stub.query, stub.args = query, args
	return stub.result, stub.err
}

func TestCandleWriterInsertsUsingJavaIdentityAndReportsDuplicate(t *testing.T) {
	stub := &candleExecStub{result: pgconn.NewCommandTag("INSERT 0 0")}
	writer, err := NewCandleWriter(stub)
	if err != nil {
		t.Fatal(err)
	}
	candle := validYahooCandle()
	inserted, err := writer.InsertIfAbsent(context.Background(), candle, "15m")
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("duplicate candle was reported as inserted")
	}
	if !strings.Contains(stub.query, "ON CONFLICT (symbol, timeframe, date, bar_time) DO NOTHING") {
		t.Fatalf("query = %s", stub.query)
	}
	if len(stub.args) != 10 || stub.args[0] != "ABC" || stub.args[1] != "15m" || stub.args[2] != "2026-10-23" || stub.args[3] != "09:30:00" || stub.args[8] != int64(100) {
		t.Fatalf("args = %#v", stub.args)
	}
	stub.result = pgconn.NewCommandTag("INSERT 0 1")
	inserted, err = writer.InsertIfAbsent(context.Background(), candle, "D")
	if err != nil || !inserted {
		t.Fatalf("inserted=%v err=%v", inserted, err)
	}
}

func TestCandleWriterRejectsInvalidInputsAndPropagatesDatabaseErrors(t *testing.T) {
	stub := &candleExecStub{result: pgconn.NewCommandTag("INSERT 0 1")}
	writer, err := NewCandleWriter(stub)
	if err != nil {
		t.Fatal(err)
	}
	candle := validYahooCandle()
	candle.High = candle.Low
	if _, err := writer.InsertIfAbsent(context.Background(), candle, "D"); err == nil {
		t.Fatal("expected invalid candle error")
	}
	if stub.query != "" {
		t.Fatal("invalid candle reached database")
	}
	wantErr := errors.New("database unavailable")
	stub.err = wantErr
	if _, err := writer.InsertIfAbsent(context.Background(), validYahooCandle(), "D"); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v", err)
	}
}

func TestNewCandleWriterRequiresDatabase(t *testing.T) {
	if _, err := NewCandleWriter(nil); err == nil {
		t.Fatal("expected nil database error")
	}
}

func validYahooCandle() YahooCandle {
	open, _ := coredecimal.Parse("10")
	high, _ := coredecimal.Parse("12")
	low, _ := coredecimal.Parse("9")
	closePrice, _ := coredecimal.Parse("11")
	return YahooCandle{Symbol: "ABC", Date: time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), BarTime: time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC), Open: open, High: high, Low: low, Close: closePrice, AdjustedClose: closePrice, Volume: 100}
}
