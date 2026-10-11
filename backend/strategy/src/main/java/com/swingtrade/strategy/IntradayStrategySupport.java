package com.swingtrade.strategy;

import com.swingtrade.domain.Signal.SignalType;

import java.math.BigDecimal;
import java.util.List;

final class IntradayStrategySupport {
    private IntradayStrategySupport() {}

    static StrategyDecision decision(boolean buy, String key, String detail, BigDecimal price) {
        List<RuleOutcome> rules = List.of(new RuleOutcome(key, buy, BigDecimal.ONE, true, detail));
        BigDecimal stop = buy ? price.multiply(new BigDecimal("0.99")) : null;
        BigDecimal target = buy ? price.multiply(new BigDecimal("1.02")) : null;
        return new StrategyDecision(buy ? SignalType.BUY : SignalType.HOLD,
            buy ? BigDecimal.ONE : BigDecimal.ZERO, rules, stop, target, detail);
    }

    static ParamSchema emptySchema() { return new ParamSchema(List.of()); }
}
