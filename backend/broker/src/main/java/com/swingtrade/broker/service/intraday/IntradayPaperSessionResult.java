package com.swingtrade.broker.service.intraday;

import com.swingtrade.strategy.IntradayBacktestTrade;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.ZonedDateTime;
import java.util.List;

/** Immutable, auditable output of one paper-only intraday session replay. */
public record IntradayPaperSessionResult(
    String runId,
    String executionMode,
    String status,
    String symbol,
    LocalDate sessionDate,
    String strategy,
    boolean killSwitchActive,
    int maxConcurrentPositions,
    int openPositionsAtEnd,
    int liveBrokerCalls,
    BigDecimal initialCapital,
    BigDecimal finalCapital,
    BigDecimal netPnl,
    List<PaperOrderRecord> orders,
    List<PaperPositionRecord> positions,
    List<IntradayBacktestTrade> trades
) {
    public IntradayPaperSessionResult {
        orders = List.copyOf(orders);
        positions = List.copyOf(positions);
        trades = List.copyOf(trades);
    }

    /** A simulated order record; no broker order ID or external submission exists. */
    public record PaperOrderRecord(String orderId, String symbol, String side,
                                   ZonedDateTime filledAt, BigDecimal fillPrice,
                                   int quantity, String status, String executionMode,
                                   String fillSource) {}

    /** Closed position row linked to its simulated entry and exit order records. */
    public record PaperPositionRecord(String positionId, String symbol,
                                      String entryOrderId, String exitOrderId,
                                      ZonedDateTime entryTime, ZonedDateTime exitTime,
                                      BigDecimal entryPrice, BigDecimal exitPrice,
                                      int quantity, String status, BigDecimal costs,
                                      BigDecimal realizedPnl, String exitReason,
                                      boolean carriedOvernight) {}
}
