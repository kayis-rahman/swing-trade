package com.swingtrade.strategy;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.util.Set;
import org.springframework.stereotype.Component;

/** Long-only session VWAP pullback/reclaim strategy; VWAP resets on each trading date. */
@Component
public final class IntradayVwapPullbackStrategy implements SignalStrategy {
    public static final String TYPE = "INTRADAY_VWAP_PULLBACK";
    @Override public String type() { return TYPE; }
    @Override public ParamSchema paramSchema() { return IntradayStrategySupport.emptySchema(); }
    @Override public Set<IndicatorKey> requiredIndicators(StrategyParamsView params) { return Set.of(); }
    @Override public int warmupBars(StrategyParamsView params) { return 2; }

    @Override public StrategyDecision evaluateEntry(MarketContext ctx, int index, StrategyParamsView params) {
        MarketContext.View view = ctx.view(index);
        int first = index;
        while (first > 0 && view.date(first - 1).equals(view.date(index))) first--;
        if (index == first) return IntradayStrategySupport.decision(false, "vwapReclaim", "Need a prior session bar", view.close());
        BigDecimal pv = BigDecimal.ZERO;
        BigDecimal volume = BigDecimal.ZERO;
        BigDecimal priorPv = BigDecimal.ZERO;
        BigDecimal priorVolume = BigDecimal.ZERO;
        for (int i = first; i <= index; i++) {
            BigDecimal typical = view.high(i).add(view.low(i)).add(view.close(i)).divide(BigDecimal.valueOf(3), 8, RoundingMode.HALF_UP);
            BigDecimal barVolume = view.volume(i);
            pv = pv.add(typical.multiply(barVolume));
            volume = volume.add(barVolume);
            if (i == index - 1) {
                priorPv = pv;
                priorVolume = volume;
            }
        }
        if (priorVolume.signum() == 0) return IntradayStrategySupport.decision(false, "vwapReclaim", "No prior session volume", view.close());
        BigDecimal priorVwap = priorPv.divide(priorVolume, 8, RoundingMode.HALF_UP);
        BigDecimal vwap = pv.divide(volume, 8, RoundingMode.HALF_UP);
        BigDecimal priorClose = view.close(index - 1);
        boolean reclaim = priorClose.compareTo(priorVwap) < 0 && view.close().compareTo(vwap) >= 0;
        return IntradayStrategySupport.decision(reclaim, "vwapReclaim", "Previous close below and current close at/above session VWAP", view.close());
    }

    @Override public ExitDecision evaluateExit(MarketContext ctx, int index, OpenPosition position, StrategyParamsView params) {
        BigDecimal close = ctx.view(index).close();
        return close.compareTo(position.stopLoss()) <= 0 ? ExitDecision.exit(ExitReason.STOP_LOSS, close, "Intraday stop")
            : close.compareTo(position.target()) >= 0 ? ExitDecision.exit(ExitReason.TARGET_HIT, close, "Intraday target")
            : ExitDecision.hold();
    }
}
