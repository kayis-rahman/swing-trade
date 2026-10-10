package com.swingtrade.data.service;

/**
 * Timeframe dimension for OHLCV candles stored in {@code ohlcv_candles}.
 *
 * <p>The swing path is daily-only: every existing row carries {@link #DAILY} and a
 * {@code bar_time} of 00:00, so daily identity stays {@code (symbol, date)}.
 * Intraday bars carry their interval code and the bar start time, giving the
 * widened identity {@code (symbol, interval, date, bar_time)}.
 *
 * <p>Provider code mappings keep the hardcoded daily resolutions out of the
 * individual clients: Yahoo's chart API interval and Fyers' history API resolution.
 */
public enum Interval {

    /** Daily bars — the swing default; one bar per trading session. */
    DAILY("D", "1d", "D"),

    /** Fifteen-minute bars — the captain-selected intraday timeframe. */
    FIFTEEN_MINUTES("15m", "15m", "15");

    private final String code;
    private final String yahooCode;
    private final String fyersCode;

    Interval(String code, String yahooCode, String fyersCode) {
        this.code = code;
        this.yahooCode = yahooCode;
        this.fyersCode = fyersCode;
    }

    /** Storage code persisted in {@code ohlcv_candles.interval}. */
    public String getCode() {
        return code;
    }

    /** Interval code for Yahoo Finance chart API requests. */
    public String toYahooCode() {
        return yahooCode;
    }

    /** Resolution code for Fyers v3 history API requests. */
    public String toFyersResolution() {
        return fyersCode;
    }

    /**
     * Resolves a storage code to an interval.
     *
     * @param code the stored interval code (e.g. "D", "15m")
     * @return the matching interval
     * @throws IllegalArgumentException if the code is unknown
     */
    public static Interval fromCode(String code) {
        if (code != null) {
            for (Interval interval : values()) {
                if (interval.code.equalsIgnoreCase(code)) {
                    return interval;
                }
            }
        }
        throw new IllegalArgumentException("Unknown candle interval code: " + code);
    }
}
