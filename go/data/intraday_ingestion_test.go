package data

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	coredecimal "github.com/kayis-rahman/swing-trade/go/core/decimal"
)

var (
	intradaySession     = time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC) // Tuesday
	intradayNextSession = time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC) // Wednesday
	intradayWindowStart = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	intradayNow         = time.Date(2026, 9, 16, 18, 0, 0, 0, time.UTC)
)

type intradayWatchlist struct {
	rows  []WatchlistEntry
	err   error
	calls int
}

func (w *intradayWatchlist) ListActiveWatchlist(context.Context) ([]WatchlistEntry, error) {
	w.calls++
	if w.err != nil {
		return nil, w.err
	}
	return w.rows, nil
}

func (w *intradayWatchlist) ListAllWatchlist(context.Context) ([]WatchlistEntry, error) {
	return w.rows, w.err
}

type intradayLatest struct {
	dates map[string]time.Time
	err   error
}

func (l *intradayLatest) LatestIntervalCandleDate(_ context.Context, symbol, timeframe string) (*time.Time, error) {
	if l.err != nil {
		return nil, l.err
	}
	date, ok := l.dates[symbol+"|"+timeframe]
	if !ok {
		return nil, nil
	}
	return &date, nil
}

type intradayHolidays struct {
	holidays map[string]Holiday
	err      error
}

func (h *intradayHolidays) GetHolidayByDate(_ context.Context, date time.Time) (*Holiday, error) {
	if h.err != nil {
		return nil, h.err
	}
	if holiday, ok := h.holidays[date.Format("2006-01-02")]; ok {
		return &holiday, nil
	}
	return nil, nil
}

func (h *intradayHolidays) ListHolidaysByDateRange(context.Context, time.Time, time.Time) ([]Holiday, error) {
	return nil, nil
}

func (h *intradayHolidays) ListUpcomingHolidays(context.Context, time.Time) ([]Holiday, error) {
	return nil, nil
}

type intradayFetch struct {
	symbol   string
	from     time.Time
	to       time.Time
	interval string
}

type intradayProvider struct {
	fetches map[string][]YahooCandle
	errs    map[string]error
	calls   []intradayFetch
}

func (p *intradayProvider) FetchCandles(_ context.Context, symbol string, from, to time.Time, interval string) ([]YahooCandle, error) {
	p.calls = append(p.calls, intradayFetch{symbol: symbol, from: from, to: to, interval: interval})
	if err, ok := p.errs[symbol]; ok {
		return nil, err
	}
	return p.fetches[symbol], nil
}

type execRecord struct {
	symbol    string
	timeframe string
	date      string
	barTime   string
}

type intradayExecQueue struct {
	results []pgconn.CommandTag
	errs    []error
	records []execRecord
	calls   int
}

func (q *intradayExecQueue) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	q.calls++
	if len(args) >= 4 {
		q.records = append(q.records, execRecord{
			symbol:    args[0].(string),
			timeframe: args[1].(string),
			date:      args[2].(string),
			barTime:   args[3].(string),
		})
	}
	if len(q.errs) > 0 && q.errs[0] != nil {
		err := q.errs[0]
		q.errs = q.errs[1:]
		return pgconn.CommandTag{}, err
	}
	if len(q.results) == 0 {
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	tag := q.results[0]
	q.results = q.results[1:]
	return tag, nil
}

func intradayBar(t *testing.T, symbol string, date, barTime time.Time, closePrice string) YahooCandle {
	t.Helper()
	closeValue := mustIntradayDecimal(t, closePrice)
	one := mustIntradayDecimal(t, "1")
	two := mustIntradayDecimal(t, "2")
	return YahooCandle{
		Symbol: symbol, Timeframe: "15m", Date: date, BarTime: barTime,
		Open: closeValue.Sub(one), High: closeValue.Add(one), Low: closeValue.Sub(two), Close: closeValue,
		AdjustedClose: closeValue, Volume: 500_000,
	}
}

func mustIntradayDecimal(t *testing.T, value string) coredecimal.Decimal {
	t.Helper()
	decimalValue, err := coredecimal.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return decimalValue
}

func newIntradayServiceForTest(t *testing.T) (*IntradayIngestion, *intradayWatchlist, *intradayLatest, *intradayHolidays, *intradayProvider, *intradayExecQueue) {
	t.Helper()
	watchlist := &intradayWatchlist{rows: []WatchlistEntry{{Symbol: "TCS"}}}
	latest := &intradayLatest{dates: map[string]time.Time{}}
	holidays := &intradayHolidays{holidays: map[string]Holiday{}}
	provider := &intradayProvider{fetches: map[string][]YahooCandle{}, errs: map[string]error{}}
	queue := &intradayExecQueue{}
	writer, err := NewCandleWriter(queue)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewIntradayIngestion(watchlist, latest, holidays, provider, writer,
		func() time.Time { return intradayNow },
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return service, watchlist, latest, holidays, provider, queue
}

func TestIngestWatchlistIntervalIsScopedToWatchlistAndInterval(t *testing.T) {
	service, watchlist, _, _, provider, queue := newIntradayServiceForTest(t)
	watchlist.rows = []WatchlistEntry{{Symbol: "TCS"}, {Symbol: "RELIANCE"}}
	provider.fetches["TCS"] = []YahooCandle{intradayBar(t, "TCS", intradaySession, barClock(9, 15), "4000")}
	provider.fetches["RELIANCE"] = nil
	queue.results = []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")}

	result, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, intradaySession)
	if err != nil {
		t.Fatalf("IngestWatchlistInterval() error = %v", err)
	}
	if result.SymbolsAttempted != 2 || result.SymbolsSucceeded != 2 || result.BarsSaved != 1 {
		t.Fatalf("result = %+v, want2 attempted,2 succeeded,1 saved", result)
	}
	if len(queue.records) != 1 {
		t.Fatalf("insert records = %+v", queue.records)
	}
	record := queue.records[0]
	if record.symbol != "TCS" || record.timeframe != "15m" || record.date != "2026-09-15" || record.barTime != "09:15:00" {
		t.Fatalf("insert identity = %+v", record)
	}
	if len(provider.calls) != 2 || provider.calls[0].interval != "15m" {
		t.Fatalf("provider calls = %+v", provider.calls)
	}
}

func TestIngestWatchlistIntervalDropsBarsOutsideSession(t *testing.T) {
	service, _, _, holidays, provider, queue := newIntradayServiceForTest(t)
	holidays.holidays["2026-09-16"] = Holiday{Date: intradayNextSession, Type: "FULL"}
	provider.fetches["TCS"] = []YahooCandle{
		intradayBar(t, "TCS", intradaySession, barClock(9, 0), "3990"),
		intradayBar(t, "TCS", intradaySession, barClock(9, 15), "3995"),
		intradayBar(t, "TCS", intradaySession, barClock(15, 15), "4100"),
		intradayBar(t, "TCS", intradaySession, barClock(15, 45), "4105"),
		intradayBar(t, "TCS", intradayNextSession, barClock(10, 0), "4200"),
	}
	queue.results = []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1"), pgconn.NewCommandTag("INSERT 0 1")}

	result, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, intradayNextSession)
	if err != nil {
		t.Fatalf("IngestWatchlistInterval() error = %v", err)
	}
	if result.BarsFetched != 5 || result.BarsSaved != 2 || result.BarsRejected != 3 {
		t.Fatalf("counts = fetched %d saved %d rejected %d, want5/2/3",
			result.BarsFetched, result.BarsSaved, result.BarsRejected)
	}
	if len(queue.records) != 2 {
		t.Fatalf("insert records = %+v", queue.records)
	}
	for _, record := range queue.records {
		if record.date != "2026-09-15" || record.timeframe != "15m" {
			t.Fatalf("persisted identity = %+v", record)
		}
	}
}

func TestIngestWatchlistIntervalResumesFromLatestStoredBar(t *testing.T) {
	service, watchlist, latest, _, provider, _ := newIntradayServiceForTest(t)
	watchlist.rows = []WatchlistEntry{{Symbol: "TCS"}, {Symbol: "RELIANCE"}}
	latest.dates["TCS|15m"] = intradaySession

	if _, err := service.IngestWatchlistInterval(context.Background(), "15m", intradayWindowStart, intradayNextSession); err != nil {
		t.Fatalf("IngestWatchlistInterval() error = %v", err)
	}
	if len(provider.calls) != 2 {
		t.Fatalf("provider calls = %+v", provider.calls)
	}
	tcs, reliance := provider.calls[0], provider.calls[1]
	if tcs.symbol != "TCS" || !tcs.from.Equal(intradaySession) || !tcs.to.Equal(intradayNextSession) {
		t.Fatalf("TCS resume fetch = %+v, want from latest stored date", tcs)
	}
	if reliance.symbol != "RELIANCE" || !reliance.from.Equal(intradayWindowStart) {
		t.Fatalf("RELIANCE fetch = %+v, want the full window", reliance)
	}
}

func TestIngestWatchlistIntervalFetchesFullWindowWhenLatestPredatesIt(t *testing.T) {
	service, _, latest, _, provider, _ := newIntradayServiceForTest(t)
	latest.dates["TCS|15m"] = time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)

	if _, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, intradayNextSession); err != nil {
		t.Fatalf("IngestWatchlistInterval() error = %v", err)
	}
	if len(provider.calls) != 1 || !provider.calls[0].from.Equal(intradaySession) {
		t.Fatalf("fetch = %+v, want the requested window start", provider.calls)
	}
}

func TestIngestWatchlistIntervalIsIdempotentAcrossRuns(t *testing.T) {
	service, _, _, _, provider, queue := newIntradayServiceForTest(t)
	provider.fetches["TCS"] = []YahooCandle{intradayBar(t, "TCS", intradaySession, barClock(9, 15), "4000")}
	queue.results = []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1"), pgconn.NewCommandTag("INSERT 0 0")}

	first, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, intradaySession)
	if err != nil {
		t.Fatalf("first run error = %v", err)
	}
	second, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, intradaySession)
	if err != nil {
		t.Fatalf("second run error = %v", err)
	}
	if first.BarsSaved != 1 || second.BarsSaved != 0 || second.SymbolsSucceeded != 1 {
		t.Fatalf("first = %+v, second = %+v", first, second)
	}
}

func TestIngestWatchlistIntervalIsolatesPerSymbolFailures(t *testing.T) {
	service, watchlist, _, _, provider, queue := newIntradayServiceForTest(t)
	watchlist.rows = []WatchlistEntry{{Symbol: "TCS"}, {Symbol: "RELIANCE"}}
	provider.errs["TCS"] = errors.New("provider timeout")
	provider.fetches["RELIANCE"] = []YahooCandle{intradayBar(t, "RELIANCE", intradaySession, barClock(9, 15), "2500")}
	queue.results = []pgconn.CommandTag{pgconn.NewCommandTag("INSERT 0 1")}

	result, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, intradaySession)
	if err != nil {
		t.Fatalf("IngestWatchlistInterval() error = %v", err)
	}
	if result.SymbolsSucceeded != 1 || result.SymbolsFailed != 1 || len(result.Errors) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if !strings.HasPrefix(result.Errors[0], "TCS") {
		t.Fatalf("error message = %q, want the symbol prefix", result.Errors[0])
	}
	if result.BarsSaved != 1 {
		t.Fatalf("bars saved = %d, want the surviving symbol's bar", result.BarsSaved)
	}
}

func TestIngestWatchlistIntervalAbortsOnUnsupportedInterval(t *testing.T) {
	service, _, _, _, provider, _ := newIntradayServiceForTest(t)
	provider.errs["TCS"] = ErrUnsupportedInterval

	_, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, intradaySession)
	if !errors.Is(err, ErrUnsupportedInterval) {
		t.Fatalf("unsupported interval error = %v", err)
	}
}

func TestIngestWatchlistIntervalRejectsInvalidWindows(t *testing.T) {
	service, _, _, _, _, _ := newIntradayServiceForTest(t)
	if _, err := service.IngestWatchlistInterval(context.Background(), "15m", intradayNextSession, intradaySession); err == nil {
		t.Fatal("reversed window accepted")
	}
	if _, err := service.IngestWatchlistInterval(context.Background(), "", intradaySession, intradaySession); err == nil {
		t.Fatal("empty interval accepted")
	}
}

func TestIngestWatchlistIntervalSkipsFutureWindows(t *testing.T) {
	service, watchlist, _, _, provider, _ := newIntradayServiceForTest(t)
	futureFrom := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	futureTo := time.Date(2099, 1, 2, 0, 0, 0, 0, time.UTC)

	result, err := service.IngestWatchlistInterval(context.Background(), "15m", futureFrom, futureTo)
	if err != nil {
		t.Fatalf("IngestWatchlistInterval() error = %v", err)
	}
	if result.SymbolsAttempted != 0 {
		t.Fatalf("attempted = %d, want0", result.SymbolsAttempted)
	}
	if watchlist.calls != 0 || len(provider.calls) != 0 {
		t.Fatalf("future window touched dependencies: watchlist=%d fetches=%d", watchlist.calls, len(provider.calls))
	}
}

func TestIngestWatchlistIntervalClampsEndToTodayIST(t *testing.T) {
	service, _, _, _, _, _ := newIntradayServiceForTest(t)
	futureTo := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	result, err := service.IngestWatchlistInterval(context.Background(), "15m", intradaySession, futureTo)
	if err != nil {
		t.Fatalf("IngestWatchlistInterval() error = %v", err)
	}
	wantTo := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	if !result.To.Equal(wantTo) {
		t.Fatalf("clamped end = %s, want %s", result.To, wantTo)
	}
}

func TestNewIntradayIngestionValidatesDependencies(t *testing.T) {
	if _, err := NewIntradayIngestion(nil, nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
}

func barClock(hour, minute int) time.Time {
	return time.Date(2000, 1, 1, hour, minute, 0, 0, time.UTC)
}
