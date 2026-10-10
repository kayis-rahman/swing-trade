package com.swingtrade.llm.service;

import com.swingtrade.domain.NewsArticle;
import com.swingtrade.domain.PersistedNewsArticle;
import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.domain.store.SentimentStore;
import com.swingtrade.domain.store.StockStore;
import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.config.SentimentPromptLoader;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDate;
import java.time.LocalTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import reactor.core.publisher.Mono;
import com.swingtrade.llm.domain.EarningsData;
import java.math.BigDecimal;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.ArgumentMatchers.intThat;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doNothing;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class SentimentServicePromptTest {

    @Mock private LlmClientProvider clientProvider;
    @Mock private LlmServerManagerProvider serverManagerProvider;
    @Mock private LlmServerManager serverManager;
    @Mock private LlmClient llmClient;
    @Mock private SentimentPromptLoader promptLoader;
    @Mock private SentimentStore sentimentStore;
    @Mock private StockStore stockStore;
    @Mock private AppSettingsStore appSettingsStore;
    @Mock private NewsIngestionService newsIngestionService;
    @Mock private PdfExtractionService pdfExtractionService;

    private SentimentService service;

    @BeforeEach
    void setUp() {
        service = new SentimentService(
                clientProvider, serverManagerProvider, promptLoader, null,
                newsIngestionService, sentimentStore, stockStore, appSettingsStore,
                org.mockito.Mockito.mock(com.swingtrade.core.metrics.LlmMetrics.class),
                org.mockito.Mockito.mock(com.swingtrade.core.metrics.SentimentMetrics.class),
                pdfExtractionService, 0.75);
        when(serverManagerProvider.getManager()).thenReturn(serverManager);
        doNothing().when(serverManager).ensureRunning();
        when(clientProvider.getClient()).thenReturn(llmClient);
        when(appSettingsStore.get("llamacpp.model")).thenReturn(Optional.of("Qwen3-4B-Instruct"));
    }

    @Test
    @DisplayName("SentimentService uses loaded system prompt in LLM call")
    void usesLoadedSystemPrompt() {
        // Given
        String systemPrompt = "You are a financial analyst specialising in Indian equity markets.";
        String userPromptTemplate = "Analyse the following for a swing trade entry decision on {symbol}.";
        when(promptLoader.getSystemPrompt()).thenReturn(systemPrompt);
        when(promptLoader.getUserPrompt()).thenReturn(userPromptTemplate);

        String llmResponse = """
            {"score": "POSITIVE", "confidence": 0.8, "summary": "Strong news",
             "red_flags": [], "catalysts": []}
            """;
        when(llmClient.generateChatCompletion(anyList(), eq(2048), eq(0.0)))
            .thenReturn(Mono.just(llmResponse));

        List<PersistedNewsArticle> articles = List.of(
                new PersistedNewsArticle(1L, new NewsArticle("RELIANCE", "Test headline", "Test URL", null, todayNoon(), "Test source", null), null));
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate(eq("RELIANCE"), org.mockito.ArgumentMatchers.nullable(LocalDate.class))).thenReturn(articles);
        when(newsIngestionService.cleanNewsText(any(NewsArticle.class))).thenReturn("Test news content");

        // When
        service.analyzeStockSentiment("RELIANCE", LocalDate.now(ZoneId.of("Asia/Kolkata")));

        // Then — verify messages passed to LLM contain the loaded prompts
        verify(llmClient).generateChatCompletion(
            argThat(msgs -> {
                Map<String, String> sysMsg = msgs.get(0);
                return "system".equals(sysMsg.get("role"))
                    && sysMsg.get("content").contains("financial analyst specialising in Indian equity markets");
            }),
            eq(2048),
            eq(0.0)
        );
    }

    @Test
    @DisplayName("SentimentService formats user prompt with symbol and news content")
    void formatsUserPromptWithPlaceholders() {
        // Given
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("Analyse {symbol}. News: {newsContent}.");

        String llmResponse = """
            {"score": "NEGATIVE", "confidence": 0.6, "summary": "Bad news",
             "red_flags": ["SEBI probe"], "catalysts": []}
            """;
        when(llmClient.generateChatCompletion(anyList(), eq(2048), eq(0.0)))
            .thenReturn(Mono.just(llmResponse));

        List<PersistedNewsArticle> articles = List.of(
                new PersistedNewsArticle(1L, new NewsArticle("TCS", "Test headline", "Test URL", null, todayNoon(), "Test source", null), null));
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate(eq("TCS"), org.mockito.ArgumentMatchers.nullable(LocalDate.class))).thenReturn(articles);
        when(newsIngestionService.cleanNewsText(any(NewsArticle.class))).thenReturn("news content");

        // When
        service.analyzeStockSentiment("TCS", LocalDate.now(ZoneId.of("Asia/Kolkata")));

        // Then — verify the formatted message contains the symbol
        verify(llmClient).generateChatCompletion(
            argThat(msgs -> {
                Map<String, String> userMsg = msgs.get(1);
                return "user".equals(userMsg.get("role"))
                    && userMsg.get("content").contains("TCS");
            }),
            eq(2048),
            eq(0.0)
        );
    }

    @Test
    @DisplayName("SentimentService uses the bounded Pi response budget")
    void usesBoundedPiResponseBudget() {
        when(clientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.PI_SSH);
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("Analyse {symbol}. News: {newsContent}.");
        when(llmClient.generateChatCompletion(anyList(), eq(128), eq(0.0)))
                .thenReturn(Mono.just("{\"score\":\"NEUTRAL\",\"confidence\":0.5,\"summary\":\"Mixed\",\"red_flags\":[],\"catalysts\":[]}"));
        List<PersistedNewsArticle> articles = List.of(
                new PersistedNewsArticle(1L, new NewsArticle("TCS", "Test headline", "Test URL", null, todayNoon(), "Test source", null), null));
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate(eq("TCS"), org.mockito.ArgumentMatchers.nullable(LocalDate.class))).thenReturn(articles);
        when(newsIngestionService.cleanNewsText(any(NewsArticle.class))).thenReturn("news content");

        service.analyzeStockSentiment("TCS", LocalDate.now(ZoneId.of("Asia/Kolkata")));

        verify(llmClient).generateChatCompletion(anyList(), eq(128), eq(0.0));
    }

    @Test
    @DisplayName("Sentiment completion budget leaves room for reasoning before the JSON object")
    void requestsABudgetReasoningCannotConsume() {
        // Real-shape regression for the stage's truncated sentiment responses.
        // max_tokens is a combined reasoning-plus-content budget on the reasoning
        // backend this pipeline runs against, so the reasoning text is reserved
        // out of the same ceiling as the JSON object. Measured by replaying a real
        // 4.8k-char news prompt: 512 tokens gave finish_reason=length with 511
        // reasoning tokens and 0 content chars; 1024 gave a complete object. The
        // truncated captures in the field (304 and 475 chars, cut off mid-string)
        // are what a partially-consumed budget looks like.
        when(clientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.OPENAI);
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("Analyse {symbol}. News: {newsContent}.");
        when(llmClient.generateChatCompletion(anyList(), intThat(tokens -> tokens >= 2048), eq(0.0)))
                .thenReturn(Mono.just("{\"score\":\"NEUTRAL\",\"confidence\":0.5,\"summary\":\"Mixed\",\"red_flags\":[],\"catalysts\":[]}"));
        List<PersistedNewsArticle> articles = List.of(
                new PersistedNewsArticle(1L, new NewsArticle("TCS", "Test headline", "Test URL", null, todayNoon(), "Test source", null), null));
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate(eq("TCS"), org.mockito.ArgumentMatchers.nullable(LocalDate.class))).thenReturn(articles);
        when(newsIngestionService.cleanNewsText(any(NewsArticle.class))).thenReturn("news content");

        service.analyzeStockSentiment("TCS", LocalDate.now(ZoneId.of("Asia/Kolkata")));

        verify(llmClient).generateChatCompletion(anyList(),
                intThat(tokens -> tokens >= 2048), eq(0.0));
        assertThat(SentimentService.DEFAULT_MAX_RESPONSE_TOKENS).isGreaterThanOrEqualTo(2048);
    }

    @Test
    @DisplayName("Sentiment completion budget fits local model context windows")
    void boundsLocalCompletionBudget() {
        when(clientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.LOCAL);
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("Analyse {symbol}. News: {newsContent}.");
        when(llmClient.generateChatCompletion(anyList(), eq(512), eq(0.0)))
                .thenReturn(Mono.just("{\"score\":\"NEUTRAL\",\"confidence\":0.5,\"summary\":\"Mixed\",\"red_flags\":[],\"catalysts\":[]}"));
        List<PersistedNewsArticle> articles = List.of(
                new PersistedNewsArticle(1L, new NewsArticle("TCS", "Test headline", "Test URL", null, todayNoon(), "Test source", null), null));
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate(eq("TCS"), org.mockito.ArgumentMatchers.nullable(LocalDate.class))).thenReturn(articles);
        when(newsIngestionService.cleanNewsText(any(NewsArticle.class))).thenReturn("news content");

        service.analyzeStockSentiment("TCS", LocalDate.now(ZoneId.of("Asia/Kolkata")));

        verify(llmClient).generateChatCompletion(anyList(), eq(512), eq(0.0));
    }

    @Test
    @DisplayName("SentimentService excludes articles published after the decision cutoff")
    void filtersFutureArticlesBeforePrompting() {
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}");
        when(llmClient.generateChatCompletion(anyList(), eq(2048), eq(0.0)))
            .thenReturn(Mono.just("{\"score\":\"NEUTRAL\",\"confidence\":0.5,\"summary\":\"Mixed\"}"));

        LocalDate decisionDate = LocalDate.of(2026, 9, 15);
        NewsArticle beforeCutoff = new NewsArticle("TCS", "Old", "u1", null,
            ZonedDateTime.of(2026, 9, 15, 15, 30, 0, 0, ZoneId.of("Asia/Kolkata")), "source", null);
        NewsArticle afterCutoff = new NewsArticle("TCS", "Future", "u2", null,
            ZonedDateTime.of(2026, 9, 15, 15, 31, 0, 0, ZoneId.of("Asia/Kolkata")), "source", null);
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate(eq("TCS"), eq(decisionDate)))
            .thenReturn(List.of(
                new PersistedNewsArticle(1L, beforeCutoff, null),
                new PersistedNewsArticle(2L, afterCutoff, null)));
        when(newsIngestionService.cleanNewsText(beforeCutoff)).thenReturn("old news");

        service.analyzeStockSentiment("TCS", decisionDate);

        verify(sentimentStore).saveOrUpdate(argThat(result -> result.articleCount() == 1));
        verify(sentimentStore).saveOrUpdate(argThat(result -> result.articleIds().contains(1L)));
        verify(newsIngestionService).cleanNewsText(beforeCutoff);
        org.mockito.Mockito.verify(newsIngestionService, org.mockito.Mockito.never())
            .cleanNewsText(afterCutoff);
        verify(llmClient).generateChatCompletion(
            argThat(msgs -> msgs.get(1).get("content").contains("old news")
                && !msgs.get(1).get("content").contains("future news")),
            eq(2048), eq(0.0));
        org.mockito.Mockito.verify(newsIngestionService, org.mockito.Mockito.never())
            .fetchStockNews("TCS");
    }

    @Test
    @DisplayName("SentimentService adds bounded structured Indian-market context when available")
    void addsStructuredMarketContext() {
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}\nContext: {marketContext}");
        when(llmClient.generateChatCompletion(anyList(), eq(2048), eq(0.0)))
                .thenReturn(Mono.just("{\"score\":\"NEUTRAL\",\"confidence\":0.5,\"summary\":\"Mixed\"}"));
        LocalDate decisionDate = LocalDate.of(2026, 9, 15);
        NewsArticle article = new NewsArticle("TCS", "Headline", "url", null,
                decisionDate.atTime(12, 0).atZone(ZoneId.of("Asia/Kolkata")), "source", null);
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of(new PersistedNewsArticle(1L, article, null)));
        when(newsIngestionService.cleanNewsText(article)).thenReturn("news content");
        when(pdfExtractionService.extractLatestEarnings("TCS")).thenReturn(new EarningsData(
                "TCS", "Q1 FY27", new BigDecimal("100"), new BigDecimal("20"),
                new BigDecimal("5"), new BigDecimal("30"), "stable guidance", decisionDate));
        when(newsIngestionService.fetchStructuredFilings("TCS")).thenReturn(List.of(
                new StructuredFiling(StructuredFiling.FilingType.DIVIDEND, decisionDate,
                        "Dividend declared", "Board approved dividend", "link")));

        service.analyzeStockSentiment("TCS", decisionDate);

        verify(llmClient).generateChatCompletion(argThat(messages -> {
            String prompt = messages.get(1).get("content");
            return prompt.contains("Structured Indian-market context")
                    && prompt.contains("Q1 FY27")
                    && prompt.contains("Dividend declared")
                    && prompt.length() < 3000;
        }), eq(2048), eq(0.0));
    }

    @Test
    @DisplayName("Keyword fallback preserves the requested historical analysis date")
    void keywordFallbackPreservesHistoricalDate() {
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}");
        when(llmClient.generateChatCompletion(anyList(), eq(2048), eq(0.0)))
                .thenReturn(Mono.error(new IllegalStateException("LLM unavailable")));

        LocalDate decisionDate = LocalDate.of(2026, 9, 15);
        NewsArticle article = new NewsArticle("TCS", "Shares surge after strong results", "url",
                null, decisionDate.atTime(12, 0).atZone(ZoneId.of("Asia/Kolkata")), "source", null);
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of(new PersistedNewsArticle(1L, article, null)));
        when(newsIngestionService.cleanNewsText(article)).thenReturn("Shares surge after strong results");

        service.analyzeStockSentiment("TCS", decisionDate);

        verify(sentimentStore).saveOrUpdate(argThat(result ->
                decisionDate.equals(result.date()) && "KEYWORD_FALLBACK".equals(result.source())));
    }

    @Test
    @DisplayName("Empty LLM responses use the auditable keyword fallback")
    void emptyLlmResponseUsesKeywordFallback() {
        when(promptLoader.getSystemPrompt()).thenReturn("System prompt");
        when(promptLoader.getUserPrompt()).thenReturn("News: {newsContent}");
        when(llmClient.generateChatCompletion(anyList(), eq(2048), eq(0.0)))
                .thenReturn(Mono.empty());

        LocalDate decisionDate = LocalDate.of(2026, 9, 15);
        NewsArticle article = new NewsArticle("TCS", "Shares surge after strong results", "url",
                null, decisionDate.atTime(12, 0).atZone(ZoneId.of("Asia/Kolkata")), "source", null);
        when(newsIngestionService.fetchPersistedStockNewsForDecisionDate("TCS", decisionDate))
                .thenReturn(List.of(new PersistedNewsArticle(1L, article, null)));
        when(newsIngestionService.cleanNewsText(article)).thenReturn("Shares surge after strong results");

        service.analyzeStockSentiment("TCS", decisionDate);

        verify(sentimentStore).saveOrUpdate(argThat(result -> "KEYWORD_FALLBACK".equals(result.source())));
    }

    private static ZonedDateTime todayNoon() {
        ZoneId zone = ZoneId.of("Asia/Kolkata");
        return LocalDate.now(zone).atTime(LocalTime.NOON).atZone(zone);
    }
}
