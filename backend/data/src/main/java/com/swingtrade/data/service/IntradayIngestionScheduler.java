package com.swingtrade.data.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.time.ZoneId;

/**
 * Scheduled intraday (15-minute) ingestion for the watchlist.
 *
 * <p>Runs on its own cron after the normal session close (15:30 IST), so every
 * bar of the session is final, and skips non-trading days via the exchange
 * holiday calendar. It is strictly additive: the end-of-day scheduler, its cron,
 * and the daily ingestion path are untouched.
 */
@Service
public class IntradayIngestionScheduler {

    private static final Logger logger = LoggerFactory.getLogger(IntradayIngestionScheduler.class);
    private static final ZoneId IST = ZoneId.of("Asia/Kolkata");

    private final IntradayIngestionService intradayIngestionService;
    private final MarketCalendar marketCalendar;

    @Value("${intraday.ingestion.enabled:true}")
    private boolean enabled;

    /** Bounded lookback for each scheduled run; the service resumes per symbol. */
    @Value("${intraday.ingestion.lookback-days:10}")
    private int lookbackDays;

    public IntradayIngestionScheduler(IntradayIngestionService intradayIngestionService,
                                      MarketCalendar marketCalendar) {
        this.intradayIngestionService = intradayIngestionService;
        this.marketCalendar = marketCalendar;
    }

    @Scheduled(cron = "0 45 15 * * MON-FRI", zone = "Asia/Kolkata")
    public void ingestSessionBars() {
        if (!enabled) {
            logger.debug("Intraday ingestion scheduler disabled");
            return;
        }
        LocalDate today = LocalDate.now(IST);
        if (!marketCalendar.isNseTradingSession(today)) {
            logger.info("Intraday ingestion skipped: {} is not an NSE trading session", today);
            return;
        }
        LocalDate from = today.minusDays(Math.max(0, lookbackDays));
        logger.info("Intraday ingestion scheduled run for window {} to {}", from, today);
        try {
            IntradayIngestionService.IntradayIngestionResult result =
                intradayIngestionService.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, from, today);
            logger.info("Intraday ingestion scheduled run completed: {} symbols succeeded, {} failed, {} bars saved",
                result.symbolsSucceeded(), result.symbolsFailed(), result.barsSaved());
        } catch (Exception e) {
            logger.warn("Intraday ingestion scheduled run failed: {}", e.getMessage());
        }
    }
}
