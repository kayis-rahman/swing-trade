package com.swingtrade.strategy;

import java.math.BigDecimal;
import java.time.ZonedDateTime;

public record IntradayBacktestTrade(String strategy, ZonedDateTime entryTime, ZonedDateTime exitTime,
                                    BigDecimal entryPrice, BigDecimal exitPrice, int quantity,
                                    BigDecimal costs, BigDecimal pnl, String exitReason) {}
