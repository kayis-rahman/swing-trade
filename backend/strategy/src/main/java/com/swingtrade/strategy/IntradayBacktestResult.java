package com.swingtrade.strategy;

import java.math.BigDecimal;
import java.time.ZonedDateTime;
import java.util.List;

public record IntradayBacktestResult(BigDecimal initialCapital, BigDecimal finalCapital,
                                     BigDecimal netPnl, List<IntradayBacktestTrade> trades,
                                     OpenPosition openPosition, ZonedDateTime openPositionEntryTime) {
    public IntradayBacktestResult { trades = List.copyOf(trades); }
}
