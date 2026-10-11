package com.swingtrade.strategy;

import java.math.BigDecimal;
import java.util.Set;
import org.springframework.stereotype.Component;

/** Long-only RSI(2) mean reversion, entering below 10 and exiting on recovery above 60. */
@Component
public final class IntradayRsi2Strategy implements SignalStrategy {
    public static final String TYPE = "INTRADAY_RSI2";
    private static final BigDecimal ENTRY = BigDecimal.TEN;
    private static final BigDecimal EXIT = BigDecimal.valueOf(60);
    @Override public String type() { return TYPE; }
    @Override public ParamSchema paramSchema() { return IntradayStrategySupport.emptySchema(); }
    @Override public Set<IndicatorKey> requiredIndicators(StrategyParamsView params) { return Set.of(); }
    @Override public int warmupBars(StrategyParamsView params) { return 3; }

    @Override public StrategyDecision evaluateEntry(MarketContext ctx, int index, StrategyParamsView params) {
        MarketContext.View view = ctx.view(index);
        int first = index;
        while (first > 0 && view.date(first - 1).equals(view.date(index))) first--;
        if (index - first < 2) return IntradayStrategySupport.decision(false, "rsi2Oversold", "Need two same-session price changes", view.close());
        BigDecimal rsi = sessionRsi2(view, index);
        boolean buy = rsi.compareTo(ENTRY) < 0;
        return IntradayStrategySupport.decision(buy, "rsi2Oversold", "Session RSI(2) below 10", view.close());
    }

    @Override public ExitDecision evaluateExit(MarketContext ctx, int index, OpenPosition position, StrategyParamsView params) {
        MarketContext.View view = ctx.view(index);
        BigDecimal close = view.close();
        if (close.compareTo(position.stopLoss()) <= 0) return ExitDecision.exit(ExitReason.STOP_LOSS, close, "Intraday stop");
        if (close.compareTo(position.target()) >= 0) return ExitDecision.exit(ExitReason.TARGET_HIT, close, "Intraday target");
        int first = index;
        while (first > 0 && view.date(first - 1).equals(view.date(index))) first--;
        return index - first >= 2 && sessionRsi2(view, index).compareTo(EXIT) >= 0
            ? ExitDecision.exit(ExitReason.SIGNAL_EXIT, close, "Session RSI(2) recovered") : ExitDecision.hold();
    }

    private BigDecimal sessionRsi2(MarketContext.View view, int index) {
        BigDecimal gains = BigDecimal.ZERO;
        BigDecimal losses = BigDecimal.ZERO;
        for (int i = index - 1; i < index + 1; i++) {
            BigDecimal change = view.close(i + 1).subtract(view.close(i));
            if (change.signum() > 0) gains = gains.add(change);
            else losses = losses.add(change.abs());
        }
        BigDecimal total = gains.add(losses);
        return total.signum() == 0 ? BigDecimal.valueOf(50)
            : gains.multiply(BigDecimal.valueOf(100)).divide(total, 8, java.math.RoundingMode.HALF_UP);
    }
}
