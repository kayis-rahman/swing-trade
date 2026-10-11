package com.swingtrade.strategy;

import com.swingtrade.data.service.MarketCalendar;
import com.swingtrade.data.service.NseSessionCalendar;
import com.swingtrade.domain.OhlcvCandle;
import org.junit.jupiter.api.Test;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class IntradayRsi2StrategyTest {
    private static final ZoneId IST = ZoneId.of("Asia/Kolkata");
    private static final LocalDate DAY = LocalDate.of(2025, 10, 6);

    @Test
    void entryDecisionUsesOnlyClosesThroughRequestedBar() {
        IntradayRsi2Strategy strategy = new IntradayRsi2Strategy();
        List<OhlcvCandle> ordinaryCandles = List.of(candle("100"), candle("99"), candle("98"), candle("150"));
        List<OhlcvCandle> changedCandles = List.of(candle("100"), candle("99"), candle("98"), candle("50"));
        List<ZonedDateTime> timestamps = List.of("09:15", "09:30", "09:45", "10:00").stream()
            .map(time -> ZonedDateTime.of(DAY, LocalTime.parse(time), IST)).toList();
        MarketContext ordinaryFuture = MarketContext.ofIntraday("TEST", ordinaryCandles, timestamps);
        MarketContext changedFuture = MarketContext.ofIntraday("TEST", changedCandles, timestamps);

        StrategyDecision ordinary = strategy.evaluateEntry(ordinaryFuture, 2, IntradayBacktestEngine.defaultParams());
        StrategyDecision changed = strategy.evaluateEntry(changedFuture, 2, IntradayBacktestEngine.defaultParams());

        assertThat(ordinary.type()).isEqualTo(changed.type());
        assertThat(ordinary.type()).isEqualTo(com.swingtrade.domain.Signal.SignalType.BUY);
    }

    @Test
    void completeSessionCanEvaluateOpenPositionAtFinalBar() {
        MarketCalendar marketCalendar = mock(MarketCalendar.class);
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(true);
        NseSessionCalendar sessions = new NseSessionCalendar(marketCalendar);
        List<IntradayBar> bars = new ArrayList<>();
        for (LocalTime time : sessions.expectedBarStartTimes(DAY)) {
            int index = bars.size();
            String close = index == 0 ? "100" : index == 1 ? "99" : "98";
            BigDecimal value = new BigDecimal(close);
            bars.add(new IntradayBar(ZonedDateTime.of(DAY, time, IST), value,
                value.add(new BigDecimal("0.1")), value.subtract(new BigDecimal("0.1")), value, 1000));
        }
        IntradayBacktestEngine engine = new IntradayBacktestEngine(sessions,
            new IntradayCostModel(BigDecimal.ZERO, BigDecimal.ZERO),
            new IntradayBacktestConfig(new BigDecimal("10000"), 1, 3, new BigDecimal("1000")));

        IntradayBacktestResult result = engine.run("TEST", bars, new IntradayRsi2Strategy(),
            IntradayBacktestEngine.defaultParams());

        assertThat(result.trades()).hasSize(1);
        assertThat(result.trades().getFirst().exitReason()).isEqualTo("FORCED_SQUARE_OFF");
        assertThat(result.trades().getFirst().exitTime()).isEqualTo(bars.getLast().timestamp());
    }

    private static OhlcvCandle candle(String close) {
        BigDecimal value = new BigDecimal(close);
        return OhlcvCandle.of("TEST", DAY, value, value, value, value, 1000L);
    }
}
