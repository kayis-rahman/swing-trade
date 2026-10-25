package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type historicalProviderStub struct {
	candles []YahooCandle
	err     error
}

func (stub historicalProviderStub) FetchCandles(context.Context, string, time.Time, time.Time, string) ([]YahooCandle, error) {
	return stub.candles, stub.err
}

type ingestionHolidayStub struct {
	dates map[string]*Holiday
	err   error
}

func (stub ingestionHolidayStub) GetHolidayByDate(_ context.Context, date time.Time) (*Holiday, error) {
	return stub.dates[date.Format(time.DateOnly)], stub.err
}
func (ingestionHolidayStub) ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]Holiday, error) {
	return []Holiday{}, nil
}
func (ingestionHolidayStub) ListUpcomingHolidays(context.Context, time.Time) ([]Holiday, error) {
	return []Holiday{}, nil
}

func TestCandleIngesterFiltersNonSessionsAndPersistsValidRows(t *testing.T) {
	friday := validYahooCandle()
	saturday := validYahooCandle()
	saturday.Date = time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	sunday := validYahooCandle()
	sunday.Date = time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	invalid := validYahooCandle()
	invalid.High = invalid.Low
	writerStub := &candleExecStub{result: pgconn.NewCommandTag("INSERT 0 1")}
	writer, err := NewCandleWriter(writerStub)
	if err != nil {
		t.Fatal(err)
	}
	ingester, err := NewCandleIngester(historicalProviderStub{candles: []YahooCandle{friday, saturday, sunday, invalid}}, writer, ingestionHolidayStub{
		dates: map[string]*Holiday{"2026-10-24": {Type: "FULL"}, "2026-10-25": {Type: "PARTIAL"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := ingester.IngestRange(context.Background(), "ABC", time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC), "1d", "D")
	if outcome.SourceOutcome != "DATA_RECEIVED" || outcome.FetchedRows != 4 || outcome.SavedRows != 2 || outcome.InvalidRows != 2 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if writerStub.args[1] != "D" {
		t.Fatalf("stored timeframe = %v", writerStub.args[1])
	}
}

func TestCandleIngesterReturnsProviderAndCalendarFailures(t *testing.T) {
	writer, err := NewCandleWriter(&candleExecStub{result: pgconn.NewCommandTag("INSERT 0 1")})
	if err != nil {
		t.Fatal(err)
	}
	providerErr := errors.New("provider unavailable")
	ingester, err := NewCandleIngester(historicalProviderStub{err: providerErr}, writer, ingestionHolidayStub{})
	if err != nil {
		t.Fatal(err)
	}
	outcome := ingester.IngestRange(context.Background(), "ABC", time.Now(), time.Now(), "1d", "D")
	if outcome.SourceOutcome != "TRANSIENT_SOURCE_FAILURE" || outcome.ErrorMessage != providerErr.Error() {
		t.Fatalf("provider outcome = %+v", outcome)
	}

	calendarErr := errors.New("holiday database unavailable")
	ingester, err = NewCandleIngester(historicalProviderStub{candles: []YahooCandle{validYahooCandle()}}, writer, ingestionHolidayStub{err: calendarErr})
	if err != nil {
		t.Fatal(err)
	}
	outcome = ingester.IngestRange(context.Background(), "ABC", validYahooCandle().Date, validYahooCandle().Date, "1d", "D")
	if outcome.SourceOutcome != "TRANSIENT_SOURCE_FAILURE" || outcome.ErrorMessage == "" {
		t.Fatalf("calendar outcome = %+v", outcome)
	}
}

func TestNewCandleIngesterRequiresPortsAndRejectsInvalidRange(t *testing.T) {
	if _, err := NewCandleIngester(nil, nil, nil); err == nil {
		t.Fatal("expected missing dependency error")
	}
	writer, err := NewCandleWriter(&candleExecStub{})
	if err != nil {
		t.Fatal(err)
	}
	ingester, err := NewCandleIngester(historicalProviderStub{}, writer, ingestionHolidayStub{})
	if err != nil {
		t.Fatal(err)
	}
	outcome := ingester.IngestRange(context.Background(), "ABC", time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), "1d", "D")
	if outcome.SourceOutcome != "INVALID_REQUEST" {
		t.Fatalf("outcome = %+v", outcome)
	}
}
