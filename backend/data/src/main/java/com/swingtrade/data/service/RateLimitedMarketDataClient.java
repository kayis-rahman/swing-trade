package com.swingtrade.data.service;

import com.swingtrade.core.metrics.DataIngestionMetrics;
import java.time.LocalDate;
import java.util.List;

/**
 * Applies a provider-wide token bucket between market-data requests. The wrapper is
 * deliberately synchronous: callers already use blocking provider APIs, and one
 * shared limiter prevents concurrent backfill workers from bypassing a quota.
 */
final class RateLimitedMarketDataClient implements MarketDataClient {

    private final MarketDataClient delegate;
    private final long refillIntervalNanos;
    private final DataIngestionMetrics metrics;
    private final String source;
    private final boolean unlimited;
    private long availableAtNanos;

    RateLimitedMarketDataClient(MarketDataClient delegate, long minimumIntervalMillis) {
        this(delegate, minimumIntervalMillis <= 0 ? 0
                : Math.max(1L, 60_000L / minimumIntervalMillis), null, "unknown");
    }

    RateLimitedMarketDataClient(MarketDataClient delegate, long requestsPerMinute,
                                DataIngestionMetrics metrics, String source) {
        this.delegate = delegate;
        long permits = Math.max(1L, requestsPerMinute);
        this.refillIntervalNanos = Math.max(1L, 60_000_000_000L / permits);
        this.metrics = metrics;
        this.source = source == null || source.isBlank() ? "unknown" : source;
        this.unlimited = requestsPerMinute <= 0;
    }

    private void acquire() {
        if (unlimited) return;
        synchronized (this) {
            long now = System.nanoTime();
            long waitNanos = availableAtNanos - now;
            if (waitNanos > 0L) {
                if (metrics != null) metrics.recordRateLimitWait(source);
                try {
                    long millis = waitNanos / 1_000_000L;
                    int nanos = (int) (waitNanos % 1_000_000L);
                    Thread.sleep(millis, nanos);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    throw new IllegalStateException("Interrupted while rate limiting market-data request", e);
                }
            }
            availableAtNanos = System.nanoTime() + refillIntervalNanos;
        }
    }

    @Override public CandleData fetchCandle(String symbol, LocalDate date) {
        acquire(); return delegate.fetchCandle(symbol, date);
    }
    @Override public Iterable<CandleData> fetchCandles(String symbol, LocalDate startDate, LocalDate endDate) {
        acquire(); return delegate.fetchCandles(symbol, startDate, endDate);
    }
    @Override public Iterable<CandleData> fetchCandles(String symbol, String exchange,
                                                        LocalDate startDate, LocalDate endDate) {
        acquire(); return delegate.fetchCandles(symbol, exchange, startDate, endDate);
    }
    @Override public Iterable<CandleData> fetchCandles(String symbol, LocalDate startDate, LocalDate endDate,
                                                      com.swingtrade.data.service.Interval interval) {
        acquire(); return delegate.fetchCandles(symbol, startDate, endDate, interval);
    }
    @Override public CandleData fetchLatestCandle(String symbol) {
        acquire(); return delegate.fetchLatestCandle(symbol);
    }
    @Override public com.swingtrade.domain.PriceBand fetchPriceBand(String symbol, LocalDate date) {
        acquire(); return delegate.fetchPriceBand(symbol, date);
    }
    @Override public InstrumentDetails fetchInstrumentDetails(String symbol) {
        acquire(); return delegate.fetchInstrumentDetails(symbol);
    }
    @Override public ChartMeta fetchChartMeta(String symbol) {
        acquire(); return delegate.fetchChartMeta(symbol);
    }
    @Override public QuoteData fetchQuote(String symbol) {
        acquire(); return delegate.fetchQuote(symbol);
    }
    @Override public List<QuoteData> fetchQuotes(List<String> symbols) {
        acquire(); return delegate.fetchQuotes(symbols);
    }
    @Override public List<SearchResult> searchSymbols(String query) {
        acquire(); return delegate.searchSymbols(query);
    }
    @Override public Iterable<String> fetchAllStockSymbols() {
        acquire(); return delegate.fetchAllStockSymbols();
    }
    @Override public boolean isConnected() {
        acquire(); return delegate.isConnected();
    }
}
