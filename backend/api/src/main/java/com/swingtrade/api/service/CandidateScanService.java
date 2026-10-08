package com.swingtrade.api.service;

import com.swingtrade.data.entity.CandidateScanResultEntity;
import com.swingtrade.data.entity.CandidateScanRunEntity;
import com.swingtrade.data.entity.StockEntity;
import com.swingtrade.data.repository.CandidateScanResultRepository;
import com.swingtrade.data.repository.CandidateScanRunRepository;
import com.swingtrade.data.repository.CandidateHistoryEligibilityRepository;
import com.swingtrade.data.entity.CandidateHistoryEligibilityEntity;
import com.swingtrade.data.repository.FyersSymbolRepository;
import com.swingtrade.data.repository.StockRepository;
import com.swingtrade.data.service.DataIngestionService;
import com.swingtrade.data.service.AppSettingsService;
import com.swingtrade.data.service.WatchlistService;
import com.swingtrade.domain.store.CandleStore;
import com.swingtrade.strategy.BacktestConfig;
import com.swingtrade.strategy.BacktestEngine;
import com.swingtrade.strategy.BacktestResult;
import com.swingtrade.strategy.PriceActionSignalEngine;
import com.swingtrade.strategy.SignalResult;
import com.swingtrade.strategy.WalkForwardEvaluation;
import jakarta.annotation.PreDestroy;
import jakarta.annotation.PostConstruct;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionSynchronization;
import org.springframework.transaction.support.TransactionSynchronizationManager;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.math.BigDecimal;
import java.io.IOException;
import java.time.LocalDateTime;
import java.time.LocalDate;
import java.time.ZoneId;
import java.util.ArrayList;
import java.util.Deque;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentLinkedDeque;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Semaphore;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicReference;
import org.springframework.data.domain.PageRequest;

@Service
public class CandidateScanService {
    private static final Logger logger = LoggerFactory.getLogger(CandidateScanService.class);
    private static final int MIN_CANDLES = 60;
    private static final ZoneId MARKET_ZONE = ZoneId.of("Asia/Kolkata");
    private static final String KEY_MIN_WIN_RATE = "candidate-scan.min-win-rate";
    private static final String KEY_MIN_TOTAL_RETURN = "candidate-scan.min-total-return";
    private static final String KEY_MAX_CONCURRENT = "candidate-scan.max-concurrent";
    private static final String KEY_BACKFILL_YEARS = "candidate-scan.backfill-years";
    private static final String KEY_MIN_TRADES = "candidate-scan.min-trades";
    private static final String KEY_OOS_DAYS = "candidate-scan.out-of-sample-days";
    private static final String KEY_OOS_FOLDS = "candidate-scan.out-of-sample-folds";
    private static final String KEY_MIN_STRATEGY_BUYS = "candidate-scan.min-strategy-buys";
    private static final double DEFAULT_MIN_WIN_RATE = 45.0;
    private static final double DEFAULT_MIN_TOTAL_RETURN = 0.0;
    private static final int DEFAULT_MIN_TRADES = 15;
    private static final int DEFAULT_OOS_DAYS = 252;
    private static final int DEFAULT_OOS_FOLDS = 3;
    private static final int DEFAULT_MIN_STRATEGY_BUYS = 2;
    record ScanTarget(String symbol, String exchange) { }

    private final FyersSymbolRepository symbolRepository;
    private final StockRepository stockRepository;
    private final CandidateScanRunRepository runRepository;
    private final CandidateScanResultRepository resultRepository;
    private final CandidateHistoryEligibilityRepository eligibilityRepository;
    private final DataIngestionService ingestionService;
    private final WatchlistService watchlistService;
    private final CandleStore candleStore;
    private final PriceActionSignalEngine signalEngine;
    private final BacktestEngine backtestEngine;
    private final CandidateStrategyEvaluator candidateStrategyEvaluator;
    private final int defaultBackfillYears;
    private final long delayMs;
    private final AppSettingsService appSettingsService;
    private volatile int maxConcurrent;
    private volatile Semaphore semaphore;
    private final ExecutorService executor;
    private final AtomicReference<UUID> activeRun = new AtomicReference<>();
    private final ConcurrentHashMap<UUID, AtomicBoolean> cancellations = new ConcurrentHashMap<>();
    private final ConcurrentHashMap<UUID, AtomicBoolean> pauses = new ConcurrentHashMap<>();
    private final Object pauseMonitor = new Object();
    private final ConcurrentHashMap<UUID, List<SseEmitter>> emitters = new ConcurrentHashMap<>();
    private final ConcurrentHashMap<UUID, Deque<ScanLogEvent>> logHistory = new ConcurrentHashMap<>();
    private static final int MAX_LOG_HISTORY = 500;

    @Autowired
    public CandidateScanService(FyersSymbolRepository symbolRepository,
                                StockRepository stockRepository,
                                CandidateScanRunRepository runRepository,
                                CandidateScanResultRepository resultRepository,
                                CandidateHistoryEligibilityRepository eligibilityRepository,
                                DataIngestionService ingestionService,
                                WatchlistService watchlistService,
                                AppSettingsService appSettingsService,
                                CandleStore candleStore,
                                PriceActionSignalEngine signalEngine,
                                BacktestEngine backtestEngine,
                                CandidateStrategyEvaluator candidateStrategyEvaluator,
                                @Value("${candidate-scan.backfill-years:5}") int backfillYears,
                                @Value("${candidate-scan.delay-ms:1000}") long delayMs,
                                @Value("${candidate-scan.max-concurrent:3}") int maxConcurrent) {
        this.symbolRepository = symbolRepository;
        this.stockRepository = stockRepository;
        this.runRepository = runRepository;
        this.resultRepository = resultRepository;
        this.eligibilityRepository = eligibilityRepository;
        this.ingestionService = ingestionService;
        this.watchlistService = watchlistService;
        this.appSettingsService = appSettingsService;
        this.candleStore = candleStore;
        this.signalEngine = signalEngine;
        this.backtestEngine = backtestEngine;
        this.candidateStrategyEvaluator = candidateStrategyEvaluator;
        this.defaultBackfillYears = backfillYears;
        this.delayMs = Math.max(0, delayMs);
        this.maxConcurrent = Math.max(1, Math.min(maxConcurrent, 12));
        this.semaphore = new Semaphore(this.maxConcurrent);
        this.executor = Executors.newThreadPerTaskExecutor(
            Thread.ofVirtual().name("candidate-scan-", 0).factory());
    }

    public CandidateScanService(FyersSymbolRepository symbolRepository,
                                CandidateScanRunRepository runRepository,
                                CandidateScanResultRepository resultRepository,
                                DataIngestionService ingestionService,
                                AppSettingsService appSettingsService,
                                CandleStore candleStore,
                                PriceActionSignalEngine signalEngine,
                                BacktestEngine backtestEngine,
                                int backfillYears, long delayMs, int maxConcurrent) {
        this(symbolRepository, null, runRepository, resultRepository, null, ingestionService, null,
            appSettingsService, candleStore, signalEngine, backtestEngine,
            null,
            backfillYears, delayMs, maxConcurrent);
    }

    @PostConstruct
    @Transactional
    void recoverInterruptedRuns() {
        List<CandidateScanRunEntity> interruptedRuns = new ArrayList<>(runRepository.findByStatus("RUNNING"));
        interruptedRuns.addAll(runRepository.findByStatus("PAUSED"));
        for (CandidateScanRunEntity run : interruptedRuns) {
            run.setStatus("CANCELLED");
            run.setCompletedAt(LocalDateTime.now(MARKET_ZONE));
            run.setErrorMessage("Scan interrupted by API restart.");
            runRepository.save(run);
        }
        if (!interruptedRuns.isEmpty()) {
            logger.info("Marked {} candidate scan(s) as cancelled after API restart.", interruptedRuns.size());
        }
    }

    @Transactional
    /** Manual scans discover candidates from NSE stock rows. */
    public CandidateScanRunEntity start() { return start(false, false); }

    /** Scheduled scans persist a handoff request for their qualified candidates. */
    @Transactional
    public CandidateScanRunEntity startScheduled() {
        CandidateScanRunEntity run = start(true, false);
        return run;
    }

    private CandidateScanRunEntity start(boolean scheduled, boolean watchlistOnly) {
        UUID existing = activeRun.get();
        if (existing != null || runRepository.existsByStatus("RUNNING")) {
            throw new IllegalStateException("Candidate scan already running: " + existing);
        }

        maxConcurrent = configuredMaxConcurrent();
        semaphore = new Semaphore(maxConcurrent);

        List<ScanTarget> sourceSymbols;
        if (watchlistOnly && watchlistService != null) {
            sourceSymbols = watchlistService.getActiveWatchlist().stream()
                .map(entry -> new ScanTarget(entry.getSymbol(), entry.getExchange())).toList();
        } else if (scheduled || stockRepository == null) {
            sourceSymbols = symbolRepository.findByExchangeIgnoreCaseOrderByTradingSymbolAsc("NSE").stream()
                .map(s -> new ScanTarget(s.getTradingSymbol(), "NSE")).toList();
        } else {
            List<StockEntity> stocks = stockRepository.findAllByOrderBySymbol();
            long skippedOtherExchanges = stocks.stream()
                .filter(stock -> !"NSE".equalsIgnoreCase(stock.getExchange() == null
                    ? "" : stock.getExchange().trim()))
                .count();
            logger.info("Manual candidate scan skipped {} stock row(s) outside NSE.", skippedOtherExchanges);
            sourceSymbols = stocks.stream()
                .filter(stock -> "NSE".equalsIgnoreCase(stock.getExchange() == null
                    ? "" : stock.getExchange().trim()))
                .map(stock -> new ScanTarget(stock.getSymbol(), stock.getExchange())).toList();
        }

        List<ScanTarget> symbols = sourceSymbols.stream()
            .map(target -> new ScanTarget(
                target.symbol() == null ? "" : target.symbol().trim().toUpperCase(),
                target.exchange() == null || target.exchange().isBlank()
                    ? "NSE" : target.exchange().trim().toUpperCase()))
            .filter(target -> !target.symbol().isBlank())
            .collect(java.util.stream.Collectors.collectingAndThen(
                java.util.stream.Collectors.toCollection(LinkedHashSet::new), ArrayList::new));

        CandidateScanRunEntity run = new CandidateScanRunEntity();
        run.setRunId(UUID.randomUUID());
        run.setStatus("RUNNING");
        run.setTotalSymbols(symbols.size());
        run.setScanScope(watchlistOnly ? "WATCHLIST" : scheduled ? "NSE_BROAD" : "ACTIVE_STOCKS");
        run.setScanTrigger(scheduled ? "SCHEDULED" : "MANUAL");
        run.setStartedAt(LocalDateTime.now(MARKET_ZONE));
        run.setOrchestrationStatus("PENDING");
        runRepository.save(run);
        activeRun.set(run.getRunId());
        cancellations.put(run.getRunId(), new AtomicBoolean(false));
        pauses.put(run.getRunId(), new AtomicBoolean(false));
        logHistory.put(run.getRunId(), new ConcurrentLinkedDeque<>());
        publish(run.getRunId(), "RUN_STARTED", null, "INFO",
            "Scanning " + symbols.size() + " stock symbols with up to " + maxConcurrent + " workers.");
        Runnable scanTask = () -> execute(run.getRunId(), symbols, scheduled);
        if (TransactionSynchronizationManager.isSynchronizationActive()) {
            TransactionSynchronizationManager.registerSynchronization(new TransactionSynchronization() {
                @Override
                public void afterCommit() {
                    executor.submit(scanTask);
                }

                @Override
                public void afterCompletion(int status) {
                    if (status != STATUS_COMMITTED) {
                        clearRunState(run.getRunId());
                    }
                }
            });
        } else {
            executor.submit(scanTask);
        }
        return run;
    }

    private void clearRunState(UUID runId) {
        activeRun.compareAndSet(runId, null);
        cancellations.remove(runId);
        pauses.remove(runId);
        logHistory.remove(runId);
    }

    private int configuredMaxConcurrent() {
        try {
            return Math.max(1, Math.min(12, Integer.parseInt(
                appSettingsService.get(KEY_MAX_CONCURRENT, String.valueOf(maxConcurrent)))));
        } catch (NumberFormatException ignored) {
            return 3;
        }
    }

    private int configuredBackfillYears() {
        try {
            return Math.max(1, Math.min(10, Integer.parseInt(
                appSettingsService.get(KEY_BACKFILL_YEARS, String.valueOf(defaultBackfillYears)))));
        } catch (NumberFormatException ignored) {
            return defaultBackfillYears;
        }
    }

    private double configuredMinWinRate() {
        try {
            return Math.max(0.0, Math.min(100.0, Double.parseDouble(
                appSettingsService.get(KEY_MIN_WIN_RATE, String.valueOf(DEFAULT_MIN_WIN_RATE)))));
        } catch (NumberFormatException ignored) {
            return DEFAULT_MIN_WIN_RATE;
        }
    }

    private double configuredMinTotalReturn() {
        try {
            return Double.parseDouble(
                appSettingsService.get(KEY_MIN_TOTAL_RETURN, String.valueOf(DEFAULT_MIN_TOTAL_RETURN)));
        } catch (NumberFormatException ignored) {
            return DEFAULT_MIN_TOTAL_RETURN;
        }
    }

    private int configuredMinTrades() {
        try {
            return Math.max(1, Math.min(1000, Integer.parseInt(
                appSettingsService.get(KEY_MIN_TRADES, String.valueOf(DEFAULT_MIN_TRADES)))));
        } catch (NumberFormatException ignored) {
            return DEFAULT_MIN_TRADES;
        }
    }

    private int configuredOosDays() {
        try {
            return Math.max(60, Math.min(1000, Integer.parseInt(
                appSettingsService.get(KEY_OOS_DAYS, String.valueOf(DEFAULT_OOS_DAYS)))));
        } catch (NumberFormatException ignored) {
            return DEFAULT_OOS_DAYS;
        }
    }

    private int configuredOosFolds() {
        try {
            return Math.max(1, Math.min(8, Integer.parseInt(
                appSettingsService.get(KEY_OOS_FOLDS, String.valueOf(DEFAULT_OOS_FOLDS)))));
        } catch (NumberFormatException ignored) {
            return DEFAULT_OOS_FOLDS;
        }
    }

    private int configuredMinStrategyBuys() {
        try {
            return Math.max(1, Math.min(12, Integer.parseInt(appSettingsService.get(KEY_MIN_STRATEGY_BUYS,
                String.valueOf(DEFAULT_MIN_STRATEGY_BUYS)))));
        } catch (NumberFormatException ignored) {
            return DEFAULT_MIN_STRATEGY_BUYS;
        }
    }

    public Map<String, String> getScanSettings() {
        Map<String, String> settings = new LinkedHashMap<>();
        settings.put(KEY_MIN_WIN_RATE, String.valueOf(configuredMinWinRate()));
        settings.put(KEY_MIN_TOTAL_RETURN, String.valueOf(configuredMinTotalReturn()));
        settings.put(KEY_MAX_CONCURRENT, String.valueOf(configuredMaxConcurrent()));
        settings.put(KEY_BACKFILL_YEARS, String.valueOf(configuredBackfillYears()));
        settings.put(KEY_MIN_TRADES, String.valueOf(configuredMinTrades()));
        settings.put(KEY_OOS_DAYS, String.valueOf(configuredOosDays()));
        settings.put(KEY_OOS_FOLDS, String.valueOf(configuredOosFolds()));
        settings.put(KEY_MIN_STRATEGY_BUYS, String.valueOf(configuredMinStrategyBuys()));
        return settings;
    }

    public Map<String, String> updateScanSettings(Map<String, String> updates) {
        if (updates.containsKey(KEY_MIN_WIN_RATE)) {
            double value = Double.parseDouble(updates.get(KEY_MIN_WIN_RATE));
            if (value < 0.0 || value > 100.0) throw new IllegalArgumentException("min-win-rate must be between 0 and 100");
            appSettingsService.set(KEY_MIN_WIN_RATE, String.valueOf(value));
        }
        if (updates.containsKey(KEY_MIN_TOTAL_RETURN)) {
            appSettingsService.set(KEY_MIN_TOTAL_RETURN, String.valueOf(Double.parseDouble(updates.get(KEY_MIN_TOTAL_RETURN))));
        }
        if (updates.containsKey(KEY_MAX_CONCURRENT)) {
            int value = Integer.parseInt(updates.get(KEY_MAX_CONCURRENT));
            if (value < 1 || value > 12) throw new IllegalArgumentException("max-concurrent must be between 1 and 12");
            appSettingsService.set(KEY_MAX_CONCURRENT, String.valueOf(value));
        }
        if (updates.containsKey(KEY_BACKFILL_YEARS)) {
            int value = Integer.parseInt(updates.get(KEY_BACKFILL_YEARS));
            if (value < 1 || value > 10) throw new IllegalArgumentException("backfill-years must be between 1 and 10");
            appSettingsService.set(KEY_BACKFILL_YEARS, String.valueOf(value));
        }
        if (updates.containsKey(KEY_MIN_TRADES)) {
            int value = Integer.parseInt(updates.get(KEY_MIN_TRADES));
            if (value < 1 || value > 1000) throw new IllegalArgumentException("min-trades must be between 1 and 1000");
            appSettingsService.set(KEY_MIN_TRADES, String.valueOf(value));
        }
        if (updates.containsKey(KEY_OOS_DAYS)) {
            int value = Integer.parseInt(updates.get(KEY_OOS_DAYS));
            if (value < 60 || value > 1000) throw new IllegalArgumentException("out-of-sample-days must be between 60 and 1000");
            appSettingsService.set(KEY_OOS_DAYS, String.valueOf(value));
        }
        if (updates.containsKey(KEY_OOS_FOLDS)) {
            int value = Integer.parseInt(updates.get(KEY_OOS_FOLDS));
            if (value < 1 || value > 8) throw new IllegalArgumentException("out-of-sample-folds must be between 1 and 8");
            appSettingsService.set(KEY_OOS_FOLDS, String.valueOf(value));
        }
        if (updates.containsKey(KEY_MIN_STRATEGY_BUYS)) {
            int value = Integer.parseInt(updates.get(KEY_MIN_STRATEGY_BUYS));
            if (value < 1 || value > 12) throw new IllegalArgumentException("min-strategy-buys must be between 1 and 12");
            appSettingsService.set(KEY_MIN_STRATEGY_BUYS, String.valueOf(value));
        }
        return getScanSettings();
    }

    public SseEmitter stream(UUID runId) {
        if (getRun(runId) == null) return null;
        SseEmitter emitter = new SseEmitter(0L);
        List<SseEmitter> runEmitters = emitters.computeIfAbsent(runId,
            ignored -> new java.util.concurrent.CopyOnWriteArrayList<>());
        runEmitters.add(emitter);
        emitter.onCompletion(() -> runEmitters.remove(emitter));
        emitter.onTimeout(() -> runEmitters.remove(emitter));
        emitter.onError(ignored -> runEmitters.remove(emitter));
        CandidateScanRunEntity run = getRun(runId);
        send(emitter, new ScanLogEvent(
            "RUN_SNAPSHOT", runId, null, "INFO",
            "Connected to candidate scan (" + run.getStatus() + ").",
            run.getCompletedSymbols(), run.getTotalSymbols(), run.getFailedSymbols(),
            run.getQualifiedSymbols(), LocalDateTime.now(MARKET_ZONE)));
        for (ScanLogEvent event : logHistory.getOrDefault(runId, new ConcurrentLinkedDeque<>())) {
            send(emitter, event);
        }
        if (!isActiveStatus(run.getStatus())) emitter.complete();
        return emitter;
    }

    public CandidateScanRunEntity getRun(UUID runId) {
        return runRepository.findByRunId(runId).orElse(null);
    }

    public boolean hasActiveRun() {
        return activeRun.get() != null || runRepository.existsByStatus("RUNNING") || runRepository.existsByStatus("PAUSED");
    }

    public boolean hasPendingHandoff() {
        return runRepository.existsByStatusAndOrchestrationStatus("COMPLETED", "PENDING");
    }

    public List<CandidateScanResultEntity> getResults(UUID runId, int offset, int limit) {
        return getResultsPage(runId, offset, limit, "", "").items();
    }

    public ResultPage getResultsPage(UUID runId, int offset, int limit, String symbol, String signalType) {
        int safeOffset = Math.max(0, offset);
        int safeLimit = Math.max(1, Math.min(limit, 100));
        var page = resultRepository.search(runId, symbol == null ? "" : symbol.trim(),
            signalType == null ? "" : signalType.trim().toUpperCase(),
            PageRequest.of(safeOffset / safeLimit, safeLimit));
        return new ResultPage(page.getContent(), page.getTotalElements(), safeOffset, safeLimit);
    }

    public List<CandidateScanRunEntity> getHistory() {
        return runRepository.findTop20ByOrderByStartedAtDesc();
    }

    @Transactional
    public boolean cancel(UUID runId) {
        CandidateScanRunEntity run = getRun(runId);
        if (run == null || !("RUNNING".equals(run.getStatus()) || "PAUSED".equals(run.getStatus()))) return false;
        AtomicBoolean cancellation = cancellations.get(runId);
        if (cancellation != null) cancellation.set(true);
        AtomicBoolean pause = pauses.get(runId);
        if (pause != null) pause.set(false);
        synchronized (pauseMonitor) { pauseMonitor.notifyAll(); }
        run.setStatus("CANCELLED");
        run.setCompletedAt(LocalDateTime.now(MARKET_ZONE));
        runRepository.save(run);
        if (runId.equals(activeRun.get())) activeRun.compareAndSet(runId, null);
        publish(runId, "RUN_CANCELLED", null, "WARN", "Scan cancellation requested.");
        completeStreams(runId);
        return true;
    }

    @Transactional
    public boolean pause(UUID runId) {
        CandidateScanRunEntity run = getRun(runId);
        if (run == null || !"RUNNING".equals(run.getStatus())) return false;
        pauses.computeIfAbsent(runId, ignored -> new AtomicBoolean()).set(true);
        run.setStatus("PAUSED");
        runRepository.save(run);
        publish(runId, "RUN_PAUSED", null, "WARN", "Scan paused. Active symbols will finish; queued symbols are waiting.");
        return true;
    }

    @Transactional
    public boolean resume(UUID runId) {
        CandidateScanRunEntity run = getRun(runId);
        if (run == null || !"PAUSED".equals(run.getStatus())) return false;
        pauses.computeIfAbsent(runId, ignored -> new AtomicBoolean()).set(false);
        run.setStatus("RUNNING");
        runRepository.save(run);
        synchronized (pauseMonitor) { pauseMonitor.notifyAll(); }
        publish(runId, "RUN_RESUMED", null, "SUCCESS", "Scan resumed.");
        return true;
    }

    private void execute(UUID runId, List<ScanTarget> symbols, boolean scheduled) {
        List<CompletableFuture<Void>> futures = symbols.stream()
            .map(target -> CompletableFuture.runAsync(() -> processSymbol(runId, target, scheduled), executor))
            .toList();
        CompletableFuture.allOf(futures.toArray(new CompletableFuture[0])).whenComplete((ignored, error) -> {
            finalizeRun(runId, error);
        });
    }

    private void processSymbol(UUID runId, ScanTarget target, boolean scheduled) {
        String symbol = target.symbol();
        if (isCancelled(runId)) return;
        if (!"NSE".equals(target.exchange())) {
            saveFailure(runId, target, new IllegalArgumentException("Unsupported exchange: " + target.exchange()));
            return;
        }
        boolean acquired = false;
        try {
            semaphore.acquire();
            acquired = true;
            awaitIfPaused(runId);
            if (isCancelled(runId)) return;
            publish(runId, "SYMBOL_STARTED", symbol, "INFO", "Processing " + symbol + ".");
            boolean qualified = scanSymbol(runId, symbol, scheduled);
            increment(runId, false, qualified);
            publish(runId, "SYMBOL_COMPLETED", symbol, qualified ? "SUCCESS" : "INFO",
                qualified ? symbol + " qualified and was activated." : symbol + " did not pass the gate.");
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        } catch (Exception e) {
            logger.warn("Candidate scan failed for {}:{}: {}", target.exchange(), symbol, e.getMessage());
            saveFailure(runId, target, e);
        } finally {
            if (acquired) semaphore.release();
        }
        if (delayMs > 0 && !isCancelled(runId)) {
            try { Thread.sleep(delayMs); }
            catch (InterruptedException e) { Thread.currentThread().interrupt(); }
        }
    }

    private void finalizeRun(UUID runId, Throwable error) {
        CandidateScanRunEntity run = getRun(runId);
        if (run != null && ("RUNNING".equals(run.getStatus()) || "PAUSED".equals(run.getStatus()))) {
            run.setStatus(isCancelled(runId) || error != null ? "CANCELLED" : "COMPLETED");
            run.setCompletedAt(LocalDateTime.now(MARKET_ZONE));
            if (error != null) run.setErrorMessage(error.getMessage());
            if ("COMPLETED".equals(run.getStatus()) && run.getQualifiedSymbols() == 0
                && !"MANUAL".equals(run.getScanTrigger())
                && "PENDING".equals(run.getOrchestrationStatus())) {
                run.setOrchestrationStatus("NOT_REQUIRED");
            }
            runRepository.save(run);
            publish(runId, run.getStatus().equals("COMPLETED") ? "RUN_COMPLETED" : "RUN_CANCELLED",
                null, error == null ? "SUCCESS" : "WARN",
                error == null ? "Candidate scan completed." : "Candidate scan stopped: " + error.getMessage());
        }
        activeRun.compareAndSet(runId, null);
        cancellations.remove(runId);
        pauses.remove(runId);
        completeStreams(runId);
    }

    private boolean scanSymbol(UUID runId, String symbol, boolean scheduled) {
        int backfillYears = configuredBackfillYears();
        int candles = (int) candleStore.countBySymbol(symbol);
        boolean fetched = false;
        DataIngestionService.BackfillOutcome outcome = null;
        CandidateHistoryEligibilityEntity eligibility = eligibilityRepository == null ? null
            : eligibilityRepository.findById(symbol).orElse(null);
        publish(runId, "STAGE_STARTED", symbol, "INFO", "Data: checking OHLCV history.");
        LocalDate today = LocalDate.now(MARKET_ZONE);
        if (scheduled && candles < MIN_CANDLES && eligibility != null && eligibility.getRetryAfter() != null
            && today.isBefore(eligibility.getRetryAfter())) {
            return saveInsufficientResult(runId, symbol, candles, eligibility, "History retry scheduled");
        }
        if (candles < MIN_CANDLES) {
            publish(runId, "STAGE_STARTED", symbol, "INFO",
                "Data: fetching " + backfillYears + " years of OHLCV history.");
            outcome = ingestionService.backfillStockDataWithOutcome(symbol, backfillYears);
            fetched = true;
            candles = (int) candleStore.countBySymbol(symbol);
        }
        publish(runId, "STAGE_COMPLETED", symbol, candles >= MIN_CANDLES ? "SUCCESS" : "WARN",
            "Data: " + candles + " candles available" + (fetched ? " after fetch." : "."));
        CandidateScanResultEntity result = new CandidateScanResultEntity();
        result.setRunId(runId);
        result.setSymbol(symbol);
        result.setCandleCount(candles);
        List<com.swingtrade.domain.OhlcvCandle> availableCandles = new ArrayList<>(candleStore.findAllBySymbolOrderByDateDesc(symbol));
        availableCandles.sort(java.util.Comparator.comparing(com.swingtrade.domain.OhlcvCandle::date));
        if (!availableCandles.isEmpty()) {
            result.setFirstAvailableDate(availableCandles.getFirst().date());
            result.setLastAvailableDate(availableCandles.getLast().date());
        }
        if (outcome != null) {
            result.setSourceOutcome(outcome.sourceOutcome());
            result.setInvalidRows(outcome.invalidRows());
        } else if (eligibility != null) {
            result.setSourceOutcome(eligibility.getSourceOutcome());
            result.setInvalidRows(eligibility.getInvalidRows());
        } else result.setSourceOutcome("EXISTING_HISTORY");
        if (candles < MIN_CANDLES) {
            if (outcome != null && "TRANSIENT_SOURCE_FAILURE".equals(outcome.sourceOutcome())) {
                result.setDataStatus("ERROR");
                result.setReason("Market-data source failed; history will be retried on the next scan");
                result.setErrorMessage(outcome.errorMessage());
                resultRepository.save(result);
                incrementFailedOnly(runId);
                return false;
            }
            result.setDataStatus("INSUFFICIENT");
            LocalDate retryAfter = today.plusDays(7);
            result.setRetryAfter(retryAfter);
            result.setReason("Insufficient OHLCV history (" + candles + " candles); retry after " + retryAfter);
            persistEligibility(symbol, result, retryAfter, outcome == null ? null : outcome.errorMessage());
            resultRepository.save(result);
            return false;
        }
        result.setDataStatus("READY");
        publish(runId, "STAGE_STARTED", symbol, "INFO", "Signal: generating technical signal.");
        SignalResult signal = signalEngine.generateSignal(symbol);
        List<CandidateStrategyEvaluator.Outcome> strategyOutcomes = candidateStrategyEvaluator == null
            ? List.of() : candidateStrategyEvaluator.evaluateWithPerformance(symbol, availableCandles, "NSE",
                BacktestConfig.defaults(), configuredOosDays(), configuredOosFolds());
        // Keeps compatibility with lightweight callers that provide the pre-performance evaluator
        // contract (notably isolated data-quality tests). Spring production always supplies the
        // non-null performance-aware result.
        if (candidateStrategyEvaluator != null && (strategyOutcomes == null || strategyOutcomes.isEmpty())) {
            strategyOutcomes = candidateStrategyEvaluator.evaluate(symbol, availableCandles);
        }
        if (strategyOutcomes == null) {
            strategyOutcomes = List.of();
        }
        int strategyBuys = (int) strategyOutcomes.stream().filter(CandidateStrategyEvaluator.Outcome::buy).count();
        int evaluatedStrategies = (int) strategyOutcomes.stream().filter(o -> !"SKIPPED".equals(o.signalType())).count();
        int minStrategyBuys = configuredMinStrategyBuys();
        boolean consensusBuy = candidateStrategyEvaluator == null ? signal.type() == com.swingtrade.domain.Signal.SignalType.BUY
            : strategyBuys >= minStrategyBuys;
        result.setStrategyBuyCount(strategyBuys);
        result.setStrategyEvaluationCount(evaluatedStrategies);
        result.setStrategyOutcomes(strategyOutcomes.stream()
            .map(strategyOutcome -> new CandidateScanResultEntity.StrategyOutcome(
                strategyOutcome.variantId(), strategyOutcome.signalType(), strategyOutcome.score(),
                strategyOutcome.detail(),
                strategyOutcome.backtestTotalTrades(), strategyOutcome.backtestWinRate(),
                strategyOutcome.backtestTotalReturn(), strategyOutcome.oosTotalTrades(),
                strategyOutcome.oosWinRate(), strategyOutcome.oosTotalReturn(),
                strategyOutcome.performanceStatus()))
            .toList());
        result.setSignalType(consensusBuy ? "BUY" : "HOLD");
        String votes = strategyOutcomes.isEmpty() ? signal.type().name()
            : strategyBuys + "/" + evaluatedStrategies + " strategies BUY (need " + minStrategyBuys + ")";
        publish(runId, "STAGE_COMPLETED", symbol, consensusBuy ? "SUCCESS" : "INFO",
            "Signal consensus: " + votes + ".");
        // Do not run a historical simulation for a symbol that already failed
        // the strategy gate. This keeps scan work bounded and avoids turning a
        // non-candidate into a misleading backtest/data-quality failure.
        if (!consensusBuy) {
            result.setQualified(false);
            result.setActivated(false);
            result.setReason(consensusReason(strategyBuys, evaluatedStrategies, minStrategyBuys, signal,
                null, null, null, 0, 0.0, 0.0));
            resultRepository.save(result);
            return false;
        }
        // Per-strategy performance was recorded above using each configured SignalStrategy.
        // Keep the established common gates for candidate qualification until the policy for
        // requiring every participating strategy to pass independently is explicitly chosen.
        publish(runId, "STAGE_STARTED", symbol, "INFO",
            "Backtest: running common default strategy (consensus signals evaluated separately).");
        BacktestResult backtest;
        try {
            backtest = backtestEngine.runBacktest(symbol, "NSE", BacktestConfig.defaults());
        } catch (IllegalStateException e) {
            if (isHistoricalMembershipFailure(e)) {
                return saveHistoricalMembershipSkip(runId, result, e.getMessage());
            }
            throw e;
        }
        result.setTotalTrades(backtest.totalTrades());
        result.setWinRate(backtest.winRate());
        result.setTotalReturn(backtest.totalReturn().doubleValue());
        result.setMaxDrawdownPct(backtest.maxDrawdownPct().doubleValue());
        publish(runId, "STAGE_COMPLETED", symbol, "INFO",
            "Backtest: " + backtest.totalTrades() + " trades, "
                + String.format(java.util.Locale.ROOT, "%.1f%% win rate, %.2f%% return.",
                    backtest.winRate(), backtest.totalReturn()));
        int oosDays = configuredOosDays();
        int oosFolds = configuredOosFolds();
        WalkForwardEvaluation walkForward;
        try {
            walkForward = backtestEngine.runWalkForward(symbol, "NSE", BacktestConfig.defaults(), oosDays, oosFolds);
        } catch (IllegalStateException e) {
            if (isHistoricalMembershipFailure(e)) {
                return saveHistoricalMembershipSkip(runId, result, e.getMessage());
            }
            result.setQualified(false);
            result.setReason("BUY rejected: " + e.getMessage());
            resultRepository.save(result);
            return false;
        }
        WalkForwardEvaluation.Fold latestFold = walkForward.folds().get(walkForward.folds().size() - 1);
        LocalDate oosStart = latestFold.startDate();
        LocalDate oosEnd = latestFold.endDate();
        BacktestResult oosBacktest = latestFold.result();
        result.setOosStartDate(oosStart);
        result.setOosEndDate(oosEnd);
        result.setOosTotalTrades(oosBacktest.totalTrades());
        result.setOosWinRate(oosBacktest.winRate());
        result.setOosTotalReturn(oosBacktest.totalReturn().doubleValue());
        result.setOosMaxDrawdownPct(oosBacktest.maxDrawdownPct().doubleValue());
        publish(runId, "STAGE_COMPLETED", symbol, "INFO",
            "Walk-forward OOS: " + oosFolds + " folds, " + walkForward.totalTrades() + " trades, "
                + String.format(java.util.Locale.ROOT, "%.1f%% average win rate, %.2f%% average return.",
                    walkForward.averageWinRate(), walkForward.averageTotalReturn()));
        double minWinRate = configuredMinWinRate();
        double minTotalReturn = configuredMinTotalReturn();
        int minTrades = configuredMinTrades();
        boolean qualified = consensusBuy
            && backtest.totalTrades() >= minTrades
            && backtest.winRate() >= minWinRate
            && backtest.totalReturn().compareTo(BigDecimal.valueOf(minTotalReturn)) > 0
            && walkForward.isComplete(oosFolds)
            && walkForward.folds().stream().allMatch(f -> f.result().totalTrades() >= minTrades
                && f.result().winRate() >= minWinRate
                && f.result().totalReturn().compareTo(BigDecimal.valueOf(minTotalReturn)) > 0);
        result.setQualified(qualified);
        result.setReason(qualified ? "At least " + minStrategyBuys + " strategies BUY and in-sample/out-of-sample backtest gates passed"
            : consensusReason(strategyBuys, evaluatedStrategies, minStrategyBuys, signal, backtest, oosBacktest,
                walkForward, minTrades, minWinRate, minTotalReturn));
        if (qualified && watchlistService != null) {
            watchlistService.addToWatchlist(symbol, symbol, "NSE");
            result.setActivated(true);
        } else {
            result.setActivated(false);
        }
        resultRepository.save(result);
        return qualified;
    }

    private boolean saveInsufficientResult(UUID runId, String symbol, int candles,
                                           CandidateHistoryEligibilityEntity eligibility, String reason) {
        CandidateScanResultEntity result = new CandidateScanResultEntity();
        result.setRunId(runId);
        result.setSymbol(symbol);
        result.setCandleCount(candles);
        result.setDataStatus("INSUFFICIENT");
        result.setSourceOutcome(eligibility.getSourceOutcome());
        result.setInvalidRows(eligibility.getInvalidRows());
        result.setFirstAvailableDate(eligibility.getFirstAvailableDate());
        result.setLastAvailableDate(eligibility.getLastAvailableDate());
        result.setRetryAfter(eligibility.getRetryAfter());
        result.setReason(reason + "; retry after " + eligibility.getRetryAfter());
        resultRepository.save(result);
        return false;
    }

    private void persistEligibility(String symbol, CandidateScanResultEntity result, LocalDate retryAfter, String error) {
        if (eligibilityRepository == null) return;
        CandidateHistoryEligibilityEntity record = eligibilityRepository.findById(symbol).orElseGet(CandidateHistoryEligibilityEntity::new);
        record.setSymbol(symbol);
        record.setCandleCount(result.getCandleCount());
        record.setFirstAvailableDate(result.getFirstAvailableDate());
        record.setLastAvailableDate(result.getLastAvailableDate());
        record.setSourceOutcome(result.getSourceOutcome());
        record.setInvalidRows(result.getInvalidRows());
        record.setRetryAfter(retryAfter);
        record.setErrorMessage(error);
        record.setUpdatedAt(LocalDateTime.now(MARKET_ZONE));
        eligibilityRepository.save(record);
    }

    private String qualificationReason(SignalResult signal, BacktestResult backtest, BacktestResult oosBacktest,
                                       WalkForwardEvaluation walkForward, int minTrades,
                                       double minWinRate, double minTotalReturn) {
        if (signal.type() != com.swingtrade.domain.Signal.SignalType.BUY) return "Current signal is " + signal.type();
        if (backtest.totalTrades() < minTrades) return "BUY rejected: fewer than " + minTrades + " trades";
        if (backtest.winRate() < minWinRate) return "BUY rejected: win rate below " + minWinRate + "%";
        if (backtest.totalReturn().compareTo(BigDecimal.valueOf(minTotalReturn)) <= 0) return "BUY rejected: backtest return not above " + minTotalReturn + "%";
        return walkForward.folds().stream()
            .filter(f -> f.result().totalTrades() < minTrades)
            .findFirst()
            .map(f -> "BUY rejected: OOS fold " + f.startDate() + " trades below " + minTrades)
            .orElseGet(() -> walkForward.folds().stream()
                .filter(f -> f.result().winRate() < minWinRate)
                .findFirst()
                .map(f -> "BUY rejected: OOS fold " + f.startDate() + " win rate below " + minWinRate + "%")
                .orElse("BUY rejected: an OOS fold return is not above " + minTotalReturn + "%"));
    }

    private boolean isHistoricalMembershipFailure(IllegalStateException error) {
        return error.getMessage() != null && error.getMessage().contains("No included historical universe membership");
    }

    private boolean saveHistoricalMembershipSkip(UUID runId, CandidateScanResultEntity result, String detail) {
        result.setDataStatus("INSUFFICIENT");
        result.setQualified(false);
        result.setActivated(false);
        result.setReason("Skipped: historical universe membership is unavailable for the backtest period");
        result.setErrorMessage(detail);
        resultRepository.save(result);
        publish(runId, "STAGE_COMPLETED", result.getSymbol(), "WARN", result.getReason());
        return false;
    }

    private String consensusReason(int buys, int evaluated, int required, SignalResult legacySignal,
                                   BacktestResult backtest, BacktestResult oosBacktest,
                                   WalkForwardEvaluation walkForward, int minTrades,
                                   double minWinRate, double minTotalReturn) {
        if (buys < required && candidateStrategyEvaluator != null) {
            return "BUY rejected: " + buys + "/" + evaluated + " strategies BUY; require " + required;
        }
        if (backtest == null || oosBacktest == null || walkForward == null) {
            return "BUY rejected: strategy consensus not reached";
        }
        return qualificationReason(legacySignal, backtest, oosBacktest, walkForward, minTrades, minWinRate,
            minTotalReturn);
    }

    private void saveFailure(UUID runId, ScanTarget target, Exception error) {
        CandidateScanResultEntity result = new CandidateScanResultEntity();
        result.setRunId(runId);
        result.setSymbol(target.symbol());
        result.setDataStatus("ERROR");
        result.setReason("Scan failed");
        result.setErrorMessage(error.getMessage());
        resultRepository.save(result);
        increment(runId, true, false);
        publish(runId, "SYMBOL_FAILED", target.symbol(), "ERROR", "Scan failed: " + error.getMessage());
    }

    private synchronized void increment(UUID runId, boolean failed, boolean qualified) {
        CandidateScanRunEntity run = getRun(runId);
        // Work that was already active when pause was requested is allowed to
        // finish, and its result must still contribute to the run counters.
        if (run == null || !isActiveStatus(run.getStatus())) return;
        run.setCompletedSymbols(run.getCompletedSymbols() + 1);
        if (failed) run.setFailedSymbols(run.getFailedSymbols() + 1);
        if (qualified) run.setQualifiedSymbols(run.getQualifiedSymbols() + 1);
        runRepository.save(run);
    }

    private synchronized void incrementFailedOnly(UUID runId) {
        CandidateScanRunEntity run = getRun(runId);
        if (run == null || !isActiveStatus(run.getStatus())) return;
        run.setFailedSymbols(run.getFailedSymbols() + 1);
        runRepository.save(run);
    }

    private boolean isCancelled(UUID runId) {
        AtomicBoolean value = cancellations.get(runId);
        return value != null && value.get();
    }

    private void awaitIfPaused(UUID runId) throws InterruptedException {
        synchronized (pauseMonitor) {
            while (isPaused(runId) && !isCancelled(runId)) pauseMonitor.wait();
        }
    }

    private boolean isPaused(UUID runId) {
        AtomicBoolean value = pauses.get(runId);
        return value != null && value.get();
    }

    private static boolean isActiveStatus(String status) {
        return "RUNNING".equals(status) || "PAUSED".equals(status);
    }

    private void publish(UUID runId, String eventType, String symbol, String level, String message) {
        CandidateScanRunEntity run = getRun(runId);
        if (run == null) return;
        ScanLogEvent event = new ScanLogEvent(eventType, runId, symbol, level, message,
            run.getCompletedSymbols(), run.getTotalSymbols(), run.getFailedSymbols(),
            run.getQualifiedSymbols(), LocalDateTime.now(MARKET_ZONE));
        Deque<ScanLogEvent> history = logHistory.computeIfAbsent(runId, ignored -> new ConcurrentLinkedDeque<>());
        history.addLast(event);
        while (history.size() > MAX_LOG_HISTORY) history.pollFirst();
        for (SseEmitter emitter : emitters.getOrDefault(runId, List.of())) send(emitter, event);
    }

    private void send(SseEmitter emitter, ScanLogEvent event) {
        try {
            emitter.send(SseEmitter.event().name("log").data(event));
        } catch (IOException | IllegalStateException e) {
            // A disconnected browser must not turn a successful symbol scan into
            // a processing failure. The next stream connection can use the run
            // snapshot and any retained events to catch up.
            logger.debug("Candidate scan SSE client disconnected: {}", e.getMessage());
        }
    }

    private void completeStreams(UUID runId) {
        List<SseEmitter> runEmitters = emitters.remove(runId);
        if (runEmitters != null) runEmitters.forEach(SseEmitter::complete);
    }

    @PreDestroy
    void shutdown() { executor.shutdownNow(); }

    public record ScanLogEvent(
        String eventType,
        UUID runId,
        String symbol,
        String level,
        String message,
        int completedSymbols,
        int totalSymbols,
        int failedSymbols,
        int qualifiedSymbols,
        LocalDateTime timestamp
    ) {}

    public record ResultPage(
        List<CandidateScanResultEntity> items,
        long total,
        int offset,
        int limit
    ) {}
}
