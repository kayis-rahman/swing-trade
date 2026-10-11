package com.swingtrade.strategy;

import com.swingtrade.domain.OhlcvCandle;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;

/** A timestamped 15-minute bar for the isolated, paper-only intraday research path. */
public record IntradayBar(ZonedDateTime timestamp, BigDecimal open, BigDecimal high,
                          BigDecimal low, BigDecimal close, long volume) {
    private static final ZoneId NSE_ZONE = ZoneId.of("Asia/Kolkata");

    public IntradayBar {
        if (timestamp == null || open == null || high == null || low == null || close == null) {
            throw new IllegalArgumentException("Timestamp and OHLC values are required");
        }
        if (volume < 0 || high.compareTo(low) < 0 || high.compareTo(open) < 0
                || high.compareTo(close) < 0 || low.compareTo(open) > 0 || low.compareTo(close) > 0) {
            throw new IllegalArgumentException("Invalid OHLCV bar");
        }
    }

    public LocalDate sessionDate() { return timestamp.withZoneSameInstant(NSE_ZONE).toLocalDate(); }
    public LocalTime sessionTime() { return timestamp.withZoneSameInstant(NSE_ZONE).toLocalTime(); }

    OhlcvCandle asCandle(String symbol) {
        return OhlcvCandle.of(symbol, sessionDate(), open, high, low, close, volume);
    }
}
