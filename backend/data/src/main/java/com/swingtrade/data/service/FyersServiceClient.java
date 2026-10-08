package com.swingtrade.data.service;

import com.swingtrade.core.metrics.DataIngestionMetrics;
import tools.jackson.databind.JsonNode;
import tools.jackson.databind.ObjectMapper;
import com.swingtrade.data.entity.FyersSymbolEntity;
import com.swingtrade.domain.PriceBand;
import com.tts.in.model.FyersClass;
import com.tts.in.model.OrderModel;
import com.tts.in.model.PositionModel;
import com.tts.in.model.TradeBookModel;
import com.tts.in.model.HoldingModel;
import com.tts.in.model.MarketDepthModel;
import com.tts.in.model.MarketStatusModel;
import com.tts.in.model.ProfileModel;
import com.tts.in.model.FundModel;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.reactive.function.client.WebClient;
import org.springframework.web.reactive.function.client.WebClientResponseException;
import org.springframework.web.util.UriComponentsBuilder;
import reactor.core.publisher.Mono;
import reactor.util.retry.Retry;
import reactor.netty.http.client.HttpClient;

import java.math.BigDecimal;
import java.time.Duration;
import java.net.URI;
import java.time.LocalDate;
import java.time.OffsetDateTime;
import java.time.ZoneId;
import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;

import io.github.resilience4j.bulkhead.Bulkhead;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.reactor.bulkhead.operator.BulkheadOperator;
import io.github.resilience4j.reactor.circuitbreaker.operator.CircuitBreakerOperator;

/**
 * Fyers v3 market data and order management client.
 *
 * Market data goes through WebClient against the documented REST endpoints.
 * Order management delegates to focused sub-services.
 *
 * Auth header: {appId}:{accessToken} (colon-separated, no "token" prefix).
 */
public class FyersServiceClient implements MarketDataClient {

    private static final Logger logger = LoggerFactory.getLogger(FyersServiceClient.class);
    private static final String BASE_URL = "https://api-t1.fyers.in";
    private static final int CHUNK_DAYS = 365;

    private final WebClient webClient;
    private final FyersAuthService authService;
    private final FyersSymbolMasterService symbolMaster;
    private final ObjectMapper objectMapper = new ObjectMapper();
    private volatile boolean missingTokenLogged = false;

    // Resilience4j fields
    private CircuitBreaker fyersCircuitBreaker;
    private Bulkhead fyersBulkhead;
    private DataIngestionMetrics ingestionMetrics;

    // Delegated order management services
    private final FyersOrderService orderService;
    private final FyersPositionService positionService;
    private final FyersTradeBookService tradeBookService;
    private final FyersHoldingService holdingService;
    private final FyersMarketDepthService marketDepthService;
    private final FyersMarketStatusService marketStatusService;
    private final FyersProfileService profileService;
    private final FyersGTTService gttService;

    public FyersServiceClient(WebClient.Builder webClientBuilder, FyersAuthService authService,
                              FyersSymbolMasterService symbolMaster) {
        this(webClientBuilder, authService, symbolMaster, BASE_URL);
    }

    static final Duration FYERS_READ_TIMEOUT = Duration.ofSeconds(15);

    // Package-private constructor for testing with a custom base URL (MockWebServer)
    FyersServiceClient(WebClient.Builder webClientBuilder, FyersAuthService authService,
                       FyersSymbolMasterService symbolMaster, String baseUrl) {
        this.webClient = webClientBuilder
            .baseUrl(baseUrl)
            .defaultHeader("Accept", "application/json")
            .clientConnector(new org.springframework.http.client.reactive.ReactorClientHttpConnector(
                HttpClient.create().responseTimeout(FYERS_READ_TIMEOUT)))
            .build();
        this.authService = authService;
        this.symbolMaster = symbolMaster;
        this.orderService = new FyersOrderService(authService);
        this.positionService = new FyersPositionService(authService);
        this.tradeBookService = new FyersTradeBookService(authService);
        this.holdingService = new FyersHoldingService(authService);
        this.marketDepthService = new FyersMarketDepthService(authService);
        this.marketStatusService = new FyersMarketStatusService(authService);
        this.profileService = new FyersProfileService(authService);
        this.gttService = new FyersGTTService(authService);
    }

    // Setter for Resilience4j injection (called by FyersResilienceConfig)
    public void setResilience4j(CircuitBreaker circuitBreaker, Bulkhead bulkhead) {
        this.fyersCircuitBreaker = circuitBreaker;
        this.fyersBulkhead = bulkhead;
    }

    public void setIngestionMetrics(DataIngestionMetrics ingestionMetrics) {
        this.ingestionMetrics = ingestionMetrics;
    }

    // -----------------------------------------------------------------------
    // Market Data — MarketDataClient interface
    // -----------------------------------------------------------------------

    @Override
    public CandleData fetchCandle(String symbol, LocalDate date) {
        List<CandleData> candles = fetchCandlesList(symbol, date, date);
        return candles.isEmpty() ? null : candles.get(0);
    }

    @Override
    public Iterable<CandleData> fetchCandles(String symbol, LocalDate startDate, LocalDate endDate) {
        return fetchCandles(symbol, "NSE", startDate, endDate);
    }

    @Override
    public Iterable<CandleData> fetchCandles(String symbol, String exchange,
                                              LocalDate startDate, LocalDate endDate) {
        return fetchCandlesList(symbol, exchange, startDate, endDate);
    }

    @Override
    public Iterable<CandleData> fetchCandles(String symbol, LocalDate startDate, LocalDate endDate,
                                             com.swingtrade.data.service.Interval interval) {
        return fetchCandlesList(symbol, startDate, endDate, interval);
    }

    // Exposed for testing
    List<CandleData> fetchCandlesList(String symbol, LocalDate startDate, LocalDate endDate) {
        return fetchCandlesList(symbol, startDate, endDate, com.swingtrade.data.service.Interval.DAILY);
    }

    List<CandleData> fetchCandlesList(String symbol, LocalDate startDate, LocalDate endDate,
                                      com.swingtrade.data.service.Interval interval) {
        return fetchCandlesList(symbol, "NSE", startDate, endDate, interval);
    }

    List<CandleData> fetchCandlesList(String symbol, String exchange, LocalDate startDate, LocalDate endDate) {
        return fetchCandlesList(symbol, exchange, startDate, endDate, com.swingtrade.data.service.Interval.DAILY);
    }

    List<CandleData> fetchCandlesList(String symbol, String exchange, LocalDate startDate, LocalDate endDate,
                                      com.swingtrade.data.service.Interval interval) {
        String fyersSymbol = exchange.toUpperCase() + ":" + symbol + "-EQ";
        // Keyed by (date, barTime): daily bars collapse to one entry per date, while
        // intraday bars keep every bar of the session.
        Map<String, CandleData> byBar = new LinkedHashMap<>();
        LocalDate windowStart = startDate;
        while (!windowStart.isAfter(endDate)) {
            LocalDate windowEnd = windowStart.plusDays(CHUNK_DAYS - 1);
            if (windowEnd.isAfter(endDate)) windowEnd = endDate;
            URI uri = UriComponentsBuilder.fromPath("/data/history")
                .queryParam("symbol", fyersSymbol)
                .queryParam("resolution", interval.toFyersResolution())
                .queryParam("date_format", "1")
                .queryParam("date[from]", windowStart)
                .queryParam("date[to]", windowEnd)
                .build().encode().toUri();
            String body = getWithAuthRetry(uri);
            if (body != null) {
                try {
                    for (CandleData candle : parseCandles(symbol, body, interval)) {
                        byBar.put(candle.date() + "|" + candle.barTime(), candle);
                    }
                } catch (Exception e) {
                    logger.error("Failed to parse candles for {}: {}", symbol, e.getMessage());
                }
            }
            windowStart = windowEnd.plusDays(1);
        }
        return new ArrayList<>(byBar.values());
    }

    private List<CandleData> parseCandles(String symbol, String responseBody,
                                          com.swingtrade.data.service.Interval interval) throws Exception {
        if (responseBody == null) return Collections.emptyList();
        JsonNode root = objectMapper.readTree(responseBody);
        if (!root.has("candles") || "error".equals(root.path("s").asText())) {
            logger.warn("Fyers API error for {}: {}", symbol,
                root.has("message") ? root.get("message").asText() : "unknown");
            return Collections.emptyList();
        }
        JsonNode candlesNode = root.get("candles");
        List<CandleData> candles = new ArrayList<>();
        for (JsonNode node : candlesNode) {
            long epochSeconds = node.get(0).asLong();
            ZoneId ist = ZoneId.of("Asia/Kolkata");
            LocalDate date = OffsetDateTime.ofInstant(
                java.time.Instant.ofEpochSecond(epochSeconds), ist).toLocalDate();
            BigDecimal open = node.get(1).decimalValue();
            BigDecimal high = node.get(2).decimalValue();
            BigDecimal low = node.get(3).decimalValue();
            BigDecimal close = node.get(4).decimalValue();
            long volume = node.get(5).asLong();
            // Daily bars are identified by trading date alone; intraday bars carry
            // their actual bar start time.
            java.time.LocalTime barTime = com.swingtrade.data.service.Interval.DAILY.equals(interval)
                ? java.time.LocalTime.MIN
                : OffsetDateTime.ofInstant(java.time.Instant.ofEpochSecond(epochSeconds), ist).toLocalTime();
            candles.add(CandleData.of(symbol, date, barTime, open, high, low, close, volume, close, interval));
        }
        return candles;
    }

    @Override
    public CandleData fetchLatestCandle(String symbol) {
        List<CandleData> candles = fetchCandlesList(symbol, LocalDate.now().minusDays(10), LocalDate.now());
        return candles.isEmpty() ? null : candles.get(candles.size() - 1);
    }

    @Override
    public InstrumentDetails fetchInstrumentDetails(String symbol) {
        Optional<FyersSymbolEntity> match = symbolMaster.findByTradingSymbol(symbol);
        if (match.isPresent()) {
            FyersSymbolEntity e = match.get();
            return InstrumentDetails.of(e.getTradingSymbol(), e.getName(), "NSE_EQ", "EQ",
                e.getLotSize(), e.getTickSize(), e.getIsin());
        }
        logger.warn("No symbol master entry for {}; returning fallback instrument details", symbol);
        return InstrumentDetails.of(symbol, symbol, "NSE_EQ", "EQ", 1, BigDecimal.valueOf(0.05), null);
    }

    @Override
    public ChartMeta fetchChartMeta(String symbol) {
        QuoteData quote = fetchQuote(symbol);
        if (quote == null) return null;
        Optional<FyersSymbolEntity> match = symbolMaster.findByTradingSymbol(symbol);
        String name = match.map(FyersSymbolEntity::getName).orElse(quote.shortName());
        return ChartMeta.of(symbol, "NSE", "EQ", "INR", name, quote.shortName(),
            quote.regularMarketPrice(), null, null, quote.regularMarketPreviousClose(), null,
            0, "Asia/Kolkata", 19800);
    }

    @Override
    public Iterable<String> fetchAllStockSymbols() {
        return symbolMaster.allTradingSymbols();
    }

    // -----------------------------------------------------------------------
    // Quotes
    // -----------------------------------------------------------------------

    @Override
    public PriceBand fetchPriceBand(String symbol, LocalDate date) {
        QuoteData quote = fetchQuote(symbol);
        if (quote == null || quote.lowerPriceBand() == null || quote.upperPriceBand() == null) return null;
        try {
            return new PriceBand(symbol, date, quote.lowerPriceBand(), quote.upperPriceBand());
        } catch (IllegalArgumentException e) {
            logger.warn("Ignoring invalid price band for {} on {}: {}", symbol, date, e.getMessage());
            return null;
        }
    }

    @Override
    public QuoteData fetchQuote(String symbol) {
        List<QuoteData> results = fetchQuotes(List.of(symbol));
        return results.isEmpty() ? null : results.get(0);
    }

    @Override
    public List<QuoteData> fetchQuotes(List<String> symbols) {
        if (symbols == null || symbols.isEmpty()) return Collections.emptyList();
        String joined = symbols.stream().map(s -> "NSE:" + s + "-EQ")
            .collect(Collectors.joining(","));
        URI uri = UriComponentsBuilder.fromPath("/data/quotes")
            .queryParam("symbols", joined).build().encode().toUri();
        String body = getWithAuthRetry(uri);
        if (body == null) return Collections.emptyList();
        try { return parseQuotes(body); }
        catch (Exception e) {
            logger.error("Failed to parse quotes for {}: {}", symbols, e.getMessage());
            return Collections.emptyList();
        }
    }

    private List<QuoteData> parseQuotes(String responseBody) throws Exception {
        JsonNode root = objectMapper.readTree(responseBody);
        if (!"success".equals(root.path("s").asText())) return Collections.emptyList();
        List<QuoteData> results = new ArrayList<>();
        for (JsonNode item : root.path("d")) {
            JsonNode v = item.path("v");
            if (v.isMissingNode()) continue;
            String rawSymbol = v.has("symbol") ? v.get("symbol").asText() : item.path("n").asText(null);
            QuoteData q = parseQuoteFromNode(extractTradingSymbol(rawSymbol), v);
            if (q != null) results.add(q);
        }
        return results;
    }

    private String extractTradingSymbol(String rawSymbol) {
        if (rawSymbol == null) return null;
        String s = rawSymbol.contains(":") ? rawSymbol.substring(rawSymbol.indexOf(':') + 1) : rawSymbol;
        return s.split("-")[0];
    }

    private QuoteData parseQuoteFromNode(String symbol, JsonNode v) {
        try {
            BigDecimal price = decimalOrNull(v, "lp");
            BigDecimal change = decimalOrNull(v, "ch");
            BigDecimal changePct = decimalOrNull(v, "chp");
            BigDecimal high = decimalOrNull(v, "high_price");
            BigDecimal low = decimalOrNull(v, "low_price");
            BigDecimal prevClose = decimalOrNull(v, "prev_close_price");
            BigDecimal lowerBand = decimalOrNull(v, "lower_ckt");
            BigDecimal upperBand = decimalOrNull(v, "upper_ckt");
            Long volume = v.has("volume") ? v.get("volume").asLong() : null;
            String shortName = v.has("short_name") ? v.get("short_name").asText() : null;
            return QuoteData.of(symbol, shortName, null, price, change, changePct,
                high, low, prevClose, null, null, volume, null, null, null, null,
                lowerBand, upperBand);
        } catch (Exception e) {
            logger.warn("Failed to parse quote for {}: {}", symbol, e.getMessage());
            return null;
        }
    }

    private BigDecimal decimalOrNull(JsonNode v, String field) {
        JsonNode node = v.get(field);
        return (node == null || node.isNull()) ? null : node.decimalValue();
    }

    // -----------------------------------------------------------------------
    // Symbol search
    // -----------------------------------------------------------------------

    @Override
    public List<SearchResult> searchSymbols(String query) {
        return symbolMaster.search(query, 20).stream()
            .map(e -> SearchResult.of(e.getTradingSymbol(), e.getName(), e.getName(), "EQUITY", "NSE", "NSE"))
            .toList();
    }

    @Override
    public boolean isConnected() {
        return authService.validateToken();
    }

    // -----------------------------------------------------------------------
    // Shared auth-retry GET helper
    // -----------------------------------------------------------------------

    private String getWithAuthRetry(URI uri) {
        String token = authService.getAccessToken();
        if (token == null) {
            if (!missingTokenLogged) {
                logger.warn("Fyers access token is missing. Data ingestion skipped — configure Fyers credentials to enable.");
                missingTokenLogged = true;
            }
            return null;
        }
        String appId = authService.getClientId();
        try {
            String body = doGet(uri, appId, token);
            if (isAuthError(body)) {
                logger.warn("Fyers auth error for {}. Refreshing and retrying...", uri.getPath());
                return retryAfterRefresh(uri, appId);
            }
            return body;
        } catch (WebClientResponseException e) {
            if (e.getStatusCode().value() == 401) {
                logger.warn("Token expired (401) for {}. Refreshing and retrying...", uri.getPath());
                return retryAfterRefresh(uri, appId);
            }
            logger.error("Fyers API error for {}: {} {}", uri.getPath(), e.getStatusCode(), e.getResponseBodyAsString());
            return null;
        } catch (Exception e) {
            logger.error("Request failed for {}: {}", uri.getPath(), e.getMessage());
            return null;
        }
    }

    private String retryAfterRefresh(URI uri, String appId) {
        authService.refreshToken();
        String newToken = authService.getAccessToken();
        if (newToken == null) return null;
        try {
            String body = doGet(uri, appId, newToken);
            if (isAuthError(body)) {
                logger.error("Fyers auth error persisted after refresh for {}", uri.getPath());
                return null;
            }
            return body;
        } catch (Exception e) {
            logger.error("Retry failed for {}: {}", uri.getPath(), e.getMessage());
            return null;
        }
    }

    private String doGet(URI uri, String appId, String token) {
        Mono<String> mono = webClient.get()
            .uri(uriBuilder -> uriBuilder.replacePath(uri.getRawPath())
                .replaceQuery(uri.getRawQuery()).build())
            .header("Authorization", appId + ":" + token)
            .retrieve().bodyToMono(String.class);
        if (fyersCircuitBreaker != null) {
            mono = mono.transformDeferred(CircuitBreakerOperator.of(fyersCircuitBreaker));
        }
        if (fyersBulkhead != null) {
            mono = mono.transformDeferred(BulkheadOperator.of(fyersBulkhead));
        }
        mono = mono.retryWhen(Retry.backoff(4, Duration.ofSeconds(1))
            .maxBackoff(Duration.ofSeconds(8))
            .filter(this::isRateLimitError)
            .doBeforeRetry(signal -> {
                if (ingestionMetrics != null) {
                    ingestionMetrics.recordRateLimitHit("fyers");
                }
                logger.warn("Fyers rate limit hit for {}; retry {}/{}", uri.getPath(),
                    signal.totalRetries() + 1, 4);
            }));
        return mono
            .onErrorResume(io.github.resilience4j.circuitbreaker.CallNotPermittedException.class,
                e -> {
                    logger.warn("Circuit breaker open for fyers");
                    return Mono.empty();
                })
            .onErrorResume(io.github.resilience4j.bulkhead.BulkheadFullException.class,
                e -> {
                    logger.warn("Bulkhead full for fyers");
                    return Mono.empty();
                })
            .block();
    }

    private boolean isRateLimitError(Throwable error) {
        Throwable current = error;
        while (current != null) {
            if (current instanceof WebClientResponseException response
                    && response.getStatusCode().value() == 429) {
                return true;
            }
            current = current.getCause();
        }
        return false;
    }

    /**
     * Fyers returns HTTP 200 with {"s":"error","code":-16} for bad/expired tokens.
     */
    private boolean isAuthError(String body) {
        if (body == null) return false;
        try {
            JsonNode root = objectMapper.readTree(body);
            if (!"error".equals(root.path("s").asText())) return false;
            int code = root.path("code").asInt(0);
            return code == -15 || code == -16 || code == -17;
        } catch (Exception e) {
            return false;
        }
    }

    // -----------------------------------------------------------------------
    // Order Management — delegated to FyersOrderService
    // -----------------------------------------------------------------------

    public String placeOrder(String symbol, int qty, int side, int orderType,
                              double limitPrice, String productType,
                              Double stopLoss, Double takeProfit, String orderTag) {
        return orderService.placeOrder(symbol, qty, side, orderType, limitPrice,
            productType, stopLoss, takeProfit, orderTag);
    }

    public boolean cancelOrder(String orderId) {
        return orderService.cancelOrder(orderId);
    }

    public List<OrderModel> getAllOrders() {
        return orderService.getAllOrders();
    }

    public List<OrderModel> getOrderHistory(String symbols) {
        return orderService.getOrderHistory(symbols);
    }

    // -----------------------------------------------------------------------
    // Position Management — delegated to FyersPositionService
    // -----------------------------------------------------------------------

    public List<PositionModel> getPositions() {
        return positionService.getPositions();
    }

    public boolean exitAllPositions() {
        return positionService.exitAllPositions();
    }

    public boolean exitPosition(String orderId) {
        return positionService.exitPosition(orderId);
    }

    public boolean convertPosition(String symbol, int side, int convertQty,
                                    String fromProduct, String toProduct) {
        return positionService.convertPosition(symbol, side, convertQty, fromProduct, toProduct);
    }

    // -----------------------------------------------------------------------
    // Trade Book — delegated to FyersTradeBookService
    // -----------------------------------------------------------------------

    public List<TradeBookModel> getTradeBook() {
        return tradeBookService.getTradeBook();
    }

    // -----------------------------------------------------------------------
    // Holdings — delegated to FyersHoldingService
    // -----------------------------------------------------------------------

    public List<HoldingModel> getHoldings() {
        return holdingService.getHoldings();
    }

    // -----------------------------------------------------------------------
    // Market Depth — delegated to FyersMarketDepthService
    // -----------------------------------------------------------------------

    public MarketDepthModel getMarketDepth(String symbol) {
        return marketDepthService.getMarketDepth(symbol);
    }

    // -----------------------------------------------------------------------
    // Market Status — delegated to FyersMarketStatusService
    // -----------------------------------------------------------------------

    public List<MarketStatusModel> getMarketStatus() {
        return marketStatusService.getMarketStatus();
    }

    // -----------------------------------------------------------------------
    // Profile & Funds — delegated to FyersProfileService
    // -----------------------------------------------------------------------

    public ProfileModel getProfile() {
        return profileService.getProfile();
    }

    public List<FundModel> getFunds() {
        return profileService.getFunds();
    }

    // -----------------------------------------------------------------------
    // GTT Orders — delegated to FyersGTTService
    // -----------------------------------------------------------------------

    public String placeGTTOrder(String symbol, int side, String productType,
                                 double triggerPrice, double limitPrice, int qty) {
        return gttService.placeGTTOrder(symbol, side, productType, triggerPrice, limitPrice, qty);
    }

    private FyersClass getSdk() {
        FyersClass sdk = FyersClass.getInstance();
        sdk.clientId = authService.getClientId();
        String token = authService.getAccessToken();
        if (token != null) {
            sdk.accessToken = token;
        }
        return sdk;
    }
}
