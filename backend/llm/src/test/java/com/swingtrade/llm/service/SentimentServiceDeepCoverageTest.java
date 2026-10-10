package com.swingtrade.llm.service;

import com.swingtrade.core.metrics.LlmMetrics;
import com.swingtrade.core.metrics.SentimentMetrics;
import com.swingtrade.domain.NewsArticle;
import com.swingtrade.domain.PersistedNewsArticle;
import com.swingtrade.domain.SentimentResult;
import com.swingtrade.domain.Stock;
import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.domain.store.SentimentStore;
import com.swingtrade.domain.store.StockStore;
import com.swingtrade.llm.SentimentOutput;
import com.swingtrade.llm.SentimentType;
import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.config.SentimentPromptLoader;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import reactor.core.publisher.Mono;

import java.math.BigDecimal;
import java.time.DayOfWeek;
import java.time.LocalDate;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.temporal.TemporalAdjusters;
import java.util.List;
import java.util.Map;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class SentimentServiceDeepCoverageTest {

    private static final ZoneId MARKET_ZONE = ZoneId.of("Asia/Kolkata");

    @Mock private LlmClientProvider clientProvider;
    @Mock private LlmServerManagerProvider serverManagerProvider;
    @Mock private SentimentPromptLoader promptLoader;
    @Mock private SentimentAnalyzer sentimentAnalyzer;
    @Mock private NewsIngestionService newsIngestionService;
    @Mock private SentimentStore sentimentStore;
    @Mock private StockStore stockStore;
    @Mock private AppSettingsStore appSettingsStore;
    @Mock private LlmMetrics llmMetrics;
    @Mock private SentimentMetrics sentimentMetrics;
    @Mock private LlmClient llmClient;

    private SentimentService service;
    private final LocalDate decisionDate = LocalDate.of(2026, 9, 15);

    @BeforeEach
    void setUp() {
        service = new SentimentService(
                clientProvider, serverManagerProvider, promptLoader, sentimentAnalyzer,
                newsIngestionService, sentimentStore, stockStore, appSettingsStore,
                llmMetrics, sentimentMetrics, 0.75);
    }

    @AfterEach
    void tearDown() {
        service.shutdown();
    }

    @Test
    void analyzesMultipleStocksAndKeepsDefaultForOneStockThatErrors() {
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of());
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("INFY", decisionDate))
                .thenThrow(new IllegalStateException("news source unavailable"));

        Map<String, SentimentResult> results = service.analyzeMultipleStocks(
                List.of("TCS", "INFY"), decisionDate);

        assertThat(results).containsOnlyKeys("TCS", "INFY");
        assertThat(results.get("TCS").score()).isEqualTo(SentimentResult.SentimentScore.NEUTRAL);
        assertThat(results.get("INFY").score()).isEqualTo(SentimentResult.SentimentScore.NEUTRAL);
        assertThat(results.get("INFY").source()).isEqualTo("DEFAULT");
        verify(sentimentStore, never()).saveOrUpdate(any());
        verify(sentimentMetrics).recordFailed();
    }

    @Test
    void admitsOnlyArticlesInsideTheSevenDayMarketWindowAndCutoff() {
        NewsArticle exactlyAtLookback = article("TCS", "lookback", decisionDate.minusDays(7)
                .atStartOfDay(MARKET_ZONE));
        NewsArticle atCutoff = article("TCS", "cutoff", decisionDate.atTime(15, 30)
                .atZone(MARKET_ZONE));
        NewsArticle afterCutoff = article("TCS", "future", decisionDate.atTime(15, 31)
                .atZone(MARKET_ZONE));
        NewsArticle beforeLookback = article("TCS", "old", decisionDate.minusDays(7)
                .atStartOfDay(MARKET_ZONE).minusMinutes(1));
        NewsArticle undated = article("TCS", "undated", null);

        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of(
                        persisted(1, exactlyAtLookback), persisted(2, atCutoff),
                        persisted(3, afterCutoff), persisted(4, beforeLookback),
                        persisted(5, undated)));
        when(newsIngestionService.cleanNewsText(exactlyAtLookback)).thenReturn("lookback news");
        when(newsIngestionService.cleanNewsText(atCutoff)).thenReturn("cutoff news");
        configureSuccessfulLlm();

        SentimentResult result = service.analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.articleCount()).isEqualTo(2);
        assertThat(result.articleIds()).containsExactlyInAnyOrder(1L, 2L);
        verify(newsIngestionService).cleanNewsText(exactlyAtLookback);
        verify(newsIngestionService).cleanNewsText(atCutoff);
        verify(newsIngestionService, never()).cleanNewsText(afterCutoff);
        verify(newsIngestionService, never()).cleanNewsText(beforeLookback);
        verify(newsIngestionService, never()).cleanNewsText(undated);
        verify(llmClient).generateChatCompletion(anyList(), eq(512), eq(0.0));
    }

    @Test
    void returnsNeutralDefaultWhenNewsFetchFailsBeforeAnalysis() {
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenThrow(new RuntimeException("provider timeout"));

        SentimentResult result = service.analyzeStockSentiment("TCS", decisionDate);

        assertThat(result).extracting(SentimentResult::symbol, SentimentResult::date,
                        SentimentResult::score, SentimentResult::source)
                .containsExactly("TCS", decisionDate, SentimentResult.SentimentScore.NEUTRAL, "DEFAULT");
        assertThat(result.confidence()).isEqualTo(0.3);
        verify(sentimentMetrics).recordFailed();
        verify(sentimentStore, never()).saveOrUpdate(any());
    }

    @Test
    void groupsUnknownSymbolsAndRanksPositiveAndNegativeSectors() {
        SentimentResult bankPositive = result("HDFCBANK", SentimentResult.SentimentScore.POSITIVE);
        SentimentResult bankNegative = result("ICICIBANK", SentimentResult.SentimentScore.NEGATIVE);
        SentimentResult itPositiveOne = result("TCS", SentimentResult.SentimentScore.POSITIVE);
        SentimentResult itPositiveTwo = result("INFY", SentimentResult.SentimentScore.POSITIVE);
        SentimentResult unknownNegative = result("MYSTERY", SentimentResult.SentimentScore.NEGATIVE);
        when(stockStore.findBySymbol("HDFCBANK")).thenReturn(Optional.of(stock(Stock.Sector.BANK)));
        when(stockStore.findBySymbol("ICICIBANK")).thenReturn(Optional.of(stock(Stock.Sector.BANK)));
        when(stockStore.findBySymbol("TCS")).thenReturn(Optional.of(stock(Stock.Sector.IT)));
        when(stockStore.findBySymbol("INFY")).thenReturn(Optional.of(stock(Stock.Sector.IT)));
        when(stockStore.findBySymbol("MYSTERY")).thenReturn(Optional.empty());

        Map<Stock.Sector, Map<SentimentResult.SentimentScore, Long>> grouped =
                service.groupBySectorAndSentiment(List.of(
                        bankPositive, bankNegative, itPositiveOne, itPositiveTwo, unknownNegative));

        assertThat(grouped.get(Stock.Sector.BANK))
                .containsEntry(SentimentResult.SentimentScore.POSITIVE, 1L)
                .containsEntry(SentimentResult.SentimentScore.NEGATIVE, 1L);
        assertThat(grouped.get(Stock.Sector.IT))
                .containsEntry(SentimentResult.SentimentScore.POSITIVE, 2L);
        assertThat(grouped.get(Stock.Sector.OTHERS))
                .containsEntry(SentimentResult.SentimentScore.NEGATIVE, 1L);
        assertThat(service.getTopSectors(grouped, 2, true))
                .containsExactly(Stock.Sector.IT, Stock.Sector.BANK);
        assertThat(service.getTopSectors(grouped, 2, false))
                .containsExactlyInAnyOrder(Stock.Sector.BANK, Stock.Sector.OTHERS);
        assertThat(service.getSectorForSymbol("MYSTERY")).isNull();
    }

    @Test
    void generatesLastWeekDigestUsingStoreRangeAndIncludesUnknownSector() {
        LocalDate today = LocalDate.now(MARKET_ZONE);
        LocalDate endDate = today.with(TemporalAdjusters.previousOrSame(DayOfWeek.SUNDAY));
        if (!today.equals(endDate)) {
            endDate = today.minusDays(1).with(TemporalAdjusters.previousOrSame(DayOfWeek.SUNDAY));
        }
        LocalDate startDate = endDate.minusDays(6);
        SentimentResult positive = result("TCS", SentimentResult.SentimentScore.POSITIVE);
        SentimentResult unknown = result("UNKNOWN", SentimentResult.SentimentScore.NEUTRAL);
        when(sentimentStore.findAllByDateBetween(startDate, endDate)).thenReturn(List.of(positive, unknown));
        when(stockStore.findBySymbol("TCS")).thenReturn(Optional.of(stock(Stock.Sector.IT)));
        when(stockStore.findBySymbol("UNKNOWN")).thenReturn(Optional.empty());

        String digest = service.generateSectorDigestForLastWeek();

        assertThat(digest).contains("Week of: " + startDate + " to " + endDate,
                "IT - 1 POS", "OTHERS - 0 POS, 1 NEU, 0 NEG", "Total stocks analyzed: 2");
        verify(sentimentStore).findAllByDateBetween(startDate, endDate);
    }

    private void configureSuccessfulLlm() {
        when(promptLoader.getSystemPrompt()).thenReturn("system");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}");
        when(clientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.LOCAL);
        when(clientProvider.getClient()).thenReturn(llmClient);
        when(llmClient.generateChatCompletion(anyList(), eq(512), eq(0.0)))
                .thenReturn(Mono.just("response"));
        when(sentimentAnalyzer.parseResponse(any(), anyInt()))
                .thenReturn(new SentimentOutput(SentimentType.POSITIVE, "good", 0.8));
        when(appSettingsStore.get("llamacpp.model")).thenReturn(Optional.empty());
    }

    private NewsArticle article(String symbol, String title, ZonedDateTime publishedDate) {
        return new NewsArticle(symbol, title, "https://example.test/" + title,
                null, publishedDate, "Reuters", null);
    }

    private PersistedNewsArticle persisted(long id, NewsArticle article) {
        return new PersistedNewsArticle(id, article, null);
    }

    private SentimentResult result(String symbol, SentimentResult.SentimentScore score) {
        return SentimentResult.create(symbol, decisionDate, score, "summary", "", 0.7);
    }

    private Stock stock(Stock.Sector sector) {
        return new Stock("TEST", Stock.Exchange.NSE, "Test", sector, "Industry", 1L,
                BigDecimal.ONE, "IN0000000001", 1, decisionDate);
    }
}
