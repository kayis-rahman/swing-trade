package com.swingtrade.data.client;

import tools.jackson.databind.JsonNode;
import tools.jackson.databind.ObjectMapper;
import com.swingtrade.data.service.CandleData;
import com.swingtrade.data.service.ChartMeta;
import com.swingtrade.data.service.InstrumentDetails;
import com.swingtrade.data.service.MarketDataClient;
import com.swingtrade.data.service.QuoteData;
import com.swingtrade.data.service.SearchResult;

import java.util.concurrent.atomic.AtomicLong;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpHeaders;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;
import reactor.util.retry.Retry;
import reactor.netty.http.client.HttpClient;
import io.github.resilience4j.bulkhead.Bulkhead;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.timelimiter.TimeLimiter;
import io.github.resilience4j.reactor.circuitbreaker.operator.CircuitBreakerOperator;
import io.github.resilience4j.reactor.bulkhead.operator.BulkheadOperator;
import org.springframework.http.client.reactive.ReactorClientHttpConnector;
import io.netty.channel.ChannelOption;

import java.math.BigDecimal;
import java.time.Duration;
import java.time.Instant;
import java.time.LocalDate;
import java.time.LocalDateTime;
import java.time.ZoneOffset;
import java.time.ZoneId;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.function.Function;
import com.swingtrade.data.service.MarketCalendar;

/**
 * Yahoo Finance API client for fetching OHLCV market data.
 * Uses Yahoo Finance's public API to retrieve historical stock data.
 *
 * Note: This uses the public yfinance API which is free and doesn't require
 * authentication. It's suitable for development and backfill scenarios.
 */
public class YahooFinanceClient implements MarketDataClient {

    private static final Logger logger = LoggerFactory.getLogger(YahooFinanceClient.class);
    private static final String YAHOO_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36";

    static final Duration READ_TIMEOUT = Duration.ofSeconds(15);
    static final Duration CONNECT_TIMEOUT = Duration.ofSeconds(30);
    static final String DEFAULT_BASE_URL = "https://query1.finance.yahoo.com";

    private final WebClient webClient;
    private final ObjectMapper objectMapper;
    private final java.time.Clock clock;
    private final MarketCalendar marketCalendar;

    // URI format strings — static to avoid repeated allocation
    private static final String CANDLE_URI_FMT = "/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d&events=history&includePrePost=false";
    private static final String CHART_META_URI_FMT = "/v8/finance/chart/%s?range=1mo&interval=1d";
    private static final String QUOTE_URI_FMT = "/v7/finance/quote?symbols=%s";
    private static final String SEARCH_URI_FMT = "/v1/finance/search?q=%s&quotesCount=6&enableFuzzyQuery=true";

    private final AtomicLong lastRequestTime = new AtomicLong(0);
    private static final long RATE_LIMIT_MS = 1000;
    private static final int MAX_RETRIES = 3;
    private static final Duration RETRY_BACKOFF = Duration.ofSeconds(1);

    // Resilience4j fields (set by constructor with Resilience4j support)
    private CircuitBreaker yahooCircuitBreaker;
    private Bulkhead yahooBulkhead;
    private TimeLimiter yahooTimeLimiter;
    private Duration yahooTimeout;

    public YahooFinanceClient() {
        this(DEFAULT_BASE_URL, new ObjectMapper(), java.time.Clock.systemUTC());
    }

    public YahooFinanceClient(ObjectMapper objectMapper, java.time.Clock clock) {
        this(DEFAULT_BASE_URL, objectMapper, clock);
    }

    // Package-private constructor for testing with MockWebServer
    YahooFinanceClient(String baseUrl) {
        this(baseUrl, new ObjectMapper(), java.time.Clock.systemUTC());
    }

    // Package-private constructor for testing with MockWebServer (daemon threads)
    YahooFinanceClient(String baseUrl, reactor.netty.resources.LoopResources loop) {
        this(baseUrl, ObjectMapper::new, java.time.Clock.systemUTC(), loop);
    }

    // Full package-private constructor for testing with custom clock
    YahooFinanceClient(String baseUrl, java.util.function.Supplier<ObjectMapper> mapperSupplier,
                       java.time.Clock clock, reactor.netty.resources.LoopResources loop) {
        this.objectMapper = mapperSupplier.get();
        this.clock = clock;
        this.marketCalendar = null;
        this.webClient = WebClient.builder()
                .clientConnector(new org.springframework.http.client.reactive.ReactorClientHttpConnector(
                        HttpClient.create().baseUrl(baseUrl).runOn(loop)))
                .defaultHeader(HttpHeaders.USER_AGENT, YAHOO_USER_AGENT)
                .build();
    }

    // Public constructor with configurable baseUrl (for Spring @Value injection)
    public YahooFinanceClient(String baseUrl, ObjectMapper objectMapper, java.time.Clock clock) {
        this(baseUrl, objectMapper, clock, null, null, null, null);
    }

    // Public constructor with Resilience4j support (injected by Spring)
    public YahooFinanceClient(String baseUrl, ObjectMapper objectMapper, java.time.Clock clock,
                              CircuitBreaker circuitBreaker, Bulkhead bulkhead, TimeLimiter timeLimiter) {
        this(baseUrl, objectMapper, clock, circuitBreaker, bulkhead, timeLimiter, null);
    }

    public YahooFinanceClient(String baseUrl, ObjectMapper objectMapper, java.time.Clock clock,
                              CircuitBreaker circuitBreaker, Bulkhead bulkhead, TimeLimiter timeLimiter,
                              MarketCalendar marketCalendar) {
        this.objectMapper = objectMapper;
        this.clock = clock;
        this.marketCalendar = marketCalendar;
        this.webClient = WebClient.builder()
                .clientConnector(new ReactorClientHttpConnector(
                    HttpClient.create()
                        .option(ChannelOption.CONNECT_TIMEOUT_MILLIS, (int) CONNECT_TIMEOUT.toMillis())
                        .responseTimeout(READ_TIMEOUT)))
                .baseUrl(baseUrl)
                .defaultHeader(HttpHeaders.USER_AGENT, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
                .build();
        this.yahooCircuitBreaker = circuitBreaker;
        this.yahooBulkhead = bulkhead;
        this.yahooTimeLimiter = timeLimiter;
        this.yahooTimeout = timeLimiter != null ? Duration.ofSeconds(15) : null;
    }

    /**
     * Wraps a WebClient request chain with Resilience4j operators.
     * Caller builds the full request (get().uri().onStatus().bodyToMono()) and this applies resilience.
     */
    private String executeWithResilience(Function<WebClient, Mono<String>> fetchFn) {
        // Defer both throttling and request creation so every retry is rate-limited too.
        Mono<String> mono = Mono.defer(() -> {
            enforceRateLimit();
            return fetchFn.apply(webClient);
        });
        // Circuit breaker and bulkhead wrap EACH individual attempt, not the whole
        // retry burst: retryWhen is applied last (outermost) so a retry re-enters
        // both operators as a fresh call. Wrapping them the other way around (as
        // this used to) makes a 4-attempt retry burst look like a single slow call
        // to the circuit breaker - it opens far later than configured - and holds
        // one bulkhead permit for the whole ~7s backoff window instead of only
        // during each attempt.
        if (yahooCircuitBreaker != null) {
            mono = mono.transformDeferred(CircuitBreakerOperator.of(yahooCircuitBreaker));
        }
        if (yahooBulkhead != null) {
            mono = mono.transformDeferred(BulkheadOperator.of(yahooBulkhead));
        }
        mono = mono.retryWhen(Retry.backoff(MAX_RETRIES, RETRY_BACKOFF)
                .scheduler(reactor.core.scheduler.Schedulers.boundedElastic())
                .filter(this::isRetryableFailure));
        if (yahooTimeout != null) {
            mono = mono.timeout(yahooTimeout);
        }
        return mono
                .onErrorResume(io.github.resilience4j.circuitbreaker.CallNotPermittedException.class,
                        e -> {
                            logger.warn("Circuit breaker open for yahoo");
                            return Mono.error(new YahooDataUnavailableException(
                                    "Yahoo circuit breaker is open", e));
                        })
                .onErrorResume(java.util.concurrent.TimeoutException.class,
                        e -> {
                            logger.warn("Time limit exceeded for yahoo");
                            return Mono.error(new YahooDataUnavailableException(
                                    "Yahoo request timed out", e));
                        })
                .onErrorResume(io.github.resilience4j.bulkhead.BulkheadFullException.class,
                        e -> {
                            logger.warn("Bulkhead full for yahoo");
                            return Mono.error(new YahooDataUnavailableException(
                                    "Yahoo request bulkhead is full", e));
                        })
                .block();
    }

    /** Retry transient DNS/connectivity failures as well as retryable HTTP responses. */
    private boolean isRetryableFailure(Throwable error) {
        if (error instanceof YahooHttpException yahoo) return yahoo.isRetryable();
        Throwable current = error;
        while (current != null) {
            if (current instanceof java.net.UnknownHostException
                    || current instanceof java.net.ConnectException
                    || current instanceof java.net.SocketTimeoutException
                    || current instanceof java.io.IOException) {
                return true;
            }
            current = current.getCause();
        }
        return false;
    }

    /**
     * Fetches a single day's OHLCV data for a stock from Yahoo Finance.
     *
     * @param symbol the stock symbol (e.g., "RELIANCE.NS" for NSE)
     * @param date the trading date
     * @return candle data or null if not found
     */
    @Override
    public CandleData fetchCandle(String symbol, LocalDate date) {
        try {
            for (CandleData candle : fetchCandles(symbol, date, date)) {
                if (date.equals(candle.date())) return candle;
            }
            logger.warn("Yahoo single-day fetch returned no valid close values for {} on {}", symbol, date);
            return null;

        } catch (Exception e) {
            logger.warn("Failed to fetch candle for {} on {}: {}", symbol, date, e.getMessage());
            return null;
        }
    }

    /**
     * Fetches multiple days of OHLCV data for a stock.
     *
     * @param symbol the stock symbol
     * @param startDate start date (inclusive)
     * @param endDate end date (inclusive)
     * @return list of candle data
     */
    @Override
    public Iterable<CandleData> fetchCandles(String symbol, LocalDate startDate, LocalDate endDate) {
        return fetchCandles(symbol, "NSE", startDate, endDate);
    }

    @Override
    public Iterable<CandleData> fetchCandles(String symbol, String exchange,
                                              LocalDate startDate, LocalDate endDate) {
        List<CandleData> candles = new ArrayList<>();

        try {
            String yfinanceSymbol = formatSymbolForYahoo(symbol, exchange);

            long period1 = startDate.atStartOfDay().toEpochSecond(java.time.ZoneOffset.UTC);
            long period2 = endDate.atStartOfDay().toEpochSecond(java.time.ZoneOffset.UTC) + 86400;

            String uri = String.format(CANDLE_URI_FMT, yfinanceSymbol, period1, period2);

            String response = executeWithResilience(client ->
                    client.get().uri(uri)
                        .retrieve()
                        // A 404 means Yahoo has no data for this symbol (delisted, unknown,
                        // or wrong exchange suffix) - treat it the same as fetchCandle() does:
                        // an empty result, not a failure. Without this, an unknown symbol
                        // went from "empty candle list" to a thrown YahooDataUnavailableException,
                        // which IngestionController turns into a 500 instead of 200-with-zero.
                        .onStatus(s -> s.value() == 404, r -> Mono.empty())
                        .onStatus(s -> s.value() >= 400,
                                r -> Mono.error(new YahooHttpException(r.statusCode().value())))
                        .bodyToMono(String.class));

            if (response == null || response.isEmpty()) return candles;
            JsonNode root = objectMapper.readTree(response);
            JsonNode result = root.path("chart").path("result");
            if (!result.isArray() || result.size() == 0) return candles;
            JsonNode quoteObj = result.get(0).path("indicators").path("quote").get(0);
            JsonNode timestamps = result.get(0).path("timestamp");
            if (!quoteObj.isObject() || !timestamps.isArray() || timestamps.size() == 0) return candles;

            JsonNode openArr = quoteObj.path("open");
            JsonNode highArr = quoteObj.path("high");
            JsonNode lowArr = quoteObj.path("low");
            JsonNode closeArr = quoteObj.path("close");
            JsonNode volumeArr = quoteObj.path("volume");
            JsonNode adjArr = result.get(0).path("indicators").path("adjclose")
                .isArray() && result.get(0).path("indicators").path("adjclose").size() > 0
                    ? result.get(0).path("indicators").path("adjclose").get(0).path("adjclose") : null;
            ZoneId exchangeZone = resolveExchangeZone(result.get(0).path("meta"));
            for (int i = 0; i < timestamps.size(); i++) {
                LocalDate resolvedDate = LocalDate.ofInstant(Instant.ofEpochSecond(timestamps.get(i).asLong()), exchangeZone);
                String rejection = null;
                if (resolvedDate.isBefore(startDate) || resolvedDate.isAfter(endDate)) rejection = "outside_requested_range";
                else if (resolvedDate.getDayOfWeek().getValue() > 5) rejection = "non_trading_day";
                else if (marketCalendar != null && !marketCalendar.isNseTradingSession(resolvedDate)) rejection = "nse_holiday";
                else if (closeArr.isMissingNode() || closeArr.get(i).isNull() || closeArr.get(i).asDouble(0) == 0) rejection = "null_or_zero_close";
                if (rejection != null) {
                    logger.warn("Rejected Yahoo candle timestamp symbol={} sourceTimestamp={} resolvedDate={} reason={}",
                        symbol, timestamps.get(i).asLong(), resolvedDate, rejection);
                    continue;
                }
                long volume = volumeArr.isNull() ? 0 : volumeArr.get(i).asLong(0);
                if (volume == 0 || openArr.get(i).isNull() || highArr.get(i).isNull() || lowArr.get(i).isNull()) {
                    logger.warn("Rejected Yahoo candle timestamp symbol={} sourceTimestamp={} resolvedDate={} reason=null_or_zero_ohlcv",
                        symbol, timestamps.get(i).asLong(), resolvedDate);
                    continue;
                }
                BigDecimal adjClose = (adjArr != null && !adjArr.isNull() && i < adjArr.size())
                    ? parseBigDecimal(adjArr.get(i)) : parseBigDecimal(closeArr.get(i));
                candles.add(CandleData.of(symbol,
                    resolvedDate,
                    parseBigDecimal(openArr.get(i)), parseBigDecimal(highArr.get(i)),
                    parseBigDecimal(lowArr.get(i)), parseBigDecimal(closeArr.get(i)), volume, adjClose));
            }
            logger.info("Fetched {} candles for {} from {} to {}", candles.size(), symbol, startDate, endDate);

        } catch (Exception e) {
            logger.error("Error fetching candles for {}: {}", symbol, e.getMessage());
            throw new YahooDataUnavailableException("Yahoo historical data unavailable for " + symbol, e);
        }
        return candles;
    }

    private ZoneId resolveExchangeZone(JsonNode meta) {
        String timezone = meta.path("timezone").asText("");
        try {
            return timezone.isBlank() ? ZoneId.of("Asia/Kolkata") : ZoneId.of(timezone);
        } catch (Exception ignored) {
            return ZoneId.of("Asia/Kolkata");
        }
    }

    /**
     * Fetches the most recent candle for a stock.
     *
     * @param symbol the stock symbol
     * @return the latest candle data
     */
    @Override
    public CandleData fetchLatestCandle(String symbol) {
        // Fetch last 30 days to get the most recent candle
        LocalDate endDate = LocalDate.now();
        LocalDate startDate = endDate.minusDays(30);

        Iterable<CandleData> candles = fetchCandles(symbol, startDate, endDate);

        CandleData latest = null;
        for (CandleData candle : candles) {
            if (latest == null || candle.date().isAfter(latest.date())) {
                latest = candle;
            }
        }

        return latest;
    }

    /**
     * Fetches instrument details for a stock.
     * Uses the chart endpoint's meta field (no separate API call needed).
     */
    @Override
    public InstrumentDetails fetchInstrumentDetails(String symbol) {
        ChartMeta meta = fetchChartMeta(symbol);
        if (meta == null) return null;

        String exchange = meta.exchange().toUpperCase() + "_EQ";
        return InstrumentDetails.of(
            symbol,
            meta.longName(),
            exchange,
            meta.instrumentType(),
            null, // Lot size not available from Yahoo Finance
            null, // Tick size not available
            null  // ISIN not available
        );
    }

    /**
     * Fetches chart metadata for a stock using the chart endpoint meta field.
     * No separate API call needed — meta is returned with every chart response.
     */
    @Override
    public ChartMeta fetchChartMeta(String symbol) {
        try {
            String yfinanceSymbol = formatSymbolForYahoo(symbol);

            String uri = String.format(CHART_META_URI_FMT, yfinanceSymbol);

            String response = executeWithResilience(client ->
                    client.get().uri(uri)
                        .retrieve()
                        .onStatus(s -> s.value() >= 400,
                                r -> Mono.error(new YahooHttpException(r.statusCode().value())))
                        .bodyToMono(String.class));

            if (response == null || response.isEmpty()) return null;
            JsonNode root = objectMapper.readTree(response);
            JsonNode result = root.path("chart").path("result");
            if (!result.isArray() || result.size() == 0) return null;
            JsonNode meta = result.get(0).path("meta");

            return ChartMeta.of(
                meta.path("symbol").asText(), meta.path("fullExchangeName").asText(),
                meta.path("instrumentType").asText(), meta.path("currency").asText(),
                meta.path("longName").asText(), meta.path("shortName").asText(),
                parseBigDecimalOrNull(meta.path("regularMarketPrice")),
                parseBigDecimalOrNull(meta.path("fiftyTwoWeekHigh")),
                parseBigDecimalOrNull(meta.path("fiftyTwoWeekLow")),
                parseBigDecimalOrNull(meta.path("chartPreviousClose")),
                meta.has("regularMarketTime")
                    ? LocalDateTime.ofInstant(
                        Instant.ofEpochSecond(meta.path("regularMarketTime").asLong()), ZoneOffset.UTC)
                    : null,
                meta.path("firstTradeDate").asInt(),
                meta.path("timezone").asText(), meta.path("gmtoffset").asInt());

        } catch (Exception e) {
            logger.warn("Failed to fetch chart meta for {}: {}", symbol, e.getMessage());
            return null;
        }
    }

    /**
     * Fetches a list of all available stocks.
     * Note: Yahoo Finance doesn't provide a complete list of all stocks.
     * This returns a limited set or empty list.
     *
     * @return list of stock symbols
     */
    @Override
    public Iterable<String> fetchAllStockSymbols() {
        // Yahoo Finance doesn't provide a complete stock list endpoint
        // Return empty - use a predefined list instead
        return List.of();
    }

    /**
     * Checks if the client is connected and working.
     *
     * @return true if connected
     */
    @Override
    public boolean isConnected() {
        try {
            // Lightweight check: fetch chart meta for a common symbol (no date params needed)
            ChartMeta meta = fetchChartMeta("RELIANCE.NS");
            return meta != null && meta.symbol() != null;
        } catch (Exception e) {
            return false;
        }
    }

    /**
     * Enforces rate limiting between requests to avoid Yahoo blocking.
     * Sleeps if the last request was less than RATE_LIMIT_MS ago.
     */
    private synchronized void enforceRateLimit() {
        long now = clock.millis();
        long elapsed = now - lastRequestTime.get();
        if (elapsed < RATE_LIMIT_MS && lastRequestTime.get() > 0) {
            try { Thread.sleep(RATE_LIMIT_MS - elapsed); }
            catch (InterruptedException e) { Thread.currentThread().interrupt(); }
        }
        lastRequestTime.set(clock.millis());
    }

    /**
     * Fetches real-time quote data for a single stock.
     * Uses the v7/finance/quote endpoint.
     *
     * @param symbol the stock symbol
     * @return quote data or null if not found
     */
    @Override
    public QuoteData fetchQuote(String symbol) {
        List<String> symbols = List.of(symbol);
        List<QuoteData> quotes = fetchQuotes(symbols);
        return quotes.isEmpty() ? null : quotes.get(0);
    }

    /**
     * Fetches real-time quote data for multiple stocks in a single request.
     *
     * @param symbols list of stock symbols
     * @return list of quote data (may contain nulls for failed symbols)
     */
    @Override
    public List<QuoteData> fetchQuotes(List<String> symbols) {
        if (symbols == null || symbols.isEmpty()) {
            return Collections.emptyList();
        }

        try {
            // Format symbols for Yahoo (add .NS suffix if needed)
            String yahooSymbols = symbols.stream()
                    .map(this::formatSymbolForYahoo)
                    .collect(java.util.stream.Collectors.joining(","));

            String uri = String.format(QUOTE_URI_FMT, yahooSymbols);

            String response = executeWithResilience(client ->
                    client.get().uri(uri)
                        .retrieve()
                        .onStatus(s -> s.value() >= 400,
                                r -> Mono.error(new YahooHttpException(r.statusCode().value())))
                        .bodyToMono(String.class));

            if (response == null || response.isEmpty()) return Collections.emptyList();
            JsonNode root = objectMapper.readTree(response);
            JsonNode result = root.path("finance").path("result");
            if (!result.isArray()) return Collections.emptyList();

            List<QuoteData> quotes = new ArrayList<>();
            for (JsonNode item : result) {
                QuoteData quote = parseQuote(item);
                if (quote != null) quotes.add(quote);
            }
            logger.info("Fetched {} quotes for symbols: {}", quotes.size(), yahooSymbols);
            return quotes;

        } catch (Exception e) {
            logger.error("Error fetching quotes for {}: {}", symbols, e.getMessage());
            return Collections.emptyList();
        }
    }

    /**
     * Parses a single quote JSON node into a QuoteData record.
     */
    private QuoteData parseQuote(JsonNode node) {
        String symbol = node.path("symbol").asText(null);
        if (symbol == null || "NONE".equals(node.path("quoteType").asText(null))) return null;
        return QuoteData.of(
            symbol, node.path("shortName").asText(null), node.path("longName").asText(null),
            parseBigDecimalOrNull(node.path("regularMarketPrice")),
            parseBigDecimalOrNull(node.path("regularMarketChange")),
            parseBigDecimalOrNull(node.path("regularMarketChangePercent")),
            parseBigDecimalOrNull(node.path("regularMarketDayHigh")),
            parseBigDecimalOrNull(node.path("regularMarketDayLow")),
            parseBigDecimalOrNull(node.path("regularMarketPreviousClose")),
            parseBigDecimalOrNull(node.path("fiftyTwoWeekHigh")),
            parseBigDecimalOrNull(node.path("fiftyTwoWeekLow")),
            node.has("regularMarketVolume") && !node.path("regularMarketVolume").isNull()
                ? node.path("regularMarketVolume").asLong(0) : null,
            node.path("currency").asText(null), node.path("marketState").asText(null),
            parseBigDecimalOrNull(node.path("fiftyDayAverage")),
            parseBigDecimalOrNull(node.path("twoHundredDayAverage")));
    }

    /**
     * Searches for stock symbols by name or ticker.
     * Uses the v1/finance/search endpoint.
     *
     * @param query search term
     * @return list of matching search results
     */
    @Override
    public List<SearchResult> searchSymbols(String query) {
        if (query == null || query.isBlank()) {
            return Collections.emptyList();
        }

        try {
            String uri = String.format(SEARCH_URI_FMT, query.trim());

            String response = executeWithResilience(client ->
                    client.get().uri(uri)
                        .retrieve()
                        .onStatus(s -> s.value() >= 400,
                                r -> Mono.error(new YahooHttpException(r.statusCode().value())))
                        .bodyToMono(String.class));

            if (response == null || response.isEmpty()) return Collections.emptyList();
            JsonNode root = objectMapper.readTree(response);
            JsonNode quotes = root.path("quotes");
            if (!quotes.isArray()) return Collections.emptyList();

            List<SearchResult> results = new ArrayList<>();
            for (JsonNode quote : quotes) {
                if (!quote.path("isYahooFinance").asBoolean(false)) continue;
                String quoteType = quote.path("quoteType").asText(null);
                if (!"EQUITY".equals(quoteType)) continue;
                String exchange = quote.path("exchange").asText(null);
                results.add(SearchResult.of(
                    quote.path("symbol").asText(null),
                    quote.path("shortname").asText(null),
                    quote.path("longname").asText(null),
                    quoteType, exchange, quote.path("exchangeName").asText(null)));
            }
            logger.info("Search for '{}' returned {} results", query, results.size());
            return results;

        } catch (Exception e) {
            logger.error("Error searching symbols for '{}': {}", query, e.getMessage());
            return Collections.emptyList();
        }
    }

    private String formatSymbolForYahoo(String symbol) {
        return formatSymbolForYahoo(symbol, "NSE");
    }

    private String formatSymbolForYahoo(String symbol, String exchange) {
        if ("NIFTY50".equalsIgnoreCase(symbol) || "NIFTY 50".equalsIgnoreCase(symbol)) {
            return "^NSEI";
        }
        if (symbol.endsWith(".NS") || symbol.endsWith(".BO")) return symbol;
        return symbol + ("BSE".equalsIgnoreCase(exchange) ? ".BO" : ".NS");
    }

    private BigDecimal parseBigDecimal(JsonNode node) {
        if (node == null || node.isNull()) return BigDecimal.ZERO;
        try { return new BigDecimal(node.asText()); }
        catch (Exception e) { return BigDecimal.ZERO; }
    }

    private BigDecimal parseBigDecimalOrNull(JsonNode node) {
        if (node == null || node.isNull() || !node.isNumber()) return null;
        try { return new BigDecimal(node.asText()); }
        catch (Exception e) { return null; }
    }

    static final class YahooHttpException extends RuntimeException {
        private final int statusCode;

        YahooHttpException(int statusCode) {
            super("Yahoo Finance returned HTTP " + statusCode);
            this.statusCode = statusCode;
        }

        boolean isRetryable() {
            return statusCode == 429 || statusCode >= 500;
        }
    }

    static final class YahooDataUnavailableException extends RuntimeException {
        YahooDataUnavailableException(String message, Throwable cause) {
            super(message, cause);
        }
    }
}
