package data

import (
	"context"
	"errors"
	"testing"
	"time"
)

type failingCandleStore struct{ err error }

func (f failingCandleStore) GetDailyDateWindow(context.Context, string) (CandleDateWindow, error) {
	return CandleDateWindow{}, f.err
}

func (f failingCandleStore) CountDailyCandles(context.Context, string) (int64, error) {
	return 0, f.err
}

func TestIngestionStatusComposesWatchlistAndDailyCoverage(t *testing.T) {
	earliest := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	synced := time.Date(2026, 10, 24, 16, 30, 0, 0, time.UTC)
	watchlist := &intradayWatchlist{rows: []WatchlistEntry{{
		Symbol: "TCS", Name: stringPtrStatus("Tata Consultancy"), Exchange: stringPtrStatus("NSE"),
		IsActive: boolPtrStatus(true), LastSyncedAt: &synced,
	}}}
	candles := candleReadStub{
		count:  1000,
		window: CandleDateWindow{Earliest: &earliest, Latest: &latest},
	}
	service, err := NewIngestionStatusService(watchlist, candles, func() time.Time {
		return time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := service.IngestionStatus(context.Background())
	if err != nil {
		t.Fatalf("IngestionStatus() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	entry := entries[0]
	if entry.Symbol != "TCS" || entry.CandleCount != 1000 || !entry.HasData ||
		entry.LastCandleDate == nil || !entry.LastCandleDate.Equal(latest) ||
		entry.EarliestCandleDate == nil || !entry.EarliestCandleDate.Equal(earliest) ||
		entry.LastSyncedAt == nil || entry.DataQuality == "no_data" ||
		entry.Name == nil || *entry.Name != "Tata Consultancy" {
		t.Fatalf("status entry = %+v", entry)
	}
}

func TestIngestionStatusReportsNoDataForEmptyCandles(t *testing.T) {
	watchlist := &intradayWatchlist{rows: []WatchlistEntry{{Symbol: "NEW"}}}
	service, err := NewIngestionStatusService(watchlist, candleReadStub{}, func() time.Time {
		return time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := service.IngestionStatus(context.Background())
	if err != nil {
		t.Fatalf("IngestionStatus() error = %v", err)
	}
	entry := entries[0]
	if entry.HasData || entry.DataQuality != "no_data" || entry.CandleCount != 0 ||
		entry.LastCandleDate != nil || entry.EarliestCandleDate != nil {
		t.Fatalf("empty status entry = %+v", entry)
	}
}

func TestIngestionStatusQualityBandsMatchJavaThresholds(t *testing.T) {
	// Two years back: int(730 / 365.25) == 1 year, so the expected day
	// count is 252 and coverage is count / 252.
	twoYearsBack := time.Date(2024, 10, 25, 0, 0, 0, 0, time.UTC)
	today := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		count    int64
		earliest *time.Time
		want     string
	}{
		"excellent at 95 percent":     {count: 240, earliest: &twoYearsBack, want: "excellent"},
		"good below 95 percent":       {count: 239, earliest: &twoYearsBack, want: "good"},
		"good at 80 percent":          {count: 202, earliest: &twoYearsBack, want: "good"},
		"partial below 80 percent":    {count: 201, earliest: &twoYearsBack, want: "partial"},
		"partial at 50 percent":       {count: 126, earliest: &twoYearsBack, want: "partial"},
		"poor below 50 percent":       {count: 125, earliest: &twoYearsBack, want: "poor"},
		"first year uses one day":     {count: 3, earliest: timePtrStatus(today.AddDate(0, 0, -100)), want: "excellent"},
		"missing earliest is no data": {count: 10, earliest: nil, want: "no_data"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := assessDataQuality(tc.earliest, tc.count, today); got != tc.want {
				t.Fatalf("assessDataQuality = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIngestionStatusPropagatesStoreFailures(t *testing.T) {
	failingWatchlist := &intradayWatchlist{err: errors.New("db down")}
	service, err := NewIngestionStatusService(failingWatchlist, candleReadStub{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.IngestionStatus(context.Background()); err == nil {
		t.Fatal("watchlist error = nil")
	}

	watchlist := &intradayWatchlist{rows: []WatchlistEntry{{Symbol: "TCS"}}}
	service, err = NewIngestionStatusService(watchlist, failingCandleStore{err: errors.New("db down")}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.IngestionStatus(context.Background()); err == nil {
		t.Fatal("candle store error = nil")
	}
}

func TestNewIngestionStatusServiceValidatesDependencies(t *testing.T) {
	if _, err := NewIngestionStatusService(nil, nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
}

func stringPtrStatus(value string) *string     { return &value }
func boolPtrStatus(value bool) *bool           { return &value }
func timePtrStatus(value time.Time) *time.Time { return &value }
