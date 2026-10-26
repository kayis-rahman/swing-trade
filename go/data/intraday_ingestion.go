package data

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// ErrUnsupportedInterval aborts a whole intraday run, mirroring Java's
// UnsupportedOperationException from a provider without interval support.
var ErrUnsupportedInterval = errors.New("provider does not support the requested interval")

// IntradayIngestionResult mirrors Java's IntradayIngestionResult record: the
// requested window (with the end clamped to today IST) and per-symbol counts.
type IntradayIngestionResult struct {
	Interval         string
	From             time.Time
	To               time.Time
	SymbolsAttempted int
	SymbolsSucceeded int
	SymbolsFailed    int
	BarsFetched      int
	BarsSaved        int
	BarsRejected     int
	Errors           []string
}

// IntervalLatestStore exposes the newest stored bar date for one timeframe.
type IntervalLatestStore interface {
	LatestIntervalCandleDate(ctx context.Context, symbol, timeframe string) (*time.Time, error)
}

// IntradayIngestion ports Java's IntradayIngestionService: watchlist-only,
// session-aware, bounded 15-minute ingestion that resumes per symbol from its
// latest stored bar and isolates per-symbol failures.
type IntradayIngestion struct {
	watchlist WatchlistReadStore
	latest    IntervalLatestStore
	holidays  HolidayReadStore
	provider  HistoricalCandleProvider
	writer    *CandleWriter
	now       func() time.Time
	logger    *slog.Logger
}

// NewIntradayIngestion wires the service over explicit ports.
func NewIntradayIngestion(
	watchlist WatchlistReadStore,
	latest IntervalLatestStore,
	holidays HolidayReadStore,
	provider HistoricalCandleProvider,
	writer *CandleWriter,
	now func() time.Time,
	logger *slog.Logger,
) (*IntradayIngestion, error) {
	if watchlist == nil || latest == nil || holidays == nil || provider == nil || writer == nil || now == nil || logger == nil {
		return nil, fmt.Errorf("watchlist, interval reader, holiday store, provider, writer, clock, and logger are required")
	}
	return &IntradayIngestion{
		watchlist: watchlist,
		latest:    latest,
		holidays:  holidays,
		provider:  provider,
		writer:    writer,
		now:       now,
		logger:    logger,
	}, nil
}

// IngestWatchlistInterval ingests one interval for every active watchlist
// symbol over the requested window, clamped to today IST. Invalid windows are
// rejected, entirely future windows skip work, an unsupported provider
// interval aborts the whole run, and every other failure stays isolated to its
// symbol with the message prefixed by the symbol like Java's errors list.
func (s *IntradayIngestion) IngestWatchlistInterval(ctx context.Context, interval string, from, to time.Time) (IntradayIngestionResult, error) {
	if interval == "" || from.IsZero() || to.IsZero() || from.After(to) {
		return IntradayIngestionResult{}, fmt.Errorf("interval and an ordered date range are required")
	}
	today := DateOnlyIST(s.now())
	effectiveTo := to
	if effectiveTo.After(today) {
		effectiveTo = today
	}
	result := IntradayIngestionResult{
		Interval: interval, From: from, To: effectiveTo, Errors: make([]string, 0),
	}
	if from.After(effectiveTo) {
		s.logger.Info("intraday ingestion skipped: window entirely in the future",
			"module", "data", "interval", interval, "from", from.Format(time.DateOnly), "to", effectiveTo.Format(time.DateOnly))
		return result, nil
	}

	entries, err := s.watchlist.ListActiveWatchlist(ctx)
	if err != nil {
		return result, fmt.Errorf("read active watchlist: %w", err)
	}
	symbols := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Symbol == "" {
			continue
		}
		symbols = append(symbols, entry.Symbol)
	}
	result.SymbolsAttempted = len(symbols)
	s.logger.Info("intraday ingestion started",
		"module", "data", "interval", interval, "symbols", len(symbols),
		"from", from.Format(time.DateOnly), "to", effectiveTo.Format(time.DateOnly))

	timeframe := storageTimeframe(interval)
	for _, symbol := range symbols {
		outcome, err := s.ingestSymbol(ctx, interval, timeframe, symbol, from, effectiveTo)
		if err != nil {
			if errors.Is(err, ErrUnsupportedInterval) {
				return result, err
			}
			result.SymbolsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", symbol, err.Error()))
			s.logger.Warn("intraday ingestion failed", "module", "data", "symbol", symbol, "error", err)
			continue
		}
		result.SymbolsSucceeded++
		result.BarsFetched += outcome.fetched
		result.BarsSaved += outcome.saved
		result.BarsRejected += outcome.rejected
	}
	s.logger.Info("intraday ingestion completed",
		"module", "data", "interval", interval,
		"succeeded", result.SymbolsSucceeded, "failed", result.SymbolsFailed, "saved", result.BarsSaved)
	return result, nil
}

type intradaySymbolOutcome struct {
	fetched  int
	saved    int
	rejected int
}

// ingestSymbol resumes from the latest stored bar inside the window so a
// restart does not re-download completed sessions, then stores only bars that
// pass the session and validation guards.
func (s *IntradayIngestion) ingestSymbol(ctx context.Context, interval, timeframe, symbol string, from, to time.Time) (intradaySymbolOutcome, error) {
	latest, err := s.latest.LatestIntervalCandleDate(ctx, symbol, timeframe)
	if err != nil {
		return intradaySymbolOutcome{}, err
	}
	effectiveFrom := from
	if latest != nil && !latest.Before(from) && !latest.After(to) {
		effectiveFrom = *latest
	}
	candles, err := s.provider.FetchCandles(ctx, symbol, effectiveFrom, to, interval)
	if err != nil {
		return intradaySymbolOutcome{}, fmt.Errorf("fetch %s candles: %w", interval, err)
	}
	outcome := intradaySymbolOutcome{fetched: len(candles)}
	for index := range candles {
		persistable, err := s.persistable(ctx, &candles[index])
		if err != nil {
			return intradaySymbolOutcome{}, err
		}
		if !persistable {
			outcome.rejected++
			continue
		}
		inserted, err := s.writer.InsertIfAbsent(ctx, candles[index], timeframe)
		if err != nil {
			return intradaySymbolOutcome{}, err
		}
		if inserted {
			outcome.saved++
		}
	}
	return outcome, nil
}

// persistable mirrors Java's guard chain: trading-session date, normal-session
// bar time, then OHLCV validation.
func (s *IntradayIngestion) persistable(ctx context.Context, candle *YahooCandle) (bool, error) {
	if candle == nil || candle.Date.IsZero() || candle.BarTime.IsZero() {
		return false, nil
	}
	trading, err := IsNSETradingSession(ctx, candle.Date, s.holidays)
	if err != nil {
		return false, err
	}
	if !trading {
		return false, nil
	}
	if !IsNormalSessionBarTime(candle.BarTime) {
		return false, nil
	}
	return IsValidCandle(&CandleData{
		Symbol: candle.Symbol, Open: &candle.Open, High: &candle.High,
		Low: &candle.Low, Close: &candle.Close, Volume: candle.Volume,
	}), nil
}
