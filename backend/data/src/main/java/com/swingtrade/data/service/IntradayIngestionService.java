package com.swingtrade.data.service;

import com.swingtrade.core.metrics.DataIngestionMetrics;
import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.entity.WatchlistEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.data.repository.WatchlistRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionTemplate;

import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.util.ArrayList;
import java.util.List;

/**
 * Bounded, session-aware ingestion path for intraday (15-minute) candles.
 *
 * <p>Scope guards, per the stage-0 data-plane contract:
 * <ul>
 *   <li><b>Watchlist only</b> — never the full symbol universe.</li>
 *   <li><b>Session-aware</b> — only trading sessions (exchange holiday calendar),
 *       and only bars whose start time falls inside the normal 09:15–15:30 window
 *       are persisted; pre-open and closing-auction bars are dropped.</li>
 *   <li><b>Bounded</b> — the window ends at today (IST) and the scheduler passes a
 *       bounded lookback; there is no unbounded or forward fetching.</li>
 *   <li><b>Rate-limit handling</b> — the provider wrapper enforces the provider
 *       token bucket; this service paces symbols and isolates per-symbol failures
 *       so one bad symbol never aborts the run.</li>
 *   <li><b>Restart-safe</b> — persistence is an idempotent upsert on the widened
 *       candle identity {@code (symbol, interval, date, bar_time)}, and each symbol
 *       resumes from its latest stored bar, so an interrupted run continues
 *       without re-downloading completed history.</li>
 * </ul>
 *
 * <p>This path is strictly additive: it never touches daily rows and shares no
 * scheduler with the end-of-day ingestion.
 */
@Service
public class IntradayIngestionService {

    private static final Logger logger = LoggerFactory.getLogger(IntradayIngestionService.class);
    private static final ZoneId IST = ZoneId.of("Asia/Kolkata");

    private final OhlcvCandleRepository candleRepository;
    private final WatchlistRepository watchlistRepository;
    private final MarketDataClientProvider marketDataClientProvider;
    private final MarketCalendar marketCalendar;
    private final NseSessionCalendar sessionCalendar;
    private final TransactionTemplate txTemplate;
    private final DataIngestionMetrics ingestionMetrics;

    /** Per-symbol pacing between provider requests, on top of the provider token bucket. */
    @Value("${intraday.ingestion.rate-limit-ms:500}")
    private long rateLimitMs;

    public IntradayIngestionService(OhlcvCandleRepository candleRepository,
                                    WatchlistRepository watchlistRepository,
                                    MarketDataClientProvider marketDataClientProvider,
                                    MarketCalendar marketCalendar,
                                    NseSessionCalendar sessionCalendar,
                                    TransactionTemplate txTemplate,
                                    DataIngestionMetrics ingestionMetrics) {
        this.candleRepository = candleRepository;
        this.watchlistRepository = watchlistRepository;
        this.marketDataClientProvider = marketDataClientProvider;
        this.marketCalendar = marketCalendar;
        this.sessionCalendar = sessionCalendar;
        this.txTemplate = txTemplate;
        this.ingestionMetrics = ingestionMetrics;
    }

    /**
     * Ingests intraday candles for every active watchlist symbol over the given window.
     *
     * @param interval the candle interval (stage 0: {@link Interval#FIFTEEN_MINUTES})
     * @param fromDate requested window start (inclusive)
     * @param toDate requested window end (inclusive); clamped to today (IST)
     * @return the run outcome, including per-symbol failures
     */
    public IntradayIngestionResult ingestWatchlistInterval(Interval interval, LocalDate fromDate, LocalDate toDate) {
        if (interval == null || fromDate == null || toDate == null || fromDate.isAfter(toDate)) {
            throw new IllegalArgumentException("interval and an ordered date range are required");
        }
        LocalDate today = LocalDate.now(IST);
        LocalDate effectiveTo = toDate.isAfter(today) ? today : toDate;
        if (fromDate.isAfter(effectiveTo)) {
            logger.info("Intraday ingestion for {} skipped: window {} to {} is entirely in the future",
                interval.getCode(), fromDate, effectiveTo);
            return new IntradayIngestionResult(interval, fromDate, effectiveTo, 0, 0, 0, 0, 0, 0, List.of());
        }

        List<String> symbols = activeWatchlistSymbols();
        logger.info("Intraday ingestion started for {}: {} watchlist symbols, window {} to {}",
            interval.getCode(), symbols.size(), fromDate, effectiveTo);

        int fetched = 0;
        int saved = 0;
        int rejected = 0;
        int succeeded = 0;
        int failed = 0;
        List<String> errors = new ArrayList<>();

        for (String symbol : symbols) {
            try {
                SymbolIngestionOutcome outcome = ingestSymbol(interval, symbol, fromDate, effectiveTo);
                fetched += outcome.fetched();
                saved += outcome.saved();
                rejected += outcome.rejected();
                succeeded++;
            } catch (UnsupportedOperationException e) {
                // A provider without interval support is a configuration error, not a
                // per-symbol transient failure — fail the whole run loudly.
                throw e;
            } catch (Exception e) {
                failed++;
                String message = symbol + ": " + e.getMessage();
                errors.add(message);
                logger.warn("Intraday ingestion failed for {}: {}", symbol, e.getMessage());
                ingestionMetrics.recordFetchFailure(marketDataClientProvider.getActiveBroker());
            }
            pace();
        }

        logger.info("Intraday ingestion completed for {}: {} symbols succeeded, {} failed, {} bars saved",
            interval.getCode(), succeeded, failed, saved);
        return new IntradayIngestionResult(interval, fromDate, effectiveTo, symbols.size(),
            succeeded, failed, fetched, saved, rejected, errors);
    }

    /**
     * Ingests one symbol, resuming from its latest stored bar so a restart does not
     * re-download completed sessions. Re-fetching the latest stored date is safe:
     * the upsert is idempotent and picks up late-arriving corrections.
     */
    private SymbolIngestionOutcome ingestSymbol(Interval interval, String symbol,
                                                LocalDate fromDate, LocalDate toDate) {
        LocalDate effectiveFrom = fromDate;
        var latest = candleRepository.findLatestBySymbolAndInterval(symbol, interval.getCode());
        if (latest.isPresent() && !latest.get().getDate().isBefore(fromDate)
            && !latest.get().getDate().isAfter(toDate)) {
            effectiveFrom = latest.get().getDate();
        }

        List<CandleData> candles = new ArrayList<>();
        marketDataClientProvider.getClient()
            .fetchCandles(symbol, effectiveFrom, toDate, interval)
            .forEach(candles::add);

        int saved = 0;
        int rejected = 0;
        for (CandleData candle : candles) {
            if (!isPersistable(candle)) {
                rejected++;
                continue;
            }
            int inserted = txTemplate.execute(status -> candleRepository.insertIfAbsent(
                symbol, interval.getCode(), candle.date(), candle.barTime(),
                candle.open(), candle.high(), candle.low(), candle.close(),
                candle.volume(), candle.adjClose()));
            if (inserted == 1) {
                saved++;
                ingestionMetrics.recordCandleIngested();
            }
        }
        return new SymbolIngestionOutcome(candles.size(), saved, rejected);
    }

    /**
     * Session-awareness guard: the bar's date must be a trading session and its
     * start time must fall inside the normal 09:15–15:30 window.
     */
    private boolean isPersistable(CandleData candle) {
        if (candle == null || candle.date() == null || candle.barTime() == null) {
            return false;
        }
        if (!marketCalendar.isNseTradingSession(candle.date())) {
            logger.debug("Rejected intraday bar {} {}: not a trading session",
                candle.symbol(), candle.date());
            return false;
        }
        if (!sessionCalendar.isNormalSessionBarTime(candle.barTime())) {
            logger.debug("Rejected intraday bar {} {} {}: outside the normal session window",
                candle.symbol(), candle.date(), candle.barTime());
            return false;
        }
        if (!CandleValidator.isValid(candle, false)) {
            logger.debug("Rejected intraday bar {} {} {}: failed validation",
                candle.symbol(), candle.date(), candle.barTime());
            return false;
        }
        return true;
    }

    private List<String> activeWatchlistSymbols() {
        List<String> symbols = new ArrayList<>();
        for (WatchlistEntity watchlist : watchlistRepository.findByIsActiveTrueOrderBySymbolAsc()) {
            if (watchlist.getSymbol() != null && !watchlist.getSymbol().isBlank()) {
                symbols.add(watchlist.getSymbol());
            }
        }
        return symbols;
    }

    private void pace() {
        if (rateLimitMs <= 0) {
            return;
        }
        try {
            Thread.sleep(rateLimitMs);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new IllegalStateException("Interrupted while pacing intraday ingestion", e);
        }
    }

    /** Per-symbol outcome. */
    private record SymbolIngestionOutcome(int fetched, int saved, int rejected) {}

    /** Run outcome for an intraday ingestion window. */
    public record IntradayIngestionResult(
        Interval interval,
        LocalDate fromDate,
        LocalDate toDate,
        int symbolsAttempted,
        int symbolsSucceeded,
        int symbolsFailed,
        int barsFetched,
        int barsSaved,
        int barsRejected,
        List<String> errors
    ) {}
}
