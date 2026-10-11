package com.swingtrade.strategy;

import java.math.BigDecimal;
import java.util.List;

public record IntradayBacktestResult(BigDecimal initialCapital, BigDecimal finalCapital,
                                     BigDecimal netPnl, List<IntradayBacktestTrade> trades) {
    public IntradayBacktestResult { trades = List.copyOf(trades); }
}
