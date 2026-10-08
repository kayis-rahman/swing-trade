package com.swingtrade.data.service;

import com.swingtrade.core.metrics.DataIngestionMetrics;
import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.data.repository.StockRepository;
import com.swingtrade.data.repository.WatchlistRepository;
import com.swingtrade.data.repository.ReconciliationAuditRepository;
import com.swingtrade.data.entity.ReconciliationAuditEntity;
import com.swingtrade.domain.PriceBand;
import com.swingtrade.domain.store.PriceBandStore;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionTemplate;

import java.time.LocalDate;
import java.time.ZoneId;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.Map;
import java.util.HashMap;
import java.util.Set;
import java.util.stream.Collectors;

/**
 * Service for ingesting and managing OHLCV market data.
 * Handles backfilling historical data and validating data quality.
 */
@Service
public class DataIngestionService {

    private static final Logger logger = LoggerFactory.getLogger(DataIngestionService.class);

    private final OhlcvCandleRepository candleRepository;
    private final StockRepository stockRepository;
    private final WatchlistRepository watchlistRepository;
    private final MarketDataClientProvider marketDataClientProvider;
    private final TransactionTemplate txTemplate;
    private final DataIngestionMetrics ingestionMetrics;
    private final MarketCalendar marketCalendar;
    private final ReconciliationAuditRepository reconciliationAuditRepository;
    private final int backfillChunkDays;
    private final PriceBandStore priceBandStore;

    @Autowired
    public DataIngestionService(
        OhlcvCandleRepository candleRepository,
        StockRepository stockRepository,
        WatchlistRepository watchlistRepository,
        MarketDataClientProvider marketDataClientProvider,
        TransactionTemplate txTemplate,
        DataIngestionMetrics ingestionMetrics,
        MarketCalendar marketCalendar,
        ReconciliationAuditRepository reconciliationAuditRepository,
        @Value("${data.backfill.chunk-days:365}") int backfillChunkDays,
        PriceBandStore priceBandStore
    ) {
        this.candleRepository = candleRepository;
        this.stockRepository = stockRepository;
        this.watchlistRepository = watchlistRepository;
        this.marketDataClientProvider = marketDataClientProvider;
        this.txTemplate = txTemplate;
        this.ingestionMetrics = ingestionMetrics;
        this.marketCalendar = marketCalendar;
        this.reconciliationAuditRepository = reconciliationAuditRepository;
        this.backfillChunkDays = Math.max(1, backfillChunkDays);
        this.priceBandStore = priceBandStore;
    }

    /** Compatibility constructor for lightweight unit tests. */
    public DataIngestionService(OhlcvCandleRepository candleRepository,
        StockRepository stockRepository, WatchlistRepository watchlistRepository,
        MarketDataClientProvider marketDataClientProvider, TransactionTemplate txTemplate,
        DataIngestionMetrics ingestionMetrics, MarketCalendar marketCalendar) {
        this(candleRepository, stockRepository, watchlistRepository, marketDataClientProvider,
            txTemplate, ingestionMetrics, marketCalendar, null, 30, null);
    }

    /** Compatibility constructor for lightweight unit tests. */
    public DataIngestionService(OhlcvCandleRepository candleRepository,
        StockRepository stockRepository, WatchlistRepository watchlistRepository,
        MarketDataClientProvider marketDataClientProvider, TransactionTemplate txTemplate,
        DataIngestionMetrics ingestionMetrics) {
        this(candleRepository, stockRepository, watchlistRepository, marketDataClientProvider,
            txTemplate, ingestionMetrics, null, null, 30, null);
    }

    /**
     * Backfill historical data for a specific stock.
     *
     * @param stockSymbol the stock symbol
     * @param yearsBack number of years to backfill
     */
    @Transactional
    public void backfillStockData(String stockSymbol, int yearsBack) {
        backfillStockDataWithOutcome(stockSymbol, yearsBack);
    }

    @Transactional
    public BackfillOutcome backfillStockDataWithOutcome(String stockSymbol, int yearsBack) {
        return backfillStockDataWithOutcome(stockSymbol, "NSE", yearsBack);
    }

    @Transactional
    public BackfillOutcome backfillStockDataWithOutcome(String stockSymbol, String exchange, int yearsBack) {
        logger.info("Starting backfill for {}:{}: {} years of historical data", exchange, stockSymbol, yearsBack);

        LocalDate toDate = LocalDate.now(ZoneId.of("Asia/Kolkata"));
        LocalDate fromDate = toDate.minusYears(yearsBack);

        BackfillOutcome outcome = processIncrementalStockData(stockSymbol, exchange, fromDate, toDate);

        logger.info("Backfill completed for {}: {}", stockSymbol, outcome.sourceOutcome());
        return outcome;
    }

    /**
     * Fetches only the unpopulated tail of a requested range. Existing interior gaps are
     * handled by the bounded repair path; this method prevents a routine backfill from
     * re-downloading years of already stored candles.
     */
    public BackfillOutcome processIncrementalStockData(String symbol, LocalDate requestedFrom,
                                                       LocalDate toDate) {
        return processIncrementalStockData(symbol, "NSE", requestedFrom, toDate);
    }

    public BackfillOutcome processIncrementalStockData(String symbol, String exchange,
                                                       LocalDate requestedFrom, LocalDate toDate) {
        if (symbol == null || symbol.isBlank() || requestedFrom == null || toDate == null
                || requestedFrom.isAfter(toDate)) {
            throw new IllegalArgumentException("symbol and an ordered date range are required");
        }
        DataWindow existing = getExistingDataWindow(symbol);
        LocalDate effectiveFrom = requestedFrom;
        // A symbol can have a recent tail without having the requested historical head
        // (for example, a newly listed or partially imported symbol). In that case using
        // latestDate + 1 would permanently preserve the short tail and every candidate scan
        // would reject it before signal generation. Re-request the full requested range;
        // insertIfAbsent makes this safe for the candles already stored.
        if ((existing.earliestDate() == null || !existing.earliestDate().isAfter(requestedFrom))
                && existing.latestDate() != null && !existing.latestDate().isBefore(requestedFrom)) {
            effectiveFrom = existing.latestDate().plusDays(1);
        }
        if (effectiveFrom.isAfter(toDate)) {
            logger.info("Skipping incremental backfill for {}: existing data reaches {}", symbol,
                existing.latestDate());
            return new BackfillOutcome("ALREADY_CURRENT", 0, 0, 0, null);
        }
        logger.info("Incremental backfill for {}: requested {} to {}, fetching {} to {} in {}-day chunks",
            symbol, requestedFrom, toDate, effectiveFrom, toDate, backfillChunkDays);
        return processStockDataInChunks(symbol, exchange, effectiveFrom, toDate);
    }

    private BackfillOutcome processStockDataInChunks(String symbol, String exchange,
                                                     LocalDate fromDate, LocalDate toDate) {
        int fetched = 0;
        int saved = 0;
        int invalid = 0;
        String firstError = null;
        boolean receivedData = false;
        LocalDate chunkStart = fromDate;
        while (!chunkStart.isAfter(toDate)) {
            LocalDate chunkEnd = chunkStart.plusDays(backfillChunkDays - 1L);
            if (chunkEnd.isAfter(toDate)) chunkEnd = toDate;
            BackfillOutcome outcome = processStockDataWithOutcome(symbol, exchange, chunkStart, chunkEnd);
            fetched += outcome.fetchedRows();
            saved += outcome.savedRows();
            invalid += outcome.invalidRows();
            if ("TRANSIENT_SOURCE_FAILURE".equals(outcome.sourceOutcome())) {
                if (firstError == null) firstError = outcome.errorMessage();
                break;
            }
            receivedData |= !"NO_USABLE_DATA".equals(outcome.sourceOutcome());
            if (firstError == null) firstError = outcome.errorMessage();
            chunkStart = chunkEnd.plusDays(1);
        }
        String sourceOutcome = firstError != null ? "TRANSIENT_SOURCE_FAILURE"
            : fetched == 0 ? "NO_USABLE_DATA"
            : invalid > 0 && saved == 0 ? "INVALID_ROWS_REJECTED"
            : receivedData ? "DATA_RECEIVED" : "NO_USABLE_DATA";
        return new BackfillOutcome(sourceOutcome, fetched, saved, invalid, firstError);
    }

    /** Returns the earliest and latest stored candle dates for a symbol. */
    public DataWindow getExistingDataWindow(String symbol) {
        if (symbol == null || symbol.isBlank()) {
            throw new IllegalArgumentException("symbol is required");
        }
        LocalDate earliest = candleRepository.findEarliestBySymbol(symbol).map(OhlcvCandleEntity::getDate).orElse(null);
        LocalDate latest = candleRepository.findLatestBySymbol(symbol).map(OhlcvCandleEntity::getDate).orElse(null);
        return new DataWindow(earliest, latest);
    }

    /**
     * Process stock data for a specific date range.
     *
     * @param symbol the stock symbol
     * @param fromDate start date
     * @param toDate end date
     */
    public void processStockData(String symbol, LocalDate fromDate, LocalDate toDate) {
        processStockDataWithOutcome(symbol, fromDate, toDate);
    }

    /** Returns source quality rather than making callers infer it from a candle count. */
    public BackfillOutcome processStockDataWithOutcome(String symbol, LocalDate fromDate, LocalDate toDate) {
        return processStockDataWithOutcome(symbol, "NSE", fromDate, toDate);
    }

    public BackfillOutcome processStockDataWithOutcome(String symbol, String exchange,
                                                       LocalDate fromDate, LocalDate toDate) {
        logger.info("Processing data for {} from {} to {}", symbol, fromDate, toDate);

        // Batch-fetch all candles in one API call
        Iterable<CandleData> candles;
        try {
            MarketDataClient client = marketDataClientProvider.getClient();
            candles = "NSE".equalsIgnoreCase(exchange)
                ? client.fetchCandles(symbol, fromDate, toDate)
                : client.fetchCandles(symbol, exchange, fromDate, toDate);
        } catch (RuntimeException e) {
            logger.warn("Market data source failed for {}: {}", symbol, e.getMessage());
            return new BackfillOutcome("TRANSIENT_SOURCE_FAILURE", 0, 0, 0, e.getMessage());
        }
        int saved = 0;
        int skipped = 0;
        int invalid = 0;
        int fetched = 0;
        for (CandleData candle : candles) {
            fetched++;
            if (isNseTradingSession(candle.date()) && CandleValidator.isValid(candle)) {
                Integer insertedResult = txTemplate.execute(status -> saveCandle(symbol, candle));
                int inserted = insertedResult == null ? 1 : insertedResult;
                if (inserted == 1) {
                    saved++;
                } else {
                    skipped++;
                }
            } else {
                skipped++;
                invalid++;
            }
        }
        logger.info("Processed {}: {} candles fetched, {} newly saved, {} rejected by validation",
            symbol, saved + skipped, saved, skipped);
        String outcome = fetched == 0 ? "NO_USABLE_DATA" : invalid > 0 && saved == 0
            ? "INVALID_ROWS_REJECTED" : "DATA_RECEIVED";
        return new BackfillOutcome(outcome, fetched, saved, invalid, null);
    }

    /**
     * Process a single stock for a specific date.
     *
     * @param symbol the stock symbol
     * @param date the trading date
     */
    public void processSingleStock(String symbol, LocalDate date) {
        logger.debug("Processing single stock: {} for date {}", symbol, date);

        if (!isNseTradingSession(date)) {
            return;
        }

        persistPriceBand(symbol, date);
        if (candleRepository.existsBySymbolAndDate(symbol, date)) {
            logger.trace("Candle already exists for {}: {}", symbol, date);
            return;
        }

        CandleData candle = marketDataClientProvider.getClient().fetchCandle(symbol, date);
        if (candle != null && CandleValidator.isValid(candle)) {
            txTemplate.execute(status -> {
                saveCandle(symbol, candle);
                ingestionMetrics.recordCandleIngested();
                return null;
            });
            logger.trace("Saved candle for {}: {}", symbol, date);
        } else if (candle != null) {
            logger.debug("Rejected invalid candle for {} on {}: {}", symbol, date, candle);
            ingestionMetrics.recordFetchFailure("unknown");
        }
    }

    private void persistPriceBand(String symbol, LocalDate date) {
        if (priceBandStore == null) return;
        try {
            PriceBand band = marketDataClientProvider.getClient().fetchPriceBand(symbol, date);
            if (band != null) priceBandStore.save(band);
        } catch (RuntimeException e) {
            // Circuit data is supplementary; a provider outage must not discard a valid candle.
            logger.warn("Price-band fetch failed for {} on {}: {}", symbol, date, e.getMessage());
        }
    }

    /**
     * Pull the latest completed date and repair missing sessions inside the
     * symbol's existing history. This is intentionally separate from the
     * multi-year backfill so the daily orchestration can close data gaps
     * without re-downloading every historical candle on every run.
     */
    private static final int GAP_REPAIR_LOOKBACK_DAYS = 30;

    public String fetchLatestAndRepairGaps(String symbol, LocalDate latestDate) {
        processSingleStock(symbol, latestDate);

        Optional<OhlcvCandleEntity> earliest = candleRepository.findEarliestBySymbol(symbol);
        if (earliest.isEmpty() || earliest.get().getDate() == null
            || earliest.get().getDate().isAfter(latestDate)) {
            return "latest date pulled; no historical range available for gap repair";
        }

        // Bounded to a recent rolling window, not the symbol's entire history: the
        // nse_holidays table is often incomplete, so a genuine holiday looks like a
        // permanent "missing" trading day. Scanning the full history on every call
        // re-requests years of already-settled data from the market data client for a
        // gap that will never close, which is what was blowing past the DATA_FETCH
        // stage timeout.
        LocalDate fromDate = earliest.get().getDate().isAfter(latestDate.minusDays(GAP_REPAIR_LOOKBACK_DAYS))
            ? earliest.get().getDate()
            : latestDate.minusDays(GAP_REPAIR_LOOKBACK_DAYS);
        List<LocalDate> expectedDates = getTradingDays(fromDate, latestDate);
        if (expectedDates.isEmpty()) {
            return "latest date pulled; no trading sessions in range";
        }

        List<OhlcvCandleEntity> existing = candleRepository.findBySymbolAndDateRange(
            symbol, fromDate, latestDate,
            org.springframework.data.domain.Pageable.unpaged());
        Set<LocalDate> existingDates = existing.stream()
            .map(OhlcvCandleEntity::getDate)
            .filter(java.util.Objects::nonNull)
            .collect(Collectors.toSet());
        List<LocalDate> missingDates = expectedDates.stream()
            .filter(date -> !existingDates.contains(date))
            .toList();

        if (missingDates.isEmpty()) {
            return "latest date pulled; no gaps found";
        }

        // One bounded range request lets the active client repair all gaps in
        // one call while insertIfAbsent preserves already stored candles.
        processStockData(symbol, fromDate, latestDate);
        return "latest date pulled; requested gap repair for " + missingDates.size() + " sessions";
    }

    // Upstox is unconfigured — this ingestion path is commented out (kept for future re-enablement).
    // /**
    //  * Pull data from Upstox API.
    //  * Uses the MarketDataClient (UpstoxServiceClient) for authenticated API calls.
    //  *
    //  * @param symbol the stock symbol
    //  * @param startDate start date
    //  * @param endDate end date
    //  * @return number of candles successfully ingested
    //  */
    // @Transactional
    // public int pullDataFromUpstox(String symbol, LocalDate startDate, LocalDate endDate) {
    //     logger.info("Pulling data from Upstox for {} from {} to {}", symbol, startDate, endDate);

    //     int count = 0;
    //     LocalDate current = startDate;

    //     while (!current.isAfter(endDate)) {
    //         if (candleRepository.existsBySymbolAndDate(symbol, current)) {
    //             logger.trace("Candle already exists for {}: {}", symbol, current);
    //             current = current.plusDays(1);
    //             continue;
    //         }

    //         CandleData candle = marketDataClientProvider.getClient().fetchCandle(symbol, current);
    //         if (candle != null && CandleValidator.isValid(candle)) {
    //             saveCandle(symbol, candle);
    //             count++;
    //             ingestionMetrics.recordCandleIngested();
    //             logger.trace("Ingested candle for {}: {}", symbol, current);
    //         } else if (candle != null) {
    //             logger.debug("Rejected invalid candle for {} on {}: {}", symbol, current, candle);
    //             ingestionMetrics.recordFetchFailure("upstox");
    //         } else {
    //             logger.warn("Failed to fetch candle for {}: {}", symbol, current);
    //             ingestionMetrics.recordFetchFailure("upstox");
    //         }

    //         current = current.plusDays(1);
    //     }

    //     logger.info("Data pull completed: {} candles ingested for {}", count, symbol);
    //     return count;
    // }

    /**
     * Save a candle to the database.
     *
     * @param symbol the stock symbol
     * @param candle the candle data
     */
    private int saveCandle(String symbol, CandleData candle) {
        return candleRepository.insertIfAbsent(symbol, candle.date(), candle.open(), candle.high(),
            candle.low(), candle.close(), candle.volume(), candle.adjClose());
    }

    private List<LocalDate> getTradingDays(LocalDate startDate, LocalDate endDate) {
        List<LocalDate> tradingDays = new ArrayList<>();
        LocalDate current = startDate;
        while (!current.isAfter(endDate)) {
            if (isNseTradingSession(current)) {
                tradingDays.add(current);
            }
            current = current.plusDays(1);
        }
        return tradingDays;
    }

    private boolean isNseTradingSession(LocalDate date) {
        return marketCalendar == null
            ? date.getDayOfWeek().getValue() <= 5
            : marketCalendar.isNseTradingSession(date);
    }

    /**
     * Validate data quality for a stock.
     * Checks for missing candles and anomalies.
     *
     * @param stockSymbol the stock symbol
     * @param fromDate start date for validation
     * @param toDate end date for validation
     * @return DataQualityReport with validation results
     */
    public DataQualityReport validateDataQuality(String stockSymbol, LocalDate fromDate, LocalDate toDate) {
        logger.info("Validating data quality for {} from {} to {}", stockSymbol, fromDate, toDate);

        DataQualityReport report = new DataQualityReport();
        report.setStockSymbol(stockSymbol);
        report.setFromDate(fromDate);
        report.setToDate(toDate);

        // Calculate expected trading days
        long expectedTradingDays = getTradingDays(fromDate, toDate).size();
        report.setExpectedTradingDays(expectedTradingDays);

        // Get actual candles
        List<OhlcvCandleEntity> candles = candleRepository.findBySymbolAndDateRange(
            stockSymbol, fromDate, toDate,
            org.springframework.data.domain.Pageable.unpaged()
        );

        Set<LocalDate> actualDates = candles.stream()
            .map(OhlcvCandleEntity::getDate)
            .collect(Collectors.toSet());
        report.setActualTradingDays(actualDates.size());

        // Find gaps
        List<LocalDate> tradingDays = getTradingDays(fromDate, toDate);
        List<DataGap> gaps = new ArrayList<>();
        LocalDate currentGapStart = null;

        for (LocalDate expectedDate : tradingDays) {
            if (!actualDates.contains(expectedDate)) {
                if (currentGapStart == null) {
                    currentGapStart = expectedDate;
                }
            } else {
                if (currentGapStart != null) {
                    gaps.add(new DataGap("Missing data", currentGapStart, expectedDate.minusDays(1)));
                    currentGapStart = null;
                }
            }
        }

        // Handle gap at end of range
        if (currentGapStart != null) {
            gaps.add(new DataGap("Missing data", currentGapStart, toDate));
        }

        report.setGaps(gaps);
        report.setHasIssues(!gaps.isEmpty());

        // Check for price anomalies
        List<PriceAnomaly> anomalies = findPriceAnomalies(candles);
        report.setAnomalies(anomalies);

        if (report.isCritical()) {
            logger.warn("Critical data quality issues for {} from {} to {}: gapRate={}%, gaps={}, anomalies={}",
                stockSymbol, fromDate, toDate, report.getGapPercentage(), gaps.size(), anomalies.size());
        }

        return report;
    }

    /**
     * Find price anomalies in candle data.
     *
     * @param candles list of candles to check
     * @return list of anomalies found
     */
    private List<PriceAnomaly> findPriceAnomalies(List<OhlcvCandleEntity> candles) {
        List<PriceAnomaly> anomalies = new ArrayList<>();

        for (OhlcvCandleEntity candle : candles) {
            // High should be >= all other prices
            if (candle.getHighPrice().compareTo(candle.getOpenPrice()) < 0 ||
                candle.getHighPrice().compareTo(candle.getLowPrice()) < 0 ||
                candle.getHighPrice().compareTo(candle.getClosePrice()) < 0) {
                anomalies.add(new PriceAnomaly(
                    candle.getSymbol(),
                    candle.getDate(),
                    "High price invalid",
                    String.format("High: %s should be >= Open: %s, Low: %s, Close: %s",
                        candle.getHighPrice(), candle.getOpenPrice(),
                        candle.getLowPrice(), candle.getClosePrice())
                ));
                continue;
            }

            // Low should be <= all other prices
            if (candle.getLowPrice().compareTo(candle.getOpenPrice()) > 0 ||
                candle.getLowPrice().compareTo(candle.getHighPrice()) > 0 ||
                candle.getLowPrice().compareTo(candle.getClosePrice()) > 0) {
                anomalies.add(new PriceAnomaly(
                    candle.getSymbol(),
                    candle.getDate(),
                    "Low price invalid",
                    String.format("Low: %s should be <= Open: %s, High: %s, Close: %s",
                        candle.getLowPrice(), candle.getOpenPrice(),
                        candle.getHighPrice(), candle.getClosePrice())
                ));
            }
        }

        return anomalies;
    }

    /**
     * Get the latest candle for a stock.
     *
     * @param symbol the stock symbol
     * @return optional containing the latest candle
     */
    public Optional<OhlcvCandleEntity> getLatestCandle(String symbol) {
        return candleRepository.findLatestBySymbol(symbol);
    }

    /** Bounded dry-run reconciliation; apply is atomic after the provider response is complete. */
    public Map<String, Object> reconcile(String symbol, LocalDate fromDate, LocalDate toDate, boolean apply) {
        if (toDate.isBefore(fromDate) || fromDate.plusDays(10).isBefore(toDate)) {
            throw new IllegalArgumentException("reconciliation range must be at most 10 calendar days");
        }
        if (!marketCalendar.isCoverageVerified(fromDate, toDate)) {
            return Map.of("symbol", symbol, "from", fromDate, "to", toDate,
                "status", "CALENDAR_INCOMPLETE");
        }
        List<LocalDate> expected = marketCalendar.expectedNseSessions(fromDate, toDate);
        List<OhlcvCandleEntity> stored = candleRepository.findBySymbolAndDateRange(
            symbol, fromDate, toDate, org.springframework.data.domain.Pageable.unpaged());
        Set<LocalDate> storedDates = stored.stream().map(OhlcvCandleEntity::getDate).collect(Collectors.toSet());
        List<LocalDate> missing = expected.stream().filter(d -> !storedDates.contains(d)).toList();
        List<LocalDate> invalidStored = stored.stream().map(OhlcvCandleEntity::getDate)
            .filter(d -> !expected.contains(d)).toList();
        List<CandleData> source = new ArrayList<>();
        marketDataClientProvider.getClient().fetchCandles(symbol, fromDate, toDate).forEach(source::add);
        Map<LocalDate, CandleData> validSource = source.stream()
            .filter(c -> expected.contains(c.date()) && CandleValidator.isValid(c, false))
            .collect(Collectors.toMap(CandleData::date, c -> c, (a, b) -> a));
        List<LocalDate> sourceIncomplete = missing.stream().filter(d -> !validSource.containsKey(d)).toList();
        String terminal = sourceIncomplete.isEmpty() ? "COMPLETED" : "SOURCE_INCOMPLETE";
        int inserted = 0;
        int removed = 0;
        if (apply && "COMPLETED".equals(terminal)) {
            final Map<LocalDate, CandleData> fetched = validSource;
            int[] counts = txTemplate.execute(status -> {
                int deletes = invalidStored.stream().mapToInt(d -> candleRepository.deleteBySymbolAndDate(symbol, d)).sum();
                int inserts = missing.stream().mapToInt(d -> saveCandle(symbol, fetched.get(d))).sum();
                return new int[]{inserts, deletes};
            });
            if (counts != null) {
                inserted = counts[0];
                removed = counts[1];
            }
        }
        Map<String, Object> result = new HashMap<>();
        Map<LocalDate, List<OhlcvCandleEntity>> byDate = stored.stream()
            .collect(Collectors.groupingBy(OhlcvCandleEntity::getDate));
        List<Map<String, Object>> duplicates = byDate.entrySet().stream()
            .filter(e -> e.getValue().size() > 1)
            .map(e -> Map.<String, Object>of("date", e.getKey(), "ids",
                e.getValue().stream().map(OhlcvCandleEntity::getId).toList(), "count", e.getValue().size()))
            .toList();
        result.put("symbol", symbol);
        result.put("from", fromDate);
        result.put("to", toDate);
        result.put("expected", expected);
        result.put("storedDates", storedDates);
        result.put("missing", missing);
        result.put("nonTradingDayDates", invalidStored);
        result.put("invalidStored", invalidStored);
        result.put("duplicateGroups", duplicates);
        result.put("sourceIncomplete", sourceIncomplete);
        result.put("inserted", inserted);
        result.put("invalidStoredRemoved", removed);
        String quality = !duplicates.isEmpty() ? "DUPLICATE"
            : !invalidStored.isEmpty() ? "INVALID_DATE"
            : !missing.isEmpty() ? "MISSING" : "GOOD";
        result.put("qualityStatus", quality);
        result.put("earliestDate", stored.stream().map(OhlcvCandleEntity::getDate).min(LocalDate::compareTo).orElse(null));
        result.put("latestDate", stored.stream().map(OhlcvCandleEntity::getDate).max(LocalDate::compareTo).orElse(null));
        result.put("count", stored.size());
        result.put("status", terminal);
        if (apply && reconciliationAuditRepository != null) {
            reconciliationAuditRepository.save(new ReconciliationAuditEntity(symbol, fromDate, toDate,
                terminal, missing.size(), inserted, removed));
        }
        return result;
    }

    /**
     * Get recent candles for a stock.
     *
     * @param symbol the stock symbol
     * @param days number of days to fetch
     * @return list of candles
     */
    public List<OhlcvCandleEntity> getRecentCandles(String symbol, int days) {
        return candleRepository.findTopBySymbolOrderByDateDesc(
            symbol,
            org.springframework.data.domain.PageRequest.of(0, days)
        );
    }

    /**
     * Data quality report for validating candle data.
     */
    public static class DataQualityReport {
        private String stockSymbol;
        private LocalDate fromDate;
        private LocalDate toDate;
        private long expectedTradingDays;
        private long actualTradingDays;
        private List<DataGap> gaps = new ArrayList<>();
        private List<PriceAnomaly> anomalies = new ArrayList<>();
        private boolean hasIssues = false;

        public String getStockSymbol() { return stockSymbol; }
        public void setStockSymbol(String stockSymbol) { this.stockSymbol = stockSymbol; }
        public LocalDate getFromDate() { return fromDate; }
        public void setFromDate(LocalDate fromDate) { this.fromDate = fromDate; }
        public LocalDate getToDate() { return toDate; }
        public void setToDate(LocalDate toDate) { this.toDate = toDate; }
        public long getExpectedTradingDays() { return expectedTradingDays; }
        public void setExpectedTradingDays(long expectedTradingDays) {
            this.expectedTradingDays = expectedTradingDays;
            refreshIssueState();
        }
        public long getActualTradingDays() { return actualTradingDays; }
        public void setActualTradingDays(long actualTradingDays) {
            this.actualTradingDays = actualTradingDays;
            refreshIssueState();
        }
        public List<DataGap> getGaps() { return gaps; }
        public void setGaps(List<DataGap> gaps) {
            this.gaps = gaps == null ? new ArrayList<>() : gaps;
            refreshIssueState();
        }
        public List<PriceAnomaly> getAnomalies() { return anomalies; }
        public void setAnomalies(List<PriceAnomaly> anomalies) {
            this.anomalies = anomalies == null ? new ArrayList<>() : anomalies;
            refreshIssueState();
        }
        public boolean hasIssues() { return hasIssues; }
        public void setHasIssues(boolean hasIssues) { this.hasIssues = hasIssues; }
        public double getGapPercentage() {
            return expectedTradingDays == 0 ? 0.0
                : ((expectedTradingDays - Math.min(expectedTradingDays, actualTradingDays)) * 100.0)
                    / expectedTradingDays;
        }
        public boolean isCritical() { return getGapPercentage() > 10.0 || !anomalies.isEmpty(); }

        private void refreshIssueState() {
            this.hasIssues = actualTradingDays < expectedTradingDays
                || !gaps.isEmpty()
                || !anomalies.isEmpty();
        }
    }

    /**
     * Represents a data gap.
     */
    public static class DataGap {
        private String description;
        private LocalDate startDate;
        private LocalDate endDate;

        public DataGap(String description, LocalDate startDate, LocalDate endDate) {
            this.description = description;
            this.startDate = startDate;
            this.endDate = endDate;
        }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }
        public LocalDate getStartDate() { return startDate; }
        public void setStartDate(LocalDate startDate) { this.startDate = startDate; }
        public LocalDate getEndDate() { return endDate; }
        public void setEndDate(LocalDate endDate) { this.endDate = endDate; }
    }

    /**
     * Represents a price anomaly.
     */
    public static class PriceAnomaly {
        private String symbol;
        private LocalDate date;
        private String type;
        private String details;

        public PriceAnomaly(String symbol, LocalDate date, String type, String details) {
            this.symbol = symbol;
            this.date = date;
            this.type = type;
            this.details = details;
        }

        public String getSymbol() { return symbol; }
        public void setSymbol(String symbol) { this.symbol = symbol; }
        public LocalDate getDate() { return date; }
        public void setDate(LocalDate date) { this.date = date; }
        public String getType() { return type; }
        public void setType(String type) { this.type = type; }
        public String getDetails() { return details; }
        public void setDetails(String details) { this.details = details; }
    }

    public record BackfillOutcome(String sourceOutcome, int fetchedRows, int savedRows,
                                  int invalidRows, String errorMessage) {}

    public record DataWindow(LocalDate earliestDate, LocalDate latestDate) {}
}
