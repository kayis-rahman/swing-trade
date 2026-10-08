package com.swingtrade.data.service;

import com.swingtrade.core.metrics.DataIngestionMetrics;
import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.entity.WatchlistEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.data.repository.WatchlistRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.mockito.junit.jupiter.MockitoSettings;
import org.mockito.quality.Strictness;
import org.springframework.transaction.support.TransactionCallback;
import org.springframework.transaction.support.TransactionTemplate;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

/**
 * Unit tests for the bounded, session-aware 15-minute ingestion path:
 * watchlist-only scope, session-window filtering, rate-limit pacing,
 * restart-safe resumption, and per-symbol failure isolation.
 */
@ExtendWith(MockitoExtension.class)
@MockitoSettings(strictness = Strictness.LENIENT)
class IntradayIngestionServiceTest {

    private static final LocalDate SESSION = LocalDate.of(2026, 9, 15); // Tuesday
    private static final LocalDate NEXT_SESSION = LocalDate.of(2026, 9, 16); // Wednesday

    @Mock private OhlcvCandleRepository candleRepository;
    @Mock private WatchlistRepository watchlistRepository;
    @Mock private MarketDataClientProvider marketDataClientProvider;
    @Mock private MarketDataClient marketDataClient;
    @Mock private MarketCalendar marketCalendar;
    @Mock private DataIngestionMetrics ingestionMetrics;
    @Mock private TransactionTemplate txTemplate;

    private IntradayIngestionService service;

    @BeforeEach
    void setUp() {
        NseSessionCalendar sessionCalendar = new NseSessionCalendar(marketCalendar);
        service = new IntradayIngestionService(candleRepository, watchlistRepository,
            marketDataClientProvider, marketCalendar, sessionCalendar, txTemplate, ingestionMetrics);
        when(marketDataClientProvider.getClient()).thenReturn(marketDataClient);
        when(marketDataClientProvider.getActiveBroker()).thenReturn("yahoo");
        // Run the callback with a null status: the repository is mocked anyway.
        when(txTemplate.execute(any())).thenAnswer(invocation ->
            ((TransactionCallback<?>) invocation.getArgument(0)).doInTransaction(null));
    }

    @SuppressWarnings("unchecked")
    private void watchlist(String... symbols) {
        List<WatchlistEntity> entities = new ArrayList<>();
        for (String symbol : symbols) {
            entities.add(new WatchlistEntity(symbol, symbol));
        }
        when(watchlistRepository.findByIsActiveTrueOrderBySymbolAsc()).thenReturn(entities);
    }

    private CandleData bar(String symbol, LocalDate date, LocalTime barTime, double close) {
        return CandleData.of(symbol, date, barTime,
            BigDecimal.valueOf(close - 1), BigDecimal.valueOf(close + 1),
            BigDecimal.valueOf(close - 2), BigDecimal.valueOf(close), 500_000L,
            BigDecimal.valueOf(close), Interval.FIFTEEN_MINUTES);
    }

    private OhlcvCandleEntity storedBar(String symbol, LocalDate date, LocalTime barTime, double close) {
        return OhlcvCandleEntity.fromCandleData(bar(symbol, date, barTime, close));
    }

    @Test
    void ingestsOnlyWatchlistSymbolsAtTheRequestedInterval() {
        watchlist("TCS", "RELIANCE");
        when(marketCalendar.isNseTradingSession(any())).thenReturn(true);
        when(marketDataClient.fetchCandles(eq("TCS"), any(), any(), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of(bar("TCS", SESSION, LocalTime.of(9, 15), 4000.0)));
        when(marketDataClient.fetchCandles(eq("RELIANCE"), any(), any(), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of());
        when(candleRepository.insertIfAbsent(anyString(), anyString(), any(), any(), any(), any(),
            any(), any(), anyLong(), any())).thenReturn(1);

        IntradayIngestionService.IntradayIngestionResult result =
            service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, SESSION, SESSION);

        assertThat(result.symbolsAttempted()).isEqualTo(2);
        assertThat(result.symbolsSucceeded()).isEqualTo(2);
        assertThat(result.barsSaved()).isEqualTo(1);
        // The interval code is passed through to persistence.
        verify(candleRepository).insertIfAbsent(eq("TCS"), eq("15m"), eq(SESSION),
            eq(LocalTime.of(9, 15)), any(), any(), any(), any(), eq(500_000L), any());
        // The daily path is never touched.
        verify(marketDataClient, never()).fetchCandles(anyString(), any(), any());
    }

    @Test
    void dropsBarsOutsideTheNormalSessionWindowAndNonTradingDates() {
        watchlist("TCS");
        when(marketCalendar.isNseTradingSession(SESSION)).thenReturn(true);
        when(marketCalendar.isNseTradingSession(NEXT_SESSION)).thenReturn(false); // holiday
        when(marketDataClient.fetchCandles(eq("TCS"), any(), any(), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of(
                bar("TCS", SESSION, LocalTime.of(9, 0), 3990.0),    // pre-open: dropped
                bar("TCS", SESSION, LocalTime.of(9, 15), 3995.0),   // first session bar: kept
                bar("TCS", SESSION, LocalTime.of(15, 15), 4100.0),  // last session bar: kept
                bar("TCS", SESSION, LocalTime.of(15, 45), 4105.0),  // closing auction: dropped
                bar("TCS", NEXT_SESSION, LocalTime.of(10, 0), 4200.0) // holiday: dropped
            ));
        when(candleRepository.insertIfAbsent(anyString(), anyString(), any(), any(), any(), any(),
            any(), any(), anyLong(), any())).thenReturn(1);

        IntradayIngestionService.IntradayIngestionResult result =
            service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, SESSION, NEXT_SESSION);

        assertThat(result.barsFetched()).isEqualTo(5);
        assertThat(result.barsSaved()).isEqualTo(2);
        assertThat(result.barsRejected()).isEqualTo(3);
        verify(candleRepository, times(2)).insertIfAbsent(eq("TCS"), eq("15m"), eq(SESSION),
            any(), any(), any(), any(), any(), eq(500_000L), any());
    }

    @Test
    void resumesFromLatestStoredBarSoRestartsDoNotRedownload() {
        watchlist("TCS", "RELIANCE");
        when(marketCalendar.isNseTradingSession(any())).thenReturn(true);
        LocalDate windowStart = LocalDate.of(2026, 9, 1);
        // TCS already has bars through 2026-10-05; RELIANCE has none.
        when(candleRepository.findLatestBySymbolAndInterval("TCS", "15m"))
            .thenReturn(Optional.of(storedBar("TCS", SESSION, LocalTime.of(15, 15), 4100.0)));
        when(candleRepository.findLatestBySymbolAndInterval("RELIANCE", "15m"))
            .thenReturn(Optional.empty());
        when(marketDataClient.fetchCandles(eq("TCS"), eq(SESSION), eq(NEXT_SESSION), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of());
        when(marketDataClient.fetchCandles(eq("RELIANCE"), eq(windowStart), eq(NEXT_SESSION), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of());

        service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, windowStart, NEXT_SESSION);

        // TCS resumes at its latest stored date instead of re-downloading the window;
        // RELIANCE, with no stored bars, fetches the full window.
        verify(marketDataClient).fetchCandles(eq("TCS"), eq(SESSION), eq(NEXT_SESSION), eq(Interval.FIFTEEN_MINUTES));
        verify(marketDataClient).fetchCandles(eq("RELIANCE"), eq(windowStart), eq(NEXT_SESSION), eq(Interval.FIFTEEN_MINUTES));
    }

    @Test
    void resumesAtTheLatestStoredDateWhenItIsInsideTheWindow() {
        watchlist("TCS");
        when(marketCalendar.isNseTradingSession(any())).thenReturn(true);
        // TCS has bars through 2026-09-14, one day before the requested window start.
        when(candleRepository.findLatestBySymbolAndInterval("TCS", "15m"))
            .thenReturn(Optional.of(storedBar("TCS", LocalDate.of(2026, 9, 14), LocalTime.of(15, 15), 3900.0)));
        when(marketDataClient.fetchCandles(eq("TCS"), eq(SESSION), eq(NEXT_SESSION), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of());

        service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, SESSION, NEXT_SESSION);

        // Latest stored date is before the window, so the full window is fetched.
        verify(marketDataClient).fetchCandles(eq("TCS"), eq(SESSION), eq(NEXT_SESSION), eq(Interval.FIFTEEN_MINUTES));
    }

    @Test
    void reRunningIsIdempotentAndRestartSafe() {
        watchlist("TCS");
        when(marketCalendar.isNseTradingSession(any())).thenReturn(true);
        when(candleRepository.findLatestBySymbolAndInterval("TCS", "15m"))
            .thenReturn(Optional.empty());
        when(marketDataClient.fetchCandles(eq("TCS"), any(), any(), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of(bar("TCS", SESSION, LocalTime.of(9, 15), 4000.0)));
        // First run saves the bar; second run finds it already stored.
        when(candleRepository.insertIfAbsent(anyString(), anyString(), any(), any(), any(), any(),
            any(), any(), anyLong(), any())).thenReturn(1).thenReturn(0);

        IntradayIngestionService.IntradayIngestionResult first =
            service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, SESSION, SESSION);
        IntradayIngestionService.IntradayIngestionResult second =
            service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, SESSION, SESSION);

        assertThat(first.barsSaved()).isEqualTo(1);
        assertThat(second.barsSaved()).isZero();
        assertThat(second.symbolsSucceeded()).isEqualTo(1);
    }

    @Test
    void isolatesPerSymbolFailuresAndContinues() {
        watchlist("TCS", "RELIANCE");
        when(marketCalendar.isNseTradingSession(any())).thenReturn(true);
        when(marketDataClient.fetchCandles(eq("TCS"), any(), any(), eq(Interval.FIFTEEN_MINUTES)))
            .thenThrow(new RuntimeException("provider timeout"));
        when(marketDataClient.fetchCandles(eq("RELIANCE"), any(), any(), eq(Interval.FIFTEEN_MINUTES)))
            .thenReturn(List.of(bar("RELIANCE", SESSION, LocalTime.of(9, 15), 2500.0)));
        when(candleRepository.insertIfAbsent(anyString(), anyString(), any(), any(), any(), any(),
            any(), any(), anyLong(), any())).thenReturn(1);

        IntradayIngestionService.IntradayIngestionResult result =
            service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, SESSION, SESSION);

        assertThat(result.symbolsSucceeded()).isEqualTo(1);
        assertThat(result.symbolsFailed()).isEqualTo(1);
        assertThat(result.errors()).hasSize(1);
        assertThat(result.errors().get(0)).startsWith("TCS");
        assertThat(result.barsSaved()).isEqualTo(1);
        verify(ingestionMetrics).recordFetchFailure("yahoo");
    }

    @Test
    void failsLoudlyWhenTheProviderHasNoIntervalSupport() {
        watchlist("TCS");
        when(marketDataClient.fetchCandles(anyString(), any(), any(), eq(Interval.FIFTEEN_MINUTES)))
            .thenThrow(new UnsupportedOperationException("no intervals"));

        assertThatThrownBy(() -> service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, SESSION, SESSION))
            .isInstanceOf(UnsupportedOperationException.class);
    }

    @Test
    void rejectsInvalidWindows() {
        assertThatThrownBy(() -> service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES, NEXT_SESSION, SESSION))
            .isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> service.ingestWatchlistInterval(null, SESSION, SESSION))
            .isInstanceOf(IllegalArgumentException.class);
    }

    @Test
    void skipsFutureWindows() {
        watchlist("TCS");
        IntradayIngestionService.IntradayIngestionResult result =
            service.ingestWatchlistInterval(Interval.FIFTEEN_MINUTES,
                LocalDate.of(2099, 1, 1), LocalDate.of(2099, 1, 2));

        assertThat(result.symbolsAttempted()).isZero();
        verify(marketDataClient, never()).fetchCandles(anyString(), any(), any(), any(Interval.class));
    }
}
