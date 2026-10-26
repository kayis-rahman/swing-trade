package data

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// ErrInvalidReconciliationRange rejects windows Java rejects with an
// IllegalArgumentException, which its global handler maps to HTTP 400.
var ErrInvalidReconciliationRange = errors.New("reconciliation range must be at most 10 calendar days")

// DuplicateGroup mirrors Java's duplicateGroups entries: a stored date with
// more than one row, plus the row ids.
type DuplicateGroup struct {
	Date  time.Time
	IDs   []int64
	Count int
}

// ReconciliationReport mirrors Java's DataIngestionService.reconcile result
// map for one symbol.
type ReconciliationReport struct {
	Symbol               string
	From                 time.Time
	To                   time.Time
	Expected             []time.Time
	StoredDates          []time.Time
	Missing              []time.Time
	NonTradingDayDates   []time.Time
	DuplicateGroups      []DuplicateGroup
	SourceIncomplete     []time.Time
	Inserted             int
	InvalidStoredRemoved int
	QualityStatus        string
	EarliestDate         *time.Time
	LatestDate           *time.Time
	Count                int
	Status               string
}

// DailyCandleRangeStore reads stored daily rows for a reconciliation window.
type DailyCandleRangeStore interface {
	DailyCandlesInRange(ctx context.Context, symbol string, from, to time.Time) ([]DailyCandleRow, error)
}

// ReconciliationPreview computes preview reconciliation reports.
type ReconciliationPreview interface {
	PreviewReconciliation(ctx context.Context, symbol string, from, to time.Time) (ReconciliationReport, error)
}

// Reconciler ports the read-only (apply=false) half of Java's
// DataIngestionService.reconcile: coverage gate, expected-session and stored
// set diffing, provider source validation, and quality classification without
// touching stored rows.
type Reconciler struct {
	coverage CalendarCoverageStore
	holidays HolidayReadStore
	stored   DailyCandleRangeStore
	provider HistoricalCandleProvider
}

// NewReconciler wires the preview over explicit ports.
func NewReconciler(
	coverage CalendarCoverageStore,
	holidays HolidayReadStore,
	stored DailyCandleRangeStore,
	provider HistoricalCandleProvider,
) (*Reconciler, error) {
	if coverage == nil || holidays == nil || stored == nil || provider == nil {
		return nil, fmt.Errorf("coverage store, holiday store, candle range store, and provider are required")
	}
	return &Reconciler{coverage: coverage, holidays: holidays, stored: stored, provider: provider}, nil
}

// PreviewReconciliation reports gaps and invalid stored days for one symbol
// over a range of at most ten calendar days. Coverage-unverified years return
// a minimal CALENDAR_INCOMPLETE report before any provider call, and provider
// failures propagate so the endpoint can answer 500 like Java.
func (reconciler *Reconciler) PreviewReconciliation(ctx context.Context, symbol string, from, to time.Time) (ReconciliationReport, error) {
	if symbol == "" || from.IsZero() || to.IsZero() || to.Before(from) || from.AddDate(0, 0, 10).Before(to) {
		return ReconciliationReport{}, fmt.Errorf("%w: a symbol and a range of at most 10 calendar days are required", ErrInvalidReconciliationRange)
	}
	report := ReconciliationReport{Symbol: symbol, From: from, To: to}
	verified, err := reconciler.coverage.IsCoverageVerified(ctx, from, to)
	if err != nil {
		return ReconciliationReport{}, err
	}
	if !verified {
		report.Status = "CALENDAR_INCOMPLETE"
		return report, nil
	}

	expected := make([]time.Time, 0, 10)
	expectedSet := make(map[string]struct{}, 10)
	sessions, err := ExpectedNSETradingSessions(ctx, from, to, reconciler.holidays)
	if err != nil {
		return ReconciliationReport{}, err
	}
	for _, session := range sessions {
		// Expected sessions are IST midnights; stored candles use the
		// UTC-midnight date convention, so normalize before set comparisons.
		date := DateOnlyIST(session)
		expected = append(expected, date)
		expectedSet[date.Format(time.DateOnly)] = struct{}{}
	}
	report.Expected = expected

	storedRows, err := reconciler.stored.DailyCandlesInRange(ctx, symbol, from, to)
	if err != nil {
		return ReconciliationReport{}, err
	}
	storedSet := make(map[string]struct{}, len(storedRows))
	perDate := make(map[string][]int64, len(storedRows))
	for _, row := range storedRows {
		key := row.Date.Format(time.DateOnly)
		storedSet[key] = struct{}{}
		perDate[key] = append(perDate[key], row.ID)
	}
	return reconciler.finishPreview(ctx, report, expected, expectedSet, storedRows, storedSet, perDate)
}

func (reconciler *Reconciler) finishPreview(
	ctx context.Context,
	report ReconciliationReport,
	expected []time.Time,
	expectedSet map[string]struct{},
	storedRows []DailyCandleRow,
	storedSet map[string]struct{},
	perDate map[string][]int64,
) (ReconciliationReport, error) {
	missing, invalidStored := splitStoredAndExpected(expected, expectedSet, storedRows, storedSet)
	sourceIncomplete, err := reconciler.incompleteSourceDates(ctx, report.Symbol, report.From, report.To, expectedSet, missing)
	if err != nil {
		return ReconciliationReport{}, err
	}

	terminal := "COMPLETED"
	if len(sourceIncomplete) > 0 {
		terminal = "SOURCE_INCOMPLETE"
	}
	quality := "GOOD"
	duplicates := buildDuplicateGroups(perDate)
	switch {
	case len(duplicates) > 0:
		quality = "DUPLICATE"
	case len(invalidStored) > 0:
		quality = "INVALID_DATE"
	case len(missing) > 0:
		quality = "MISSING"
	}

	report.Missing = missing
	report.NonTradingDayDates = invalidStored
	report.SourceIncomplete = sourceIncomplete
	report.DuplicateGroups = duplicates
	report.QualityStatus = quality
	report.StoredDates = sortedStoredDates(storedSet)
	report.Count = len(storedRows)
	report.Status = terminal
	if len(storedRows) > 0 {
		earliest := storedRows[len(storedRows)-1].Date
		latest := storedRows[0].Date
		report.EarliestDate = &earliest
		report.LatestDate = &latest
	}
	return report, nil
}

// splitStoredAndExpected derives missing sessions (expected order) and stored
// non-session dates (newest first, like Java's descending range query).
func splitStoredAndExpected(
	expected []time.Time,
	expectedSet map[string]struct{},
	storedRows []DailyCandleRow,
	storedSet map[string]struct{},
) (missing, invalidStored []time.Time) {
	missing = make([]time.Time, 0)
	for _, date := range expected {
		if _, ok := storedSet[date.Format(time.DateOnly)]; !ok {
			missing = append(missing, date)
		}
	}
	invalidStored = make([]time.Time, 0)
	for _, row := range storedRows {
		if _, ok := expectedSet[row.Date.Format(time.DateOnly)]; !ok {
			invalidStored = append(invalidStored, row.Date)
		}
	}
	return missing, invalidStored
}

// incompleteSourceDates fetches the window and reports missing dates the
// provider cannot fill with a valid, expected-session candle.
func (reconciler *Reconciler) incompleteSourceDates(
	ctx context.Context,
	symbol string,
	from, to time.Time,
	expectedSet map[string]struct{},
	missing []time.Time,
) ([]time.Time, error) {
	source, err := reconciler.provider.FetchCandles(ctx, symbol, from, to, "1d")
	if err != nil {
		return nil, fmt.Errorf("fetch source candles: %w", err)
	}
	validSource := make(map[string]YahooCandle, len(source))
	for _, candle := range source {
		key := utcDateOnly(candle.Date).Format(time.DateOnly)
		if _, ok := expectedSet[key]; !ok {
			continue
		}
		if !IsValidCandle(&CandleData{
			Symbol: candle.Symbol, Open: &candle.Open, High: &candle.High,
			Low: &candle.Low, Close: &candle.Close, Volume: candle.Volume,
		}) {
			continue
		}
		if _, duplicate := validSource[key]; !duplicate {
			validSource[key] = candle // Java's toMap merge keeps the first value.
		}
	}
	incomplete := make([]time.Time, 0)
	for _, date := range missing {
		if _, ok := validSource[date.Format(time.DateOnly)]; !ok {
			incomplete = append(incomplete, date)
		}
	}
	return incomplete, nil
}

// buildDuplicateGroups returns dates stored more than once, sorted by date.
func buildDuplicateGroups(perDate map[string][]int64) []DuplicateGroup {
	duplicates := make([]DuplicateGroup, 0)
	for key, ids := range perDate {
		if len(ids) < 2 {
			continue
		}
		date, _ := time.Parse(time.DateOnly, key)
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		duplicates = append(duplicates, DuplicateGroup{Date: date, IDs: ids, Count: len(ids)})
	}
	sort.Slice(duplicates, func(i, j int) bool { return duplicates[i].Date.Before(duplicates[j].Date) })
	return duplicates
}

func sortedStoredDates(storedSet map[string]struct{}) []time.Time {
	storedDates := make([]time.Time, 0, len(storedSet))
	for key := range storedSet {
		date, _ := time.Parse(time.DateOnly, key)
		storedDates = append(storedDates, date)
	}
	sort.Slice(storedDates, func(i, j int) bool { return storedDates[i].Before(storedDates[j]) })
	return storedDates
}

// utcDateOnly maps a provider candle instant to the stored UTC-midnight date.
func utcDateOnly(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
