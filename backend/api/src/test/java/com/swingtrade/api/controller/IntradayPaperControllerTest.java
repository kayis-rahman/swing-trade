package com.swingtrade.api.controller;

import com.swingtrade.api.dto.IntradayPaperSessionRequest;
import com.swingtrade.broker.service.intraday.IntradayPaperExecutionService;
import com.swingtrade.broker.service.intraday.IntradayPaperSessionResult;
import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.strategy.IntradayBacktestConfig;
import com.swingtrade.strategy.IntradayOpeningRangeBreakoutStrategy;
import com.swingtrade.strategy.StrategyTypeRegistry;
import org.junit.jupiter.api.Test;
import org.springframework.data.domain.Pageable;
import org.springframework.http.HttpStatus;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class IntradayPaperControllerTest {
    private static final LocalDate DAY = LocalDate.of(2026, 10, 9);

    @Test
    void readsOnlyStoredFifteenMinuteBarsAndReturnsThePaperAudit() {
        OhlcvCandleRepository repository = mock(OhlcvCandleRepository.class);
        IntradayPaperExecutionService execution = mock(IntradayPaperExecutionService.class);
        StrategyTypeRegistry registry = new StrategyTypeRegistry(List.of(new IntradayOpeningRangeBreakoutStrategy()));
        OhlcvCandleEntity candle = candle();
        when(repository.findBySymbolAndIntervalAndDateRange("TCS", "15m", DAY, DAY, Pageable.unpaged()))
            .thenReturn(List.of(candle));
        IntradayPaperSessionResult expected = new IntradayPaperSessionResult("run-1", "PAPER_SIMULATION",
            "COMPLETED", "TCS", DAY, 25, 25, IntradayOpeningRangeBreakoutStrategy.TYPE, false,
            0, 0, 0, BigDecimal.TEN, BigDecimal.TEN, BigDecimal.ZERO, List.of(), List.of(), List.of());
        when(execution.runSession(eq("TCS"), eq(DAY), any(), any(), any(), eq(IntradayBacktestConfig.defaults())))
            .thenReturn(expected);
        IntradayPaperController controller = new IntradayPaperController(repository, registry, execution);

        var response = controller.runSession(new IntradayPaperSessionRequest(
            "tcs", DAY, IntradayOpeningRangeBreakoutStrategy.TYPE, null));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.OK);
        assertThat(response.getBody()).isSameAs(expected);
        verify(repository).findBySymbolAndIntervalAndDateRange("TCS", "15m", DAY, DAY, Pageable.unpaged());
        verify(execution).runSession(eq("TCS"), eq(DAY), any(), any(), any(), eq(IntradayBacktestConfig.defaults()));
    }

    @Test
    void rejectsNonIntradayStrategiesWithoutLoadingBarsOrExecuting() {
        OhlcvCandleRepository repository = mock(OhlcvCandleRepository.class);
        IntradayPaperExecutionService execution = mock(IntradayPaperExecutionService.class);
        StrategyTypeRegistry registry = new StrategyTypeRegistry(List.of(new IntradayOpeningRangeBreakoutStrategy()));
        IntradayPaperController controller = new IntradayPaperController(repository, registry, execution);

        var response = controller.runSession(new IntradayPaperSessionRequest("TCS", DAY, "BREAKOUT", null));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.BAD_REQUEST);
        verify(repository, never()).findBySymbolAndIntervalAndDateRange(any(), any(), any(), any(), any());
        verify(execution, never()).runSession(any(), any(), any(), any(), any(), any());
    }

    @Test
    void sendsEmptyStoredSessionToExecutionSoItCanReturnIncompleteData() {
        OhlcvCandleRepository repository = mock(OhlcvCandleRepository.class);
        IntradayPaperExecutionService execution = mock(IntradayPaperExecutionService.class);
        StrategyTypeRegistry registry = new StrategyTypeRegistry(List.of(new IntradayOpeningRangeBreakoutStrategy()));
        IntradayPaperSessionResult incomplete = new IntradayPaperSessionResult("run-2", "PAPER_SIMULATION",
            "INCOMPLETE_DATA", "TCS", DAY, 25, 0, IntradayOpeningRangeBreakoutStrategy.TYPE, false,
            0, 0, 0, BigDecimal.TEN, BigDecimal.TEN, BigDecimal.ZERO, List.of(), List.of(), List.of());
        when(execution.runSession(eq("TCS"), eq(DAY), any(), any(), any(), eq(IntradayBacktestConfig.defaults())))
            .thenReturn(incomplete);
        IntradayPaperController controller = new IntradayPaperController(repository, registry, execution);

        var response = controller.runSession(new IntradayPaperSessionRequest(
            "TCS", DAY, IntradayOpeningRangeBreakoutStrategy.TYPE, null));

        assertThat(response.getStatusCode()).isEqualTo(HttpStatus.OK);
        assertThat(response.getBody().status()).isEqualTo("INCOMPLETE_DATA");
        verify(execution).runSession(eq("TCS"), eq(DAY), any(), any(), any(), eq(IntradayBacktestConfig.defaults()));
    }

    private static OhlcvCandleEntity candle() {
        OhlcvCandleEntity candle = new OhlcvCandleEntity();
        candle.setSymbol("TCS");
        candle.setDate(DAY);
        candle.setTimeframe("15m");
        candle.setBarTime(LocalTime.of(9, 15));
        candle.setOpenPrice(new BigDecimal("100"));
        candle.setHighPrice(new BigDecimal("101"));
        candle.setLowPrice(new BigDecimal("99"));
        candle.setClosePrice(new BigDecimal("100"));
        candle.setVolume(1_000L);
        return candle;
    }
}
