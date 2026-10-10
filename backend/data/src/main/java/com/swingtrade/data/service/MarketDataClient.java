package com.swingtrade.data.service;

import java.time.LocalDate;
import java.util.List;

/**
 * Interface for market data clients.
 * Implementations can use Upstox, NSE, or any other data provider.
 */
public interface MarketDataClient {

    /** Returns an authoritative exchange price band, or null when the provider has none. */
    default com.swingtrade.domain.PriceBand fetchPriceBand(String symbol, LocalDate date) {
        return null;
    }

    /**
     * Fetches a single day's OHLCV data for a stock.
     *
     * @param symbol the stock symbol
     * @param date the trading date
     * @return candle data or null if not found
     */
    CandleData fetchCandle(String symbol, LocalDate date);

    /**
     * Fetches multiple days of OHLCV data for a stock.
     *
     * @param symbol the stock symbol
     * @param startDate start date (inclusive)
     * @param endDate end date (inclusive)
     * @return list of candle data
     */
    Iterable<CandleData> fetchCandles(String symbol, LocalDate startDate, LocalDate endDate);

    /**
     * Fetches OHLCV candles for a stock at a specific interval/timeframe.
     *
     * <p>This is the interval-aware path for the intraday data plane. The default
     * implementation throws {@link UnsupportedOperationException} so a provider
     * without interval support fails loudly instead of silently returning daily
     * bars mislabeled as intraday — providers that support intervals (Yahoo, Fyers)
     * override this method.
     *
     * @param symbol the stock symbol
     * @param startDate start date (inclusive)
     * @param endDate end date (inclusive)
     * @param interval the requested candle interval
     * @return list of candle data carrying the interval and bar start time
     */
    default Iterable<CandleData> fetchCandles(String symbol, LocalDate startDate, LocalDate endDate,
                                              Interval interval) {
        throw new UnsupportedOperationException(
            "Market data provider " + getClass().getSimpleName() + " does not support interval " + interval);
    }

    /**
     * Fetches the most recent candle for a stock.
     *
     * @param symbol the stock symbol
     * @return the latest candle data
     */
    CandleData fetchLatestCandle(String symbol);

    /**
     * Fetches the instrument details for a stock.
     *
     * @param symbol the stock symbol
     * @return instrument details or null if not found
     */
    InstrumentDetails fetchInstrumentDetails(String symbol);

    /**
     * Fetches chart metadata for a stock (price, exchange info, 52-week range).
     *
     * @param symbol the stock symbol
     * @return chart metadata or null if not found
     */
    ChartMeta fetchChartMeta(String symbol);

    /**
     * Fetches real-time quote data for a stock.
     * Implementation-dependent: Yahoo Finance uses chart meta field (v7/quote endpoint is dead).
     * Fyers implementation uses GetStockQuotes SDK call.
     *
     * @param symbol the stock symbol
     * @return quote data or null if not found
     */
    QuoteData fetchQuote(String symbol);

    /**
     * Fetches real-time quote data for multiple stocks in a single request.
     *
     * @param symbols list of stock symbols
     * @return list of quote data (may contain nulls for failed symbols)
     */
    List<QuoteData> fetchQuotes(List<String> symbols);

    /**
     * Searches for stock symbols by name or ticker.
     * Uses the v1/finance/search endpoint.
     *
     * @param query search term (company name or symbol)
     * @return list of matching symbols
     */
    List<SearchResult> searchSymbols(String query);

    /**
     * Fetches a list of all available stocks.
     *
     * @return list of stock symbols
     */
    Iterable<String> fetchAllStockSymbols();

    /**
     * Checks if the client is connected and working.
     *
     * @return true if connected
     */
    boolean isConnected();
}
