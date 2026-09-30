package com.swingtrade.domain.service;

import com.swingtrade.domain.Order;
import com.swingtrade.domain.Position;

import java.math.BigDecimal;
import java.util.List;
import java.util.Optional;

/**
 * Port through which the trading engine persists and restores its in-memory state.
 * Implemented by {@code PaperTradingStateService} in the broker module.
 *
 * <p>The dependency is one-way: the engine depends on this port, and the implementation
 * never depends on the engine. Everything the implementation needs from the engine is
 * passed in as a value ({@link PortfolioState}, {@link SnapshotState}), and restored
 * state is handed back as a {@link PersistedState} for the engine to apply.
 */
public interface TradingStatePersistence {

    /** Portfolio-level figures persisted to the single default-portfolio row. */
    record PortfolioState(
        BigDecimal currentCapital,
        BigDecimal initialCapital,
        BigDecimal totalRealizedPnl,
        BigDecimal totalUnrealizedPnl,
        int openPositionCount
    ) {
    }

    /** One point-in-time portfolio valuation, appended to the snapshot history. */
    record SnapshotState(
        BigDecimal totalValue,
        BigDecimal cashBalance,
        BigDecimal marketValue,
        BigDecimal totalPnl,
        BigDecimal returnPct,
        int openPositions
    ) {
    }

    /**
     * State read back from storage at startup.
     *
     * @param currentCapital persisted cash balance, or null when no portfolio row exists
     * @param initialCapital persisted starting capital, or null when no portfolio row exists
     * @param openPositions open positions to re-register with the engine
     * @param pendingOrders PENDING/ACCEPTED orders to re-register with the engine
     * @param maxPositionIdSuffix highest {@code POS_<n>} suffix ever persisted (any status)
     */
    record PersistedState(
        BigDecimal currentCapital,
        BigDecimal initialCapital,
        List<Position> openPositions,
        List<Order> pendingOrders,
        long maxPositionIdSuffix
    ) {
        public static PersistedState empty() {
            return new PersistedState(null, null, List.of(), List.of(), 0L);
        }
    }

    /** Loads persisted state; never throws, falling back to whatever could be read. */
    PersistedState loadState();

    void savePortfolio(PortfolioState state);

    void saveSnapshot(SnapshotState state);

    void saveOrder(Order order);

    void savePosition(Position position);

    void savePosition(Position position, Long signalId);

    /** Saves a signal-originated position together with its entry commission for trade audit. */
    default void savePosition(Position position, Long signalId, BigDecimal entryCommission) {
        savePosition(position, signalId);
    }

    void closePosition(String positionId, Position closedPosition);

    /**
     * Resolves the engine-side position id ({@code POS_<n>}) of a persisted position.
     *
     * @param databaseId the database primary key
     * @return the position id, or empty if the row is missing or has no position id
     */
    Optional<String> findPositionId(Long databaseId);

    /** Total portfolio value of every persisted snapshot, newest first. */
    List<BigDecimal> getSnapshotTotalValues();
}
