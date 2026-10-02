package com.swingtrade.data.service;

import com.swingtrade.data.entity.WatchlistEntity;
import com.swingtrade.data.repository.WatchlistRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.ApplicationArguments;
import org.springframework.boot.ApplicationRunner;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.time.ZoneId;
import java.util.List;

/**
 * Daily end-of-day scheduler: fetches the latest candle for every watchlist stock.
 * Runs at 16:30 IST on weekdays (after market close).
 */
@Service
public class EodIngestionScheduler implements ApplicationRunner {

    private static final Logger logger = LoggerFactory.getLogger(EodIngestionScheduler.class);
    private static final ZoneId IST = ZoneId.of("Asia/Kolkata");
    /** Persisted benchmark symbol required for portfolio and excess-return metrics. */
    static final String NIFTY_50_SYMBOL = "NIFTY50";

    private final DataIngestionService dataIngestionService;
    private final WatchlistRepository watchlistRepository;
    private final NseHolidayService holidayService;
    /** Overridable in tests so weekday-only behaviour does not depend on the day the suite runs. */
    java.time.Clock clock = java.time.Clock.system(IST);

    @Value("${yahoo.finance.rate-limit-ms:500}")
    private long rateLimitMs;

    /**
     * Years of NIFTY50 history required before backtests over past windows can find an
     * authoritative benchmark. Daily forward ingestion alone only accumulates from the day the
     * scheduler first ran, which is insufficient for backtests over historical date ranges.
     */
    @Value("${benchmark.backfill.years:10}")
    private int benchmarkBackfillYears = 10;

    @Value("${benchmark.backfill.on-startup:true}")
    boolean benchmarkBackfillOnStartup = true;

    public EodIngestionScheduler(DataIngestionService dataIngestionService, WatchlistRepository watchlistRepository, NseHolidayService holidayService) {
        this.dataIngestionService = dataIngestionService;
        this.watchlistRepository = watchlistRepository;
        this.holidayService = holidayService;
    }

    /**
     * Ensures the NIFTY50 benchmark has enough historical depth for backtests run over past
     * date ranges, not just the forward-daily ticks the scheduled job appends. Runs once the
     * application context is fully up (not during bean construction, which would block server
     * startup on network I/O); the underlying incremental backfill is a no-op once the window
     * is already covered, so repeated application restarts do not re-download history.
     */
    @Override
    public void run(ApplicationArguments args) {
        if (!benchmarkBackfillOnStartup) {
            logger.info("Startup NIFTY50 benchmark backfill disabled");
            return;
        }
        ensureNiftyBenchmarkHistory();
    }

    void ensureNiftyBenchmarkHistory() {
        try {
            LocalDate cutoff = LocalDate.now(IST).minusYears(benchmarkBackfillYears);
            DataIngestionService.DataWindow window = dataIngestionService.getExistingDataWindow(NIFTY_50_SYMBOL);
            if (window.earliestDate() == null || window.earliestDate().isAfter(cutoff)) {
                logger.info("Backfilling NIFTY50 benchmark history: {} years (earliest stored date: {})",
                        benchmarkBackfillYears, window.earliestDate());
                dataIngestionService.backfillStockData(NIFTY_50_SYMBOL, benchmarkBackfillYears);
            } else {
                logger.debug("NIFTY50 benchmark history already covers {} years (earliest stored date: {})",
                        benchmarkBackfillYears, window.earliestDate());
            }
        } catch (Exception e) {
            logger.warn("NIFTY50 benchmark historical backfill failed at startup: {}", e.getMessage());
        }
    }

    @Scheduled(cron = "0 30 16 * * MON-FRI", zone = "Asia/Kolkata")
    public void ingestLatestForAll() {
        LocalDate today = LocalDate.now(clock);
        LocalDate yesterday = today.minusDays(1);

        // Skip weekends (safety net)
        int day = today.getDayOfWeek().getValue();
        if (day > 5) {
            logger.debug("Today is {} — skipping scheduled EOD ingestion", today);
            return;
        }

        // Skip NSE holidays
        if (holidayService.isMarketClosed(yesterday)) {
            logger.info("EOD ingestion skipped: {} is an NSE market holiday", yesterday);
            return;
        }

        logger.info("EOD ingestion started at {} for date {}", LocalDate.now(IST), yesterday);
        long start = System.currentTimeMillis();

        List<WatchlistEntity> watchlist = watchlistRepository.findByIsActiveTrueOrderBySymbolAsc();
        List<String> symbols = new java.util.ArrayList<>(watchlist.stream()
                .map(WatchlistEntity::getSymbol)
                .filter(symbol -> symbol != null && !symbol.isBlank())
                .toList());
        if (!symbols.contains(NIFTY_50_SYMBOL)) symbols.add(NIFTY_50_SYMBOL);
        logger.info("EOD ingestion scheduled for {} watchlist symbols plus benchmark {}",
                watchlist.size(), NIFTY_50_SYMBOL);

        int success = 0;
        int failures = 0;

        for (String symbol : symbols) {
            try {
                dataIngestionService.processSingleStock(symbol, yesterday);
                success++;
            } catch (Exception e) {
                logger.warn("EOD ingestion failed for {}: {}", symbol, e.getMessage());
                failures++;
            }

            try {
                Thread.sleep(rateLimitMs);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            }
        }

        long elapsed = System.currentTimeMillis() - start;
        logger.info("EOD ingestion completed: {} succeeded, {} failed, {}ms", success, failures, elapsed);
    }
}
