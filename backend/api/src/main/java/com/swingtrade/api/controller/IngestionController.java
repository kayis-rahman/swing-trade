package com.swingtrade.api.controller;

import com.swingtrade.api.dto.ApiResponse;
import com.swingtrade.data.entity.NseHolidayEntity;
import com.swingtrade.data.service.DataIngestionService;
import com.swingtrade.data.service.NseHolidayService;
import com.swingtrade.data.service.WatchlistService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.beans.factory.annotation.Value;

import java.time.LocalDate;
import java.time.ZoneId;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * REST endpoints for manual data ingestion triggers.
 */
@RestController
@RequestMapping("/api")
public class IngestionController {

    private enum OperationStatus {
        STARTED, COMPLETED, ADDED, REMOVED, UNKNOWN;

        @Override
        public String toString() {
            return name().toLowerCase(java.util.Locale.ROOT);
        }

        static OperationStatus from(String value) {
            try {
                return value == null ? UNKNOWN : valueOf(value.toUpperCase(java.util.Locale.ROOT));
            } catch (IllegalArgumentException ignored) {
                return UNKNOWN;
            }
        }
    }

    private static final Logger logger = LoggerFactory.getLogger(IngestionController.class);

    private final DataIngestionService dataIngestionService;
    private final WatchlistService watchlistService;
    private final NseHolidayService holidayService;
    @Value("${ingestion.reconcile.apply.enabled:false}")
    private boolean reconcileApplyEnabled;
    @Value("${reconcile.apply.token:}")
    private String reconcileApplyToken;

    public IngestionController(DataIngestionService dataIngestionService, WatchlistService watchlistService, NseHolidayService holidayService) {
        this.dataIngestionService = dataIngestionService;
        this.watchlistService = watchlistService;
        this.holidayService = holidayService;
    }

    /**
     * Backfill historical data for a single stock.
     */
    @PostMapping("/ingestion/backfill")
    public ResponseEntity<ApiResponse<Map<String, Object>>> backfillStock(
            @RequestParam String symbol,
            @RequestParam(defaultValue = "5") int years) {

        logger.info("Manual backfill requested for {} ({} years)", symbol, years);

        try {
            dataIngestionService.backfillStockData(symbol, years);

            long count = watchlistService.getIngestionStatus().stream()
                .filter(s -> symbol.equals(s.get("symbol")))
                .mapToLong(s -> ((Number) s.get("candleCount")).longValue())
                .findFirst()
                .orElse(0);

            Map<String, Object> data = Map.of(
                "symbol", symbol,
                "years", years,
                "candleCount", count,
                "status", OperationStatus.COMPLETED
            );
            return ResponseEntity.ok(ApiResponse.ok(data));

        } catch (Exception e) {
            logger.error("Backfill failed for {}: {}", symbol, e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to backfill: " + e.getMessage()));
        }
    }

    /**
     * Backfill all watchlist stocks.
     */
    @PostMapping("/ingestion/backfill-all")
    public ResponseEntity<ApiResponse<Map<String, Object>>> backfillAll(@RequestParam(defaultValue = "5") int years) {
        logger.info("Manual backfill-all requested ({} years)", years);

        try {
            String pullId = watchlistService.startPullAll(years);
            return ResponseEntity.ok(ApiResponse.ok(Map.of(
                "pullId", pullId,
                "status", OperationStatus.STARTED
            )));

        } catch (Exception e) {
            logger.error("Backfill-all failed: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to start backfill: " + e.getMessage()));
        }
    }

    @PostMapping("/ingestion/range")
    public ResponseEntity<ApiResponse<Map<String, Object>>> ingestRange(
            @RequestParam LocalDate from, @RequestParam LocalDate to) {
        logger.info("Manual range pull requested from {} to {}", from, to);
        try {
            String pullId = watchlistService.startPullAll(from, to);
            return ResponseEntity.ok(ApiResponse.ok(Map.of("pullId", pullId, "status", OperationStatus.STARTED)));
        } catch (Exception e) {
            logger.error("Range pull failed: {}", e.getMessage(), e);
            return ResponseEntity.badRequest().body(ApiResponse.error("Failed to start range pull: " + e.getMessage()));
        }
    }

    /** Performs an incremental pull for one symbol, skipping its already stored tail. */
    @PostMapping("/data/pull/incremental")
    public ResponseEntity<ApiResponse<Map<String, Object>>> incrementalPull(
            @RequestParam String symbol, @RequestParam LocalDate from, @RequestParam LocalDate to) {
        String normalized = symbol.trim().toUpperCase(java.util.Locale.ROOT);
        try {
            DataIngestionService.BackfillOutcome outcome =
                dataIngestionService.processIncrementalStockData(normalized, from, to);
            Map<String, Object> payload = new HashMap<>();
            payload.put("symbol", normalized);
            payload.put("from", from);
            payload.put("to", to);
            payload.put("sourceOutcome", outcome.sourceOutcome());
            payload.put("fetchedRows", outcome.fetchedRows());
            payload.put("savedRows", outcome.savedRows());
            payload.put("invalidRows", outcome.invalidRows());
            return ResponseEntity.ok(ApiResponse.ok(payload));
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().body(ApiResponse.error(e.getMessage()));
        } catch (Exception e) {
            logger.error("Incremental pull failed for {}: {}", normalized, e.getMessage(), e);
            return ResponseEntity.internalServerError().body(ApiResponse.error("Incremental pull failed: " + e.getMessage()));
        }
    }

    @PostMapping("/ingestion/date")
    public ResponseEntity<ApiResponse<Map<String, Object>>> ingestDate(
            @RequestParam String symbol, @RequestParam LocalDate date) {
        logger.info("Manual single-date pull requested for {} on {}", symbol, date);
        try {
            String normalizedSymbol = symbol.trim().toUpperCase();
            dataIngestionService.processSingleStock(normalizedSymbol, date);
            return ResponseEntity.ok(ApiResponse.ok(Map.of(
                "symbol", normalizedSymbol, "date", date, "status", OperationStatus.COMPLETED)));
        } catch (Exception e) {
            logger.error("Single-date pull failed for {} on {}: {}", symbol, date, e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to pull selected date: " + e.getMessage()));
        }
    }

    /**
     * Ingest only the latest candle for a stock.
     */
    @PostMapping("/ingestion/latest")
    public ResponseEntity<ApiResponse<Map<String, Object>>> ingestLatest(
            @RequestParam String symbol,
            @RequestParam(defaultValue = "NSE") String exchange) {

        logger.info("Latest candle ingestion requested for {} ({})", symbol, exchange);

        try {
            dataIngestionService.processSingleStock(symbol, LocalDate.now());

            Map<String, Object> data = Map.of(
                "symbol", symbol,
                "exchange", exchange,
                "status", OperationStatus.COMPLETED
            );
            return ResponseEntity.ok(ApiResponse.ok(data));

        } catch (Exception e) {
            logger.error("Latest ingestion failed for {}: {}", symbol, e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to ingest latest: " + e.getMessage()));
        }
    }

    /**
     * Get candle count per symbol for all watchlist stocks.
     */
    @GetMapping("/ingestion/status")
    public ResponseEntity<ApiResponse<List<Map<String, Object>>>> getIngestionStatus() {
        try {
            List<Map<String, Object>> status = watchlistService.getIngestionStatus();
            return ResponseEntity.ok(ApiResponse.ok(status));
        } catch (Exception e) {
            logger.error("Failed to get ingestion status: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to get status: " + e.getMessage()));
        }
    }

    @GetMapping("/ingestion/reconcile")
    public ResponseEntity<ApiResponse<Map<String, Object>>> reconcile(
            @RequestParam LocalDate from, @RequestParam LocalDate to,
            @RequestParam(required = false) String symbol) {
        if (symbol != null) {
            return ResponseEntity.ok(ApiResponse.ok(dataIngestionService.reconcile(symbol.toUpperCase(), from, to, false)));
        }
        List<Map<String, Object>> reports = watchlistService.getActiveWatchlist().stream()
            .map(w -> dataIngestionService.reconcile(w.getSymbol(), from, to, false)).toList();
        OperationStatus status = reports.stream()
            .map(r -> OperationStatus.from(String.valueOf(r.get("status"))))
            .filter(s -> s != OperationStatus.COMPLETED)
            .findFirst().orElse(OperationStatus.COMPLETED);
        return ResponseEntity.ok(ApiResponse.ok(Map.of("reports", reports, "status", status)));
    }

    @PostMapping("/ingestion/reconcile")
    public ResponseEntity<ApiResponse<Map<String, Object>>> applyReconcile(
            @RequestParam LocalDate from, @RequestParam LocalDate to,
            @RequestParam String symbol, @RequestParam boolean apply,
            @RequestHeader(value = "X-Reconcile-Token", required = false) String token) {
        if (!apply || !reconcileApplyEnabled || !watchlistService.getActiveWatchlist().stream()
                .anyMatch(w -> w.getSymbol().equalsIgnoreCase(symbol))) {
            return ResponseEntity.badRequest().body(ApiResponse.error("apply requires one active symbol and the local/stage feature gate"));
        }
        if (!reconcileApplyToken.isBlank() && !reconcileApplyToken.equals(token)) {
            return ResponseEntity.status(403).body(ApiResponse.error("invalid reconciliation token"));
        }
        return ResponseEntity.ok(ApiResponse.ok(dataIngestionService.reconcile(symbol.toUpperCase(), from, to, true)));
    }

    /**
     * Check if today is an NSE holiday.
     */
    @GetMapping("/holidays/today")
    public ResponseEntity<ApiResponse<Map<String, Object>>> checkToday() {
        boolean closed = holidayService.isTodayMarketClosed();
        LocalDate today = LocalDate.now(ZoneId.of("Asia/Kolkata"));

        Map<String, Object> reason;
        if (closed) {
            Optional<NseHolidayEntity> opt = holidayService.getHoliday(today);
            if (opt.isPresent()) {
                NseHolidayEntity h = opt.get();
                reason = Map.of("occasion", h.getOccasion(), "type", h.getHolidayType());
            } else {
                reason = Map.of("reason", "weekend");
            }
        } else {
            reason = Map.of();
        }

        Map<String, Object> data = Map.of(
            "date", today.toString(),
            "marketClosed", closed,
            "reason", reason
        );
        return ResponseEntity.ok(ApiResponse.ok(data));
    }

    /**
     * List all NSE holidays.
     */
    @GetMapping("/holidays")
    public ResponseEntity<ApiResponse<Map<String, Object>>> listHolidays(
            @RequestParam(required = false) String from,
            @RequestParam(required = false) String to) {

        try {
            List<NseHolidayEntity> holidays;
            if (from != null && to != null) {
                holidays = holidayService.getHolidays(LocalDate.parse(from), LocalDate.parse(to));
            } else {
                holidays = holidayService.getUpcomingHolidays(LocalDate.now(ZoneId.of("Asia/Kolkata")));
            }

            List<Map<String, Object>> result = holidays.stream().map(h -> Map.<String, Object>of(
                "date", h.getHolidayDate().toString(),
                "occasion", h.getOccasion(),
                "type", h.getHolidayType()
            )).toList();

            Map<String, Object> payload = new HashMap<>();
            payload.put("count", result.size());
            payload.put("holidays", result);
            return ResponseEntity.ok(ApiResponse.ok(payload));
        } catch (Exception e) {
            logger.error("Failed to list holidays: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to list holidays: " + e.getMessage()));
        }
    }

    /**
     * Add a new NSE holiday.
     */
    @PostMapping("/holidays")
    public ResponseEntity<ApiResponse<Map<String, Object>>> addHoliday(
            @RequestParam LocalDate date,
            @RequestParam String occasion,
            @RequestParam(defaultValue = "FULL") String type) {

        try {
            NseHolidayEntity entity = holidayService.addHoliday(date, occasion, type);
            Map<String, Object> data = Map.of(
                "date", entity.getHolidayDate().toString(),
                "occasion", entity.getOccasion(),
                "type", entity.getHolidayType(),
                "status", OperationStatus.ADDED
            );
            return ResponseEntity.ok(ApiResponse.ok(data));
        } catch (Exception e) {
            logger.error("Failed to add holiday: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to add holiday: " + e.getMessage()));
        }
    }

    /**
     * Remove an NSE holiday.
     */
    @DeleteMapping("/holidays/{date}")
    public ResponseEntity<ApiResponse<Map<String, Object>>> removeHoliday(@PathVariable LocalDate date) {
        try {
            boolean removed = holidayService.removeHoliday(date);
            if (removed) {
                return ResponseEntity.ok(ApiResponse.ok(Map.of(
                    "date", date.toString(),
                    "status", OperationStatus.REMOVED
                )));
            }
            return ResponseEntity.badRequest()
                .body(ApiResponse.error("No holiday found on " + date));
        } catch (Exception e) {
            logger.error("Failed to remove holiday: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to remove holiday: " + e.getMessage()));
        }
    }
}
