package com.swingtrade.broker.service.intraday;

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
    int expectedBars,
    int receivedBars,
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
    List<PaperTradeRecord> trades
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

    /** Paper trade row with explicit links to its one closed position and both simulated orders. */
    public record PaperTradeRecord(String positionId, String entryOrderId, String exitOrderId,
                                   String strategy, ZonedDateTime entryTime, ZonedDateTime exitTime,
                                   BigDecimal entryPrice, BigDecimal exitPrice, int quantity,
                                   BigDecimal costs, BigDecimal pnl, String exitReason) {}

    /** Closed position row linked to its simulated entry and exit order records. */
    public record PaperPositionRecord(String positionId, String symbol,
                                      String entryOrderId, String exitOrderId,
                                      ZonedDateTime entryTime, ZonedDateTime exitTime,
                                      BigDecimal entryPrice, BigDecimal exitPrice, BigDecimal currentPrice,
                                      int quantity, String status, BigDecimal costs,
                                      BigDecimal realizedPnl, BigDecimal unrealizedPnl, String exitReason,
                                      boolean carriedOvernight) {}
}
