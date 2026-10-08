package com.swingtrade.api.service;

import com.swingtrade.data.entity.CandidateScanResultEntity;
import com.swingtrade.data.entity.CandidateScanRunEntity;
import com.swingtrade.data.repository.CandidateHistoryEligibilityRepository;
import com.swingtrade.data.repository.CandidateScanResultRepository;
import com.swingtrade.data.repository.CandidateScanRunRepository;
import com.swingtrade.data.repository.FyersSymbolRepository;
import com.swingtrade.data.service.AppSettingsService;
import com.swingtrade.data.service.DataIngestionService;
import com.swingtrade.data.service.WatchlistService;
import com.swingtrade.domain.OhlcvCandle;
import com.swingtrade.domain.Signal.SignalType;
import com.swingtrade.domain.store.CandleStore;
import com.swingtrade.strategy.BacktestEngine;
import com.swingtrade.strategy.PriceActionSignalEngine;
import com.swingtrade.strategy.SignalResult;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicReference;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class CandidateScanDataQualityTest {
    private final CandidateScanRunRepository runs = mock(CandidateScanRunRepository.class);
    private final CandidateScanResultRepository results = mock(CandidateScanResultRepository.class);
    private final CandidateHistoryEligibilityRepository eligibility = mock(CandidateHistoryEligibilityRepository.class);
    private final DataIngestionService ingestion = mock(DataIngestionService.class);
    private final CandleStore candles = mock(CandleStore.class);
    private final PriceActionSignalEngine signalEngine = mock(PriceActionSignalEngine.class);
    private final BacktestEngine backtest = mock(BacktestEngine.class);
    private final CandidateStrategyEvaluator evaluator = mock(CandidateStrategyEvaluator.class);
    private CandidateScanService service;

    @BeforeEach
    void setUp() {
        service = new CandidateScanService(
            mock(FyersSymbolRepository.class), null, runs, results, eligibility, ingestion,
            mock(WatchlistService.class), mock(AppSettingsService.class), candles,
            signalEngine, backtest, evaluator, 3, 0, 1);
    }

    @AfterEach
    void tearDown() {
        service.shutdown();
    }

    @Test
    void classifiesMissingHistoricalUniverseMembershipAsInsufficientInsteadOfScanFailure() {
        UUID runId = UUID.randomUUID();
        CandidateScanRunEntity run = runningRun(runId);
        when(runs.findByRunId(runId)).thenReturn(Optional.of(run));
        when(eligibility.findById("INFY")).thenReturn(Optional.empty());
        when(candles.countBySymbol("INFY")).thenReturn(60L);
        when(candles.findAllBySymbolOrderByDateDesc("INFY")).thenReturn(sampleCandles("INFY", 60));
        when(signalEngine.generateSignal("INFY")).thenReturn(signal("INFY", SignalType.BUY));
        when(evaluator.evaluate(eq("INFY"), any())).thenReturn(List.of(
            new CandidateStrategyEvaluator.Outcome("breakout", "BUY", BigDecimal.ONE, null),
            new CandidateStrategyEvaluator.Outcome("pullback", "BUY", BigDecimal.ONE, null)));
        when(backtest.runBacktest(eq("INFY"), eq("NSE"), any()))
            .thenThrow(new IllegalStateException("No included historical universe membership for INFY on 2023-08-31"));
        AtomicReference<CandidateScanResultEntity> saved = new AtomicReference<>();
        when(results.save(any(CandidateScanResultEntity.class))).thenAnswer(invocation -> {
            CandidateScanResultEntity result = invocation.getArgument(0);
            saved.set(result);
            return result;
        });

        assertThat(invokeScan(runId, "INFY")).isFalse();

        assertThat(saved.get()).isNotNull();
        assertThat(saved.get().getDataStatus()).isEqualTo("INSUFFICIENT");
        assertThat(saved.get().isQualified()).isFalse();
        assertThat(saved.get().isActivated()).isFalse();
        assertThat(saved.get().getReason()).contains("historical universe membership");
        assertThat(saved.get().getErrorMessage()).contains("No included historical universe membership");
        verify(backtest, never()).runWalkForward(any(), any(), any(), any(Integer.class), any(Integer.class));
    }

    @Test
    void recordsInsufficientHistoryWithoutCallingStrategyOrBacktest() {
        UUID runId = UUID.randomUUID();
        CandidateScanRunEntity run = runningRun(runId);
        when(runs.findByRunId(runId)).thenReturn(Optional.of(run));
        when(eligibility.findById("SMALL")).thenReturn(Optional.empty());
        when(candles.countBySymbol("SMALL")).thenReturn(12L);
        when(ingestion.backfillStockDataWithOutcome("SMALL", 3))
            .thenReturn(new DataIngestionService.BackfillOutcome("NO_USABLE_DATA", 12, 0, 0, null));
        when(candles.findAllBySymbolOrderByDateDesc("SMALL")).thenReturn(List.of());
        AtomicReference<CandidateScanResultEntity> saved = new AtomicReference<>();
        when(results.save(any(CandidateScanResultEntity.class))).thenAnswer(invocation -> {
            CandidateScanResultEntity result = invocation.getArgument(0);
            saved.set(result);
            return result;
        });

        assertThat(invokeScan(runId, "SMALL")).isFalse();

        assertThat(saved.get().getDataStatus()).isEqualTo("INSUFFICIENT");
        assertThat(saved.get().getCandleCount()).isEqualTo(12);
        assertThat(saved.get().getReason()).contains("Insufficient OHLCV history");
        verify(signalEngine, never()).generateSignal(any());
        verify(backtest, never()).runBacktest(any(), any(), any());
    }

    @Test
    void countsTransientProviderFailureWithoutDoubleCountingCompletedSymbol() {
        UUID runId = UUID.randomUUID();
        CandidateScanRunEntity run = runningRun(runId);
        when(runs.findByRunId(runId)).thenReturn(Optional.of(run));
        when(eligibility.findById("TRANSIENT")).thenReturn(Optional.empty());
        when(candles.countBySymbol("TRANSIENT")).thenReturn(0L);
        when(ingestion.backfillStockDataWithOutcome("TRANSIENT", 3))
            .thenReturn(new DataIngestionService.BackfillOutcome(
                "TRANSIENT_SOURCE_FAILURE", 0, 0, 0, "Yahoo Finance returned HTTP 400"));
        when(candles.findAllBySymbolOrderByDateDesc("TRANSIENT")).thenReturn(List.of());
        when(results.save(any(CandidateScanResultEntity.class))).thenAnswer(invocation -> invocation.getArgument(0));

        assertThat(invokeScan(runId, "TRANSIENT")).isFalse();

        assertThat(run.getFailedSymbols()).isEqualTo(1);
        assertThat(run.getCompletedSymbols()).isZero();
        verify(results).save(any(CandidateScanResultEntity.class));
    }

    private boolean invokeScan(UUID runId, String symbol) {
        try {
            Method method = CandidateScanService.class.getDeclaredMethod(
                "scanSymbol", UUID.class, String.class, boolean.class);
            method.setAccessible(true);
            return (boolean) method.invoke(service, runId, symbol, true);
        } catch (InvocationTargetException e) {
            Throwable cause = e.getCause();
            if (cause instanceof RuntimeException runtime) throw runtime;
            throw new AssertionError(cause);
        } catch (ReflectiveOperationException e) {
            throw new AssertionError(e);
        }
    }

    private static CandidateScanRunEntity runningRun(UUID runId) {
        CandidateScanRunEntity run = new CandidateScanRunEntity();
        run.setRunId(runId);
        run.setStatus("RUNNING");
        run.setTotalSymbols(1);
        return run;
    }

    private static SignalResult signal(String symbol, SignalType type) {
        BigDecimal value = BigDecimal.ONE;
        return new SignalResult(symbol, LocalDate.now(), type, value, value, value, value, "test");
    }

    private static List<OhlcvCandle> sampleCandles(String symbol, int count) {
        List<OhlcvCandle> candles = new ArrayList<>();
        for (int index = 0; index < count; index++) {
            BigDecimal close = BigDecimal.valueOf(100 + index);
            candles.add(OhlcvCandle.of(symbol, LocalDate.of(2023, 1, 1).plusDays(index),
                close, close, close, close, 1_000L));
        }
        return candles;
    }
}
