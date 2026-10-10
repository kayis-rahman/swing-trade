package com.swingtrade.llm.service;

import com.swingtrade.core.metrics.LlmMetrics;
import com.swingtrade.core.metrics.SentimentMetrics;
import com.swingtrade.data.entity.SentimentClassificationLogEntity;
import com.swingtrade.data.repository.SentimentClassificationLogRepository;
import com.swingtrade.domain.NewsArticle;
import com.swingtrade.domain.PersistedNewsArticle;
import com.swingtrade.domain.SentimentResult;
import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.domain.store.SentimentStore;
import com.swingtrade.domain.store.StockStore;
import com.swingtrade.llm.SentimentOutput;
import com.swingtrade.llm.SentimentType;
import com.swingtrade.llm.client.LayaClient;
import com.swingtrade.llm.client.LayaResult;
import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.config.LayaProperties;
import com.swingtrade.llm.config.SentimentPromptLoader;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import reactor.core.publisher.Mono;

import java.time.LocalDate;
import java.time.ZoneId;
import java.util.List;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

/**
 * Covers {@code SentimentService}'s Laya local pre-filter branching, added
 * ahead of the existing Qwen/LLM call in {@code performSentimentAnalysis}.
 */
@ExtendWith(MockitoExtension.class)
class SentimentServiceLayaTest {

    @Mock private LlmClientProvider clientProvider;
    @Mock private LlmServerManagerProvider serverManagerProvider;
    @Mock private SentimentPromptLoader promptLoader;
    @Mock private SentimentAnalyzer sentimentAnalyzer;
    @Mock private NewsIngestionService newsIngestionService;
    @Mock private SentimentStore sentimentStore;
    @Mock private StockStore stockStore;
    @Mock private AppSettingsStore appSettingsStore;
    @Mock private LlmClient llmClient;
    @Mock private LayaClient layaClient;
    @Mock private SentimentClassificationLogRepository classificationLogRepository;

    private final LocalDate decisionDate = LocalDate.of(2026, 9, 15);
    private LayaProperties layaProperties;

    @BeforeEach
    void setUp() {
        layaProperties = new LayaProperties();
    }

    private SentimentService newService() {
        return new SentimentService(
                clientProvider, serverManagerProvider, promptLoader, sentimentAnalyzer,
                newsIngestionService, sentimentStore, stockStore, appSettingsStore,
                mock(LlmMetrics.class), mock(SentimentMetrics.class),
                null, null, null,
                layaClient, layaProperties, classificationLogRepository, 0.75);
    }

    /** Wires the article-fetch and Qwen-call mocks common to every test here. */
    private void stubArticleAndQwenCall(SentimentType qwenSentiment) {
        NewsArticle article = new NewsArticle("TCS", "Headline", "https://example.test/1",
                null, decisionDate.atTime(12, 0).atZone(ZoneId.of("Asia/Kolkata")), "Reuters", null);
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of(new PersistedNewsArticle(1L, article, null)));
        when(newsIngestionService.cleanNewsText(article)).thenReturn("Company reports record profit");
        when(newsIngestionService.fetchStructuredFilings(any())).thenReturn(List.of());
        when(clientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.LOCAL);
        when(clientProvider.getClient()).thenReturn(llmClient);
        when(promptLoader.getSystemPrompt()).thenReturn("system");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}");
        when(llmClient.generateChatCompletion(anyList(), eq(512), eq(0.0)))
                .thenReturn(Mono.just("qwen-response"));
        when(sentimentAnalyzer.parseResponse(any(), eq(1)))
                .thenReturn(new SentimentOutput(qwenSentiment, "qwen reasoning", 0.6));
    }

    @Test
    void shadowModeLogsBothAndReturnsQwenResult() {
        layaProperties.setEnabled(true);
        layaProperties.setShadowMode(true);
        stubArticleAndQwenCall(SentimentType.NEGATIVE);
        when(layaClient.classify(any())).thenReturn(Optional.of(new LayaResult(SentimentType.POSITIVE, 0.95)));

        SentimentResult result = newService().analyzeStockSentiment("TCS", decisionDate);

        // Shadow mode always returns Qwen's result unchanged, regardless of Laya's output.
        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.NEGATIVE);
        assertThat(result.source()).isEqualTo("LLM");
        verify(layaClient).classify(any());
        verify(classificationLogRepository, times(2)).save(any());
        verify(classificationLogRepository).save(argThatModelUsed(SentimentClassificationLogEntity.ModelUsed.LAYA, true));
        verify(classificationLogRepository).save(argThatModelUsed(SentimentClassificationLogEntity.ModelUsed.QWEN, true));
    }

    @Test
    void liveModeHighConfidenceLayaSkipsQwen() {
        layaProperties.setEnabled(true);
        layaProperties.setShadowMode(false);
        layaProperties.setConfidenceThreshold(0.75);
        NewsArticle article = new NewsArticle("TCS", "Headline", "https://example.test/1",
                null, decisionDate.atTime(12, 0).atZone(ZoneId.of("Asia/Kolkata")), "Reuters", null);
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of(new PersistedNewsArticle(1L, article, null)));
        when(newsIngestionService.cleanNewsText(article)).thenReturn("Company reports record profit");
        when(layaClient.classify(any())).thenReturn(Optional.of(new LayaResult(SentimentType.POSITIVE, 0.9)));

        SentimentResult result = newService().analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.POSITIVE);
        assertThat(result.source()).isEqualTo("LAYA");
        verify(llmClient, never()).generateChatCompletion(anyList(), any(Integer.class), any(Double.class));
        verify(classificationLogRepository, times(1)).save(any());
        verify(classificationLogRepository).save(argThatModelUsed(SentimentClassificationLogEntity.ModelUsed.LAYA, false));
    }

    @Test
    void liveModeLowConfidenceLayaFallsBackToQwen() {
        layaProperties.setEnabled(true);
        layaProperties.setShadowMode(false);
        layaProperties.setConfidenceThreshold(0.75);
        stubArticleAndQwenCall(SentimentType.NEUTRAL);
        when(layaClient.classify(any())).thenReturn(Optional.of(new LayaResult(SentimentType.POSITIVE, 0.5)));

        SentimentResult result = newService().analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.NEUTRAL);
        assertThat(result.source()).isEqualTo("LLM");
        verify(llmClient).generateChatCompletion(anyList(), eq(512), eq(0.0));
        verify(classificationLogRepository, times(2)).save(any());
        verify(classificationLogRepository).save(argThatModelUsed(SentimentClassificationLogEntity.ModelUsed.LAYA, false));
        verify(classificationLogRepository).save(argThatModelUsed(SentimentClassificationLogEntity.ModelUsed.QWEN, false));
    }

    @Test
    void layaUnreachableFallsBackToQwen() {
        layaProperties.setEnabled(true);
        layaProperties.setShadowMode(false);
        stubArticleAndQwenCall(SentimentType.POSITIVE);
        when(layaClient.classify(any())).thenReturn(Optional.empty());

        SentimentResult result = newService().analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.POSITIVE);
        assertThat(result.source()).isEqualTo("LLM");
        verify(llmClient).generateChatCompletion(anyList(), eq(512), eq(0.0));
        // No Laya row logged (nothing to compare), but Qwen's run is still logged.
        verify(classificationLogRepository, times(1)).save(any());
        verify(classificationLogRepository).save(argThatModelUsed(SentimentClassificationLogEntity.ModelUsed.QWEN, false));
    }

    @Test
    void layaDisabledBehavesIdenticallyToBeforeChange() {
        layaProperties.setEnabled(false);
        stubArticleAndQwenCall(SentimentType.POSITIVE);

        SentimentResult result = newService().analyzeStockSentiment("TCS", decisionDate);

        assertThat(result.score()).isEqualTo(SentimentResult.SentimentScore.POSITIVE);
        assertThat(result.source()).isEqualTo("LLM");
        verify(layaClient, never()).classify(any());
        verify(classificationLogRepository, never()).save(any());
    }

    private SentimentClassificationLogEntity argThatModelUsed(
            SentimentClassificationLogEntity.ModelUsed model, boolean shadowMode) {
        return org.mockito.ArgumentMatchers.argThat(entity ->
                entity != null
                        && model.name().equals(entity.getModelUsed())
                        && shadowMode == Boolean.TRUE.equals(entity.getShadowMode()));
    }
}
