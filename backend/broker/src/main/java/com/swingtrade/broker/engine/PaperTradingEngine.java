package com.swingtrade.broker.engine;

import com.swingtrade.broker.config.PaperTradingProperties;
import com.swingtrade.broker.manager.OrderManager;
import com.swingtrade.broker.manager.PositionManager;
import com.swingtrade.core.metrics.TradeMetrics;
import com.swingtrade.domain.Order;
import com.swingtrade.domain.OrderStatus;
import com.swingtrade.domain.service.TradingService;
import com.swingtrade.domain.service.TradingStatePersistence;
import com.swingtrade.domain.service.TradingStatePersistence.PersistedState;
import com.swingtrade.domain.service.TradingStatePersistence.PortfolioState;
import com.swingtrade.domain.service.TradingStatePersistence.SnapshotState;
import com.swingtrade.domain.OhlcvCandle;
import com.swingtrade.domain.PriceBand;
import com.swingtrade.domain.Position;
import com.swingtrade.domain.PositionStatus;
import com.swingtrade.domain.Signal;
import com.swingtrade.broker.model.Portfolio;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.event.EventListener;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.time.LocalDate;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Core trading engine for paper trading functionality.
 * Integrates order management, position management, and risk controls.
 * Orchestrates trade execution based on signals from the SignalEngine.
 */
@Component
public class PaperTradingEngine implements TradingService {

    private static final Logger logger = LoggerFactory.getLogger(PaperTradingEngine.class);

    // Managers
    private final OrderManager orderManager;
    private final PositionManager positionManager;

    // Portfolio state
    private final Portfolio portfolio;

    // Guards every read-modify-write of portfolio.currentCapital plus its immediate DB
    // persist, so concurrent symbol-processing threads (e.g. JobOrchestratorService
    // closing several SELL signals at once) serialize onto the single shared
    // PaperTradingPortfolioEntity row instead of racing on it — the optimistic-lock
    // retries in OptimisticLockRetryHelper only tolerate occasional contention, not a
    // handful of threads hitting the same row in the same instant.
    private final Object portfolioLock = new Object();

    // Configuration
    private final PaperTradingProperties properties;
    private final BigDecimal commissionRate;
    private final BigDecimal initialCapital;

    // Position counter for unique IDs
    private final AtomicLong positionCounter;
    private final AtomicLong orderCounter;

    // Metrics
    private final TradeMetrics tradeMetrics;

    // DB persistence port (implemented by PaperTradingStateService, which does not depend on the engine)
    private final TradingStatePersistence persistence;

    /**
     * Restores persisted state (capital, open positions, pending orders, position ID
     * counter) once the application is ready.
     */
    @EventListener(ApplicationReadyEvent.class)
    public void initState() {
        restoreState(persistence.loadState());
    }

    private void restoreState(PersistedState state) {
        if (state.currentCapital() != null) {
            portfolio.setCurrentCapital(state.currentCapital());
        }
        if (state.initialCapital() != null) {
            portfolio.setInitialCapital(state.initialCapital());
        }
        for (Position position : state.openPositions()) {
            portfolio.addPosition(position);
            positionManager.getPositions().put(position.positionId(), position);
        }
        for (Order order : state.pendingOrders()) {
            orderManager.getOrders().put(order.getOrderId(), order);
        }
        seedPositionCounter(state.maxPositionIdSuffix());
        logger.info("Seeded position ID counter to {} (max POS_ suffix found in DB)", state.maxPositionIdSuffix());
    }

    /** Persists the current portfolio figures (cash, realized/unrealized P&L, open count). */
    public void savePortfolio() {
        persistence.savePortfolio(new PortfolioState(
            portfolio.getCurrentCapital(),
            portfolio.getInitialCapital(),
            getTotalRealizedPnL(),
            getTotalUnrealizedPnL(),
            getOpenPositionCount()));
    }

    /** Appends a point-in-time portfolio valuation to the snapshot history. */
    public void saveSnapshot() {
        BigDecimal totalValue = portfolio.getTotalValue();
        BigDecimal cash = getCurrentCash();
        persistence.saveSnapshot(new SnapshotState(
            totalValue,
            cash,
            totalValue.subtract(cash),
            getTotalPnL(),
            getReturnPercentage(),
            getOpenPositionCount()));
    }

    private Position findInMemoryPosition(Long databaseId) {
        String positionId = persistence.findPositionId(databaseId)
            .orElseThrow(() -> new IllegalArgumentException("Position not found: " + databaseId));
        Position position = positionManager.getPosition(positionId);
        if (position == null) {
            throw new IllegalArgumentException("Position not found in memory: " + positionId);
        }
        return position;
    }

    /**
     * Creates a new PaperTradingEngine with specified configuration.
     *
     * @param orderManager the order manager
     * @param positionManager the position manager
     * @param properties the paper trading configuration
     * @param tradeMetrics the trade metrics recorder
     * @param persistence the persistence port used to save and restore engine state
     */
    @Autowired
    public PaperTradingEngine(OrderManager orderManager,
                              PositionManager positionManager,
                              PaperTradingProperties properties,
                              TradeMetrics tradeMetrics,
                              TradingStatePersistence persistence) {
        this.orderManager = orderManager;
        this.positionManager = positionManager;
        this.properties = properties;
        this.initialCapital = properties.getInitialBalance();
        this.portfolio = new Portfolio("default-portfolio", this.initialCapital);
        this.commissionRate = BigDecimal.valueOf(0.05); // 5 paise per share
        this.positionCounter = new AtomicLong(0);
        this.orderCounter = new AtomicLong(0);
        this.tradeMetrics = tradeMetrics;
        this.persistence = persistence;
    }

    /**
     * Places a market buy order based on a signal.
     * This is the primary method for initiating trades.
     *
     * @param signal the trading signal
     * @param currentPrice the current market price
     * @return the created order if valid
     * @throws IllegalStateException if position limits are reached
     */
    public Order executeSignal(Signal signal, BigDecimal currentPrice) {
        Order order = queueSignal(signal, currentPrice);
        if (order == null) return null;

        // This method remains the explicit immediate-execution API used by direct callers.
        return executePendingOrder(order.getOrderId(), currentPrice);
    }

    /**
     * Queues a signal without filling it. The pending-order scheduler settles the
     * market order at the next session open.
     */
    @Override
    public Order queueSignal(Signal signal, BigDecimal referencePrice) {
        if (signal == null || !signal.isBuySignal()) {
            logger.debug("Ignoring non-BUY signal: {}", signal);
            return null;
        }

        if (signal.id() != null) {
            String signalId = signal.id().toString();
            Order existing = orderManager.getPendingOrders().stream()
                .filter(candidate -> candidate.getAdditionalProperties() != null
                    && signalId.equals(candidate.getAdditionalProperties().get("signalId")))
                .findFirst().orElse(null);
            if (existing != null) return existing;
        }

        // Calculate position size based on risk
        BigDecimal quantity = calculatePositionSize(signal.symbol(), referencePrice, signal.stopLoss());
        if (quantity == null || quantity.intValue() <= 0
                || !validatePositionCapacity(referencePrice, quantity.intValue())) {
            logger.warn("Cannot execute signal for {}: calculated position capacity exceeded", signal.symbol());
            return null;
        }
        tradeMetrics.recordTradeOpen();
        tradeMetrics.recordSymbolTrade(signal.symbol());

        // Create order
        Order order = orderManager.createBuyOrder(signal.symbol(), quantity.intValue(), referencePrice);

        // Attach signal metadata
        java.util.Map<String, Object> signalProps = new java.util.HashMap<>();
        signalProps.put("signalId", signal.id() != null ? signal.id().toString() : "");
        signalProps.put("signalReason", signal.reasoning() != null ? signal.reasoning() : "");
        signalProps.put("confidence", signal.confidence() != null ? signal.confidence().toString() : "0");
        signalProps.put("stopLoss", signal.stopLoss() != null ? signal.stopLoss().toString() : "0");
        signalProps.put("target", signal.target() != null ? signal.target().toString() : "0");
        signalProps.put("riskReward", signal.riskReward() != null ? signal.riskReward().toString() : "0");
        signalProps.put("entryDate", LocalDate.now().toString());
        order.setAdditionalProperties(signalProps);

        // Persist before the orchestrator marks the source signal processed so a restart cannot
        // lose a queued trade between those two operations.
        persistence.saveOrder(order);

        logger.info("Queued BUY order for {} using reference price {}: signal confidence={}, SL={}, Target={}",
            signal.symbol(), referencePrice, signal.confidence(), signal.stopLoss(), signal.target());
        return order;
    }

    /**
     * Validates if new position can be opened based on capacity constraints.
     *
     * @param entryPrice the intended entry price
     * @param quantity the intended quantity
     * @return true if position can be opened
     */
    public boolean validatePositionCapacity(BigDecimal entryPrice, int quantity) {
        if (positionManager.hasReachedPositionLimit()) {
            logger.warn("Cannot open position: maximum positions ({}) reached", positionManager.getMaxPositions());
            return false;
        }

        BigDecimal positionValue = entryPrice.multiply(BigDecimal.valueOf(quantity));
        BigDecimal currentCapital = portfolio.getCurrentCapital();
        if (currentCapital.compareTo(BigDecimal.ZERO) <= 0) {
            logger.warn("Cannot open position: portfolio capital is depleted ({})", currentCapital);
            return false;
        }
        BigDecimal capitalRatio = positionValue.divide(currentCapital, 4, RoundingMode.HALF_UP);

        BigDecimal maxCapitalRatio = maxCapitalPerPositionDiv100();
        if (capitalRatio.compareTo(maxCapitalRatio) > 0) {
            logger.warn("Cannot open position for {}: position value {} exceeds {}% of capital",
                entryPrice, positionValue, properties.getMaxCapitalPerPosition());
            return false;
        }

        return true;
    }

    /**
     * Calculates position size based on risk parameters.
     * Uses ATR-based position sizing to limit risk to a fixed percentage of capital.
     *
     * @param symbol the trading symbol
     * @param entryPrice the entry price
     * @param stopLoss the stop loss price
     * @return the calculated position size
     */
    public BigDecimal calculatePositionSize(String symbol, BigDecimal entryPrice, BigDecimal stopLoss) {
        // Default position size calculation (simplified)
        // In production, this would use ATR and risk-per-trade parameters

        BigDecimal riskPerTrade = BigDecimal.valueOf(1).divide(BigDecimal.valueOf(100), 4, RoundingMode.HALF_UP); // 1% max risk per trade
        if (stopLoss == null) {
            return BigDecimal.valueOf(100);
        }
        BigDecimal riskPerShare = entryPrice.subtract(stopLoss);
        BigDecimal maxRiskAmount = portfolio.getCurrentCapital().multiply(riskPerTrade);

        if (riskPerShare.compareTo(BigDecimal.ZERO) <= 0) {
            // Default to 100 shares if risk calculation is invalid
            return BigDecimal.valueOf(100);
        }

        BigDecimal quantity = maxRiskAmount.divide(riskPerShare, 0, RoundingMode.DOWN);
        return quantity.max(BigDecimal.valueOf(100)); // Minimum 100 shares
    }

    /**
     * Updates positions with new candle data and checks for SL/TP triggers.
     *
     * @param candleData the OHLCV candle data
     */
    public void updatePositions(OhlcvCandle candleData) {
        List<Position> updatedPositions = positionManager.updatePositionsWithCandleData(
            candleData.symbol(), candleData);

        for (Position position : updatedPositions) {
            logger.debug("Updated position {} for {} at price {}: P&L={}",
                position.positionId(), position.symbol(), position.currentPrice(),
                position.unrealizedPnL());
        }
    }

    /**
     * Checks all open positions against candle data for SL/TP triggers.
     *
     * @param symbol the trading symbol
     * @param candle the candle data
     */
    public void checkPositionTriggers(String symbol, OhlcvCandle candle) {
        List<Position> openPositions = positionManager.getOpenPositions();
        for (Position position : openPositions) {
            if (position.symbol().equals(symbol)) {
                positionManager.checkPositionTriggers(position, candle);
            }
        }
    }

    /**
     * Executes a pending order at the given price.
     *
     * @param orderId the order to execute
     * @param executionPrice the price at which to execute
     * @return the executed order
     */
    public Order executePendingOrder(String orderId, BigDecimal executionPrice) {
        Order order = orderManager.executeOrder(orderId, executionPrice);

        // Create position from filled order
        if (order.getStatus() == OrderStatus.FILLED) {
            synchronized (portfolioLock) {
                Position position = createPositionFromOrder(order);

                // Update portfolio
                updatePortfolioAfterEntry(order);

                logger.info("Position {} created from order {} at {}",
                    position.positionId(), orderId, executionPrice);

                // Persist
                persistence.saveOrder(order);
                Long signalId = signalId(order);
                if (signalId == null) persistence.savePosition(position);
                else persistence.savePosition(position, signalId, calculateCommission(order));
                savePortfolio();
            }
        }

        return order;
    }

    /**
     * Creates a position from an executed order.
     *
     * @param order the executed order
     * @return the created position
     */
    public Position createPositionFromOrder(Order order) {
        if (order.getStatus() != OrderStatus.FILLED) {
            throw new IllegalStateException("Can only create position from FILLED order: " + order.getStatus());
        }

        String positionId = generatePositionId();

        // Extract ATR if available, otherwise use default
        BigDecimal atr = getATRFromOrder(order);

        return positionManager.createPosition(
            positionId,
            order.getSymbol(),
            order.getDirection(),
            order.getQuantity().intValue(),
            order.getPrice(),
            atr,
            order.getAdditionalProperties() != null
                ? (String) order.getAdditionalProperties().get("signalReason")
                : null
        );
    }

    private static Long signalId(Order order) {
        if (order == null || order.getAdditionalProperties() == null) return null;
        Object raw = order.getAdditionalProperties().get("signalId");
        if (raw == null) return null;
        try {
            long value = Long.parseLong(raw.toString());
            return value > 0 ? value : null;
        } catch (NumberFormatException ignored) {
            return null;
        }
    }

    /**
     * Calculates stop loss price based on ATR.
     *
     * @param order the order
     * @return calculated stop loss
     */
    public BigDecimal calculateStopLossFromOrder(Order order) {
        if (order == null || order.getAdditionalProperties() == null) {
            return null;
        }

        // Check if stop loss is provided in signal metadata
        Object stopLossObj = order.getAdditionalProperties().get("stopLoss");
        if (stopLossObj instanceof String) {
            return new BigDecimal((String) stopLossObj);
        }

        // Calculate from ATR if available
        BigDecimal atr = getATRFromOrder(order);
        if (atr != null && atr.compareTo(BigDecimal.ZERO) > 0) {
            return positionManager.calculateStopLoss(order.getPrice(), order.getDirection(), atr);
        }

        return null;
    }

    /**
     * Calculates target price based on risk-reward ratio.
     *
     * @param order the order
     * @return calculated target
     */
    public BigDecimal calculateTargetFromOrder(Order order) {
        if (order == null || order.getAdditionalProperties() == null) {
            return null;
        }

        // Check if target is provided in signal metadata
        Object targetObj = order.getAdditionalProperties().get("target");
        if (targetObj instanceof String) {
            return new BigDecimal((String) targetObj);
        }

        // Calculate from stop loss if available
        BigDecimal stopLoss = calculateStopLossFromOrder(order);
        if (stopLoss != null) {
            return positionManager.calculateTarget(order.getPrice(), stopLoss, order.getDirection());
        }

        return null;
    }

    /**
     * Extracts ATR from order metadata.
     *
     * @param order the order
     * @return ATR value or null
     */
    public BigDecimal getATRFromOrder(Order order) {
        if (order == null || order.getAdditionalProperties() == null) {
            return null;
        }

        Object atrObj = order.getAdditionalProperties().get("atr");
        if (atrObj instanceof String) {
            return new BigDecimal((String) atrObj);
        }

        return null;
    }

    /**
     * Updates portfolio after position entry.
     *
     * @param order the filled order
     */
    private void updatePortfolioAfterEntry(Order order) {
        BigDecimal positionValue = order.getPrice().multiply(order.getQuantity());
        BigDecimal commission = calculateCommission(order);

        // Update portfolio capital
        portfolio.setCurrentCapital(
            portfolio.getCurrentCapital()
                .subtract(positionValue)
                .subtract(commission)
        );
    }

    /**
     * Calculates commission for an order.
     *
     * @param order the order
     * @return calculated commission
     */
    public BigDecimal calculateCommission(Order order) {
        if (order == null) {
            return BigDecimal.ZERO;
        }

        BigDecimal quantity = order.getQuantity() != null ? order.getQuantity() : BigDecimal.ZERO;
        return quantity.multiply(commissionRate);
    }

    @Override
    public BigDecimal calculateEntryCommission(int quantity) {
        return BigDecimal.valueOf(quantity).multiply(commissionRate);
    }

    /**
     * Performs partial exit of a position.
     *
     * @param positionId the position to exit
     * @param exitRatio the ratio to exit (0.5 = 50%)
     * @param exitPrice the exit price
     * @return the updated position
     */
    public Position partialExitPosition(String positionId, BigDecimal exitRatio, BigDecimal exitPrice) {
        synchronized (portfolioLock) {
            // Capture quantity before partial exit (this is the original at call time)
            Position currentPosition = positionManager.getPosition(positionId);
            if (currentPosition == null) {
                throw new IllegalArgumentException("Position not found: " + positionId);
            }
            if (exitRatio == null || exitRatio.compareTo(BigDecimal.ZERO) <= 0
                    || exitRatio.compareTo(BigDecimal.ONE) > 0) {
                throw new IllegalArgumentException("Exit ratio must be between 0 and 1: " + exitRatio);
            }
            BigDecimal currentQty = BigDecimal.valueOf(currentPosition.quantity());
            BigDecimal exitedQuantity = currentQty.multiply(exitRatio)
                .setScale(0, RoundingMode.DOWN);
            if (exitedQuantity.signum() <= 0) {
                throw new IllegalArgumentException("Exit ratio must sell at least one share: " + exitRatio);
            }

            Position position = positionManager.partialExitPosition(positionId, exitRatio, exitPrice);

            // Update portfolio with cash proceeds from exited shares
            BigDecimal exitValue = exitedQuantity.multiply(exitPrice);
            BigDecimal commission = exitedQuantity.multiply(commissionRate);
            portfolio.setCurrentCapital(portfolio.getCurrentCapital().add(exitValue.subtract(commission)));

            logger.info("Partial exit completed for position {}: exited={}, remaining={}",
                positionId, exitedQuantity, position.quantity());

            // Persist
            persistence.savePosition(position);
            savePortfolio();

            return position;
        }
    }

    /**
     * Closes a position by its database ID.
     * Looks up the persisted position from DB to get its positionId string
     * (the in-memory counter may have reset on restart).
     *
     * <p>Intentionally NOT {@code @Transactional}: this method only performs
     * an in-memory lookup before delegating to {@link #closePosition(String, BigDecimal, String)}
     * for the actual (already transactional) close-and-persist logic. Callers
     * (e.g. {@code PositionService}) run their own outer transaction and rely
     * on catching failures from this method without having the outer
     * transaction marked rollback-only — annotating this wrapper would cause
     * any exception thrown here (e.g. an unresolved in-memory position) to be
     * intercepted by Spring's transaction advice and poison the caller's
     * transaction even though the caller catches it.
     *
     * @param positionId the database position ID
     * @return the closed position
     * @throws IllegalArgumentException if position not found (in DB or in memory)
     */
    public Position closePosition(Long positionId) {
        // Load position from DB to get its positionId (counter may have reset)
        Position position = findInMemoryPosition(positionId);

        BigDecimal exitPrice = position.currentPrice();
        String reason = "manual_close";

        closePosition(position.positionId(), exitPrice, reason);
        return position;
    }

    /**
     * Closes a position by its database ID with explicit exit price and reason.
     * Implements TradingService interface method.
     *
     * <p>Intentionally NOT {@code @Transactional} — see {@link #closePosition(Long)}
     * for rationale.
     *
     * @param positionId the database position ID
     * @param exitPrice the exit price
     * @param reason the reason for closing
     * @return the closed position
     * @throws IllegalArgumentException if position not found (in DB or in memory)
     */
    @Override
    public Position closePosition(Long positionId, BigDecimal exitPrice, String reason) {
        Position position = findInMemoryPosition(positionId);

        closePosition(position.positionId(), exitPrice, reason);
        return position;
    }

    /**
     * Closes a position completely.
     *
     * <p>Intentionally NOT {@code @Transactional}: {@link PositionManager} is a
     * pure in-memory store with no JPA/DB participation, so there is no
     * transactional resource here to protect. If this method were annotated
     * (directly or transitively via a transactional {@code PositionManager}
     * method), a failure here (e.g. an unresolved in-memory position lookup)
     * would mark the caller's ambient transaction (e.g.
     * {@code PositionService.closePosition()}) rollback-only — surfacing as an
     * {@code UnexpectedRollbackException} even when the caller catches and
     * handles the failure gracefully.
     *
     * @param positionId the position to close
     * @param exitPrice the exit price
     * @param reason the reason for closing
     */
    public void closePosition(String positionId, BigDecimal exitPrice, String reason) {
        synchronized (portfolioLock) {
            Position position = positionManager.closePosition(positionId, exitPrice, reason);
            tradeMetrics.recordTradeClose(reason);

            // Update portfolio
            BigDecimal exitValue = exitPrice.multiply(BigDecimal.valueOf(position.quantity()));
            BigDecimal commission = calculateCommissionForPosition(position);
            BigDecimal netProceeds = exitValue.subtract(commission);

            portfolio.setCurrentCapital(portfolio.getCurrentCapital().add(netProceeds));

            logger.info("Position {} closed: P&L={}, Reason={}",
                positionId, position.realizedPnL(), reason);

            // Persist
            persistence.closePosition(positionId, position);
            savePortfolio();
        }
    }

    /**
     * Calculates commission for position closure.
     *
     * @param position the position
     * @return commission amount
     */
    private BigDecimal calculateCommissionForPosition(Position position) {
        BigDecimal quantity = BigDecimal.valueOf(position.quantity() != null ? position.quantity() : 0);
        return quantity.multiply(commissionRate);
    }

    /**
     * Generates a unique position ID.
     *
     * @return unique position ID
     */
    private String generatePositionId() {
        return "POS_" + String.format("%08d", positionCounter.incrementAndGet());
    }

    /**
     * Seeds the position ID counter so that subsequently generated IDs never
     * collide with any position (open or closed) already persisted in the
     * database. The counter always starts at 0 on JVM startup (in-memory
     * state), so without this seeding step every restart would reissue IDs
     * from POS_00000001 and silently overwrite unrelated historical rows via
     * {@code PaperTradingStateService.savePosition()}'s upsert-by-positionId
     * logic. Called once from {@link #initState()} with the
     * maximum suffix reported by {@code TradingStatePersistence.loadState()}, before any new position can be created.
     *
     * <p>Only raises the counter — never lowers it — so repeated or
     * out-of-order calls are safe.
     *
     * @param minValue the minimum counter value, typically the max POS_
     *     numeric suffix found in the database
     */
    public void seedPositionCounter(long minValue) {
        positionCounter.updateAndGet(current -> Math.max(current, minValue));
    }

    /**
     * Gets the current portfolio.
     *
     * @return the portfolio
     */
    public Portfolio getPortfolio() {
        return portfolio;
    }

    /**
     * Gets all open positions.
     *
     * @return list of open positions
     */
    public List<Position> getOpenPositions() {
        return positionManager.getOpenPositions();
    }

    /**
     * Finds an open position by symbol.
     *
     * @param symbol the trading symbol
     * @return the position if found, null otherwise
     */
    public Position findOpenPositionBySymbol(String symbol) {
        return positionManager.findOpenPositionBySymbol(symbol);
    }

    /**
     * Gets a specific position by ID.
     *
     * @param positionId the position ID
     * @return the position if found
     */
    public Optional<Position> getPosition(String positionId) {
        return Optional.ofNullable(positionManager.getPosition(positionId));
    }

    /**
     * Gets all closed positions.
     *
     * @return list of closed positions
     */
    public List<Position> getClosedPositions() {
        return positionManager.getClosedPositions();
    }

    /**
     * Gets the total unrealized P&L.
     *
     * @return total unrealized P&L
     */
    public BigDecimal getTotalUnrealizedPnL() {
        return positionManager.getTotalUnrealizedPnL();
    }

    /**
     * Gets the total realized P&L.
     *
     * @return total realized P&L
     */
    public BigDecimal getTotalRealizedPnL() {
        return positionManager.getTotalRealizedPnL();
    }

    /**
     * Gets the total P&L (realized + unrealized).
     *
     * @return total P&L
     */
    public BigDecimal getTotalPnL() {
        return getTotalRealizedPnL().add(getTotalUnrealizedPnL());
    }

    /**
     * Gets the return percentage.
     *
     * @return return percentage
     */
    public BigDecimal getReturnPercentage() {
        if (initialCapital.compareTo(BigDecimal.ZERO) == 0) {
            return BigDecimal.ZERO;
        }
        return getTotalPnL()
            .divide(initialCapital, 4, RoundingMode.HALF_UP)
            .multiply(BigDecimal.valueOf(100));
    }

    /**
     * Gets the current cash balance.
     *
     * @return current cash balance
     */
    public BigDecimal getCurrentCash() {
        return portfolio.getCurrentCapital();
    }

    public BigDecimal getTotalValue() {
        Portfolio p = getPortfolio();
        return p != null ? p.getTotalValue() : null;
    }

    @Override
    public BigDecimal getPortfolioMaxDrawdown() {
        BigDecimal peak = null;
        BigDecimal maxDrawdown = BigDecimal.ZERO;
        List<BigDecimal> snapshots = persistence.getSnapshotTotalValues();
        for (int i = snapshots.size() - 1; i >= 0; i--) {
            BigDecimal value = snapshots.get(i);
            if (value == null || value.signum() <= 0) continue;
            if (peak == null || value.compareTo(peak) > 0) peak = value;
            BigDecimal drawdown = peak.subtract(value)
                .divide(peak, 8, RoundingMode.HALF_UP)
                .multiply(BigDecimal.valueOf(100));
            if (drawdown.compareTo(maxDrawdown) > 0) maxDrawdown = drawdown;
        }
        return peak == null ? null : maxDrawdown.setScale(2, RoundingMode.HALF_UP);
    }

    /**
     * Gets the maximum number of concurrent positions.
     *
     * @return maximum positions
     */
    public int getMaxConcurrentPositions() {
        return properties.getMaxConcurrentPositions();
    }

    /**
     * Gets the maximum capital percentage per position.
     *
     * @return max capital percentage
     */
    public BigDecimal getMaxCapitalPerPosition() {
        return properties.getMaxCapitalPerPosition();
    }

    private BigDecimal maxCapitalPerPositionDiv100() {
        return properties.getMaxCapitalPerPosition().divide(BigDecimal.valueOf(100), 4, RoundingMode.HALF_UP);
    }

    /**
     * Checks if there is capacity for a new position.
     *
     * @param entryPrice the entry price
     * @param quantity the quantity
     * @return true if position can be opened
     */
    public boolean canOpenPosition(BigDecimal entryPrice, int quantity) {
        return validatePositionCapacity(entryPrice, quantity);
    }

    /**
     * Gets the number of open positions.
     *
     * @return position count
     */
    public int getOpenPositionCount() {
        return positionManager.getOpenPositionCount();
    }

    /**
     * Gets the initial capital.
     *
     * @return initial capital
     */
    public BigDecimal getInitialCapital() {
        return initialCapital;
    }

    /**
     * Updates all positions with new candle data from domain object.
     *
     * @param candle the domain candle
     */
    public void updatePositionsFromDomain(OhlcvCandle candle) {
        List<Position> updated = positionManager.updatePositionsWithCandleData(candle.symbol(), candle);
        handleUpdatedPositions(updated);
    }

    /** Updates positions using optional explicit exchange price-band data. */
    public void updatePositionsFromDomain(OhlcvCandle candle, PriceBand priceBand) {
        List<Position> updated = positionManager.updatePositionsWithCandleData(
            candle.symbol(), candle, priceBand);

        handleUpdatedPositions(updated);
    }

    private void handleUpdatedPositions(List<Position> updated) {
        // Check triggers for each updated position
        for (Position position : updated) {
            if (position.status() == PositionStatus.OPEN) {
                // Persist updated price/P&L
                persistence.savePosition(position);
                continue;
            }

            logger.info("Position {} status changed to: {}",
                position.positionId(), position.status());

            // Persist closed position
            synchronized (portfolioLock) {
                persistence.closePosition(position.positionId(), position);
                savePortfolio();
            }
            tradeMetrics.recordTradeClose(position.status().name().toLowerCase());
        }
    }

    /**
     * Gets pending orders.
     *
     * @return list of pending orders
     */
    public List<Order> getPendingOrders() {
        return orderManager.getPendingOrders();
    }

    /**
     * Cancels a pending order.
     *
     * @param orderId the order ID
     * @return true if cancelled
     */
    public boolean cancelOrder(String orderId) {
        return orderManager.cancelOrder(orderId);
    }

    /**
     * Clears all state (for testing).
     */
    public void clearAll() {
        positionManager.clearAllPositions();
        orderManager.clearAllOrders();
        portfolio.setCurrentCapital(initialCapital);
        positionCounter.set(0);
        orderCounter.set(0);
    }
}
