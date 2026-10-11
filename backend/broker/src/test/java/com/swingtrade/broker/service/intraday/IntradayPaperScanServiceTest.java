package com.swingtrade.broker.service.intraday;

import com.swingtrade.broker.risk.KillSwitchService;
import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.entity.WatchlistEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.data.repository.WatchlistRepository;
import com.swingtrade.data.service.MarketCalendar;
import com.swingtrade.data.service.NseSessionCalendar;
import com.swingtrade.strategy.IntradayBacktestConfig;
import com.swingtrade.strategy.IntradayBar;
import com.swingtrade.strategy.IntradayOpeningRangeBreakoutStrategy;
import com.swingtrade.strategy.IntradayRsi2Strategy;
import com.swingtrade.strategy.IntradayVwapPullbackStrategy;
import com.swingtrade.strategy.SignalStrategy;
import com.swingtrade.strategy.StrategyParamsView;
import com.swingtrade.strategy.StrategyTypeRegistry;
import org.junit.jupiter.api.Test;
import org.springframework.data.domain.Pageable;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class IntradayPaperScanServiceTest {
    private static final LocalDate DAY = LocalDate.of(2026, 10, 9);
    private static final LocalTime AS_OF = LocalTime.of(10, 0);
    private static final ZoneId NSE = ZoneId.of("Asia/Kolkata");

    @Test
    void scansOnlyActiveNseWatchlistAcrossRegisteredIntradayStrategies() {
        WatchlistRepository watchlist = mock(WatchlistRepository.class);
        OhlcvCandleRepository candles = mock(OhlcvCandleRepository.class);
        IntradayPaperExecutionService execution = mock(IntradayPaperExecutionService.class);
        StrategyTypeRegistry registry = new StrategyTypeRegistry(List.of(
            new IntradayOpeningRangeBreakoutStrategy(), new IntradayVwapPullbackStrategy(), new IntradayRsi2Strategy()));
        when(watchlist.findByIsActiveTrueOrderBySymbolAsc()).thenReturn(List.of(
            symbol("TCS", "NSE"), symbol("INFY", "NSE"), symbol("BSECO", "BSE")));
        when(candles.findBySymbolAndIntervalAndDateRange(anyString(), eq("15m"), eq(DAY), eq(DAY), eq(Pageable.unpaged())))
            .thenReturn(List.of(), List.of(candle("TCS")));
        when(execution.runSession(anyString(), eq(DAY), org.mockito.ArgumentMatchers.<IntradayBar>anyList(),
            any(SignalStrategy.class), any(StrategyParamsView.class), any(IntradayBacktestConfig.class), eq(AS_OF)))
            .thenAnswer(invocation -> {
                String stock = invocation.getArgument(0);
                SignalStrategy strategy = invocation.getArgument(3);
                boolean candidate = stock.equals("TCS") && strategy.type().equals("INTRADAY_ORB");
                return candidate ? openCandidate(stock, strategy.type()) : noCandidate(stock, strategy.type());
            });

        IntradayPaperScanService service = new IntradayPaperScanService(watchlist, candles, registry, execution, 10);
        IntradayPaperScanResult result = service.scan(DAY, AS_OF, null);

        assertThat(result.mode()).isEqualTo("PAPER_SCAN_AS_OF");
        assertThat(result.universe()).isEqualTo("ACTIVE_NSE_WATCHLIST");
        assertThat(result.watchlistSize()).isEqualTo(3);
        assertThat(result.scannedSymbols()).isEqualTo(2);
        assertThat(result.candidateCount()).isEqualTo(1);
        assertThat(result.liveBrokerCalls()).isZero();
        assertThat(result.excludedSymbols()).containsExactly("BSECO");
        assertThat(result.results()).hasSize(6);
        assertThat(result.results()).filteredOn(row -> row.status().equals("CANDIDATE"))
            .extracting(IntradayPaperScanResult.Candidate::symbol,
                IntradayPaperScanResult.Candidate::strategy).containsExactly(
                    org.assertj.core.groups.Tuple.tuple("TCS", "INTRADAY_ORB"));
        verify(candles).findBySymbolAndIntervalAndDateRange("TCS", "15m", DAY, DAY, Pageable.unpaged());
        verify(candles).findBySymbolAndIntervalAndDateRange("INFY", "15m", DAY, DAY, Pageable.unpaged());
        verify(candles, never()).findBySymbolAndIntervalAndDateRange("BSECO", "15m", DAY, DAY, Pageable.unpaged());
        verify(execution, times(3)).runSession(eq("TCS"), eq(DAY),
            org.mockito.ArgumentMatchers.<List<IntradayBar>>argThat(bars -> bars.size() == 1
                && bars.stream().noneMatch(bar -> bar.sessionTime().isAfter(AS_OF))),
            any(SignalStrategy.class), any(StrategyParamsView.class), any(IntradayBacktestConfig.class), eq(AS_OF));
    }

    @Test
    void refusesAnOversizedWatchlistBeforeReadingAnyBars() {
        WatchlistRepository watchlist = mock(WatchlistRepository.class);
        OhlcvCandleRepository candles = mock(OhlcvCandleRepository.class);
        IntradayPaperScanService service = new IntradayPaperScanService(watchlist, candles,
            new StrategyTypeRegistry(List.of(new IntradayOpeningRangeBreakoutStrategy())),
            mock(IntradayPaperExecutionService.class), 1);
        when(watchlist.findByIsActiveTrueOrderBySymbolAsc()).thenReturn(List.of(symbol("A", "NSE"), symbol("B", "NSE")));

        assertThatThrownBy(() -> service.scan(DAY, AS_OF, IntradayBacktestConfig.defaults()))
            .isInstanceOf(IllegalStateException.class).hasMessageContaining("scan limit is 1");
        verify(candles, never()).findBySymbolAndIntervalAndDateRange(anyString(), anyString(),
            any(LocalDate.class), any(LocalDate.class), any(Pageable.class));
    }

    @Test
    void rejectsNonTradingDateWhenWatchlistIsEmpty() {
        WatchlistRepository watchlist = mock(WatchlistRepository.class);
        IntradayPaperExecutionService execution = executionService(false);
        IntradayPaperScanService service = new IntradayPaperScanService(watchlist,
            mock(OhlcvCandleRepository.class), new StrategyTypeRegistry(List.of()), execution, 10);

        assertThatThrownBy(() -> service.scan(LocalDate.of(2026, 10, 10), AS_OF, null))
            .isInstanceOf(IllegalArgumentException.class).hasMessageContaining("not an NSE trading session");
        verify(watchlist, never()).findByIsActiveTrueOrderBySymbolAsc();
    }

    @Test
    void rejectsUnexpectedAsOfTimeWhenWatchlistIsEmpty() {
        WatchlistRepository watchlist = mock(WatchlistRepository.class);
        IntradayPaperExecutionService execution = executionService(true);
        IntradayPaperScanService service = new IntradayPaperScanService(watchlist,
            mock(OhlcvCandleRepository.class), new StrategyTypeRegistry(List.of()), execution, 10);

        assertThatThrownBy(() -> service.scan(DAY, LocalTime.of(10, 7), null))
            .isInstanceOf(IllegalArgumentException.class).hasMessageContaining("expected NSE session bar start");
        verify(watchlist, never()).findByIsActiveTrueOrderBySymbolAsc();
    }

    private static IntradayPaperExecutionService executionService(boolean tradingDay) {
        MarketCalendar marketCalendar = mock(MarketCalendar.class);
        when(marketCalendar.isNseTradingSession(DAY)).thenReturn(tradingDay);
        return new IntradayPaperExecutionService(new NseSessionCalendar(marketCalendar), mock(KillSwitchService.class));
    }

    private static WatchlistEntity symbol(String value, String exchange) {
        WatchlistEntity entity = new WatchlistEntity(value, value);
        entity.setExchange(exchange);
        return entity;
    }

    private static OhlcvCandleEntity candle(String symbol) {
        OhlcvCandleEntity candle = new OhlcvCandleEntity();
        candle.setSymbol(symbol);
        candle.setDate(DAY);
        candle.setTimeframe("15m");
        candle.setBarTime(LocalTime.of(9, 15));
        candle.setOpenPrice(BigDecimal.TEN);
        candle.setHighPrice(BigDecimal.TEN);
        candle.setLowPrice(BigDecimal.TEN);
        candle.setClosePrice(BigDecimal.TEN);
        candle.setVolume(1_000L);
        return candle;
    }

    private static IntradayPaperSessionResult openCandidate(String symbol, String strategy) {
        var entry = java.time.ZonedDateTime.of(DAY, AS_OF, NSE);
        var position = new IntradayPaperSessionResult.PaperPositionRecord("p", symbol, "e", null,
            entry, null, BigDecimal.TEN, null, BigDecimal.TEN, 1, "OPEN", BigDecimal.ZERO,
            BigDecimal.ZERO, BigDecimal.ZERO, "OPEN", false);
        return new IntradayPaperSessionResult("r", "PAPER_SIMULATION", "COMPLETED", symbol, DAY,
            4, 4, strategy, false, 1, 1, 0, BigDecimal.TEN, BigDecimal.TEN, BigDecimal.ZERO,
            List.of(), List.of(position), List.of());
    }

    private static IntradayPaperSessionResult noCandidate(String symbol, String strategy) {
        return new IntradayPaperSessionResult("r", "PAPER_SIMULATION", "COMPLETED", symbol, DAY,
            4, 4, strategy, false, 0, 0, 0, BigDecimal.TEN, BigDecimal.TEN, BigDecimal.ZERO,
            List.of(), List.of(), List.of());
    }
}
