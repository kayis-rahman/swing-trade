package com.swingtrade.data.service;

import com.swingtrade.data.entity.WatchlistEntity;
import com.swingtrade.data.repository.WatchlistRepository;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.springframework.boot.DefaultApplicationArguments;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDate;
import java.util.List;

import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class EodIngestionSchedulerTest {

    @Mock private DataIngestionService dataIngestionService;
    @Mock private WatchlistRepository watchlistRepository;
    @Mock private NseHolidayService holidayService;

    @Test
    void alwaysIngestsNiftyBenchmarkAlongsideWatchlist() throws Exception {
        WatchlistEntity tcs = new WatchlistEntity("TCS", "TCS");
        when(watchlistRepository.findByIsActiveTrueOrderBySymbolAsc()).thenReturn(List.of(tcs));
        when(holidayService.isMarketClosed(org.mockito.ArgumentMatchers.any())).thenReturn(false);

        EodIngestionScheduler scheduler = new EodIngestionScheduler(
                dataIngestionService, watchlistRepository, holidayService);
        // Pin a Wednesday: the scheduler deliberately skips weekends.
        scheduler.clock = java.time.Clock.fixed(
                java.time.Instant.parse("2026-09-16T11:00:00Z"), java.time.ZoneId.of("Asia/Kolkata"));
        scheduler.ingestLatestForAll();

        verify(dataIngestionService).processSingleStock(eq("TCS"), org.mockito.ArgumentMatchers.any());
        verify(dataIngestionService).processSingleStock(eq(EodIngestionScheduler.NIFTY_50_SYMBOL),
                org.mockito.ArgumentMatchers.any());
    }

    @Test
    void backfillsNiftyBenchmarkHistoryAtStartupWhenAbsent() {
        when(dataIngestionService.getExistingDataWindow(EodIngestionScheduler.NIFTY_50_SYMBOL))
                .thenReturn(new DataIngestionService.DataWindow(null, null));

        EodIngestionScheduler scheduler = new EodIngestionScheduler(
                dataIngestionService, watchlistRepository, holidayService);
        scheduler.ensureNiftyBenchmarkHistory();

        verify(dataIngestionService).backfillStockData(eq(EodIngestionScheduler.NIFTY_50_SYMBOL), eq(10));
    }

    @Test
    void skipsStartupBackfillWhenDisabled() throws Exception {
        EodIngestionScheduler scheduler = new EodIngestionScheduler(
                dataIngestionService, watchlistRepository, holidayService);
        scheduler.benchmarkBackfillOnStartup = false;

        scheduler.run(new DefaultApplicationArguments(new String[0]));

        verify(dataIngestionService, never()).getExistingDataWindow(EodIngestionScheduler.NIFTY_50_SYMBOL);
        verify(dataIngestionService, never()).backfillStockData(
                eq(EodIngestionScheduler.NIFTY_50_SYMBOL), org.mockito.ArgumentMatchers.anyInt());
    }

    @Test
    void backfillsNiftyBenchmarkHistoryAtStartupWhenExistingHistoryIsTooShallow() {
        LocalDate recentEarliest = LocalDate.now().minusYears(1);
        when(dataIngestionService.getExistingDataWindow(EodIngestionScheduler.NIFTY_50_SYMBOL))
                .thenReturn(new DataIngestionService.DataWindow(recentEarliest, LocalDate.now()));

        EodIngestionScheduler scheduler = new EodIngestionScheduler(
                dataIngestionService, watchlistRepository, holidayService);
        scheduler.ensureNiftyBenchmarkHistory();

        verify(dataIngestionService).backfillStockData(eq(EodIngestionScheduler.NIFTY_50_SYMBOL), eq(10));
    }

    @Test
    void skipsNiftyBenchmarkBackfillWhenSufficientHistoryAlreadyExists() {
        LocalDate deepEarliest = LocalDate.now().minusYears(11);
        when(dataIngestionService.getExistingDataWindow(EodIngestionScheduler.NIFTY_50_SYMBOL))
                .thenReturn(new DataIngestionService.DataWindow(deepEarliest, LocalDate.now()));

        EodIngestionScheduler scheduler = new EodIngestionScheduler(
                dataIngestionService, watchlistRepository, holidayService);
        scheduler.ensureNiftyBenchmarkHistory();

        verify(dataIngestionService, never()).backfillStockData(eq(EodIngestionScheduler.NIFTY_50_SYMBOL), org.mockito.ArgumentMatchers.anyInt());
    }
}
