package data

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeCoverage struct {
	verified bool
	err      error
	calls    int
}

func (f *fakeCoverage) IsCoverageVerified(context.Context, time.Time, time.Time) (bool, error) {
	f.calls++
	return f.verified, f.err
}

type fakeRangeStore struct {
	rows []DailyCandleRow
	err  error
}

func (f *fakeRangeStore) DailyCandlesInRange(context.Context, string, time.Time, time.Time) ([]DailyCandleRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func newReconcilerForTest(t *testing.T, coverage *fakeCoverage, stored *fakeRangeStore, provider HistoricalCandleProvider) *Reconciler {
	t.Helper()
	reconciler, err := NewReconciler(coverage, &intradayHolidays{holidays: map[string]Holiday{}}, stored, provider)
	if err != nil {
		t.Fatal(err)
	}
	return reconciler
}

func utcDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestReconciliationPreviewRejectsInvalidRanges(t *testing.T) {
	reconciler := newReconcilerForTest(t, &fakeCoverage{verified: true}, &fakeRangeStore{}, &intradayProvider{})
	for name, tc := range map[string]struct {
		symbol string
		from   time.Time
		to     time.Time
	}{
		"reversed range": {symbol: "TCS", from: utcDay(2026, 9, 18), to: utcDay(2026, 9, 14)},
		"eleven days":    {symbol: "TCS", from: utcDay(2026, 9, 1), to: utcDay(2026, 9, 12)},
		"empty symbol":   {symbol: "", from: utcDay(2026, 9, 14), to: utcDay(2026, 9, 18)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := reconciler.PreviewReconciliation(context.Background(), tc.symbol, tc.from, tc.to); !errors.Is(err, ErrInvalidReconciliationRange) {
				t.Fatalf("error = %v, want ErrInvalidReconciliationRange", err)
			}
		})
	}
}

func TestReconciliationPreviewStopsWhenCoverageUnverified(t *testing.T) {
	coverage := &fakeCoverage{verified: false}
	provider := &intradayProvider{}
	reconciler := newReconcilerForTest(t, coverage, &fakeRangeStore{rows: []DailyCandleRow{{ID: 1, Date: utcDay(2026, 9, 15)}}}, provider)

	report, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 18))
	if err != nil {
		t.Fatalf("PreviewReconciliation() error = %v", err)
	}
	if report.Status != "CALENDAR_INCOMPLETE" || len(report.Expected) != 0 || report.Count != 0 {
		t.Fatalf("calendar-incomplete report = %+v", report)
	}
	if coverage.calls != 1 || len(provider.calls) != 0 {
		t.Fatalf("coverage gate bypassed: coverage=%d fetches=%d", coverage.calls, len(provider.calls))
	}
}

func TestReconciliationPreviewComputesCompletedReport(t *testing.T) {
	provider := &intradayProvider{fetches: map[string][]YahooCandle{}}
	reconciler := newReconcilerForTest(t, &fakeCoverage{verified: true}, &fakeRangeStore{rows: []DailyCandleRow{
		{ID: 5, Date: utcDay(2026, 9, 18)},
		{ID: 4, Date: utcDay(2026, 9, 17)},
		{ID: 3, Date: utcDay(2026, 9, 16)},
		{ID: 2, Date: utcDay(2026, 9, 15)},
		{ID: 1, Date: utcDay(2026, 9, 14)},
	}}, provider)

	report, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 18))
	if err != nil {
		t.Fatalf("PreviewReconciliation() error = %v", err)
	}
	if report.Status != "COMPLETED" || report.QualityStatus != "GOOD" || report.Count != 5 {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Expected) != 5 || len(report.Missing) != 0 || len(report.NonTradingDayDates) != 0 ||
		len(report.DuplicateGroups) != 0 || len(report.SourceIncomplete) != 0 {
		t.Fatalf("unexpected gaps: %+v", report)
	}
	if report.EarliestDate == nil || !report.EarliestDate.Equal(utcDay(2026, 9, 14)) ||
		report.LatestDate == nil || !report.LatestDate.Equal(utcDay(2026, 9, 18)) {
		t.Fatalf("range endpoints = %+v", report)
	}
	// The source is always fetched for the window, even when nothing is missing.
	if len(provider.calls) != 1 || provider.calls[0].interval != "1d" {
		t.Fatalf("provider calls = %+v", provider.calls)
	}
}

func TestReconciliationPreviewClassifiesInvalidAndSourceIncompleteData(t *testing.T) {
	provider := &intradayProvider{fetches: map[string][]YahooCandle{
		"TCS": {intradayBar(t, "TCS", utcDay(2026, 9, 15), barClock(15, 15), "4000")},
	}}
	provider.fetches["TCS"][0].Timeframe = "1d"
	reconciler := newReconcilerForTest(t, &fakeCoverage{verified: true}, &fakeRangeStore{rows: []DailyCandleRow{
		{ID: 7, Date: utcDay(2026, 9, 17)},
		{ID: 6, Date: utcDay(2026, 9, 13)}, // Sunday stored
		{ID: 5, Date: utcDay(2026, 9, 12)}, // Saturday stored
	}}, provider)

	report, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 18))
	if err != nil {
		t.Fatalf("PreviewReconciliation() error = %v", err)
	}
	if report.Status != "SOURCE_INCOMPLETE" {
		t.Fatalf("status = %q, want SOURCE_INCOMPLETE", report.Status)
	}
	if report.QualityStatus != "INVALID_DATE" {
		t.Fatalf("quality = %q, want INVALID_DATE", report.QualityStatus)
	}
	wantInvalid := []time.Time{utcDay(2026, 9, 13), utcDay(2026, 9, 12)}
	if len(report.NonTradingDayDates) != 2 || !report.NonTradingDayDates[0].Equal(wantInvalid[0]) ||
		!report.NonTradingDayDates[1].Equal(wantInvalid[1]) {
		t.Fatalf("invalid stored dates (want newest first) = %s", report.NonTradingDayDates)
	}
	// Sep15 arrives from the source; Sep14, Sep16, and Sep18 stay incomplete.
	wantIncomplete := []time.Time{utcDay(2026, 9, 14), utcDay(2026, 9, 16), utcDay(2026, 9, 18)}
	if len(report.Missing) != 4 || len(report.SourceIncomplete) != 3 {
		t.Fatalf("missing = %s, incomplete = %s", report.Missing, report.SourceIncomplete)
	}
	for i := range wantIncomplete {
		if !report.SourceIncomplete[i].Equal(wantIncomplete[i]) {
			t.Fatalf("source incomplete = %s", report.SourceIncomplete)
		}
	}
}

func TestReconciliationPreviewReportsMissingWhenSourceEmpty(t *testing.T) {
	reconciler := newReconcilerForTest(t, &fakeCoverage{verified: true}, &fakeRangeStore{}, &intradayProvider{})

	report, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 15))
	if err != nil {
		t.Fatalf("PreviewReconciliation() error = %v", err)
	}
	if report.QualityStatus != "MISSING" || report.Status != "SOURCE_INCOMPLETE" || report.Count != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.EarliestDate != nil || report.LatestDate != nil {
		t.Fatalf("empty range endpoints = %+v", report)
	}
}

func TestReconciliationPreviewGroupsDuplicateStoredDates(t *testing.T) {
	reconciler := newReconcilerForTest(t, &fakeCoverage{verified: true}, &fakeRangeStore{rows: []DailyCandleRow{
		{ID: 9, Date: utcDay(2026, 9, 15)},
		{ID: 4, Date: utcDay(2026, 9, 15)},
		{ID: 2, Date: utcDay(2026, 9, 14)},
	}}, &intradayProvider{})

	report, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 15))
	if err != nil {
		t.Fatalf("PreviewReconciliation() error = %v", err)
	}
	if report.QualityStatus != "DUPLICATE" || len(report.DuplicateGroups) != 1 {
		t.Fatalf("duplicate handling = %+v", report)
	}
	group := report.DuplicateGroups[0]
	if group.Count != 2 || len(group.IDs) != 2 || group.IDs[0] != 4 || group.IDs[1] != 9 ||
		!group.Date.Equal(utcDay(2026, 9, 15)) {
		t.Fatalf("duplicate group = %+v", group)
	}
}

func TestReconciliationPreviewPropagatesStoreAndProviderFailures(t *testing.T) {
	t.Run("coverage failure", func(t *testing.T) {
		reconciler := newReconcilerForTest(t, &fakeCoverage{err: errors.New("db down")}, &fakeRangeStore{}, &intradayProvider{})
		if _, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 18)); err == nil {
			t.Fatal("coverage error = nil")
		}
	})
	t.Run("range store failure", func(t *testing.T) {
		reconciler := newReconcilerForTest(t, &fakeCoverage{verified: true}, &fakeRangeStore{err: errors.New("db down")}, &intradayProvider{})
		if _, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 18)); err == nil {
			t.Fatal("range error = nil")
		}
	})
	t.Run("provider failure", func(t *testing.T) {
		provider := &intradayProvider{errs: map[string]error{"TCS": errors.New("provider down")}}
		reconciler := newReconcilerForTest(t, &fakeCoverage{verified: true}, &fakeRangeStore{}, provider)
		if _, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 18)); err == nil {
			t.Fatal("provider error = nil")
		}
	})
	t.Run("holiday failure", func(t *testing.T) {
		reconciler, err := NewReconciler(&fakeCoverage{verified: true},
			&intradayHolidays{err: errors.New("db down")}, &fakeRangeStore{}, &intradayProvider{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reconciler.PreviewReconciliation(context.Background(), "TCS", utcDay(2026, 9, 14), utcDay(2026, 9, 18)); err == nil {
			t.Fatal("holiday error = nil")
		}
	})
}

func TestNewReconcilerValidatesDependencies(t *testing.T) {
	if _, err := NewReconciler(nil, nil, nil, nil); err == nil {
		t.Fatal("nil dependencies accepted")
	}
}
