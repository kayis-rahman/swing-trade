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
import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.config.SentimentPromptLoader;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import reactor.core.publisher.Mono;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.List;
import java.util.Map;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class SentimentServiceCoverageTest {

    @Mock private LlmClientProvider clientProvider;
    @Mock private LlmServerManagerProvider serverManagerProvider;
    @Mock private SentimentPromptLoader promptLoader;
    @Mock private SentimentAnalyzer sentimentAnalyzer;
    @Mock private NewsIngestionService newsIngestionService;
    @Mock private SentimentStore sentimentStore;
    @Mock private StockStore stockStore;
    @Mock private AppSettingsStore appSettingsStore;
    @Mock private LlmClient llmClient;

    private SentimentService service;
    private final LocalDate decisionDate = LocalDate.of(2026, 9, 15);

    @BeforeEach
    void setUp() {
        service = new SentimentService(
                clientProvider, serverManagerProvider, promptLoader, sentimentAnalyzer,
                newsIngestionService, sentimentStore, stockStore, appSettingsStore,
                org.mockito.Mockito.mock(LlmMetrics.class),
                org.mockito.Mockito.mock(SentimentMetrics.class),
                null, 0.75);
    }

    @Test
    void returnsNeutralDefaultWhenNoArticlesAreInTheDecisionWindow() {
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of());

        SentimentResult result = service.analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.NEUTRAL);
        assertThat(result.source()).isEqualTo("DEFAULT");
        assertThat(result.confidence()).isEqualTo(0.3);
        verify(llmClient, never()).generateChatCompletion(anyList(), any(Integer.class), any(Double.class));
        verify(sentimentStore, never()).saveOrUpdate(any());
    }

    @Test
    void returnsNeutralDefaultWhenAllArticleTextIsRemovedByCleaning() {
        NewsArticle article = article("TCS", "Headline", "Reuters");
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of(new PersistedNewsArticle(7L, article, null)));
        when(newsIngestionService.cleanNewsText(article)).thenReturn("  ");

        SentimentResult result = service.analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.NEUTRAL);
        assertThat(result.summary()).contains("No news articles available");
        verify(newsIngestionService).cleanNewsText(article);
        verify(sentimentStore, never()).saveOrUpdate(any());
    }

    @Test
    void usesKeywordFallbackAndContinuesWhenPersistenceFails() {
        NewsArticle positive = article("RELIANCE", "Positive", "NSE");
        NewsArticle negative = article("RELIANCE", "Negative", "BSE");
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("RELIANCE", decisionDate))
                .thenReturn(List.of(
                        new PersistedNewsArticle(1L, positive, null),
                        new PersistedNewsArticle(2L, negative, null)));
        when(newsIngestionService.cleanNewsText(positive)).thenReturn("Shares surge after partnership and upgrade");
        when(newsIngestionService.cleanNewsText(negative)).thenReturn("Company faces fraud probe");
        when(newsIngestionService.fetchStructuredFilings(any())).thenReturn(List.of());
        when(clientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.LOCAL);
        when(promptLoader.getSystemPrompt()).thenReturn("system");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}");
        when(newsIngestionService.fetchStructuredFilings(any())).thenReturn(List.of());
        when(clientProvider.getClient()).thenReturn(llmClient);
        when(llmClient.generateChatCompletion(anyList(), eq(512), eq(0.0)))
                .thenReturn(Mono.error(new IllegalStateException("provider unavailable")));
        doThrow(new IllegalStateException("database unavailable"))
                .when(sentimentStore).saveOrUpdate(any());

        SentimentResult result = service.analyzeStockSentiment("RELIANCE", decisionDate);

        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.NEUTRAL);
        assertThat(result.source()).isEqualTo("KEYWORD_FALLBACK");
        assertThat(result.degradedReason()).contains("provider unavailable");
        assertThat(result.redFlags()).containsExactly("Regulatory/legal investigation");
        assertThat(result.catalysts()).contains("M&A/Partnership catalyst", "Analyst positive action");
        verify(sentimentStore).saveOrUpdate(any());
    }

    @Test
    void usesPiLimitsForArticleCountAndResponseTokens() {
        when(clientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.PI_SSH);
        when(clientProvider.getClient()).thenReturn(llmClient);
        when(promptLoader.getSystemPrompt()).thenReturn("system");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}");
        when(sentimentAnalyzer.parseResponse(any(), eq(6)))
                .thenReturn(new com.swingtrade.llm.SentimentOutput(
                        com.swingtrade.llm.SentimentType.POSITIVE, "good", 0.8));
        when(llmClient.generateChatCompletion(anyList(), eq(128), eq(0.0)))
                .thenReturn(Mono.just("response"));

        List<PersistedNewsArticle> articles = java.util.stream.IntStream.rangeClosed(1, 7)
                .mapToObj(index -> {
                    NewsArticle article = article("TCS", "Headline " + index, "source");
                    return new PersistedNewsArticle(index, article, null);
                }).toList();
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(articles);
        for (PersistedNewsArticle persisted : articles) {
            when(newsIngestionService.cleanNewsText(persisted.article())).thenReturn("news " + persisted.id());
        }

        SentimentResult result = service.analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.articleCount()).isEqualTo(6);
        verify(llmClient).generateChatCompletion(anyList(), eq(128), eq(0.0));
        verify(sentimentAnalyzer).parseResponse(any(), eq(6));
    }

    @Test
    void formatsEmptyAndGroupedSectorDigestsIncludingUnknownSymbols() {
        when(sentimentStore.findAllByDateBetween(decisionDate.minusDays(6), decisionDate))
                .thenReturn(List.of());
        String empty = service.generateSectorDigest(decisionDate.minusDays(6), decisionDate);
        assertThat(empty).contains("No sentiment data available", "Total stocks analyzed: 0");

        SentimentResult positive = result("TCS", SentimentResult.SentimentScore.POSITIVE);
        SentimentResult negative = result("UNKNOWN", SentimentResult.SentimentScore.NEGATIVE);
        when(sentimentStore.findAllByDateBetween(decisionDate.minusDays(6), decisionDate))
                .thenReturn(List.of(positive, negative));
        when(stockStore.findBySymbol("TCS")).thenReturn(Optional.of(stock(Stock.Sector.IT)));
        when(stockStore.findBySymbol("UNKNOWN")).thenReturn(Optional.empty());

        String digest = service.generateSectorDigest(decisionDate.minusDays(6), decisionDate);

        assertThat(digest).contains("IT - 1 POS", "OTHERS - 0 POS, 0 NEU, 1 NEG",
                "Total stocks analyzed: 2", "Positive signals: 1", "Negative: 1");
        assertThat(service.groupBySectorAndSentiment(List.of(positive, negative)))
                .containsKeys(Stock.Sector.IT, Stock.Sector.OTHERS);
    }

    private NewsArticle article(String symbol, String title, String source) {
        return new NewsArticle(symbol, title, "https://example.test/" + title,
                null, decisionDate.atTime(12, 0).atZone(ZoneId.of("Asia/Kolkata")), source, null);
    }

    private SentimentResult result(String symbol, SentimentResult.SentimentScore score) {
        return SentimentResult.create(symbol, decisionDate, score, "summary", "", 0.7);
    }

    private Stock stock(Stock.Sector sector) {
        return new Stock("TCS", Stock.Exchange.NSE, "Test", sector, "Industry", 1L,
                BigDecimal.ONE, "IN0000000001", 1, decisionDate);
    }
}
