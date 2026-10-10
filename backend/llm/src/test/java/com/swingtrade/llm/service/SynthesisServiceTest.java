package com.swingtrade.llm.service;

import com.swingtrade.domain.CompositeAnalysis;
import com.swingtrade.domain.SynthesisResult;
import com.swingtrade.llm.config.SynthesisPromptLoader;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import reactor.core.publisher.Mono;

import java.math.BigDecimal;
import java.time.LocalDate;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyDouble;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.anyList;
import static org.mockito.ArgumentMatchers.argThat;
import static org.mockito.ArgumentMatchers.intThat;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.mockito.Mockito.lenient;

/**
 * Unit tests for SynthesisService testing BeanOutputConverter integration,
 * fallback parsing, and error handling.
 */
@ExtendWith(MockitoExtension.class)
class SynthesisServiceTest {

    @Mock
    private LlmClientProvider llmClientProvider;
    @Mock
    private com.swingtrade.llm.client.LlmClient llmClient;
    @Mock
    private SynthesisPromptLoader promptLoader;
    @Mock
    private LlmServerManagerProvider serverManagerProvider;
    @Mock
    private SynthesisEvaluationService evaluationService;

    private SynthesisService service;
    private CompositeAnalysis composite;

    @BeforeEach
    void setUp() {
        when(promptLoader.getSystemPrompt()).thenReturn("You are a financial analyst.");
        lenient().when(llmClientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.OPENAI);

        service = new SynthesisService(llmClientProvider, promptLoader, serverManagerProvider, evaluationService);

        composite = new CompositeAnalysis(
                "RELIANCE",
                LocalDate.of(2026, 8, 15),
                72, "BUY", new BigDecimal("0.75"), List.of(),
                new CompositeAnalysis.NewsScore(65, "Positive earnings momentum",
                        List.of("new contracts"), List.of("regulatory risk"), 5),
                new CompositeAnalysis.TechnicalScore(78, "BUY", 0.8,
                        List.of("MACD cross", "RSI bullish")),
                new CompositeAnalysis.FundamentalScore(70, List.of("strong ROE", "low debt")),
                new CompositeAnalysis.BacktestScore(45, 0.62, 1.8, 12.5, 28.3, 6.3, true),
                "Composite analysis indicates favorable conditions",
                null
        );
    }

    // ===== BeanOutputConverter Tests =====

    @Nested
    @DisplayName("BeanOutputConverter structured output")
    class BeanOutputConverterTests {

        @Test
        void shouldSynthesizeWithValidJsonResponse() {
            // Arrange
            String validJson = """
                    {
                      "narrative": "RELIANCE shows strong momentum",
                      "recommendation": "BUY",
                      "confidence": 0.75,
                      "keyDrivers": ["earnings growth", "oil demand"],
                      "bullishFactors": ["strong fundamentals", "technical breakout"],
                      "bearishFactors": ["valuation concerns"]
                    }
                    """;
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just(validJson));

            // Act
            SynthesisResult result = service.synthesize(composite);

            // Assert
            assertThat(result).isNotNull();
            assertThat(result.narrative()).isEqualTo("RELIANCE shows strong momentum");
            assertThat(result.recommendation()).isEqualTo("BUY");
            assertThat(result.confidence()).isEqualTo(0.75);
            assertThat(result.keyDrivers()).hasSize(2);
            assertThat(result.bullishFactors()).hasSize(2);
            assertThat(result.bearishFactors()).hasSize(1);
            assertThat(result.success()).isTrue();
        }

        @Test
        void shouldHandleBeanOutputConverterFallbackToJsonParsing() {
            // Arrange — BeanOutputConverter may fail for non-standard JSON
            // The fallback should use brace-counting extraction
            String responseWithReasoning = """
                    Let me analyze this:
                    {
                      "narrative": "Mixed signals",
                      "recommendation": "HOLD",
                      "confidence": 0.5,
                      "keyDrivers": ["sector rotation"],
                      "bullishFactors": ["low P/E"],
                      "bearishFactors": ["high debt"]
                    }
                    """;
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just(responseWithReasoning));

            // Act
            SynthesisResult result = service.synthesize(composite);

            // Assert
            assertThat(result).isNotNull();
            assertThat(result.narrative()).isEqualTo("Mixed signals");
            assertThat(result.success()).isTrue();
        }

        @Test
        void shouldReturnFallbackWhenLlmReturnsEmpty() {
            // Arrange
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just(""));

            // Act
            SynthesisResult result = service.synthesize(composite);

            // Assert
            assertThat(result).isNotNull();
            assertThat(result.success()).isFalse();
            assertThat(result.narrative()).isNull();
            assertThat(result.recommendation()).isNull();
            assertThat(result.confidence()).isEqualTo(0.0);
        }

        @Test
        void shouldReturnFallbackForCapturedTruncatedJsonResponse() {
            String capturedTruncatedResponse = "{\n"
                    + "  \"narrative\": \"The overall outlook for INFY is bearish, driven primarily by "
                    + "strongly negative technical indicators and poor backtest performance. [TECHNICAL] shows a "
                    + "SELL signal with a score of -100/100 and 100% confidence, characterized by bearish EMA "
                    + "alignment (1000.20 / 1061.42 / 1092.95), a bearish RSI of 30.0, and a price 40.9% away from "
                    + "its 52-week high. [BACKTEST] further reinforces the negative outlook with a 25% win rate, "
                    + "a profit factor of 0.68, and";
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just(capturedTruncatedResponse));

            SynthesisResult result = service.synthesize(composite);

            assertThat(result.success()).isFalse();
            assertThat(result.recommendation()).isNull();
            assertThat(result.confidence()).isEqualTo(0.0);
        }

        @Test
        void shouldReturnFallbackWhenLlmReturnsNull() {
            // Arrange
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.empty());

            // Act
            SynthesisResult result = service.synthesize(composite);

            // Assert
            assertThat(result).isNotNull();
            assertThat(result.success()).isFalse();
        }

        @Test
        void shouldReturnFallbackOnLlmError() {
            // Arrange
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.error(new RuntimeException("LLM unavailable")));

            // Act
            SynthesisResult result = service.synthesize(composite);

            // Assert
            assertThat(result).isNotNull();
            assertThat(result.success()).isFalse();
        }
    }

    // ===== Prompt Building Tests =====

    @Nested
    @DisplayName("Prompt building")
    class PromptBuildingTests {

        /**
         * Real-shape regression for the DEGRADED LLM_ANALYSIS stage.
         *
         * <p>On the OpenAI-compatible reasoning backend this pipeline runs on
         * (longcat-2.5-preview-free), replaying a real synthesis prompt measured:</p>
         * <pre>
         *   max_tokens=1024 -&gt; finish_reason=length, reasoning_tokens=1023, content=0 chars
         *   max_tokens=4096 -&gt; finish_reason=stop,   reasoning_tokens= 454, complete parseable JSON
         * </pre>
         * <p>{@code max_tokens} is a combined reasoning-plus-content budget, so a budget the
         * reasoning can absorb whole starves the object entirely, and a partially-consumed
         * one cuts the JSON off mid-narrative — the 466-char partial capture behind the
         * NO_RECOMMENDATION degradations. The budget must clear the observed reasoning
         * spend plus the whole object, not merely the object.</p>
         */
        @Test
        void shouldRequestACompletionBudgetReasoningCannotConsume() {
            when(llmClientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.OPENAI);
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just("{}"));

            service.synthesize(composite);

            verify(llmClient).generateChatCompletion(anyList(),
                    intThat(tokens -> tokens >= 2048), eq(0.0));
        }

        @Test
        void shouldBoundCompletionBudgetForLocalContextWindows() {
            when(llmClientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.LOCAL);
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just("{}"));

            service.synthesize(composite);

            verify(llmClient).generateChatCompletion(anyList(), eq(512), eq(0.0));
        }

        @Test
        void shouldNotClaimATokenLimitForPiAgent() {
            when(llmClientProvider.getBackend()).thenReturn(LlmBackendSelector.Backend.PI_AGENT);
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just("{}"));

            service.synthesize(composite);

            verify(llmClient).generateChatCompletion(anyList(), eq(0), eq(0.0));
        }

        @Test
        void shouldIncludeStockSymbolInPrompt() {
            // Arrange
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just("{}"));

            // Act
            service.synthesize(composite);

            // Assert — verify LLM was called
            org.mockito.Mockito.verify(llmClient).generateChatCompletion(
                    org.mockito.ArgumentMatchers.anyList(),
                    org.mockito.ArgumentMatchers.anyInt(),
                    org.mockito.ArgumentMatchers.anyDouble()
            );
        }

        @Test
        void shouldIncludeAnalysisDateInPrompt() {
            // Arrange
            when(llmClientProvider.getClient(any(LlmBackendSelector.Backend.class))).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just("{}"));

            // Act
            service.synthesize(composite);

            // Assert
            org.mockito.Mockito.verify(llmClient).generateChatCompletion(
                    org.mockito.ArgumentMatchers.anyList(),
                    org.mockito.ArgumentMatchers.eq(SynthesisService.MAX_TOKENS),
                    eq(0.0)
            );
        }
    }
}
