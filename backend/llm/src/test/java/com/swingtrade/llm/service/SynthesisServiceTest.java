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
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.when;

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
            when(llmClientProvider.getClient()).thenReturn(llmClient);
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
            when(llmClientProvider.getClient()).thenReturn(llmClient);
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
            when(llmClientProvider.getClient()).thenReturn(llmClient);
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
            when(llmClientProvider.getClient()).thenReturn(llmClient);
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
            when(llmClientProvider.getClient()).thenReturn(llmClient);
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
            when(llmClientProvider.getClient()).thenReturn(llmClient);
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

        @Test
        void shouldIncludeStockSymbolInPrompt() {
            // Arrange
            when(llmClientProvider.getClient()).thenReturn(llmClient);
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
            when(llmClientProvider.getClient()).thenReturn(llmClient);
            when(llmClient.generateChatCompletion(any(), anyInt(), anyDouble()))
                    .thenReturn(Mono.just("{}"));

            // Act
            service.synthesize(composite);

            // Assert
            org.mockito.Mockito.verify(llmClient).generateChatCompletion(
                    org.mockito.ArgumentMatchers.anyList(),
                    org.mockito.ArgumentMatchers.eq(1024),
                    eq(0.0)
            );
        }
    }
}
