package com.swingtrade.broker.service.intraday;

import com.swingtrade.broker.risk.KillSwitchService;
import com.swingtrade.data.service.MarketCalendar;
import com.swingtrade.data.service.NseSessionCalendar;
import com.swingtrade.domain.Signal.SignalType;
import com.swingtrade.strategy.ExitDecision;
import com.swingtrade.strategy.ExitReason;
import com.swingtrade.strategy.IndicatorKey;
import com.swingtrade.strategy.IntradayBacktestConfig;
import com.swingtrade.strategy.IntradayBacktestEngine;
import com.swingtrade.strategy.IntradayBar;
import com.swingtrade.strategy.IntradayOpeningRangeBreakoutStrategy;
import com.swingtrade.strategy.ParamSchema;
import com.swingtrade.strategy.RuleOutcome;
import com.swingtrade.strategy.SignalStrategy;
import com.swingtrade.strategy.StrategyDecision;
import com.swingtrade.strategy.StrategyParamsView;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.UncheckedIOException;
import java.math.BigDecimal;
import java.nio.charset.StandardCharsets;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.List;
import java.util.Set;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class IntradayPaperExecutionServiceTest {
    private static final ZoneId IST = ZoneId.of("Asia/Kolkata");
    private static final LocalDate DAY = LocalDate.of(2026, 10, 9);

    private KillSwitchService killSwitch;
    private NseSessionCalendar sessions;
    private IntradayPaperExecutionService service;

    @BeforeEach
    void setUp() {
        MarketCalendar marketCalendar = mock(MarketCalendar.class);
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(true);
        sessions = new NseSessionCalendar(marketCalendar);
        killSwitch = mock(KillSwitchService.class);
        when(killSwitch.isActive()).thenReturn(false);
        service = new IntradayPaperExecutionService(sessions, killSwitch);
    }

    @Test
    void runsOneSessionToClosedPaperRecordsWithoutAnyLiveBrokerCalls() {
        IntradayPaperSessionResult result = service.runSession("TCS", DAY, fixtureBars(),
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams(), config(1, "1000"));

        assertThat(result.executionMode()).isEqualTo("PAPER_SIMULATION");
        assertThat(result.status()).isEqualTo("COMPLETED");
        assertThat(result.expectedBars()).isEqualTo(25);
        assertThat(result.receivedBars()).isEqualTo(25);
        assertThat(result.liveBrokerCalls()).isZero();
        assertThat(result.trades()).hasSize(1);
        assertThat(result.trades().getFirst().exitReason()).isEqualTo("TARGET_HIT");
        assertThat(result.trades().getFirst().costs()).isEqualByComparingTo("47.57");
        assertThat(result.trades().getFirst().pnl()).isEqualByComparingTo("-27.6802601000");
        assertThat(result.orders()).hasSize(2).allSatisfy(order -> {
            assertThat(order.status()).isEqualTo("FILLED");
            assertThat(order.executionMode()).isEqualTo("PAPER_SIMULATION");
        });
        assertThat(result.orders()).extracting(IntradayPaperSessionResult.PaperOrderRecord::side)
            .containsExactly("BUY", "SELL");
        assertThat(result.orders().getFirst().fillPrice()).isEqualByComparingTo("102.051");
        assertThat(result.orders().getLast().fillPrice()).isEqualByComparingTo("104.0399739900");
        assertThat(result.positions()).singleElement().satisfies(position -> {
            assertThat(position.status()).isEqualTo("CLOSED");
            assertThat(position.carriedOvernight()).isFalse();
            assertThat(position.entryTime().toLocalDate()).isEqualTo(DAY);
            assertThat(position.exitTime().toLocalDate()).isEqualTo(DAY);
            assertThat(position.entryOrderId()).isEqualTo(result.orders().getFirst().orderId());
            assertThat(position.exitOrderId()).isEqualTo(result.orders().getLast().orderId());
        });
        assertTradeLinks(result);
        assertThat(result.maxConcurrentPositions()).isEqualTo(1);
        assertThat(result.openPositionsAtEnd()).isZero();
        assertThat(result.netPnl()).isEqualByComparingTo(result.trades().getFirst().pnl());
    }

    @Test
    void forcesOpenPositionFlatAtTheIntradaySquareOffBar() {
        IntradayPaperSessionResult result = service.runSession("TCS", DAY, completeSession(squareOffBars()),
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams(), config(1, "1000"));

        assertThat(result.trades()).hasSize(1);
        assertThat(result.trades().getFirst().exitReason()).isEqualTo("FORCED_SQUARE_OFF");
        assertThat(result.positions()).singleElement().satisfies(position -> {
            assertThat(position.status()).isEqualTo("CLOSED");
            assertThat(position.carriedOvernight()).isFalse();
            assertThat(position.exitTime().toLocalTime()).isEqualTo(LocalTime.of(15, 15));
        });
        assertTradeLinks(result);
        assertThat(result.openPositionsAtEnd()).isZero();
        assertThat(result.maxConcurrentPositions()).isEqualTo(1);
    }

    @Test
    void killSwitchBlocksTheSessionBeforeAnyPaperOrderIsCreated() {
        when(killSwitch.isActive()).thenReturn(true);

        IntradayPaperSessionResult result = service.runSession("TCS", DAY, fixtureBars(),
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams(), config(3, "1000"));

        assertThat(result.status()).isEqualTo("BLOCKED_KILL_SWITCH");
        assertThat(result.killSwitchActive()).isTrue();
        assertThat(result.orders()).isEmpty();
        assertThat(result.positions()).isEmpty();
        assertThat(result.liveBrokerCalls()).isZero();
        assertThat(result.openPositionsAtEnd()).isZero();
    }

    @Test
    void dailyLossAndTradeCountLimitsConstrainThePaperLedger() {
        List<IntradayBar> bars = completeSession(List.of(
            barAt("09:15", "100", "100", "100", "100"),
            barAt("09:30", "99", "99", "98", "98"),
            barAt("09:45", "98", "98", "97", "97"),
            barAt("10:00", "97", "97", "96", "96"),
            barAt("10:15", "96", "96", "95", "95")
        ));
        IntradayPaperSessionResult lossLimited = service.runSession("TCS", DAY, bars,
            new LossAfterEachBar(), StrategyParamsView.of(java.util.Map.of()), config(5, "1"));
        IntradayPaperSessionResult tradeLimited = service.runSession("TCS", DAY, bars,
            new LossAfterEachBar(), StrategyParamsView.of(java.util.Map.of()), config(1, "1000"));

        assertThat(lossLimited.trades()).hasSize(1);
        assertThat(lossLimited.netPnl()).isNegative();
        assertThat(tradeLimited.trades()).hasSize(1);
        assertThat(tradeLimited.orders()).hasSize(2);
        assertTradeLinks(lossLimited);
        assertTradeLinks(tradeLimited);
    }

    @Test
    void partialSessionReturnsIncompleteWithoutSimulatedLedgerOrForcedExit() {
        IntradayPaperSessionResult result = service.runSession("TCS", DAY, breakoutBars(),
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams(), config(3, "1000"));

        assertThat(result.status()).isEqualTo("INCOMPLETE_DATA");
        assertThat(result.expectedBars()).isEqualTo(25);
        assertThat(result.receivedBars()).isEqualTo(4);
        assertThat(result.orders()).isEmpty();
        assertThat(result.positions()).isEmpty();
        assertThat(result.trades()).isEmpty();
        assertThat(result.netPnl()).isZero();
        assertThat(result.finalCapital()).isEqualByComparingTo(result.initialCapital());
        assertThat(result.openPositionsAtEnd()).isZero();
    }

    @Test
    void emptySessionReturnsIncompleteData() {
        IntradayPaperSessionResult result = service.runSession("TCS", DAY, List.of(),
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams(), config(3, "1000"));

        assertThat(result.status()).isEqualTo("INCOMPLETE_DATA");
        assertThat(result.orders()).isEmpty();
        assertThat(result.positions()).isEmpty();
        assertThat(result.trades()).isEmpty();
    }

    @Test
    void rejectsBarsSpanningMultipleSessionDates() {
        List<IntradayBar> bars = List.of(bar("100", "100"), new IntradayBar(
            ZonedDateTime.of(DAY.plusDays(1), LocalTime.of(9, 15), IST), bd("100"), bd("101"),
            bd("99"), bd("100"), 1_000));

        assertThatThrownBy(() -> service.runSession("TCS", DAY, bars,
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams(), config(3, "1000")))
            .isInstanceOf(IllegalArgumentException.class)
            .hasMessageContaining("one session date");
    }

    private static List<IntradayBar> fixtureBars() {
        String resource = "/com/swingtrade/broker/intraday/tcs-2026-10-09-15m.csv";
        try (var stream = IntradayPaperExecutionServiceTest.class.getResourceAsStream(resource)) {
            if (stream == null) throw new IllegalStateException("Missing intraday acceptance fixture: " + resource);
            try (var reader = new BufferedReader(new InputStreamReader(stream, StandardCharsets.UTF_8))) {
                return reader.lines().skip(1).map(line -> {
                    String[] fields = line.split(",");
                    return new IntradayBar(ZonedDateTime.of(LocalDate.parse(fields[0]),
                        LocalTime.parse(fields[1]), IST), bd(fields[2]), bd(fields[3]), bd(fields[4]),
                        bd(fields[5]), Long.parseLong(fields[6]));
                }).toList();
            }
        } catch (IOException e) {
            throw new UncheckedIOException(e);
        }
    }

    private static void assertTradeLinks(IntradayPaperSessionResult result) {
        assertThat(result.positions()).hasSize(result.trades().size());
        assertThat(result.orders()).hasSize(result.trades().size() * 2);
        Set<String> orderIds = new java.util.HashSet<>();
        result.orders().forEach(order -> assertThat(orderIds.add(order.orderId())).isTrue());
        for (var trade : result.trades()) {
            var positions = result.positions().stream()
                .filter(position -> position.positionId().equals(trade.positionId())).toList();
            assertThat(positions).hasSize(1);
            var position = positions.getFirst();
            assertThat(position.entryOrderId()).isEqualTo(trade.entryOrderId());
            assertThat(position.exitOrderId()).isEqualTo(trade.exitOrderId());
            var entryOrders = result.orders().stream()
                .filter(order -> order.orderId().equals(trade.entryOrderId())).toList();
            var exitOrders = result.orders().stream()
                .filter(order -> order.orderId().equals(trade.exitOrderId())).toList();
            assertThat(entryOrders).singleElement().satisfies(order -> assertThat(order.side()).isEqualTo("BUY"));
            assertThat(exitOrders).singleElement().satisfies(order -> assertThat(order.side()).isEqualTo("SELL"));
            assertThat(position.realizedPnl()).isEqualByComparingTo(trade.pnl());
        }
    }

    private static List<IntradayBar> squareOffBars() {
        java.util.ArrayList<IntradayBar> bars = new java.util.ArrayList<>();
        bars.add(barAt("09:15", "100", "101", "99", "100"));
        bars.add(barAt("09:30", "100", "100.5", "99.5", "100"));
        bars.add(barAt("09:45", "100", "102", "100", "101.5"));
        for (LocalTime time = LocalTime.of(10, 0); !time.isAfter(LocalTime.of(15, 15)); time = time.plusMinutes(15)) {
            bars.add(new IntradayBar(ZonedDateTime.of(DAY, time, IST), bd("101.5"), bd("101.9"),
                bd("101.3"), bd("101.6"), 1_000));
        }
        return List.copyOf(bars);
    }

    private List<IntradayBar> completeSession(List<IntradayBar> supplied) {
        java.util.Map<LocalTime, IntradayBar> byTime = new java.util.HashMap<>();
        supplied.forEach(bar -> byTime.put(bar.sessionTime(), bar));
        return sessions.expectedBarStartTimes(DAY).stream()
            .map(time -> byTime.getOrDefault(time, barAt(time.toString(), "100", "100", "100", "100")))
            .toList();
    }

    private static List<IntradayBar> breakoutBars() {
        return List.of(
            barAt("09:15", "100", "101", "99", "100"),
            barAt("09:30", "100", "100.5", "99.5", "100"),
            barAt("09:45", "100", "103", "100", "102"),
            barAt("10:00", "102", "106", "102", "105")
        );
    }

    private static IntradayBacktestConfig config(int maxTrades, String maxLoss) {
        return new IntradayBacktestConfig(new BigDecimal("10000"), 10, maxTrades, new BigDecimal(maxLoss));
    }

    private static IntradayBar bar(String open, String close) {
        return barAt("09:15", open, open, close, close);
    }

    private static IntradayBar barAt(String time, String open, String high, String low, String close) {
        return new IntradayBar(ZonedDateTime.of(DAY, LocalTime.parse(time), IST), bd(open), bd(high),
            bd(low), bd(close), 1_000);
    }

    private static BigDecimal bd(String value) { return new BigDecimal(value); }

    private static final class LossAfterEachBar implements SignalStrategy {
        @Override public String type() { return "TEST_LOSS_AFTER_EACH_BAR"; }
        @Override public ParamSchema paramSchema() { return new ParamSchema(List.of()); }
        @Override public Set<IndicatorKey> requiredIndicators(StrategyParamsView params) { return Set.of(); }
        @Override public int warmupBars(StrategyParamsView params) { return 0; }
        @Override public StrategyDecision evaluateEntry(com.swingtrade.strategy.MarketContext context, int index,
                                                        StrategyParamsView params) {
            return new StrategyDecision(SignalType.BUY, BigDecimal.ONE,
                List.of(new RuleOutcome("alwaysBuy", true, BigDecimal.ONE, true, "fixture")),
                bd("1"), bd("2"), "fixture");
        }
        @Override public ExitDecision evaluateExit(com.swingtrade.strategy.MarketContext context, int index,
                                                   com.swingtrade.strategy.OpenPosition position,
                                                   StrategyParamsView params) {
            return ExitDecision.exit(ExitReason.SIGNAL_EXIT, context.view(index).close(), "fixture loss");
        }
    }
}
