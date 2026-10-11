package com.swingtrade.broker.service.intraday;

import com.swingtrade.broker.risk.KillSwitchService;
import com.swingtrade.data.service.NseSessionCalendar;
import com.swingtrade.strategy.IntradayBacktestConfig;
import com.swingtrade.strategy.IntradayBacktestEngine;
import com.swingtrade.strategy.IntradayBacktestResult;
import com.swingtrade.strategy.IntradayBacktestTrade;
import com.swingtrade.strategy.IntradayBar;
import com.swingtrade.strategy.IntradayCostModel;
import com.swingtrade.strategy.SignalStrategy;
import com.swingtrade.strategy.StrategyParamsView;
import org.springframework.stereotype.Service;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.UUID;

/** Executes an intraday session through deterministic bar fills only; it has no broker client. */
@Service
public class IntradayPaperExecutionService {
    public static final String EXECUTION_MODE = "PAPER_SIMULATION";
    private static final String FILLED = "FILLED";

    private final NseSessionCalendar sessions;
    private final KillSwitchService killSwitch;

    public IntradayPaperExecutionService(NseSessionCalendar sessions, KillSwitchService killSwitch) {
        this.sessions = sessions;
        this.killSwitch = killSwitch;
    }

    public IntradayPaperSessionResult runSession(String symbol, LocalDate sessionDate,
                                                  List<IntradayBar> bars, SignalStrategy strategy,
                                                  StrategyParamsView params,
                                                  IntradayBacktestConfig config) {
        return runSession(symbol, sessionDate, bars, strategy, params, config, null);
    }

    public IntradayPaperSessionResult runSession(String symbol, LocalDate sessionDate,
                                                  List<IntradayBar> bars, SignalStrategy strategy,
                                                  StrategyParamsView params,
                                                  IntradayBacktestConfig config, LocalTime requestedAsOf) {
        if (symbol == null || symbol.isBlank() || sessionDate == null || bars == null
                || strategy == null || params == null || config == null) {
            throw new IllegalArgumentException("Symbol, session date, bars, strategy, params and risk config are required");
        }
        if (bars.stream().anyMatch(bar -> !sessionDate.equals(bar.sessionDate()))) {
            throw new IllegalArgumentException("Paper intraday execution accepts one session date per run");
        }
        String runId = UUID.randomUUID().toString();
        if (killSwitch.isActive()) {
            return emptyResult(runId, "BLOCKED_KILL_SWITCH", symbol, sessionDate, strategy.type(), true,
                0, 0, config);
        }
        if (!sessions.isTradingSession(sessionDate)) {
            throw new IllegalArgumentException("The requested date is not an NSE trading session");
        }
        Set<java.time.LocalTime> providedBarStarts = new HashSet<>();
        bars.forEach(bar -> providedBarStarts.add(bar.sessionTime()));
        Set<java.time.LocalTime> expectedBarStarts = new HashSet<>(sessions.expectedBarStartTimes(sessionDate));
        int receivedExpectedBars = (int) providedBarStarts.stream().filter(expectedBarStarts::contains).count();
        if (!providedBarStarts.containsAll(expectedBarStarts)) {
            return emptyResult(runId, "INCOMPLETE_DATA", symbol, sessionDate, strategy.type(), false,
                expectedBarStarts.size(), receivedExpectedBars, config);
        }

        List<LocalTime> sessionTimes = sessions.expectedBarStartTimes(sessionDate);
        LocalTime asOf = requestedAsOf == null ? sessionTimes.getLast() : requestedAsOf;
        if (!expectedBarStarts.contains(asOf)) {
            throw new IllegalArgumentException("As-of time must be an expected NSE session bar start");
        }

        IntradayCostModel costModel = new IntradayCostModel();
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions, costModel, config);
        IntradayBacktestResult replay = requestedAsOf == null
            ? engine.run(symbol, bars, strategy, params)
            : engine.runAsOf(symbol, bars, strategy, params, sessionDate, asOf);
        List<IntradayPaperSessionResult.PaperOrderRecord> orders = new ArrayList<>(replay.trades().size() * 2);
        List<IntradayPaperSessionResult.PaperPositionRecord> positions = new ArrayList<>(replay.trades().size() + 1);
        List<IntradayPaperSessionResult.PaperTradeRecord> paperTrades = new ArrayList<>(replay.trades().size());
        int maxConcurrent = 0;
        BigDecimal realizedPnl = BigDecimal.ZERO;
        BigDecimal unrealizedPnl = BigDecimal.ZERO;
        for (int index = 0; index < replay.trades().size(); index++) {
            IntradayBacktestTrade trade = replay.trades().get(index);
            String positionId = runId + "-position-" + (index + 1);
            String entryOrderId = runId + "-order-" + (index * 2 + 1);
            String exitOrderId = runId + "-order-" + (index * 2 + 2);
            orders.add(new IntradayPaperSessionResult.PaperOrderRecord(entryOrderId, symbol, "BUY",
                trade.entryTime(), trade.entryPrice(), trade.quantity(), FILLED, EXECUTION_MODE, "NEXT_BAR_OPEN"));
            orders.add(new IntradayPaperSessionResult.PaperOrderRecord(exitOrderId, symbol, "SELL",
                trade.exitTime(), trade.exitPrice(), trade.quantity(), FILLED, EXECUTION_MODE, trade.exitReason()));
            positions.add(new IntradayPaperSessionResult.PaperPositionRecord(positionId, symbol,
                entryOrderId, exitOrderId, trade.entryTime(), trade.exitTime(), trade.entryPrice(),
                trade.exitPrice(), trade.exitPrice(), trade.quantity(), "CLOSED", trade.costs(), trade.pnl(),
                BigDecimal.ZERO, trade.exitReason(), false));
            paperTrades.add(new IntradayPaperSessionResult.PaperTradeRecord(positionId, entryOrderId,
                exitOrderId, trade.strategy(), trade.entryTime(), trade.exitTime(), trade.entryPrice(),
                trade.exitPrice(), trade.quantity(), trade.costs(), trade.pnl(), trade.exitReason()));
            realizedPnl = realizedPnl.add(trade.pnl());
            maxConcurrent = 1;
        }
        if (replay.openPosition() != null) {
            var open = replay.openPosition();
            String positionId = runId + "-position-" + (replay.trades().size() + 1);
            String entryOrderId = runId + "-order-" + (replay.trades().size() * 2 + 1);
            ZonedDateTime entryTime = replay.openPositionEntryTime();
            BigDecimal currentPrice = bars.stream().filter(bar -> bar.sessionTime().equals(asOf))
                .findFirst().orElseThrow().open();
            BigDecimal estimatedSellFill = costModel.sellFill(currentPrice);
            BigDecimal estimatedCosts = costModel.roundTripCost(open.entryPrice(), estimatedSellFill, open.quantity());
            BigDecimal openPnl = estimatedSellFill.subtract(open.entryPrice())
                .multiply(BigDecimal.valueOf(open.quantity())).subtract(estimatedCosts);
            orders.add(new IntradayPaperSessionResult.PaperOrderRecord(entryOrderId, symbol, "BUY",
                entryTime, open.entryPrice(), open.quantity(), FILLED, EXECUTION_MODE, "NEXT_BAR_OPEN"));
            positions.add(new IntradayPaperSessionResult.PaperPositionRecord(positionId, symbol,
                entryOrderId, null, entryTime, null, open.entryPrice(), null, currentPrice,
                open.quantity(), "OPEN", estimatedCosts, BigDecimal.ZERO, openPnl, "OPEN", false));
            unrealizedPnl = openPnl;
            maxConcurrent = 1;
        }
        return new IntradayPaperSessionResult(runId, EXECUTION_MODE, "COMPLETED", symbol, sessionDate,
            expectedBarStarts.size(), receivedExpectedBars, strategy.type(), false,
            maxConcurrent, (int) positions.stream().filter(position -> position.status().equals("OPEN")).count(), 0,
            replay.initialCapital(), replay.initialCapital().add(realizedPnl).add(unrealizedPnl),
            realizedPnl.add(unrealizedPnl), orders, positions, paperTrades);
    }

    private static IntradayPaperSessionResult emptyResult(String runId, String status, String symbol,
                                                           LocalDate sessionDate, String strategy,
                                                           boolean killSwitchActive, int expectedBars,
                                                           int receivedBars, IntradayBacktestConfig config) {
        return new IntradayPaperSessionResult(runId, EXECUTION_MODE, status, symbol, sessionDate,
            expectedBars, receivedBars, strategy, killSwitchActive, 0, 0, 0,
            config.initialCapital(), config.initialCapital(),
            BigDecimal.ZERO, List.of(), List.of(), List.of());
    }
}
