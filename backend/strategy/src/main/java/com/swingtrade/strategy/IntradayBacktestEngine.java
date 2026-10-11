package com.swingtrade.strategy;

import com.swingtrade.data.service.NseSessionCalendar;
import com.swingtrade.domain.Signal.SignalType;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;

/** Session-aware, single-symbol intraday replay. This path never calls the delivery backtest. */
public final class IntradayBacktestEngine {
    private static final LocalTime LAST_ENTRY_BAR = LocalTime.of(15, 0);
    private static final LocalTime SQUARE_OFF_BAR = LocalTime.of(15, 15);
    private final NseSessionCalendar calendar;
    private final IntradayCostModel costs;
    private final IntradayBacktestConfig config;

    public IntradayBacktestEngine(NseSessionCalendar calendar, IntradayCostModel costs,
                                  IntradayBacktestConfig config) {
        this.calendar = calendar;
        this.costs = costs;
        this.config = config;
    }

    public IntradayBacktestResult run(String symbol, List<IntradayBar> input,
                                      SignalStrategy strategy, StrategyParamsView params) {
        return run(symbol, input, strategy, params, true);
    }

    public IntradayBacktestResult runAsOf(String symbol, List<IntradayBar> input,
                                          SignalStrategy strategy, StrategyParamsView params,
                                          LocalDate sessionDate, LocalTime asOf) {
        if (input == null || sessionDate == null || asOf == null
                || !calendar.expectedBarStartTimes(sessionDate).contains(asOf)) {
            throw new IllegalArgumentException("As-of time must be an expected session bar start");
        }
        List<IntradayBar> visibleBars = input.stream()
            .filter(bar -> sessionDate.equals(bar.sessionDate()) && !bar.sessionTime().isAfter(asOf)
                && calendar.expectedBarStartTimes(sessionDate).contains(bar.sessionTime()))
            .map(bar -> bar.sessionTime().equals(asOf)
                ? new IntradayBar(bar.timestamp(), bar.open(), bar.open(), bar.open(), bar.open(), 0)
                : bar)
            .toList();
        if (visibleBars.stream().noneMatch(bar -> bar.sessionTime().equals(asOf))) {
            throw new IllegalArgumentException("The selected as-of bar is missing");
        }
        return run(symbol, visibleBars, strategy, params, false);
    }

    private IntradayBacktestResult run(String symbol, List<IntradayBar> input,
                                       SignalStrategy strategy, StrategyParamsView params,
                                       boolean forceFinalSquareOff) {
        if (symbol == null || symbol.isBlank() || input == null || strategy == null || params == null) {
            throw new IllegalArgumentException("Symbol, bars, strategy and params are required");
        }
        List<IntradayBar> bars = input.stream().filter(bar -> calendar.isTradingSession(bar.sessionDate())
                && calendar.isWithinNormalSession(bar.sessionDate(), bar.sessionTime())
                && calendar.expectedBarStartTimes(bar.sessionDate()).contains(bar.sessionTime()))
            .sorted(Comparator.comparing(IntradayBar::timestamp)).toList();
        if (bars.isEmpty()) return new IntradayBacktestResult(config.initialCapital(), config.initialCapital(),
            BigDecimal.ZERO, List.of(), null, null);
        for (int i = 1; i < bars.size(); i++) {
            if (!bars.get(i).timestamp().isAfter(bars.get(i - 1).timestamp())) {
                throw new IllegalArgumentException("Bars must have unique, increasing timestamps");
            }
        }
        MarketContext context = MarketContext.ofIntraday(symbol, bars.stream().map(bar -> bar.asCandle(symbol)).toList(),
            bars.stream().map(IntradayBar::timestamp).toList());
        List<IntradayBacktestTrade> trades = new ArrayList<>();
        Map<LocalDate, Integer> tradeCounts = new java.util.HashMap<>();
        Map<LocalDate, BigDecimal> dayPnl = new java.util.HashMap<>();
        OpenPosition position = null;
        int entryAt = -1;
        BigDecimal capital = config.initialCapital();
        for (int i = 0; i < bars.size(); i++) {
            IntradayBar bar = bars.get(i);
            LocalDate date = bar.sessionDate();
            if (position != null && position.entryDate().equals(date)) {
                boolean stopHit = bar.low().compareTo(position.stopLoss()) <= 0;
                boolean targetHit = !stopHit && bar.high().compareTo(position.target()) >= 0;
                boolean barrierExit = stopHit || targetHit;
                ExitDecision exit = stopHit
                    ? ExitDecision.exit(ExitReason.STOP_LOSS, position.stopLoss().min(bar.open()), "Intraday stop")
                    : targetHit
                        ? ExitDecision.exit(ExitReason.TARGET_HIT, position.target().max(bar.open()), "Intraday target")
                        : strategy.evaluateExit(context, i, position, params);
                boolean cutoff = !bar.sessionTime().isBefore(NseSessionCalendar.SQUARE_OFF_CUTOFF);
                boolean cutoffBar = bar.sessionTime().equals(SQUARE_OFF_BAR);
                boolean sessionEnd = forceFinalSquareOff && i + 1 == bars.size()
                    || i + 1 < bars.size() && !bars.get(i + 1).sessionDate().equals(date);
                if (exit.exit() || cutoff || cutoffBar || sessionEnd) {
                    // The cutoff is 15:20 inside the 15:15–15:30 bar; open is the reproducible
                    // bar-only proxy for the cutoff fill, avoiding a close after the cutoff.
                    BigDecimal reference = barrierExit ? exit.exitPrice()
                        : cutoff || cutoffBar ? bar.open() : exit.exit() ? exit.exitPrice() : bar.close();
                    String reason = barrierExit ? exit.reason().name()
                        : cutoff || cutoffBar ? "FORCED_SQUARE_OFF"
                        : exit.exit() ? exit.reason().name() : "FORCED_SESSION_END";
                    BigDecimal sell = costs.sellFill(reference);
                    BigDecimal fees = costs.roundTripCost(position.entryPrice(), sell, position.quantity());
                    BigDecimal pnl = sell.subtract(position.entryPrice()).multiply(BigDecimal.valueOf(position.quantity())).subtract(fees);
                    IntradayBacktestTrade trade = new IntradayBacktestTrade(strategy.type(), bars.get(entryAt).timestamp(),
                        bar.timestamp(), position.entryPrice(), sell, position.quantity(), fees, pnl, reason);
                    trades.add(trade);
                    capital = capital.add(pnl);
                    dayPnl.merge(date, pnl, BigDecimal::add);
                    tradeCounts.merge(date, 1, Integer::sum);
                    position = null;
                    entryAt = -1;
                }
            }
            if (position == null && i + 1 < bars.size() && tradeCounts.getOrDefault(date, 0) < config.maxTradesPerDay()
                    && dayPnl.getOrDefault(date, BigDecimal.ZERO).compareTo(config.maxDailyLoss().negate()) > 0
                    && !bar.sessionTime().isAfter(LAST_ENTRY_BAR)
                    && bars.get(i + 1).sessionDate().equals(date)
                    && bars.get(i + 1).sessionTime().isBefore(SQUARE_OFF_BAR)
                    && i + 1 >= strategy.warmupBars(params)) {
                StrategyDecision decision = strategy.evaluateEntry(context, i, params);
                if (decision.type() == SignalType.BUY) {
                    IntradayBar fillBar = bars.get(i + 1);
                    BigDecimal buy = costs.buyFill(fillBar.open());
                    BigDecimal risk = buy.multiply(new BigDecimal("0.01"));
                    position = OpenPosition.open(i + 1, date, buy, buy.subtract(risk),
                        buy.multiply(new BigDecimal("1.02")), config.quantity());
                    entryAt = i + 1;
                }
            }
        }
        return new IntradayBacktestResult(config.initialCapital(), capital,
            capital.subtract(config.initialCapital()), trades, position,
            position == null ? null : bars.get(entryAt).timestamp());
    }

    public static StrategyParamsView defaultParams() { return StrategyParamsView.of(Map.of()); }
}
