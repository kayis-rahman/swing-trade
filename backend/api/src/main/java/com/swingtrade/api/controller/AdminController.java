package com.swingtrade.api.controller;

import com.swingtrade.api.dto.ApiResponse;
import com.swingtrade.api.dto.DataQualityValidationRequest;
import com.swingtrade.api.dto.KillSwitchRequest;
import com.swingtrade.broker.risk.KillSwitchService;
import com.swingtrade.data.repository.FyersSymbolRepository;
import com.swingtrade.data.service.DataIngestionService;
import com.swingtrade.data.service.FyersSymbolMasterService;
import jakarta.validation.Valid;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.reactive.function.client.WebClient;

// Only used by the commented-out Upstox pull-historical endpoint below.
// import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Admin controller for system management operations.
 * Provides endpoints for kill switch and other administrative functions.
 */
@RestController
@RequestMapping("/api/admin")
public class AdminController {

    private static final Logger logger = LoggerFactory.getLogger(AdminController.class);

    private final KillSwitchService killSwitchService;
    private final DataIngestionService dataIngestionService;
    private final FyersSymbolMasterService symbolMasterService;

    public AdminController(KillSwitchService killSwitchService, DataIngestionService dataIngestionService,
                           WebClient.Builder webClientBuilder, FyersSymbolRepository symbolRepository) {
        this.killSwitchService = killSwitchService;
        this.dataIngestionService = dataIngestionService;
        this.symbolMasterService = new FyersSymbolMasterService(webClientBuilder, symbolRepository);
        logger.info("AdminController initialized with kill switch and data ingestion services");
    }

    /**
     * Refresh the Fyers symbol master table from the public NSE_CM.csv feed.
     *
     * @return API response with the number of symbols loaded
     */
    @PostMapping("/symbols/refresh")
    public ResponseEntity<ApiResponse<Map<String, Object>>> refreshSymbolMaster() {
        try {
            int count = symbolMasterService.refresh();
            return ResponseEntity.ok(ApiResponse.ok(Map.of("count", count)));
        } catch (Exception e) {
            logger.error("Failed to refresh Fyers symbol master: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to refresh symbol master: " + e.getMessage()));
        }
    }

    /**
     * Get current Fyers symbol master status (row count).
     *
     * @return API response with symbol count
     */
    @GetMapping("/symbols/status")
    public ResponseEntity<ApiResponse<Map<String, Object>>> getSymbolMasterStatus() {
        try {
            Map<String, Object> data = new LinkedHashMap<>();
            data.put("count", symbolMasterService.count());
            return ResponseEntity.ok(ApiResponse.ok(data));
        } catch (Exception e) {
            logger.error("Failed to get symbol master status: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                .body(ApiResponse.error("Failed to get symbol master status: " + e.getMessage()));
        }
    }

    /**
     * Enable or disable the kill switch.
     * When active, all new trades are blocked and existing positions are closed.
     *
     * @param request kill switch request with enabled flag and optional reason
     * @return API response with status
     */
    @PostMapping("/kill-switch")
    public ResponseEntity<ApiResponse<Map<String, Object>>> killSwitch(@Valid @RequestBody KillSwitchRequest request) {
        logger.info("Kill switch {} requested. Reason: {}",
                request.enabled() ? "enable" : "disable", request.getReason());

        try {
            if (request.enabled()) {
                killSwitchService.enableKillSwitch(request.getReason());
                logger.warn("Kill switch ENABLED by admin request. Reason: {}", request.getReason());
            } else {
                killSwitchService.disableKillSwitch();
                logger.info("Kill switch DISABLED by admin request");
            }

            Map<String, Object> data = new LinkedHashMap<>();
            data.put("active", killSwitchService.isActive());
            data.put("enabledAt", killSwitchService.getEnabledAt());
            data.put("reason", request.getReason());

            return ResponseEntity.ok(ApiResponse.ok(data, null));

        } catch (Exception e) {
            logger.error("Error toggling kill switch: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                    .body(ApiResponse.error("Failed to toggle kill switch: " + e.getMessage()));
        }
    }

    /**
     * Get current kill switch status.
     *
     * @return kill switch status response
     */
    @GetMapping("/kill-switch/status")
    public ResponseEntity<ApiResponse<KillSwitchService.KillSwitchState>> getKillSwitchStatus() {
        logger.debug("Fetching kill switch status");

        try {
            KillSwitchService.KillSwitchState status = killSwitchService.getKillSwitchState();
            return ResponseEntity.ok(ApiResponse.ok(status));

        } catch (Exception e) {
            logger.error("Error fetching kill switch status: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                    .body(ApiResponse.error("Failed to get kill switch status: " + e.getMessage()));
        }
    }

    /**
     * Toggle kill switch (alternative endpoint for quick toggle).
     *
     * @param enable true to enable, false to disable
     * @return API response with status
     */
    @PostMapping("/kill-switch/toggle")
    public ResponseEntity<ApiResponse<Map<String, Object>>> toggleKillSwitch(@RequestParam boolean enable) {
        logger.info("Toggle kill switch to: {}", enable);

        try {
            killSwitchService.toggle(enable);
            Map<String, Object> data = new LinkedHashMap<>();
            data.put("active", killSwitchService.isActive());
            data.put("enabledAt", killSwitchService.getEnabledAt());
            data.put("reason", killSwitchService.getReason());

            return ResponseEntity.ok(ApiResponse.ok(data, null));

        } catch (Exception e) {
            logger.error("Error toggling kill switch: {}", e.getMessage(), e);
            return ResponseEntity.internalServerError()
                    .body(ApiResponse.error("Failed to toggle kill switch: " + e.getMessage()));
        }
    }

    /**
     * Health check endpoint for admin services.
     *
     * @return API response with health status
     */
    @GetMapping("/health")
    public ResponseEntity<ApiResponse<Map<String, Object>>> adminHealth() {
        Map<String, Object> data = new ConcurrentHashMap<>();
        data.put("service", "admin");
        data.put("status", "healthy");
        data.put("killSwitchActive", killSwitchService.isActive());
        data.put("killSwitchEnabled", killSwitchService.isKillSwitchEnabled());
        data.put("timestamp", LocalDateTime.now());

        return ResponseEntity.ok(ApiResponse.ok(data));
    }

    /**
     * Validate persisted OHLCV data for one symbol and an inclusive date range.
     *
     * @param request symbol and inclusive validation window
     * @return gap and price-anomaly report
     */
    @PostMapping("/data/validate")
    public ResponseEntity<ApiResponse<DataIngestionService.DataQualityReport>> validateData(
            @RequestBody DataQualityValidationRequest request) {
        if (request == null || request.symbol() == null || request.symbol().isBlank()
                || request.fromDate() == null || request.toDate() == null
                || request.fromDate().isAfter(request.toDate())) {
            return ResponseEntity.badRequest().body(ApiResponse.error(
                    "symbol, fromDate, and toDate are required and the date range must be ordered"));
        }

        try {
            DataIngestionService.DataQualityReport report = dataIngestionService.validateDataQuality(
                    request.symbol(), request.fromDate(), request.toDate());
            return ResponseEntity.ok(ApiResponse.ok(report));
        } catch (Exception e) {
            logger.error("Error validating data quality for {}: {}", request.symbol(), e.getMessage(), e);
            return ResponseEntity.internalServerError().body(ApiResponse.error(
                    "Failed to validate data quality: " + e.getMessage()));
        }
    }

    // Upstox is unconfigured — this endpoint (backed by DataIngestionService.pullDataFromUpstox)
    // is commented out along with that method (kept for future re-enablement).
    // /**
    //  * Pull historical data from Upstox for a stock.
    //  *
    //  * @param symbol stock symbol
    //  * @param startDate start date (YYYY-MM-DD)
    //  * @param endDate end date (YYYY-MM-DD)
    //  * @return API response with ingestion result
    //  */
    // @PostMapping("/data/pull-historical")
    // public ResponseEntity<ApiResponse<Map<String, Object>>> pullHistoricalData(
    //         @RequestParam String symbol,
    //         @RequestParam String startDate,
    //         @RequestParam String endDate) {

    //     logger.info("Pulling historical data for {}: {} to {}", symbol, startDate, endDate);

    //     try {
    //         LocalDate from = LocalDate.parse(startDate);
    //         LocalDate to = LocalDate.parse(endDate);

    //         int ingested = dataIngestionService.pullDataFromUpstox(symbol, from, to);

    //         Map<String, Object> data = new ConcurrentHashMap<>();
    //         data.put("symbol", symbol);
    //         data.put("startDate", startDate);
    //         data.put("endDate", endDate);
    //         data.put("ingested", ingested);

    //         return ResponseEntity.ok(ApiResponse.ok(data));

    //     } catch (Exception e) {
    //         logger.error("Error pulling historical data for {}: {}", symbol, e.getMessage(), e);
    //         return ResponseEntity.internalServerError()
    //                 .body(ApiResponse.error("Failed to pull historical data: " + e.getMessage()));
    //     }
    // }

    /**
     * Get data ingestion status for a stock.
     *
     * @param symbol stock symbol
     * @param days number of days to check
     * @return API response with data status
     */
    @GetMapping("/data/status")
    public ResponseEntity<ApiResponse<Map<String, Object>>> getDataStatus(
            @RequestParam String symbol,
            @RequestParam int days) {

        logger.debug("Fetching data status for {}: {} days", symbol, days);

        try {
            var candles = dataIngestionService.getRecentCandles(symbol, days);
            var latest = dataIngestionService.getLatestCandle(symbol);

            Map<String, Object> data = new ConcurrentHashMap<>();
            data.put("symbol", symbol);
            data.put("daysChecked", days);
            data.put("candleCount", candles.size());
            data.put("latestCandleDate", latest.map(c -> c.getDate().toString()).orElse(null));

            return ResponseEntity.ok(ApiResponse.ok(data));

        } catch (Exception e) {
            logger.error("Error fetching data status for {}: {}", symbol, e.getMessage(), e);
            return ResponseEntity.internalServerError()
                    .body(ApiResponse.error("Failed to get data status: " + e.getMessage()));
        }
    }
}
