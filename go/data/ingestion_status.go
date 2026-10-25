package data

import (
	"context"
	"fmt"
	"time"
)

// IngestionStatusEntry mirrors Java's WatchlistService.getIngestionStatus map
// rows: watchlist identity plus daily-candle coverage metadata.
type IngestionStatusEntry struct {
	Symbol             string
	Name               *string
	Exchange           *string
	IsActive           *bool
	CandleCount        int64
	LastCandleDate     *time.Time
	EarliestCandleDate *time.Time
	LastSyncedAt       *time.Time
	HasData            bool
	DataQuality        string
}

// IngestionStatusLookup builds per-symbol ingestion status rows for the
// read-only status endpoint.
type IngestionStatusLookup interface {
	IngestionStatus(ctx context.Context) ([]IngestionStatusEntry, error)
}

// IngestionStatusService composes the active watchlist with the daily-candle
// reads, exactly like Java's WatchlistService.getIngestionStatus.
type IngestionStatusService struct {
	watchlist WatchlistReadStore
	candles   CandleReadStore
	now       func() time.Time
}

// NewIngestionStatusService wires the status read over explicit ports.
func NewIngestionStatusService(watchlist WatchlistReadStore, candles CandleReadStore, now func() time.Time) (*IngestionStatusService, error) {
	if watchlist == nil || candles == nil || now == nil {
		return nil, fmt.Errorf("watchlist store, candle store, and clock are required")
	}
	return &IngestionStatusService{watchlist: watchlist, candles: candles, now: now}, nil
}

// IngestionStatus returns one row per active watchlist symbol. All candle
// metadata is daily-only, matching the Java repository queries it replaces.
func (service *IngestionStatusService) IngestionStatus(ctx context.Context) ([]IngestionStatusEntry, error) {
	entries, err := service.watchlist.ListActiveWatchlist(ctx)
	if err != nil {
		return nil, fmt.Errorf("read active watchlist: %w", err)
	}
	statuses := make([]IngestionStatusEntry, 0, len(entries))
	for _, entry := range entries {
		count, err := service.candles.CountDailyCandles(ctx, entry.Symbol)
		if err != nil {
			return nil, fmt.Errorf("count daily candles for %s: %w", entry.Symbol, err)
		}
		window, err := service.candles.GetDailyDateWindow(ctx, entry.Symbol)
		if err != nil {
			return nil, fmt.Errorf("read daily candle window for %s: %w", entry.Symbol, err)
		}
		statuses = append(statuses, IngestionStatusEntry{
			Symbol:             entry.Symbol,
			Name:               entry.Name,
			Exchange:           entry.Exchange,
			IsActive:           entry.IsActive,
			CandleCount:        count,
			LastCandleDate:     window.Latest,
			EarliestCandleDate: window.Earliest,
			LastSyncedAt:       entry.LastSyncedAt,
			HasData:            count > 0,
			DataQuality:        assessDataQuality(window.Earliest, count, DateOnlyIST(service.now())),
		})
	}
	return statuses, nil
}

// assessDataQuality mirrors Java's WatchlistService.assessDataQuality: compare
// the daily count against an approximate 252-trading-day year from the
// earliest stored candle to today.
func assessDataQuality(earliest *time.Time, count int64, today time.Time) string {
	if earliest == nil {
		return "no_data"
	}
	days := int64(today.Sub(*earliest).Hours() / 24)
	years := int64(float64(days) / 365.25)
	expectedDays := years * 252
	if expectedDays < 1 {
		expectedDays = 1
	}
	coverage := float64(count) / float64(expectedDays)
	switch {
	case coverage >= 0.95:
		return "excellent"
	case coverage >= 0.80:
		return "good"
	case coverage >= 0.50:
		return "partial"
	default:
		return "poor"
	}
}
