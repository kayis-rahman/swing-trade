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
	failAt  int
	calls   [][2]time.Time
}

func (stub *historicalProviderStub) FetchCandles(_ context.Context, _ string, from, to time.Time, _ string) ([]YahooCandle, error) {
	stub.calls = append(stub.calls, [2]time.Time{from, to})
	if stub.failAt > 0 {
		if len(stub.calls) == stub.failAt {
			return nil, stub.err
		}
		return stub.candles, nil
	}
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
	ingester, err := NewCandleIngester(&historicalProviderStub{candles: []YahooCandle{friday, saturday, sunday, invalid}}, writer, ingestionHolidayStub{
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
	ingester, err := NewCandleIngester(&historicalProviderStub{err: providerErr}, writer, ingestionHolidayStub{})
	if err != nil {
		t.Fatal(err)
	}
	outcome := ingester.IngestRange(context.Background(), "ABC", time.Now(), time.Now(), "1d", "D")
	if outcome.SourceOutcome != "TRANSIENT_SOURCE_FAILURE" || outcome.ErrorMessage != providerErr.Error() {
		t.Fatalf("provider outcome = %+v", outcome)
	}

	calendarErr := errors.New("holiday database unavailable")
	ingester, err = NewCandleIngester(&historicalProviderStub{candles: []YahooCandle{validYahooCandle()}}, writer, ingestionHolidayStub{err: calendarErr})
	if err != nil {
		t.Fatal(err)
	}
	outcome = ingester.IngestRange(context.Background(), "ABC", validYahooCandle().Date, validYahooCandle().Date, "1d", "D")
	if outcome.SourceOutcome != "TRANSIENT_SOURCE_FAILURE" || outcome.ErrorMessage == "" {
		t.Fatalf("calendar outcome = %+v", outcome)
	}
}

func TestCandleIngesterUses365DayChunksAndStopsAfterProviderFailure(t *testing.T) {
	providerErr := errors.New("provider timeout")
	provider := &historicalProviderStub{candles: []YahooCandle{validYahooCandle()}, err: providerErr, failAt: 2}
	writer, err := NewCandleWriter(&candleExecStub{result: pgconn.NewCommandTag("INSERT 0 1")})
	if err != nil {
		t.Fatal(err)
	}
	ingester, err := NewCandleIngester(provider, writer, ingestionHolidayStub{})
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 365)
	candle := validYahooCandle()
	candle.Date = from
	provider.candles = []YahooCandle{candle}
	outcome := ingester.IngestRange(context.Background(), "ABC", from, to, "1d", "D")
	if outcome.SourceOutcome != "TRANSIENT_SOURCE_FAILURE" || outcome.FetchedRows != 1 || outcome.SavedRows != 1 || outcome.ErrorMessage != providerErr.Error() {
		t.Fatalf("outcome = %+v", outcome)
	}
	if len(provider.calls) != 2 {
		t.Fatalf("provider calls = %d, want first window plus failed second window", len(provider.calls))
	}
	if got := provider.calls[0][1].Sub(provider.calls[0][0]); got != 364*24*time.Hour {
		t.Fatalf("first chunk length = %v", got)
	}
	if !provider.calls[1][0].Equal(provider.calls[0][1].AddDate(0, 0, 1)) {
		t.Fatalf("second chunk start = %v", provider.calls[1][0])
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
	ingester, err := NewCandleIngester(&historicalProviderStub{}, writer, ingestionHolidayStub{})
	if err != nil {
		t.Fatal(err)
	}
	outcome := ingester.IngestRange(context.Background(), "ABC", time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC), "1d", "D")
	if outcome.SourceOutcome != "INVALID_REQUEST" {
		t.Fatalf("outcome = %+v", outcome)
	}
}
