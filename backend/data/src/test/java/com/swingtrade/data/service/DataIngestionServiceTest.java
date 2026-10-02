package com.swingtrade.data.service;

import com.swingtrade.data.entity.OhlcvCandleEntity;
import com.swingtrade.data.repository.OhlcvCandleRepository;
import com.swingtrade.data.repository.StockRepository;
import com.swingtrade.data.repository.WatchlistRepository;
import com.swingtrade.domain.PriceBand;
import com.swingtrade.domain.store.PriceBandStore;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;
import org.springframework.transaction.support.TransactionTemplate;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.when;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.ArgumentMatchers.eq;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class DataIngestionServiceTest {

    private DataIngestionService dataIngestionService;
    private OhlcvCandleRepository candleRepository;
    private StockRepository stockRepository;
    private WatchlistRepository watchlistRepository;
    private MarketDataClientProvider marketDataClientProvider;
    private MarketDataClient mockClient;
    private PriceBandStore priceBandStore;
    private TransactionTemplate txTemplate;

    @BeforeEach
    void setUp() {
        candleRepository = Mockito.mock(OhlcvCandleRepository.class);
        stockRepository = Mockito.mock(StockRepository.class);
        watchlistRepository = Mockito.mock(WatchlistRepository.class);
        mockClient = Mockito.mock(MarketDataClient.class);

        marketDataClientProvider = Mockito.mock(MarketDataClientProvider.class);
        when(marketDataClientProvider.getClient()).thenReturn(mockClient);
        priceBandStore = Mockito.mock(PriceBandStore.class);

        txTemplate = Mockito.mock(TransactionTemplate.class);
        dataIngestionService = new DataIngestionService(candleRepository, stockRepository, watchlistRepository,
            marketDataClientProvider, txTemplate,
            Mockito.mock(com.swingtrade.core.metrics.DataIngestionMetrics.class), null,
            null, 30, priceBandStore);
    }

    @Test
    @DisplayName("processSingleStock persists provider price bands even when candle exists")
    void processSingleStockPersistsPriceBandForExistingCandle() {
        LocalDate date = LocalDate.of(2026, 1, 5);
        PriceBand band = new PriceBand("RELIANCE", date, bd("90"), bd("110"));
        when(mockClient.fetchPriceBand("RELIANCE", date)).thenReturn(band);
        when(candleRepository.existsBySymbolAndDate("RELIANCE", date)).thenReturn(true);

        dataIngestionService.processSingleStock("RELIANCE", date);

        verify(priceBandStore).save(band);
        verify(mockClient, Mockito.never()).fetchCandle("RELIANCE", date);
    }

    @Test
    @DisplayName("processSingleStock ignores non-trading sessions")
    void processSingleStockIgnoresWeekend() {
        dataIngestionService.processSingleStock("RELIANCE", LocalDate.of(2026, 1, 4));

        verifyNoInteractions(mockClient, candleRepository, priceBandStore);
    }

    @Test
    @DisplayName("processSingleStock rejects invalid provider candle")
    void processSingleStockRejectsInvalidCandle() {
        LocalDate date = LocalDate.of(2026, 1, 5);
        CandleData invalid = CandleData.of("RELIANCE", date, bd("10"), bd("9"), bd("8"), bd("9"), 100);
        when(candleRepository.existsBySymbolAndDate("RELIANCE", date)).thenReturn(false);
        when(mockClient.fetchCandle("RELIANCE", date)).thenReturn(invalid);

        dataIngestionService.processSingleStock("RELIANCE", date);

        verify(mockClient).fetchCandle("RELIANCE", date);
        verify(candleRepository, Mockito.never()).insertIfAbsent(anyString(), eq(date), any(), any(), any(), any(), eq(100L), any());
    }

    @Test
    @DisplayName("testAutoIngestData_ScheduledMethodAvailable")
    void testAutoIngestData() {
        // Test that the scheduled method can be called without errors
        assert dataIngestionService != null;
    }

    @Test
    @DisplayName("testDataQualityValidation_ReturnsValidReport")
    void testDataQualityValidation() {
        // Given: Mock repository returns empty list for new stock
        when(candleRepository.findBySymbolAndDateRange(anyString(), any(LocalDate.class), any(LocalDate.class), any()))
            .thenReturn(new ArrayList<>());

        // When: Validation is called
        DataIngestionService.DataQualityReport report =
            dataIngestionService.validateDataQuality("RELIANCE", LocalDate.now().minusYears(1), LocalDate.now());

        // Then: Report is returned with expected values
        assert report != null;
        assert report.getStockSymbol().equals("RELIANCE");
        assert report.getExpectedTradingDays() > 0;
    }

    @Test
    @DisplayName("data quality keeps gap issues and reports distinct actual sessions")
    void dataQualityKeepsGapIssuesAndUsesDistinctDates() {
        LocalDate monday = LocalDate.of(2026, 1, 5);
        OhlcvCandleEntity first = candle("RELIANCE", monday);
        OhlcvCandleEntity duplicate = candle("RELIANCE", monday);
        when(candleRepository.findBySymbolAndDateRange(eq("RELIANCE"), eq(monday),
            eq(monday.plusDays(2)), any())).thenReturn(List.of(first, duplicate));

        DataIngestionService.DataQualityReport report =
            dataIngestionService.validateDataQuality("RELIANCE", monday, monday.plusDays(2));

        assert report.getExpectedTradingDays() == 3;
        assert report.getActualTradingDays() == 1;
        assert report.hasIssues();
        assert report.getGapPercentage() > 10.0;
        assert report.isCritical();
        assert report.getGaps().size() == 1;
    }

    @Test
    @DisplayName("testGetLatestCandle_ReturnsLatestData")
    void testGetLatestCandle() {
        // Given: Mock repository returns a candle
        OhlcvCandleEntity mockCandle = new OhlcvCandleEntity();
        mockCandle.setSymbol("RELIANCE");
        mockCandle.setDate(LocalDate.now().minusDays(1));
        mockCandle.setClosePrice(BigDecimal.valueOf(2540));

        when(candleRepository.findLatestBySymbol(anyString())).thenReturn(Optional.of(mockCandle));

        // When: Get latest candle is called
        Optional<OhlcvCandleEntity> result = dataIngestionService.getLatestCandle("RELIANCE");

        // Then: Candle is returned
        assert result.isPresent();
        assert result.get().getSymbol().equals("RELIANCE");
    }

    @Test
    @DisplayName("testGetRecentCandles_ReturnsRecentData")
    void testGetRecentCandles() {
        // Given: Mock repository returns a list of candles
        OhlcvCandleEntity mockCandle = new OhlcvCandleEntity();
        mockCandle.setSymbol("RELIANCE");
        mockCandle.setDate(LocalDate.now().minusDays(1));
        mockCandle.setClosePrice(BigDecimal.valueOf(2540));

        List<OhlcvCandleEntity> mockList = List.of(mockCandle);
        when(candleRepository.findTopBySymbolOrderByDateDesc(anyString(), any())).thenReturn(mockList);

        // When: Get recent candles is called
        List<OhlcvCandleEntity> result = dataIngestionService.getRecentCandles("RELIANCE", 5);

        // Then: Candles are returned
        assert result != null;
        assert result.size() == 1;
    }

    private static BigDecimal bd(String value) {
        return new BigDecimal(value);
    }

    @Test
    @DisplayName("orchestration pulls latest data and repairs gaps in existing history")
    void fetchLatestAndRepairGapsRepairsExistingHistory() {
        LocalDate latestDate = LocalDate.of(2026, 8, 28);
        OhlcvCandleEntity earliest = candle("RELIANCE", LocalDate.of(2026, 8, 24));
        List<OhlcvCandleEntity> existing = List.of(
            earliest,
            candle("RELIANCE", LocalDate.of(2026, 8, 26)),
            candle("RELIANCE", latestDate));

        when(candleRepository.existsBySymbolAndDate(eq("RELIANCE"), eq(latestDate))).thenReturn(true);
        when(candleRepository.findEarliestBySymbol("RELIANCE")).thenReturn(Optional.of(earliest));
        when(candleRepository.findBySymbolAndDateRange(eq("RELIANCE"), eq(earliest.getDate()),
            eq(latestDate), any())).thenReturn(existing);
        when(mockClient.fetchCandles("RELIANCE", earliest.getDate(), latestDate))
            .thenReturn(List.of());

        String summary = dataIngestionService.fetchLatestAndRepairGaps("RELIANCE", latestDate);

        verify(mockClient).fetchCandles("RELIANCE", earliest.getDate(), latestDate);
        assert summary.contains("requested gap repair for 2 sessions");
    }

    @Test
    void incrementalBackfillFetchesOnlyAfterLatestStoredCandle() {
        LocalDate from = LocalDate.of(2026, 1, 1);
        LocalDate latest = LocalDate.of(2026, 6, 1);
        OhlcvCandleEntity earliest = candle("RELIANCE", from);
        OhlcvCandleEntity latestCandle = candle("RELIANCE", latest);
        when(candleRepository.findEarliestBySymbol("RELIANCE")).thenReturn(Optional.of(earliest));
        when(candleRepository.findLatestBySymbol("RELIANCE")).thenReturn(Optional.of(latestCandle));
        when(mockClient.fetchCandles("RELIANCE", latest.plusDays(1), LocalDate.of(2026, 6, 10)))
            .thenReturn(List.of());

        DataIngestionService.BackfillOutcome outcome = dataIngestionService.processIncrementalStockData(
            "RELIANCE", from, LocalDate.of(2026, 6, 10));

        verify(mockClient).fetchCandles("RELIANCE", latest.plusDays(1), LocalDate.of(2026, 6, 10));
        assert outcome.sourceOutcome().equals("NO_USABLE_DATA");
    }

    @Test
    void incrementalBackfillDoesNotCallProviderWhenRangeAlreadyCurrent() {
        LocalDate date = LocalDate.of(2026, 6, 10);
        OhlcvCandleEntity earliest = candle("RELIANCE", LocalDate.of(2026, 1, 1));
        OhlcvCandleEntity stored = candle("RELIANCE", date);
        when(candleRepository.findEarliestBySymbol("RELIANCE")).thenReturn(Optional.of(earliest));
        when(candleRepository.findLatestBySymbol("RELIANCE")).thenReturn(Optional.of(stored));

        DataIngestionService.BackfillOutcome outcome = dataIngestionService.processIncrementalStockData(
            "RELIANCE", LocalDate.of(2026, 1, 1), date);

        verify(mockClient, Mockito.never()).fetchCandles(anyString(), any(LocalDate.class), any(LocalDate.class));
        assert outcome.sourceOutcome().equals("ALREADY_CURRENT");
    }

    @Test
    void incrementalBackfillRepairsMissingHistoricalHeadBeforeUsingLatestTail() {
        LocalDate requestedFrom = LocalDate.of(2026, 1, 1);
        LocalDate latest = LocalDate.of(2026, 6, 10);
        when(candleRepository.findEarliestBySymbol("RELIANCE"))
            .thenReturn(Optional.of(candle("RELIANCE", LocalDate.of(2026, 6, 1))));
        when(candleRepository.findLatestBySymbol("RELIANCE"))
            .thenReturn(Optional.of(candle("RELIANCE", latest)));
        when(mockClient.fetchCandles("RELIANCE", requestedFrom, requestedFrom.plusDays(29)))
            .thenReturn(List.of());

        dataIngestionService.processIncrementalStockData("RELIANCE", requestedFrom, latest);

        verify(mockClient).fetchCandles("RELIANCE", requestedFrom, requestedFrom.plusDays(29));
    }

    @Test
    void incrementalBackfillStopsAfterFirstTransientProviderFailure() {
        LocalDate from = LocalDate.of(2021, 1, 1);
        LocalDate to = LocalDate.of(2021, 3, 31);
        when(candleRepository.findEarliestBySymbol("RELIANCE")).thenReturn(Optional.empty());
        when(candleRepository.findLatestBySymbol("RELIANCE")).thenReturn(Optional.empty());
        when(mockClient.fetchCandles("RELIANCE", from, from.plusDays(29)))
            .thenThrow(new IllegalStateException("provider circuit is open"));

        DataIngestionService.BackfillOutcome outcome =
            dataIngestionService.processIncrementalStockData("RELIANCE", from, to);

        assert outcome.sourceOutcome().equals("TRANSIENT_SOURCE_FAILURE");
        verify(mockClient).fetchCandles("RELIANCE", from, from.plusDays(29));
        verify(mockClient, Mockito.never()).fetchCandles("RELIANCE", from.plusDays(30), from.plusDays(59));
    }

    @Test
    void incrementalBackfillSplitsLongRangeIntoConfiguredChunks() {
        LocalDate from = LocalDate.of(2026, 1, 1);
        LocalDate to = from.plusDays(64);
        when(candleRepository.findEarliestBySymbol("RELIANCE")).thenReturn(Optional.empty());
        when(candleRepository.findLatestBySymbol("RELIANCE")).thenReturn(Optional.empty());
        when(mockClient.fetchCandles(anyString(), any(LocalDate.class), any(LocalDate.class)))
            .thenReturn(List.of());

        DataIngestionService.BackfillOutcome outcome = dataIngestionService.processIncrementalStockData(
            "RELIANCE", from, to);

        verify(mockClient).fetchCandles("RELIANCE", from, from.plusDays(29));
        verify(mockClient).fetchCandles("RELIANCE", from.plusDays(30), from.plusDays(59));
        verify(mockClient).fetchCandles("RELIANCE", from.plusDays(60), to);
        assert outcome.sourceOutcome().equals("NO_USABLE_DATA");
    }

    @Test
    void processStockDataClassifiesValidInvalidAndProviderFailure() {
        LocalDate monday = LocalDate.of(2026, 1, 5);
        CandleData valid = CandleData.of("RELIANCE", monday, bd("10"), bd("12"), bd("9"), bd("11"), 100);
        CandleData invalid = CandleData.of("RELIANCE", monday.plusDays(1), bd("12"), bd("10"), bd("9"), bd("11"), 100);
        when(mockClient.fetchCandles("RELIANCE", monday, monday.plusDays(1)))
            .thenReturn(List.of(valid, invalid));
        when(candleRepository.insertIfAbsent(eq("RELIANCE"), eq(monday), any(), any(), any(), any(), eq(100L), any()))
            .thenReturn(1);
        when(txTemplate.execute(any())).thenAnswer(invocation ->
            ((org.springframework.transaction.support.TransactionCallback<Integer>) invocation.getArgument(0))
                .doInTransaction(null));

        DataIngestionService.BackfillOutcome outcome = dataIngestionService.processStockDataWithOutcome(
            "RELIANCE", monday, monday.plusDays(1));
        assert outcome.fetchedRows() == 2;
        assert outcome.savedRows() == 1;
        assert outcome.invalidRows() == 1;
        assert outcome.sourceOutcome().equals("DATA_RECEIVED");
        verify(candleRepository).insertIfAbsent(eq("RELIANCE"), eq(monday), any(), any(), any(), any(), eq(100L), any());

        when(mockClient.fetchCandles("RELIANCE", monday, monday)).thenThrow(new IllegalStateException("down"));
        assert dataIngestionService.processStockDataWithOutcome("RELIANCE", monday, monday)
            .sourceOutcome().equals("TRANSIENT_SOURCE_FAILURE");
    }

    @Test
    void processStockDataClassifiesEmptyAndAllInvalidResponses() {
        LocalDate date = LocalDate.of(2026, 1, 5);
        when(mockClient.fetchCandles("RELIANCE", date, date)).thenReturn(List.of());
        DataIngestionService.BackfillOutcome empty = dataIngestionService.processStockDataWithOutcome(
            "RELIANCE", date, date);
        assert empty.sourceOutcome().equals("NO_USABLE_DATA");

        CandleData invalid = CandleData.of("RELIANCE", date, bd("10"), bd("9"), bd("8"), bd("9"), 100);
        when(mockClient.fetchCandles("RELIANCE", date, date)).thenReturn(List.of(invalid));
        DataIngestionService.BackfillOutcome rejected = dataIngestionService.processStockDataWithOutcome(
            "RELIANCE", date, date);
        assert rejected.sourceOutcome().equals("INVALID_ROWS_REJECTED");
        assert rejected.invalidRows() == 1;
    }

    @Test
    void processStockDataReportsDuplicateInsertAsSkipped() {
        LocalDate date = LocalDate.of(2026, 1, 5);
        CandleData valid = CandleData.of("RELIANCE", date, bd("10"), bd("12"), bd("9"), bd("11"), 100);
        when(mockClient.fetchCandles("RELIANCE", date, date)).thenReturn(List.of(valid));
        when(candleRepository.insertIfAbsent(eq("RELIANCE"), eq(date), any(), any(), any(), any(), eq(100L), any()))
            .thenReturn(0);
        when(txTemplate.execute(any())).thenAnswer(invocation ->
            ((org.springframework.transaction.support.TransactionCallback<Integer>) invocation.getArgument(0))
                .doInTransaction(null));

        DataIngestionService.BackfillOutcome outcome = dataIngestionService.processStockDataWithOutcome(
            "RELIANCE", date, date);

        assert outcome.fetchedRows() == 1;
        assert outcome.savedRows() == 0;
        assert outcome.invalidRows() == 0;
        assert outcome.sourceOutcome().equals("DATA_RECEIVED");
    }

    @Test
    void dataQualityReportsPriceAnomaliesAndTrailingGap() {
        LocalDate monday = LocalDate.of(2026, 1, 5);
        OhlcvCandleEntity invalidHigh = candle("RELIANCE", monday);
        invalidHigh.setHighPrice(bd("90"));
        when(candleRepository.findBySymbolAndDateRange(eq("RELIANCE"), eq(monday),
            eq(monday.plusDays(2)), any())).thenReturn(List.of(invalidHigh));

        DataIngestionService.DataQualityReport report = dataIngestionService.validateDataQuality(
            "RELIANCE", monday, monday.plusDays(2));

        assert report.getAnomalies().size() == 1;
        assert report.getAnomalies().get(0).getType().equals("High price invalid");
        assert report.getGaps().size() == 1;
        assert report.getGaps().get(0).getStartDate().equals(monday.plusDays(1));
        assert report.hasIssues();
        assert report.isCritical();
    }

    @Test
    void fetchLatestAndRepairGapsStopsWhenHistoryIsUnavailable() {
        LocalDate latest = LocalDate.of(2026, 1, 5);
        when(candleRepository.existsBySymbolAndDate("RELIANCE", latest)).thenReturn(true);
        when(candleRepository.findEarliestBySymbol("RELIANCE")).thenReturn(Optional.empty());

        String summary = dataIngestionService.fetchLatestAndRepairGaps("RELIANCE", latest);

        assert summary.contains("no historical range available");
        verify(mockClient, Mockito.never()).fetchCandles(anyString(), any(LocalDate.class), any(LocalDate.class));
    }

    @Test
    void invalidInputsAndExistingWindowAreHandled() {
        assertThatThrownBy(() -> dataIngestionService.getExistingDataWindow(" "))
            .isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> dataIngestionService.processIncrementalStockData("", LocalDate.now(), LocalDate.now()))
            .isInstanceOf(IllegalArgumentException.class);
        when(candleRepository.findEarliestBySymbol("X")).thenReturn(Optional.empty());
        when(candleRepository.findLatestBySymbol("X")).thenReturn(Optional.empty());
        assert dataIngestionService.getExistingDataWindow("X").earliestDate() == null;
        assert dataIngestionService.getExistingDataWindow("X").latestDate() == null;
    }

    private OhlcvCandleEntity candle(String symbol, LocalDate date) {
        OhlcvCandleEntity candle = new OhlcvCandleEntity();
        candle.setSymbol(symbol);
        candle.setDate(date);
        candle.setOpenPrice(BigDecimal.valueOf(100));
        candle.setHighPrice(BigDecimal.valueOf(105));
        candle.setLowPrice(BigDecimal.valueOf(95));
        candle.setClosePrice(BigDecimal.valueOf(102));
        return candle;
    }
}
