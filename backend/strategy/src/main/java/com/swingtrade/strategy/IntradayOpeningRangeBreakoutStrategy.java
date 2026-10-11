package com.swingtrade.strategy;

import java.math.BigDecimal;
import java.time.LocalTime;
import java.util.Set;
import org.springframework.stereotype.Component;

/** Long-only opening-range breakout: first two 15-minute bars define the range. */
@Component
public final class IntradayOpeningRangeBreakoutStrategy implements SignalStrategy {
    public static final String TYPE = "INTRADAY_ORB";
    @Override public String type() { return TYPE; }
    @Override public ParamSchema paramSchema() { return IntradayStrategySupport.emptySchema(); }
    @Override public Set<IndicatorKey> requiredIndicators(StrategyParamsView params) { return Set.of(); }
    @Override public int warmupBars(StrategyParamsView params) { return 3; }

    @Override public StrategyDecision evaluateEntry(MarketContext ctx, int index, StrategyParamsView params) {
        MarketContext.View view = ctx.view(index);
        int first = index;
        while (first > 0 && view.date(first - 1).equals(view.date(index))) first--;
        if (index - first < 2) return IntradayStrategySupport.decision(false, "openingRangeBreakout", "Opening range not complete", view.close());
        if (view.sessionTime(first).filter(LocalTime.of(9, 15)::equals).isEmpty()
                || view.sessionTime(first + 1).filter(LocalTime.of(9, 30)::equals).isEmpty()) {
            return IntradayStrategySupport.decision(false, "openingRangeBreakout", "Opening bars unavailable", view.close());
        }
        BigDecimal rangeHigh = view.high(first).max(view.high(first + 1));
        boolean breakout = view.close().compareTo(rangeHigh) > 0;
        return IntradayStrategySupport.decision(breakout, "openingRangeBreakout", "Close above first two 15-minute bars' high", view.close());
    }

    @Override public ExitDecision evaluateExit(MarketContext ctx, int index, OpenPosition position, StrategyParamsView params) {
        BigDecimal close = ctx.view(index).close();
        return close.compareTo(position.stopLoss()) <= 0 ? ExitDecision.exit(ExitReason.STOP_LOSS, close, "Intraday stop")
            : close.compareTo(position.target()) >= 0 ? ExitDecision.exit(ExitReason.TARGET_HIT, close, "Intraday target")
            : ExitDecision.hold();
    }
}
