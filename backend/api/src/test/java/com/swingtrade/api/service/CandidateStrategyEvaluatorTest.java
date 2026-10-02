package com.swingtrade.api.service;

import com.swingtrade.domain.Signal.SignalType;
import com.swingtrade.domain.StrategyConfig;
import com.swingtrade.strategy.BacktestEngine;
import com.swingtrade.strategy.PriceActionSignalEngine;
import com.swingtrade.strategy.ResolvedStrategy;
import com.swingtrade.strategy.SignalResult;
import com.swingtrade.strategy.StrategyResolver;
import com.swingtrade.strategy.TradingStrategy;
import org.junit.jupiter.api.Test;

import java.math.BigDecimal;
import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.ArgumentMatchers.same;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class CandidateStrategyEvaluatorTest {
    @Test
    void evaluatesLegacyConfiguredVariantsInsteadOfSilentlySkippingThem() {
        ActiveVariantService variants = mock(ActiveVariantService.class);
        StrategyResolver resolver = mock(StrategyResolver.class);
        BacktestEngine backtestEngine = mock(BacktestEngine.class);
        PriceActionSignalEngine signalEngine = mock(PriceActionSignalEngine.class);
        TradingStrategy strategy = mock(TradingStrategy.class);
        StrategyConfig config = StrategyConfig.create("price-action", 1, "PRICE_ACTION", Map.of(), Map.of(),
            StrategyConfig.Mode.SHADOW, BigDecimal.valueOf(100_000), true, "stage pilot", LocalDateTime.now());

        when(variants.active()).thenReturn(List.of(config));
        when(resolver.resolve(config)).thenReturn(new ResolvedStrategy.Legacy(strategy));
        when(signalEngine.analyze(eq("INFY"), anyList(), same(strategy))).thenReturn(
            new SignalResult("INFY", LocalDateTime.now().toLocalDate(), SignalType.BUY,
                BigDecimal.ONE, BigDecimal.ONE, BigDecimal.ONE, BigDecimal.ONE, "legacy rules passed"));

        CandidateStrategyEvaluator evaluator = new CandidateStrategyEvaluator(
            variants, resolver, backtestEngine, signalEngine);

        var outcome = evaluator.evaluate("INFY", List.of()).getFirst();

        assertThat(outcome.variantId()).isEqualTo("price-action");
        assertThat(outcome.signalType()).isEqualTo("BUY");
        assertThat(outcome.detail()).isEqualTo("legacy rules passed");
    }
}
