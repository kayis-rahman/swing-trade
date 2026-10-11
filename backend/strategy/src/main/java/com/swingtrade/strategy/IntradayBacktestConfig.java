package com.swingtrade.strategy;

import java.math.BigDecimal;

/** Per-symbol intraday paper-backtest limits. */
public record IntradayBacktestConfig(BigDecimal initialCapital, int quantity,
                                     int maxTradesPerDay, BigDecimal maxDailyLoss) {
    public IntradayBacktestConfig {
        if (initialCapital == null || maxDailyLoss == null || initialCapital.signum() <= 0
                || quantity <= 0 || maxTradesPerDay <= 0 || maxDailyLoss.signum() <= 0) {
            throw new IllegalArgumentException("Capital, quantity and limits must be positive");
        }
    }
    public static IntradayBacktestConfig defaults() {
        return new IntradayBacktestConfig(new BigDecimal("100000"), 1, 3, new BigDecimal("1000"));
    }
}
