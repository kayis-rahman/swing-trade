package com.swingtrade.strategy;

import com.swingtrade.data.service.MarketCalendar;
import com.swingtrade.data.service.NseSessionCalendar;
import org.junit.jupiter.api.Test;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import com.swingtrade.domain.Signal.SignalType;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class IntradayBacktestEngineTest {
    private static final ZoneId IST = ZoneId.of("Asia/Kolkata");
    private static final LocalDate DAY = LocalDate.of(2025, 10, 6);
    private final MarketCalendar marketCalendar = mock(MarketCalendar.class);
    private final NseSessionCalendar sessions = new NseSessionCalendar(marketCalendar);

    @Test
    void orbHandCheckedRoundTripUsesIndependentIntradayCosts() {
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(true);
        List<IntradayBar> bars = List.of(
            bar(DAY, "09:15", "100", "101", "99", "100", 1000),
            bar(DAY, "09:30", "100", "100.5", "99.5", "100", 1000),
            bar(DAY, "09:45", "100", "103", "100", "102", 1000),
            bar(DAY, "10:00", "102", "106", "101", "105", 1000)
        );
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions,
            new IntradayCostModel(new BigDecimal("20"), BigDecimal.ZERO),
            new IntradayBacktestConfig(new BigDecimal("10000"), 10, 3, new BigDecimal("1000")));

        IntradayBacktestResult result = engine.run("TEST", bars,
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams());

        assertThat(result.trades()).hasSize(1);
        IntradayBacktestTrade trade = result.trades().getFirst();
        assertThat(trade.entryPrice()).isEqualByComparingTo("102");
        assertThat(trade.exitPrice()).isEqualByComparingTo("104.04");
        assertThat(trade.costs()).isEqualByComparingTo("47.57");
        assertThat(trade.pnl()).isEqualByComparingTo("-27.17");
        assertThat(result.netPnl()).isEqualByComparingTo("-27.17");
    }

    @Test
    void holidayAndMuhuratOutOfBandBarsAreNotTraded() {
        LocalDate holiday = LocalDate.of(2025, 10, 2);
        LocalDate muhurat = LocalDate.of(2025, 10, 21);
        when(marketCalendar.isNseTradingSession(holiday)).thenReturn(false);
        when(marketCalendar.isNseTradingSession(muhurat)).thenReturn(true);
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions, new IntradayCostModel(),
            IntradayBacktestConfig.defaults());
        List<IntradayBar> bars = List.of(
            bar(holiday, "09:15", "100", "102", "99", "101", 1000),
            bar(muhurat, "18:00", "100", "102", "99", "101", 1000)
        );

        IntradayBacktestResult result = engine.run("TEST", bars,
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams());

        assertThat(result.trades()).isEmpty();
        assertThat(result.netPnl()).isZero();
    }

    @Test
    void forcedSquareOffUsesTheCutoffBarOpenAndNeverCarriesOvernight() {
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(true);
        List<IntradayBar> bars = List.of(
            bar(DAY, "09:15", "100", "101", "99", "100", 1000),
            bar(DAY, "09:30", "100", "100.5", "99.5", "100", 1000),
            bar(DAY, "09:45", "100", "103", "100", "102", 1000),
            bar(DAY, "15:15", "103", "104", "102", "103.5", 1000)
        );
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions,
            new IntradayCostModel(BigDecimal.ZERO, BigDecimal.ZERO),
            new IntradayBacktestConfig(new BigDecimal("10000"), 1, 3, new BigDecimal("1000")));

        IntradayBacktestResult result = engine.run("TEST", bars,
            new IntradayOpeningRangeBreakoutStrategy(), IntradayBacktestEngine.defaultParams());

        assertThat(result.trades()).hasSize(1);
        assertThat(result.trades().getFirst().exitReason()).isEqualTo("FORCED_SQUARE_OFF");
        assertThat(result.trades().getFirst().exitPrice()).isEqualByComparingTo("103");
    }

    @Test
    void barrierExitsUseIntrabarExtremesAndPreferStopWhenBothAreHit() {
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(true);

        IntradayBacktestTrade stop = tradeForExitBar("101", "98", "100");
        IntradayBacktestTrade target = tradeForExitBar("103", "99.5", "100");
        IntradayBacktestTrade both = tradeForExitBar("103", "98", "100");

        assertThat(stop.exitReason()).isEqualTo("STOP_LOSS");
        assertThat(stop.exitPrice()).isEqualByComparingTo("99");
        assertThat(target.exitReason()).isEqualTo("TARGET_HIT");
        assertThat(target.exitPrice()).isEqualByComparingTo("102");
        assertThat(both.exitReason()).isEqualTo("STOP_LOSS");
        assertThat(both.exitPrice()).isEqualByComparingTo("99");
    }

    private IntradayBacktestTrade tradeForExitBar(String high, String low, String close) {
        List<IntradayBar> bars = List.of(
            bar(DAY, "09:15", "100", "100.5", "99.5", "100", 1000),
            bar(DAY, "09:30", "100", high, low, close, 1000)
        );
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions,
            new IntradayCostModel(BigDecimal.ZERO, BigDecimal.ZERO),
            new IntradayBacktestConfig(new BigDecimal("10000"), 1, 3, new BigDecimal("1000")));
        return engine.run("TEST", bars, new AlwaysBuyAndHoldStrategy(), IntradayBacktestEngine.defaultParams())
            .trades().getFirst();
    }

    @Test
    void perDayTradeLimitStopsFurtherEntries() {
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(true);
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions,
            new IntradayCostModel(BigDecimal.ZERO, BigDecimal.ZERO),
            new IntradayBacktestConfig(new BigDecimal("10000"), 1, 1, new BigDecimal("1000")));
        IntradayBacktestResult result = engine.run("TEST", repeatedBars(DAY, new BigDecimal("100"), new BigDecimal("102")),
            new AlwaysBuyOneBarStrategy(), IntradayBacktestEngine.defaultParams());
        assertThat(result.trades()).hasSize(1);
    }

    @Test
    void perDayLossLimitStopsFurtherEntries() {
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(true);
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions,
            new IntradayCostModel(BigDecimal.ZERO, BigDecimal.ZERO),
            new IntradayBacktestConfig(new BigDecimal("10000"), 1, 5, new BigDecimal("1")));
        IntradayBacktestResult result = engine.run("TEST", repeatedBars(DAY, new BigDecimal("100"), new BigDecimal("99")),
            new AlwaysBuyOneBarStrategy(), IntradayBacktestEngine.defaultParams());
        assertThat(result.trades()).hasSize(1);
        assertThat(result.netPnl()).isNegative();
    }

    @Test
    void dailyMarketContextStillKeepsOneBarPerDateAndDailyViews() {
        var context = MarketContext.of("TEST", List.of(
            com.swingtrade.domain.OhlcvCandle.of("TEST", DAY, new BigDecimal("10"), new BigDecimal("11"),
                new BigDecimal("9"), new BigDecimal("10.5"), 100L),
            com.swingtrade.domain.OhlcvCandle.of("TEST", DAY.plusDays(1), new BigDecimal("10.5"), new BigDecimal("12"),
                new BigDecimal("10"), new BigDecimal("11.5"), 200L)));
        assertThat(context.barCount()).isEqualTo(2);
        assertThat(context.view(1).date()).isEqualTo(DAY.plusDays(1));
        assertThat(context.view(0).close()).isEqualByComparingTo("10.5");
    }

    private static List<IntradayBar> repeatedBars(LocalDate date, BigDecimal open, BigDecimal close) {
        List<IntradayBar> bars = new ArrayList<>();
        for (int i = 0; i < 10; i++) {
            LocalTime time = LocalTime.of(9, 15).plusMinutes(i * 15L);
            bars.add(new IntradayBar(ZonedDateTime.of(date, time, IST), open,
                open.max(close).add(BigDecimal.ONE), open.min(close).subtract(BigDecimal.ONE), close, 1000));
        }
        return bars;
    }

    private static final class AlwaysBuyOneBarStrategy implements SignalStrategy {
        @Override public String type() { return "TEST_ALWAYS_BUY"; }
        @Override public ParamSchema paramSchema() { return new ParamSchema(List.of()); }
        @Override public Set<IndicatorKey> requiredIndicators(StrategyParamsView params) { return Set.of(); }
        @Override public int warmupBars(StrategyParamsView params) { return 1; }
        @Override public StrategyDecision evaluateEntry(MarketContext ctx, int index, StrategyParamsView params) {
            return new StrategyDecision(SignalType.BUY, BigDecimal.ONE, List.of(), null, null, "fixture");
        }
        @Override public ExitDecision evaluateExit(MarketContext ctx, int index, OpenPosition position, StrategyParamsView params) {
            return index > position.entryIndex()
                ? ExitDecision.exit(ExitReason.SIGNAL_EXIT, ctx.view(index).close(), "fixture exit")
                : ExitDecision.hold();
        }
    }

    private static final class AlwaysBuyAndHoldStrategy implements SignalStrategy {
        @Override public String type() { return "TEST_ALWAYS_BUY_HOLD"; }
        @Override public ParamSchema paramSchema() { return new ParamSchema(List.of()); }
        @Override public Set<IndicatorKey> requiredIndicators(StrategyParamsView params) { return Set.of(); }
        @Override public int warmupBars(StrategyParamsView params) { return 1; }
        @Override public StrategyDecision evaluateEntry(MarketContext ctx, int index, StrategyParamsView params) {
            return new StrategyDecision(SignalType.BUY, BigDecimal.ONE, List.of(), null, null, "fixture");
        }
        @Override public ExitDecision evaluateExit(MarketContext ctx, int index, OpenPosition position, StrategyParamsView params) {
            return ExitDecision.hold();
        }
    }

    private static IntradayBar bar(LocalDate date, String time, String open, String high,
                                   String low, String close, long volume) {
        return new IntradayBar(ZonedDateTime.of(date, LocalTime.parse(time), IST), new BigDecimal(open),
            new BigDecimal(high), new BigDecimal(low), new BigDecimal(close), volume);
    }
}
