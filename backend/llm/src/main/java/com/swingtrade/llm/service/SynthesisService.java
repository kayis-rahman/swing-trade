package com.swingtrade.llm.service;

import com.swingtrade.domain.CompositeAnalysis;
import com.swingtrade.domain.SynthesisResult;
import com.swingtrade.llm.SynthesisOutput;
import com.swingtrade.llm.config.SynthesisPromptLoader;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.converter.BeanOutputConverter;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.List;
import java.util.Map;

@Service
public class SynthesisService {

    private static final Logger logger = LoggerFactory.getLogger(SynthesisService.class);
    /**
     * Completion budget for the synthesis call.
     *
     * <p>On the reasoning backends this pipeline actually runs against, {@code max_tokens}
     * is a <em>combined</em> reasoning-plus-content budget — the provider reserves the
     * reasoning text inside the same ceiling. Synthesis asks for the largest object in the
     * pipeline (six arrays plus a narrative), so this is the budget most exposed to that
     * reservation.</p>
     *
     * <p>Measured by replaying a real synthesis prompt against the deployed
     * OpenAI-compatible reasoning backend (longcat-2.5-preview-free):</p>
     * <pre>
     *   max_tokens=1024 → finish_reason=length, reasoning_tokens=1023, content=0 chars
     *   max_tokens=4096 → finish_reason=stop,   reasoning_tokens= 454, complete, parseable JSON
     * </pre>
     *
     * <p>At 1024 the model spent its entire budget on reasoning and returned no object at
     * all; anything left of it produced a partial object cut off mid-narrative (the
     * 466-character capture) that parses to a {@code NO_RECOMMENDATION} degradation. Raising
     * the budget is the fix — not lenient parsing, which cannot recover fields the model
     * was never given tokens to emit. Observed reasoning spend is ~450-1023 tokens, so 4096
     * leaves &gt;3000 tokens for the object after reasoning.</p>
     */
    static final int MAX_TOKENS = 4096;
    private static final int LOCAL_MAX_TOKENS = 512;
    private static final int PI_SSH_MAX_TOKENS = 1024;
    /** Synthesis is persisted as an evaluation input; deterministic output keeps reruns comparable. */
    private static final double TEMPERATURE = 0.0;
    // Must stay comfortably above LlmConfig's LOCAL_LLAMA_TIMEOUT (2850s) for the
    // CPU-bound local backends, or this outer deadline cuts the call off before
    // the client's own timeout ever gets a chance to fire. Widened alongside
    // SentimentService.ANALYSIS_TIMEOUT_SECONDS — see LOCAL_LLAMA_TIMEOUT's
    // javadoc for the measured prompt-eval-vs-decode throughput this is sized
    // from, including mid-request thermal-throttling decay (decode measured as
    // low as 0.54 tok/s on this Pi under sustained load).
    private static final long TIMEOUT_SECONDS = 2880;

    private final LlmClientProvider clientProvider;
    private final SynthesisPromptLoader promptLoader;
    private final LlmServerManagerProvider serverManagerProvider;
    private final SynthesisEvaluationService evaluationService;
    private final com.swingtrade.llm.config.LlmProperties llmProperties;

    public SynthesisService(LlmClientProvider clientProvider, SynthesisPromptLoader promptLoader,
                            LlmServerManagerProvider serverManagerProvider) {
        this(clientProvider, promptLoader, serverManagerProvider, new SynthesisEvaluationService());
    }

    public SynthesisService(LlmClientProvider clientProvider, SynthesisPromptLoader promptLoader,
                            LlmServerManagerProvider serverManagerProvider,
                            SynthesisEvaluationService evaluationService) {
        this(clientProvider, promptLoader, serverManagerProvider, evaluationService, null);
    }

    @org.springframework.beans.factory.annotation.Autowired
    public SynthesisService(LlmClientProvider clientProvider, SynthesisPromptLoader promptLoader,
                            LlmServerManagerProvider serverManagerProvider,
                            SynthesisEvaluationService evaluationService,
                            com.swingtrade.llm.config.LlmProperties llmProperties) {
        this.llmProperties = llmProperties;
        this.clientProvider = clientProvider;
        this.promptLoader = promptLoader;
        this.serverManagerProvider = serverManagerProvider;
        this.evaluationService = evaluationService;
    }

    public SynthesisResult synthesize(CompositeAnalysis composite) {
        String symbol = composite.symbol();
        logger.info("Generating LLM synthesis for {}", symbol);

        String userPrompt = buildPrompt(composite);

        List<Map<String, String>> messages = List.of(
            Map.of("role", "system", "content", promptLoader.getSystemPrompt()),
            Map.of("role", "user", "content", userPrompt)
        );

        try {
            // Synthesis is the longest generation in the pipeline, so it needs the
            // same in-flight protection as sentiment — previously it didn't even
            // call ensureRunning(), and the idle monitor could stop llama-server
            // mid-synthesis.
            LlmServerManager manager = serverManagerProvider.getManager();
            if (manager != null) {
                manager.ensureRunning();
                manager.beginRequest();
            }
            String llmResponse;
            try {
                var backend = clientProvider.getBackend();
                int maxTokens = backend == null ? MAX_TOKENS : switch (backend) {
                    case LOCAL, OLLAMA -> LOCAL_MAX_TOKENS;
                    case PI_SSH -> PI_SSH_MAX_TOKENS;
                    default -> MAX_TOKENS;
                };
                llmResponse = clientProvider.getClient()
                    .generateChatCompletion(messages, maxTokens, TEMPERATURE)
                    .block(llmProperties != null && llmProperties.getStageTimeout() != null
                        ? llmProperties.getStageTimeout() : Duration.ofSeconds(TIMEOUT_SECONDS));
            } finally {
                if (manager != null) {
                    manager.endRequest();
                }
            }

            if (llmResponse == null || llmResponse.isBlank()) {
                logger.warn("Empty LLM response for synthesis: {}", symbol);
                return fallbackSynthesis(composite);
            }

            return parseResponse(llmResponse, composite);
        } catch (Exception e) {
            logger.warn("LLM synthesis failed for {}: {}: {}, using fallback",
                symbol, e.getClass().getName(), LlmErrorUtils.describeError(e));
            logger.debug("LLM synthesis failure stack trace for {}", symbol, e);
            return fallbackSynthesis(composite);
        }
    }

    private String buildPrompt(CompositeAnalysis c) {
        return String.format("""
            Stock: %s | Analysis Date: %s

            === NEWS SENTIMENT ===
            Score: %d/100 | Articles analyzed: %d
            Summary: %s
            Catalysts: %s
            Red Flags: %s

            === TECHNICAL ANALYSIS ===
            Signal: %s | Score: %d/100 | Confidence: %.0f%%
            Indicators: %s

            === FUNDAMENTALS ===
            Score: %d/100 | Signal: %s
            Factors: %s

            === BACKTEST ===
            Total Trades: %d | Win Rate: %.1f%% | Profit Factor: %.2f
            Max Drawdown: %.1f%% | Total Return: %.1f%% | Expectancy: %.1f%%

            === COMPOSITE ===
            Score: %d/100 | Signal: %s | Confidence: %.0f%%
            Reasoning: %s
        """,
            c.symbol(), c.date(),
            c.news().score(), c.news().articleCount(),
            c.news().summary(), c.news().catalysts(), c.news().redFlags(),
            c.technical().signal(), c.technical().score(), c.technical().confidence() * 100, c.technical().indicators(),
            c.fundamentals().score(),
            c.fundamentals().score() > 0 ? "BULLISH" : c.fundamentals().score() < 0 ? "BEARISH" : "NEUTRAL",
            c.fundamentals().factors(),
            c.backtest().totalTrades(), c.backtest().winRate(), c.backtest().profitFactor(),
            c.backtest().maxDrawdown(), c.backtest().totalReturn(), c.backtest().expectancy(),
            c.compositeScore(), c.compositeSignal(), c.compositeConfidence().doubleValue() * 100,
            c.reasoning()
        );
    }

    private SynthesisResult parseResponse(String response, CompositeAnalysis composite) {
        try {
            BeanOutputConverter<SynthesisOutput> converter = new BeanOutputConverter<>(SynthesisOutput.class);
            SynthesisOutput output = converter.convert(response);
            evaluationService.record(composite, output);

            return new SynthesisResult(
                output.getNarrative(),
                output.getRecommendation(),
                output.getConfidence() != null ? output.getConfidence() : 0.0,
                output.getKeyDrivers() != null ? output.getKeyDrivers() : List.of(),
                output.getBullishFactors() != null ? output.getBullishFactors() : List.of(),
                output.getBearishFactors() != null ? output.getBearishFactors() : List.of(),
                true
            );
        } catch (Exception e) {
            logger.debug("BeanOutputConverter failed, falling back to Jackson parsing: {}: {}",
                e.getClass().getName(), e.getMessage());
            logger.debug("BeanOutputConverter failure stack trace", e);
            return parseWithFallback(response, composite);
        }
    }

    /**
     * Fallback: manual JSON parsing with brace-counting extraction.
     */
    private SynthesisResult parseWithFallback(String response, CompositeAnalysis composite) {
        String json = extractJson(response);
        if (json == null) {
            logger.warn("No JSON found in LLM response for synthesis: {} was {} chars, preview: {}",
                composite.symbol(), response.length(), preview(response));
            return fallbackSynthesis(composite);
        }

        try {
            tools.jackson.databind.ObjectMapper mapper = new tools.jackson.databind.ObjectMapper();
            LlmResponseDTO dto = mapper.readValue(json, LlmResponseDTO.class);
            SynthesisOutput output = new SynthesisOutput();
            output.setRecommendation(dto.getRecommendation());
            output.setConfidence(dto.getConfidence());
            output.setConflictDetected(dto.isConflictDetected());
            output.setEventRiskDetected(dto.isEventRiskDetected());
            output.setEventRiskReason(dto.getEventRiskReason());
            evaluationService.record(composite, output);
            return new SynthesisResult(
                dto.getNarrative(), dto.getRecommendation(), dto.getConfidence(),
                dto.getKeyDrivers(), dto.getBullishFactors(), dto.getBearishFactors(), true
            );
        } catch (Exception e) {
            logger.warn("Failed to parse synthesis JSON: {}: {}, fallback",
                e.getClass().getName(), e.getMessage());
            logger.debug("Synthesis JSON parse failure stack trace", e);
            return fallbackSynthesis(composite);
        }
    }

    /**
     * Bounded, single-line preview of a raw LLM response, so diagnosing unparseable
     * output does not require reproducing it. Truncation mid-object and reasoning
     * text before JSON are the two recurring causes, and both are invisible in the
     * log without seeing the response's head.
     */
    private static String preview(String response) {
        String flattened = response.replaceAll("\\s+", " ").trim();
        return flattened.length() <= 300 ? flattened : flattened.substring(0, 300) + "...";
    }

    private String extractJson(String response) {
        int open = findFirstBrace(response, '{', 0);
        if (open < 0) {
            return null;
        }

        int balance = 0;
        boolean inString = false;
        boolean escaped = false;

        for (int i = open; i < response.length(); i++) {
            char ch = response.charAt(i);
            if (escaped) {
                escaped = false;
                continue;
            }
            if (ch == '\\') {
                escaped = true;
                continue;
            }
            if (ch == '"') {
                inString = !inString;
                continue;
            }
            if (!inString) {
                if (ch == '{') {
                    balance++;
                } else if (ch == '}') {
                    balance--;
                    if (balance == 0) {
                        return response.substring(open, i + 1);
                    }
                }
            }
        }
        return null;
    }

    private int findFirstBrace(String text, char c, int from) {
        boolean inString = false;
        boolean escaped = false;
        for (int i = from; i < text.length(); i++) {
            if (escaped) {
                escaped = false;
                continue;
            }
            char ch = text.charAt(i);
            if (ch == '\\') {
                escaped = true;
                continue;
            }
            if (ch == '"') {
                inString = !inString;
                continue;
            }
            if (!inString && ch == c) {
                return i;
            }
        }
        return -1;
    }

    private SynthesisResult fallbackSynthesis(CompositeAnalysis c) {
        return new SynthesisResult(
            null, null, 0.0, List.of(), List.of(), List.of(), false
        );
    }

    private static class LlmResponseDTO {
        private String narrative;
        private String recommendation;
        private double confidence;
        private List<String> keyDrivers;
        private List<String> bullishFactors;
        private List<String> bearishFactors;
        private boolean conflictDetected;
        private boolean eventRiskDetected;
        private String eventRiskReason;

        public String getNarrative() { return narrative; }
        public String getRecommendation() { return recommendation; }
        public double getConfidence() { return confidence; }
        public List<String> getKeyDrivers() { return keyDrivers; }
        public List<String> getBullishFactors() { return bullishFactors; }
        public List<String> getBearishFactors() { return bearishFactors; }
        public boolean isConflictDetected() { return conflictDetected; }
        public boolean isEventRiskDetected() { return eventRiskDetected; }
        public String getEventRiskReason() { return eventRiskReason; }
    }
}
