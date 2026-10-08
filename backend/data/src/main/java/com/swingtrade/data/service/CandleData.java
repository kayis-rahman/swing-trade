package com.swingtrade.data.service;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;

/**
 * Immutable DTO representing OHLCV candle data.
 * Used for transferring market data between layers.
 *
 * <p>Daily candles (the swing default) carry {@link Interval#DAILY} and a
 * {@code barTime} of 00:00. Intraday candles carry their interval and the bar
 * start time, so a candle's full identity is {@code (symbol, interval, date, barTime)}.
 */
public record CandleData(
    String symbol,
    LocalDate date,
    BigDecimal open,
    BigDecimal high,
    BigDecimal low,
    BigDecimal close,
    long volume,
    BigDecimal adjClose,
    Interval interval,
    LocalTime barTime
) {
    /**
     * Defaults the interval dimension so daily callers keep their exact shape:
     * daily bars are {@link Interval#DAILY} at 00:00, matching the storage default.
     */
    public CandleData {
        if (interval == null) {
            interval = Interval.DAILY;
        }
        if (barTime == null) {
            barTime = LocalTime.MIN;
        }
    }

    /**
     * Creates a candle with normalized values.
     *
     * @param symbol the stock symbol
     * @param date the trading date
     * @param open open price
     * @param high high price
     * @param low low price
     * @param close close price
     * @param volume trading volume
     * @return candle data
     */
    public static CandleData of(String symbol, LocalDate date,
                                 BigDecimal open, BigDecimal high, BigDecimal low,
                                 BigDecimal close, long volume) {
        return new CandleData(symbol, date, open, high, low, close, volume, close,
            Interval.DAILY, LocalTime.MIN);
    }

    /**
     * Creates a candle with adjusted close.
     */
    public static CandleData of(String symbol, LocalDate date,
                                 BigDecimal open, BigDecimal high, BigDecimal low,
                                 BigDecimal close, long volume, BigDecimal adjClose) {
        return new CandleData(symbol, date, open, high, low, close, volume, adjClose,
            Interval.DAILY, LocalTime.MIN);
    }

    /**
     * Creates a candle for a specific interval and bar start time.
     *
     * @param symbol the stock symbol
     * @param date the trading date
     * @param barTime the bar start time (00:00 for daily bars)
     * @param open open price
     * @param high high price
     * @param low low price
     * @param close close price
     * @param volume trading volume
     * @param adjClose adjusted close price
     * @param interval the candle interval/timeframe
     * @return candle data
     */
    public static CandleData of(String symbol, LocalDate date, LocalTime barTime,
                                 BigDecimal open, BigDecimal high, BigDecimal low,
                                 BigDecimal close, long volume, BigDecimal adjClose,
                                 Interval interval) {
        return new CandleData(symbol, date, open, high, low, close, volume, adjClose, interval, barTime);
    }
}
