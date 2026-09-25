package com.swingtrade.llm.service;

import com.swingtrade.core.metrics.LlmMetrics;
import com.swingtrade.core.metrics.SentimentMetrics;
import com.swingtrade.domain.NewsArticle;
import com.swingtrade.domain.PersistedNewsArticle;
import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.domain.store.SentimentStore;
import com.swingtrade.domain.store.StockStore;
import com.swingtrade.domain.SentimentResult;
import com.swingtrade.domain.Stock;
import com.swingtrade.data.entity.LlmAnalysisAuditEntity;
import com.swingtrade.data.entity.SentimentClassificationLogEntity;
import com.swingtrade.data.repository.LlmAnalysisAuditRepository;
import com.swingtrade.data.repository.SentimentClassificationLogRepository;
import com.swingtrade.llm.SentimentOutput;
import com.swingtrade.llm.SentimentType;
import com.swingtrade.llm.client.LayaClient;
import com.swingtrade.llm.client.LayaResult;
import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.config.LayaProperties;
import com.swingtrade.llm.config.SentimentPromptLoader;
import com.swingtrade.llm.config.LlmProperties;
import com.swingtrade.llm.domain.EarningsData;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.time.LocalDate;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.UUID;
import java.time.ZoneId;
import java.time.temporal.TemporalAdjusters;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Future;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

/**
 * Main service for sentiment analysis of stocks using LLM.
 * Orchestrates news ingestion, sentiment analysis, caching, and signal generation.
 */
@Service
public class SentimentService {

    private static final Logger logger = LoggerFactory.getLogger(SentimentService.class);

    // Must stay comfortably above LlmConfig's LOCAL_LLAMA_TIMEOUT (2850s) for the
    // CPU-bound local backends, or this outer deadline cuts the call off before
    // the client's own timeout ever gets a chance to fire. See LOCAL_LLAMA_TIMEOUT's
    // javadoc for the measured prompt-eval-vs-decode throughput this is sized
    // from — including the mid-request thermal-throttling decay that made the
    // first two attempts at this number (930s, then 1830s) both too tight.
    private static final long ANALYSIS_TIMEOUT_SECONDS = 2880;
    /** Result source when the LLM failed and keyword analysis substituted for it. */
    static final String KEYWORD_FALLBACK_SOURCE = "KEYWORD_FALLBACK";
    /** Result source when Laya's high-confidence classification substituted for Qwen (live mode only). */
    static final String LAYA_SOURCE = "LAYA";
    private static final int MAX_ARTICLES_FOR_LLM = 10;
    private static final int PI_MAX_ARTICLES_FOR_LLM = 6;
    private static final int PI_MAX_ARTICLE_CHARS = 450;
    private static final int DEFAULT_MAX_RESPONSE_TOKENS = 512;
    private static final int PI_MAX_RESPONSE_TOKENS = 128;
    private static final int MAX_CONTEXT_FILINGS = 5;
    private static final int MAX_CONTEXT_CHARS = 2400;
    private static final int MAX_CONTEXT_FIELD_CHARS = 360;

    // Per-article character cap. This exists ONLY to keep the worst case (10
    // articles, all at the cap) under llamacpp.context (8192) — it is not a
    // throughput/heat knob. Deliberately NOT sized from prompt-eval speed: the
    // prompt asks for catalysts and red flags, and those often sit mid-article
    // (a margin caveat or downgrade after a positive lead), so trimming for
    // speed biases the result toward whatever the opening sentence frames.
    // Longer analysis time is the accepted trade-off, backstopped by
    // PiThermalGuard (temperature-based backoff) rather than by cutting content.
    //
    // Math, from measured throughput on this Pi (Qwen3-4B, llamacpp.threads=2):
    //   * this scraped news text tokenizes at ~1.9 chars/token
    //   * fixed prompt scaffolding (system + user template, no articles) is
    //     ~1341 chars
    //   * budget = context(8192) - response(512 max_tokens) = 7680 tokens
    //     -> ~14,592 chars total prompt -> ~13,251 chars left for article
    //     content after scaffolding -> ~1325 chars/article across 10 articles
    // 1200 leaves an ~8% margin under that ceiling for token-ratio variance,
    // instead of cutting into it. At 7 tok/s prompt eval that's a worst case of
    // ~17 min of prompt eval alone (ANALYSIS_TIMEOUT_SECONDS and
    // LlmConfig.LOCAL_LLAMA_TIMEOUT are both sized to cover it) — slower, not
    // shallower. Raising this further needs a larger llamacpp.context, not a
    // smaller margin.
    private static final int MAX_ARTICLE_CHARS = 1200;
    private static final ZoneId MARKET_ZONE = ZoneId.of("Asia/Kolkata");
    private static final int NEWS_LOOKBACK_DAYS = 7;

    // Thread pool for async operations
    private final ExecutorService analysisExecutor;

    private final LlmClientProvider clientProvider;
    private final LlmServerManagerProvider serverManagerProvider;
    private final SentimentPromptLoader promptLoader;
    private final SentimentAnalyzer sentimentAnalyzer;
    private final NewsIngestionService newsIngestionService;
    private final SentimentStore sentimentStore;
    private final StockStore stockStore;
    private final AppSettingsStore appSettingsStore;
    private final LlmMetrics llmMetrics;
    private final SentimentMetrics sentimentMetrics;
    private final LlmAnalysisAuditRepository auditRepository;
    private final LlmProperties llmProperties;
    private final PdfExtractionService pdfExtractionService;
    private final LayaClient layaClient;
    private final LayaProperties layaProperties;
    private final SentimentClassificationLogRepository classificationLogRepository;

    private final double defaultConfidence;

    /**
     * Constructs SentimentService with required dependencies.
     */
    @Autowired
    public SentimentService(
            LlmClientProvider clientProvider,
            LlmServerManagerProvider serverManagerProvider,
            SentimentPromptLoader promptLoader,
            SentimentAnalyzer sentimentAnalyzer,
            NewsIngestionService newsIngestionService,
            SentimentStore sentimentStore,
            StockStore stockStore,
            AppSettingsStore appSettingsStore,
            LlmMetrics llmMetrics,
            SentimentMetrics sentimentMetrics,
            LlmProperties llmProperties,
            LlmAnalysisAuditRepository auditRepository,
            PdfExtractionService pdfExtractionService,
            LayaClient layaClient,
            LayaProperties layaProperties,
            SentimentClassificationLogRepository classificationLogRepository,
            @Value("${llm.sentiment.default-confidence:0.75}") double defaultConfidence) {

        this.clientProvider = clientProvider;
        this.serverManagerProvider = serverManagerProvider;
        this.promptLoader = promptLoader;
        this.sentimentAnalyzer = sentimentAnalyzer;
        this.newsIngestionService = newsIngestionService;
        this.sentimentStore = sentimentStore;
        this.stockStore = stockStore;
        this.appSettingsStore = appSettingsStore;
        this.llmMetrics = llmMetrics;
        this.sentimentMetrics = sentimentMetrics;
        this.auditRepository = auditRepository;
        this.llmProperties = llmProperties;
        this.pdfExtractionService = pdfExtractionService;
        this.layaClient = layaClient;
        this.layaProperties = layaProperties;
        this.classificationLogRepository = classificationLogRepository;
        this.defaultConfidence = defaultConfidence;

        // Initialize thread pool with bounded capacity
        this.analysisExecutor = new ThreadPoolExecutor(
                2,  // 2 core threads
                5,  // 5 max threads
                60L, TimeUnit.SECONDS,
                new LinkedBlockingQueue<>(10),
                new ThreadPoolExecutor.CallerRunsPolicy()
        );
    }

    /**
     * Trims a cleaned article to {@link #MAX_ARTICLE_CHARS}, cutting on a word
     * boundary where one is available near the limit so the text doesn't end
     * mid-token.
     */
    private static String capArticleLength(String text, int maxArticleChars) {
        if (text.length() <= maxArticleChars) {
            return text;
        }
        String truncated = text.substring(0, maxArticleChars);
        int lastSpace = truncated.lastIndexOf(' ');
        if (lastSpace > maxArticleChars - 60) {
            truncated = truncated.substring(0, lastSpace);
        }
        return truncated + "...";
    }

    /**
     * Analyzes sentiment for a single stock using news from RSS feeds.
     *
     * @param stockSymbol the stock symbol (e.g., "RELIANCE", "TCS")
     * @param date the date of analysis
     * @return SentimentResult with analysis findings
     */
    public SentimentResult analyzeStockSentiment(String stockSymbol, LocalDate date) {
        logger.info("Starting sentiment analysis for stock: {} on date: {}", stockSymbol, date);
        long start = System.currentTimeMillis();
        String auditRequestId = UUID.randomUUID().toString();

        try {
            // Fetch news articles
            List<PersistedNewsArticle> persistedArticles =
                    newsIngestionService.fetchPersistedStockNewsForDecisionDate(stockSymbol, date);
            List<PersistedNewsArticle> articles = persistedArticles.stream()
                    .filter(article -> article.article().publishedDate() != null
                        && !article.article().publishedDate().isBefore(
                            date.minusDays(NEWS_LOOKBACK_DAYS).atStartOfDay(MARKET_ZONE))
                        && !article.article().publishedDate().isAfter(
                            date.atTime(15, 30).atZone(MARKET_ZONE)))
                    .toList();

            // Source completion order is nondeterministic. Rank and collapse
            // syndicated headlines before the LLM article budget is applied.
            articles = NewsIngestionService.rankAndDeduplicateForLlm(articles);

            if (articles.isEmpty()) {
                logger.warn("No news articles found for stock: {}", stockSymbol);
                return createDefaultSentimentResult(stockSymbol, date, SentimentType.NEUTRAL);
            }

            logger.info("Found {} articles for {}: {}", articles.size(), stockSymbol, stockSymbol);

            boolean piBackend = clientProvider.getBackend() == LlmBackendSelector.Backend.PI_SSH;
            int maxArticleChars = piBackend ? PI_MAX_ARTICLE_CHARS : MAX_ARTICLE_CHARS;
            int maxArticles = piBackend ? PI_MAX_ARTICLES_FOR_LLM : MAX_ARTICLES_FOR_LLM;

            // Clean and prepare news content. Each article is capped: capping the
            // article *count* alone isn't enough, since full article bodies pushed
            // the prompt to ~8.5k tokens and llama.cpp rejected it outright with
            // "request (8556 tokens) exceeds the available context size (4096)".
            // The lead of an article carries the sentiment signal, so trimming the
            // tail costs little and cuts prompt-eval time (and heat) substantially.
            Map<PersistedNewsArticle, String> cleanedArticles = new java.util.LinkedHashMap<>();
            List<PersistedNewsArticle> usableArticles = articles.stream()
                    .filter(article -> {
                        String cleaned = newsIngestionService.cleanNewsText(article.article());
                        if (cleaned == null || cleaned.trim().isEmpty()) return false;
                        cleanedArticles.put(article, cleaned);
                        return true;
                    }).toList();
            List<PersistedNewsArticle> preparedArticles = usableArticles;
            List<String> newsContent = java.util.stream.IntStream.range(0, preparedArticles.size())
                    .mapToObj(index -> {
                        NewsArticle article = preparedArticles.get(index).article();
                        String cleaned = cleanedArticles.get(preparedArticles.get(index));
                        if (cleaned == null || cleaned.trim().isEmpty()) return null;
                        String published = article.publishedDate().withZoneSameInstant(MARKET_ZONE)
                            .toLocalDate().toString();
                        String source = article.source() == null || article.source().isBlank()
                                ? "source unknown" : article.source();
                        return "[" + (index + 1) + "] " + published + " | " + source + " | "
                            + capArticleLength(cleaned, maxArticleChars);
                    })
                    .filter(Objects::nonNull)
                    .toList();

            if (newsContent.isEmpty()) {
                logger.warn("No valid news content after cleaning for stock: {}", stockSymbol);
                return createDefaultSentimentResult(stockSymbol, date, SentimentType.NEUTRAL);
            }

            // Limit articles to fit within LLM context window
            if (newsContent.size() > maxArticles) {
                logger.info("Truncating {} articles to {} for {} LLM analysis",
                        newsContent.size(), maxArticles, piBackend ? "Pi" : "configured");
                newsContent = newsContent.subList(0, maxArticles);
                usableArticles = usableArticles.subList(0, maxArticles);
            }
            List<Long> articleIds = usableArticles.stream().map(PersistedNewsArticle::id).toList();

            // Perform sentiment analysis
            SentimentOutput analysisResult;
            try {
                analysisResult = performSentimentAnalysis(stockSymbol, date, newsContent, auditRequestId);
            } catch (Exception llmEx) {
                logger.warn("LLM unavailable for {}, falling back to keyword analysis: {}", stockSymbol, LlmErrorUtils.describeError(llmEx));
                // Build a simple result from headlines
                List<String> headlines = newsContent.stream().toList();
                SentimentResult fallback = keywordBasedSentiment(stockSymbol, date, headlines);
                analysisResult = new SentimentOutput(
                        switch (fallback.score()) {
                            case POSITIVE -> SentimentType.POSITIVE;
                            case NEGATIVE -> SentimentType.NEGATIVE;
                            default -> SentimentType.NEUTRAL;
                        },
                        fallback.summary(), fallback.confidence(),
                        fallback.redFlags(), fallback.catalysts(), KEYWORD_FALLBACK_SOURCE
                );
                analysisResult.setDegradedReason(
                        "LLM unavailable: " + LlmErrorUtils.describeError(llmEx));
            }

            // Build and cache result
            var selectedBackend = clientProvider.getBackend();
            String provider = selectedBackend == null ? "unknown" : selectedBackend.getKey();
            SentimentResult result = buildSentimentResult(stockSymbol, date, analysisResult,
                    articleIds.size(), articleIds, provider, auditRequestId);

            // Persist to database
            try {
                sentimentStore.saveOrUpdate(result);
                logger.debug("Persisted sentiment result for {} on {}", stockSymbol, date);
            } catch (Exception e) {
                logger.warn("Failed to persist sentiment result for {}: {}", stockSymbol, e.getMessage());
                // Don't fail the analysis if persistence fails
            }

            logger.info("Sentiment analysis complete for {}: {} (confidence: {})",
                    stockSymbol, analysisResult.getSentiment(), analysisResult.getConfidence());

            long duration = System.currentTimeMillis() - start;
            sentimentMetrics.recordCompleted();
            sentimentMetrics.recordDuration(Duration.ofMillis(duration));
            return result;

        } catch (Exception e) {
            long duration = System.currentTimeMillis() - start;
            sentimentMetrics.recordFailed();
            sentimentMetrics.recordDuration(Duration.ofMillis(duration));
            logger.error("Error analyzing sentiment for {}: {}", stockSymbol, e.getMessage(), e);
            // Return default neutral result on error
            return createDefaultSentimentResult(stockSymbol, date, SentimentType.NEUTRAL);
        }
    }

    /** Compatibility constructor for lightweight unit tests. */
    public SentimentService(
            LlmClientProvider clientProvider, LlmServerManagerProvider serverManagerProvider,
            SentimentPromptLoader promptLoader, SentimentAnalyzer sentimentAnalyzer,
            NewsIngestionService newsIngestionService, SentimentStore sentimentStore,
            StockStore stockStore, AppSettingsStore appSettingsStore, LlmMetrics llmMetrics,
            SentimentMetrics sentimentMetrics, PdfExtractionService pdfExtractionService,
            double defaultConfidence) {
        this(clientProvider, serverManagerProvider, promptLoader, sentimentAnalyzer,
                newsIngestionService, sentimentStore, stockStore, appSettingsStore,
                llmMetrics, sentimentMetrics, null, null, pdfExtractionService,
                null, null, null, defaultConfidence);
    }

    /** Compatibility constructor for lightweight unit tests. */
    public SentimentService(
            LlmClientProvider clientProvider, LlmServerManagerProvider serverManagerProvider,
            SentimentPromptLoader promptLoader, SentimentAnalyzer sentimentAnalyzer,
            NewsIngestionService newsIngestionService, SentimentStore sentimentStore,
            StockStore stockStore, AppSettingsStore appSettingsStore, LlmMetrics llmMetrics,
            SentimentMetrics sentimentMetrics, double defaultConfidence) {
        this(clientProvider, serverManagerProvider, promptLoader, sentimentAnalyzer,
                newsIngestionService, sentimentStore, stockStore, appSettingsStore,
                llmMetrics, sentimentMetrics, null, null, null,
                null, null, null, defaultConfidence);
    }

    private Duration stageTimeout() {
        return llmProperties != null && llmProperties.getStageTimeout() != null
                ? llmProperties.getStageTimeout() : Duration.ofSeconds(ANALYSIS_TIMEOUT_SECONDS);
    }

    /**
     * Performs LLM-based sentiment analysis on news content.
     *
     * @param stockSymbol the stock symbol
     * @param newsContent list of cleaned news texts
     * @return sentiment analysis result
     */
    private SentimentOutput performSentimentAnalysis(String stockSymbol, LocalDate analysisDate,
                                                     List<String> newsContent, String requestId) {
        logger.debug("Performing LLM sentiment analysis for {} with {} articles", stockSymbol, newsContent.size());

        if (newsContent.isEmpty()) {
            return new SentimentOutput(
                    SentimentType.NEUTRAL,
                    "No news content available for analysis",
                    0.0
            );
        }

        // Combine all news content into a single prompt for comprehensive analysis
        String combinedContent = String.join("\n\n---\n\n", newsContent);

        // Laya local pre-filter: a true no-op when disabled (laya.enabled=false is the
        // shipped default), so nothing below this block runs or logs in that case. Laya
        // sees exactly combinedContent — the same text Qwen sees below — so the two are
        // directly comparable in the agreement-rate report.
        boolean layaEnabled = layaProperties != null && layaProperties.isEnabled() && layaClient != null;
        if (layaEnabled) {
            SentimentOutput layaOnly = tryLaya(stockSymbol, analysisDate, combinedContent);
            if (layaOnly != null) {
                return layaOnly;
            }
        }

        // Create prompt using loaded templates
        String formattedUser = promptLoader.getUserPrompt()
                .replace("{symbol}", stockSymbol)
                .replace("{newsContent}", combinedContent)
                .replace("{marketContext}", buildMarketContext(stockSymbol, analysisDate));
        List<Map<String, String>> messages = List.of(
                Map.of("role", "system", "content", promptLoader.getSystemPrompt().replace("{symbol}", stockSymbol)),
                Map.of("role", "user", "content", formattedUser)
        );

        var backend = clientProvider.getBackend();
        int maxResponseTokens = backend == LlmBackendSelector.Backend.PI_SSH
                ? PI_MAX_RESPONSE_TOKENS : DEFAULT_MAX_RESPONSE_TOKENS;
        String provider = backend != null ? backend.getKey() : "unknown";
        String modelVersion = configuredModel(provider);
        String promptHash = computePromptHash();
        OffsetDateTime startedAt = OffsetDateTime.now(ZoneOffset.UTC);

        // Call LLM for sentiment analysis
        String llmResponse;
        long llmStart = System.currentTimeMillis();
        try {
            LlmServerManager manager = serverManagerProvider.getManager();
            if (manager != null) {
                manager.ensureRunning();
                manager.beginRequest();
            }
            LlmClient client = clientProvider.getClient();
            try {
                llmResponse = client.generateChatCompletion(messages, maxResponseTokens, 0.0)
                        .block(stageTimeout());
            } finally {
                // Release the in-flight marker so the idle monitor can retire the
                // server again; without the pairing it would stay pinned forever.
                if (manager != null) {
                    manager.endRequest();
                }
            }
            long latencyMs = System.currentTimeMillis() - llmStart;
            llmMetrics.recordCall(Duration.ofMillis(System.currentTimeMillis() - llmStart), true);
            llmMetrics.recordSentimentAnalyzed();
            if (llmResponse == null || llmResponse.isBlank()) {
                // Treat an empty response as provider failure so the caller uses the
                // auditable keyword fallback. Returning UNKNOWN here bypassed that path,
                // mislabeled the audit as SUCCESS, and made an LLM outage look like a
                // successful neutral/default analysis.
                throw new IllegalStateException("LLM returned an empty response");
            }
            SentimentOutput parsed = sentimentAnalyzer.parseResponse(llmResponse, newsContent.size());
            persistAudit(requestId, stockSymbol, analysisDate, provider, modelVersion,
                    promptHash, messages, llmResponse, parsed, "SUCCESS", null, maxResponseTokens,
                    false, startedAt, latencyMs);
            if (layaEnabled) {
                logClassification(stockSymbol, analysisDate, SentimentClassificationLogEntity.ModelUsed.QWEN,
                        parsed.getSentiment(), parsed.getConfidence(), latencyMs, layaProperties.isShadowMode());
            }
            return parsed;
        } catch (Exception e) {
            String errorDetail = LlmErrorUtils.describeError(e);
            persistAudit(requestId, stockSymbol, analysisDate, provider, modelVersion, promptHash,
                    messages, null, null, "FAILED", errorDetail, maxResponseTokens, true, startedAt,
                    System.currentTimeMillis() - llmStart);
            llmMetrics.recordCall(Duration.ofMillis(System.currentTimeMillis() - llmStart), false);
            if (e.getMessage() != null && e.getMessage().contains("timeout")) {
                logger.error("Timeout analyzing sentiment for {}: analysis took more than {} seconds",
                        stockSymbol, stageTimeout().toSeconds());
                return new SentimentOutput(
                        SentimentType.UNKNOWN,
                        "Analysis timed out - unable to process news content",
                        0.0, List.of(), List.of(), "DEFAULT"
                );
            }
            logger.error("Error during LLM sentiment analysis: {}", errorDetail, e);
            throw e;
        }

    }

    /**
     * Runs the Laya local pre-filter ahead of the Qwen/LLM call. Only called
     * when {@code laya.enabled=true}; the caller ({@link #performSentimentAnalysis})
     * guards this so a disabled Laya adds no overhead or logging.
     *
     * <p>In shadow mode ({@code laya.shadow-mode=true}, the shipped default once
     * enabled), this always returns {@code null} so the caller falls through to
     * the existing Qwen call unchanged — Laya's result is only logged, never
     * used, and a Laya failure here has no effect on the Qwen path.
     *
     * <p>In live mode ({@code laya.shadow-mode=false}), a high-confidence Laya
     * response (confidence &gt;= {@code laya.confidence-threshold}) is returned
     * directly, skipping Qwen entirely. A low-confidence or unreachable Laya
     * response returns {@code null}, falling through to Qwen as in shadow mode.
     *
     * @return a Laya-sourced {@link SentimentOutput} to use in place of Qwen, or
     *         {@code null} to fall through to the existing Qwen call
     */
    private SentimentOutput tryLaya(String stockSymbol, LocalDate analysisDate, String combinedContent) {
        long layaStart = System.currentTimeMillis();
        Optional<LayaResult> layaOpt;
        try {
            layaOpt = layaClient.classify(combinedContent);
        } catch (Exception e) {
            // LayaClient contracts to never throw, but this stage must never fail
            // the sentiment pipeline over Laya regardless.
            logger.warn("Unexpected exception calling Laya for {}: {}", stockSymbol, e.getMessage());
            layaOpt = Optional.empty();
        }
        long layaLatencyMs = System.currentTimeMillis() - layaStart;
        LayaResult layaResult = layaOpt.orElse(null);

        if (layaProperties.isShadowMode()) {
            if (layaResult != null) {
                logClassification(stockSymbol, analysisDate, SentimentClassificationLogEntity.ModelUsed.LAYA,
                        layaResult.sentiment(), layaResult.confidence(), layaLatencyMs, true);
                logger.info("Laya shadow-mode result for {}: {} (confidence={}); proceeding with Qwen as usual",
                        stockSymbol, layaResult.sentiment(), layaResult.confidence());
            } else {
                logger.info("Laya unreachable/failed for {} in shadow mode; proceeding with Qwen as usual",
                        stockSymbol);
            }
            return null;
        }

        // Live mode
        if (layaResult != null && layaResult.confidence() >= layaProperties.getConfidenceThreshold()) {
            logClassification(stockSymbol, analysisDate, SentimentClassificationLogEntity.ModelUsed.LAYA,
                    layaResult.sentiment(), layaResult.confidence(), layaLatencyMs, false);
            logger.info("Laya live-mode result for {}: {} (confidence={} >= threshold {}); skipping Qwen",
                    stockSymbol, layaResult.sentiment(), layaResult.confidence(),
                    layaProperties.getConfidenceThreshold());
            return new SentimentOutput(layaResult.sentiment(), "Laya local pre-filter classification",
                    layaResult.confidence(), List.of(), List.of(), LAYA_SOURCE);
        }
        if (layaResult != null) {
            logClassification(stockSymbol, analysisDate, SentimentClassificationLogEntity.ModelUsed.LAYA,
                    layaResult.sentiment(), layaResult.confidence(), layaLatencyMs, false);
            logger.info("Laya live-mode result for {} below confidence threshold ({} < {}); falling back to Qwen",
                    stockSymbol, layaResult.confidence(), layaProperties.getConfidenceThreshold());
        } else {
            logger.info("Laya unreachable/failed for {} in live mode; falling back to Qwen", stockSymbol);
        }
        return null;
    }

    /** Persists one row to {@code sentiment_classification_log}. Never throws. */
    private void logClassification(String symbol, LocalDate analysisDate,
                                    SentimentClassificationLogEntity.ModelUsed model,
                                    SentimentType sentiment, Double confidence, long latencyMs,
                                    boolean shadowMode) {
        if (classificationLogRepository == null) return;
        try {
            classificationLogRepository.save(new SentimentClassificationLogEntity(
                    symbol, analysisDate, model, sentiment != null ? sentiment.name() : "UNKNOWN",
                    confidence, latencyMs, shadowMode));
        } catch (Exception e) {
            logger.warn("Failed to persist sentiment classification log for {}: {}", symbol, e.getMessage());
        }
    }

    /**
     * Builds a bounded, factual context block from structured Indian-market
     * sources. Missing context is represented by an empty string so existing
     * news-only behavior is preserved.
     */
    private String buildMarketContext(String symbol, LocalDate decisionDate) {
        if (decisionDate == null) return "";
        StringBuilder context = new StringBuilder();
        try {
            if (pdfExtractionService != null) {
                EarningsData earnings = pdfExtractionService.extractLatestEarnings(symbol);
                if (earnings != null && earnings.extractionDate() != null
                        && !earnings.extractionDate().isAfter(decisionDate)) {
                    context.append("Earnings: quarter=").append(safe(earnings.quarter()))
                            .append(", revenue=").append(safe(earnings.revenue()))
                            .append(", netProfit=").append(safe(earnings.netProfit()))
                            .append(", EPS=").append(safe(earnings.eps()))
                            .append(", EBITDA=").append(safe(earnings.ebitda()))
                            .append(", guidance=").append(cap(safe(earnings.guidance()), MAX_CONTEXT_FIELD_CHARS))
                            .append("\n");
                }
            }
        } catch (Exception e) {
            logger.debug("Structured earnings context unavailable for {}: {}", symbol, e.getMessage());
        }
        try {
            LocalDate from = decisionDate.minusDays(NEWS_LOOKBACK_DAYS);
            newsIngestionService.fetchStructuredFilings(symbol).stream()
                    .filter(Objects::nonNull)
                    .filter(f -> f.date() != null && !f.date().isBefore(from) && !f.date().isAfter(decisionDate))
                    .sorted((a, b) -> b.date().compareTo(a.date()))
                    .limit(MAX_CONTEXT_FILINGS)
                    .forEach(f -> {
                        context.append("Filing: ").append(f.date()).append(" | ")
                                .append(f.typeLabel()).append(" | ")
                                .append(cap(safe(f.title()), MAX_CONTEXT_FIELD_CHARS));
                        String description = cap(safe(f.description()), MAX_CONTEXT_FIELD_CHARS);
                        if (!description.isBlank()) context.append(" — ").append(description);
                        context.append("\n");
                    });
        } catch (Exception e) {
            logger.debug("Structured filing context unavailable for {}: {}", symbol, e.getMessage());
        }
        if (context.isEmpty()) return "";
        if (context.length() > MAX_CONTEXT_CHARS) {
            context.setLength(MAX_CONTEXT_CHARS);
            context.append("\n[context truncated]");
        }
        return "\n=== Structured Indian-market context (reference only) ===\n"
                + context + "Use only the supplied values; do not infer missing figures.\n";
    }

    private static String safe(Object value) {
        return value == null ? "not available" : value.toString().replaceAll("[\\r\\n]+", " ").trim();
    }

    private static String cap(String value, int limit) {
        return value.length() <= limit ? value : value.substring(0, limit) + "...";
    }

    private void persistAudit(String requestId, String symbol, LocalDate analysisDate,
                              String provider, String model,
                              String promptHash, List<Map<String, String>> messages,
                              String rawResponse, SentimentOutput parsed, String status,
                              String error, int maxResponseTokens, boolean fallbackUsed,
                              OffsetDateTime startedAt, long latencyMs) {
        if (auditRepository == null) return;
        String score = parsed == null ? null : parsed.getSentiment().name();
        Double confidence = parsed == null ? null : parsed.getConfidence();
        auditRepository.save(new LlmAnalysisAuditEntity(requestId, symbol, analysisDate,
                provider, model, promptHash, messages.get(0).get("content"),
                messages.get(1).get("content"), rawResponse, score, confidence, status,
                    error, fallbackUsed, maxResponseTokens, 0.0, startedAt,
                OffsetDateTime.now(ZoneOffset.UTC), latencyMs));
    }

    private String configuredModel(String provider) {
        Optional<String> legacySetting = appSettingsStore.get("llamacpp.model");
        if (("pi_ssh".equals(provider) || "local".equals(provider))
                && legacySetting != null && legacySetting.isPresent() && !legacySetting.get().isBlank()) {
            return legacySetting.get();
        }
        if (llmProperties != null) {
            String configured = switch (provider) {
                case "pi_ssh" -> llmProperties.getProviders().getPiSsh().getModel();
                case "openai" -> llmProperties.getProviders().getOpenai().getModel();
                case "ollama" -> llmProperties.getProviders().getOllama().getModel();
                default -> llmProperties.getProviders().getLocal().getModel();
            };
            if (configured != null && !configured.isBlank()) return configured;
        }
        return legacySetting != null ? legacySetting.orElse("unknown") : "unknown";
    }

    /**
     * Builds SentimentResult from analysis result.
     */
    private SentimentResult buildSentimentResult(
            String stockSymbol,
            LocalDate date,
            SentimentOutput analysisResult,
            int articleCount,
            List<Long> articleIds,
            String provider) {
        return buildSentimentResult(stockSymbol, date, analysisResult, articleCount, articleIds,
            provider, null);
    }

    private SentimentResult buildSentimentResult(
            String stockSymbol,
            LocalDate date,
            SentimentOutput analysisResult,
            int articleCount,
            List<Long> articleIds,
            String provider,
            String auditRequestId) {

        String modelVersion = configuredModel(provider);
        String promptHash = computePromptHash();

        SentimentResult.SentimentScore score;
        switch (analysisResult.getSentiment()) {
            case POSITIVE:
                score = SentimentResult.SentimentScore.POSITIVE;
                break;
            case NEGATIVE:
                score = SentimentResult.SentimentScore.NEGATIVE;
                break;
            case UNKNOWN:
                score = SentimentResult.SentimentScore.UNKNOWN;
                break;
            default:
                score = SentimentResult.SentimentScore.NEUTRAL;
        }

        return new SentimentResult(
                null,
                stockSymbol,
                date,
                score,
                analysisResult.getReasoning(),
                "",
                analysisResult.getConfidence(),
                LocalDate.now(),
                analysisResult.getRedFlags() != null ? analysisResult.getRedFlags() : List.of(),
                analysisResult.getCatalysts() != null ? analysisResult.getCatalysts() : List.of(),
                promptHash,
                modelVersion,
                articleCount,
                analysisResult.getSource(),
                articleIds,
                auditRequestId,
                analysisResult.getDegradedReason()
        );
    }

    /**
     * Enforces the information boundary for sentiment analyses. An article without a publication
     * timestamp is not admissible because its relationship to the decision cutoff is unknowable.
     */
    private List<NewsArticle> filterPointInTimeArticles(List<NewsArticle> articles, LocalDate date) {
        if (articles == null || articles.isEmpty() || date == null) return List.of();
        var from = date.minusDays(NEWS_LOOKBACK_DAYS).atStartOfDay(MARKET_ZONE);
        var cutoff = date.atTime(15, 30).atZone(MARKET_ZONE);
        return articles.stream()
            .filter(Objects::nonNull)
            .filter(article -> article.publishedDate() != null
                && !article.publishedDate().isBefore(from)
                && !article.publishedDate().isAfter(cutoff))
            .toList();
    }

    /**
     * Computes SHA-256 hash of the prompt template for A/B tracking.
     */
    private String computePromptHash() {
        try {
            java.security.MessageDigest digest = java.security.MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(promptLoader.getSystemPrompt().getBytes("UTF-8"));
            StringBuilder sb = new StringBuilder();
            for (byte b : hash) {
                sb.append(String.format("%02x", b));
            }
            return sb.toString();
        } catch (Exception e) {
            logger.warn("Failed to compute prompt hash: {}", e.getMessage());
            return "unavailable";
        }
    }

    /**
     * Creates default sentiment result with neutral sentiment.
     *
     * @param stockSymbol the stock symbol
     * @param date the date
     * @param sentimentType the sentiment type
     * @return default sentiment result
     */
    private SentimentResult createDefaultSentimentResult(
            String stockSymbol,
            LocalDate date,
            SentimentType sentimentType) {

        String defaultReasoning = switch (sentimentType) {
            case POSITIVE -> "No negative news found - potentially positive outlook";
            case NEGATIVE -> "No significant news found - potentially neutral to negative";
            default -> "No news articles available for analysis - defaulting to neutral";
        };

        return SentimentResult.create(
                stockSymbol,
                date,
                sentimentType == SentimentType.POSITIVE ?
                        SentimentResult.SentimentScore.POSITIVE :
                        sentimentType == SentimentType.NEGATIVE ?
                                SentimentResult.SentimentScore.NEGATIVE :
                                SentimentResult.SentimentScore.NEUTRAL,
                defaultReasoning,
                "",
                0.3,
                List.of(),
                List.of()
        );
    }

    /**
     * Analyzes sentiment for multiple stocks asynchronously.
     *
     * @param stockSymbols list of stock symbols
     * @param date the analysis date
     * @return map of stock symbol to sentiment result
     */
    public Map<String, SentimentResult> analyzeMultipleStocks(
            List<String> stockSymbols,
            LocalDate date) {

        logger.info("Starting batch sentiment analysis for {} stocks", stockSymbols.size());

        Map<String, Future<SentimentResult>> futures = new ConcurrentHashMap<>();

        // Submit all analysis tasks
        for (String symbol : stockSymbols) {
            Future<SentimentResult> future = analysisExecutor.submit(() ->
                    analyzeStockSentiment(symbol, date));
            futures.put(symbol, future);
        }

        // Collect results
        Map<String, SentimentResult> results = new HashMap<>();
        List<String> failedSymbols = new ArrayList<>();

        for (Map.Entry<String, Future<SentimentResult>> entry : futures.entrySet()) {
            try {
                SentimentResult result = entry.getValue().get(stageTimeout().toMillis(), TimeUnit.MILLISECONDS);
                results.put(entry.getKey(), result);
            } catch (Exception e) {
                logger.error("Error analyzing sentiment for {}: {}", entry.getKey(), e.getMessage());
                failedSymbols.add(entry.getKey());
                // Add default result for failed analysis
                results.put(entry.getKey(), createDefaultSentimentResult(
                        entry.getKey(),
                        date,
                        SentimentType.NEUTRAL
                ));
            }
        }

        logger.info("Batch analysis complete: {} successful, {} failed",
                results.size() - failedSymbols.size(), failedSymbols.size());

        return results;
    }

    /**
     * Gracefully shuts down the service and thread pool.
     */
    public void shutdown() {
        logger.info("Shutting down sentiment analysis service");
        analysisExecutor.shutdown();
        try {
            if (!analysisExecutor.awaitTermination(30, TimeUnit.SECONDS)) {
                analysisExecutor.shutdownNow();
            }
        } catch (InterruptedException e) {
            analysisExecutor.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    /**
     * Keyword-based sentiment fallback when LLM is unavailable.
     * Classifies each headline by its overall sentiment using phrase matching,
     * then aggregates into a composite result.
     */
    private SentimentResult keywordBasedSentiment(String symbol, LocalDate analysisDate,
                                                  List<String> headlines) {
        int posCount = 0, negCount = 0, neuCount = 0;
        List<String> posHeadlines = new ArrayList<>();
        List<String> negHeadlines = new ArrayList<>();
        List<String> allFlags = new ArrayList<>();
        List<String> allCatalysts = new ArrayList<>();

        for (String h : headlines) {
            String lower = h.toLowerCase();
            HeadlineResult result = classifyHeadline(lower, h);

            if (result.classification() == Classification.POSITIVE) {
                posCount++;
                posHeadlines.add(h);
            } else if (result.classification() == Classification.NEGATIVE) {
                negCount++;
                negHeadlines.add(h);
            } else {
                neuCount++;
            }
            allFlags.addAll(result.extractedFlags());
            allCatalysts.addAll(result.extractedCatalysts());
        }

        // Build reasoning from actual headline content
        StringBuilder reasoning = new StringBuilder();
        if (posCount > 0 && negCount == 0) {
            reasoning.append("All ").append(posCount).append(" articles carry positive price momentum or growth signals");
            // Summarize the positive themes
            Set<String> themes = extractThemes(posHeadlines);
            if (!themes.isEmpty()) {
                reasoning.append(" (").append(String.join(", ", themes)).append(")");
            }
        } else if (negCount > 0 && posCount == 0) {
            reasoning.append("All ").append(negCount).append(" articles carry negative price pressure or risk signals");
        } else if (posCount > negCount) {
            reasoning.append(posCount).append(" positive vs ").append(negCount).append(" negative articles");
        } else if (negCount > posCount) {
            reasoning.append(negCount).append(" negative vs ").append(posCount).append(" positive articles");
        } else {
            reasoning.append("Mixed signals: ").append(posCount).append(" positive, ").append(negCount).append(" negative, ").append(neuCount).append(" neutral");
        }

        // Determine score and confidence
        SentimentResult.SentimentScore score;
        double confidence;

        if (posCount == 0 && negCount == 0 && neuCount > 0) {
            score = SentimentResult.SentimentScore.NEUTRAL;
            confidence = 0.5;
        } else if (posCount > negCount) {
            score = SentimentResult.SentimentScore.POSITIVE;
            confidence = Math.min(0.85, 0.5 + posCount * 0.15);
        } else if (negCount > posCount) {
            score = SentimentResult.SentimentScore.NEGATIVE;
            confidence = Math.min(0.85, 0.5 + (negCount - posCount) * 0.15);
        } else {
            score = SentimentResult.SentimentScore.NEUTRAL;
            confidence = 0.4;
        }

        // Deduplicate red flags and catalysts
        List<String> uniqueFlags = allFlags.stream().distinct().toList();
        List<String> uniqueCatalysts = allCatalysts.stream().distinct().toList();

        return SentimentResult.create(symbol, analysisDate, score,
                reasoning.toString(), "", confidence,
                uniqueFlags.isEmpty() ? List.of() : uniqueFlags,
                uniqueCatalysts.isEmpty() ? List.of() : uniqueCatalysts);
    }

    /**
     * Classifies a lowercase headline into POSITIVE, NEGATIVE, or NEUTRAL.
     * Uses multi-word phrases for accuracy, not just single keywords.
     */
    private HeadlineResult classifyHeadline(String lower, String original) {
        boolean hasPositiveSignal = false;
        boolean hasNegativeSignal = false;
        List<String> extractedCatalysts = new ArrayList<>();
        List<String> extractedFlags = new ArrayList<>();

        // === POSITIVE phrases (price movement, growth, bullish) ===
        if (lower.contains("shares climb") || lower.contains("shares surge") || lower.contains("shares jump") ||
            lower.contains("stock rallies") || lower.contains("stock gains") || lower.contains("stock rises") ||
            lower.contains("share price") && (lower.contains("climb") || lower.contains("surge") || lower.contains("rally") || lower.contains("gain") || lower.contains("rise") || lower.contains("jump")) ||
            lower.contains("up nearly") || lower.contains("up over") || lower.contains("up more than") ||
            lower.contains("beat estimates") || lower.contains("beat forecasts") || lower.contains("beats estimates") ||
            lower.contains("profit") && (lower.contains("record") || lower.contains("high") || lower.contains("rise") || lower.contains("jump")) ||
            lower.contains("strong") && (lower.contains("demand") || lower.contains("buy") || lower.contains("inflow") || lower.contains("fii") || lower.contains("di")) ||
            lower.contains("fii buy") || lower.contains("di buy") || lower.contains("inflow") ||
            lower.contains("upgrade") || lower.contains("raised") || lower.contains("revised") ||
            lower.contains("bullish") || lower.contains("outperform") || lower.contains("overweight") ||
            lower.contains("new record") || lower.contains("record high") || lower.contains("set stage") ||
            lower.contains("gains most") || lower.contains("stand to gain") ||
            lower.contains("near launch") && (lower.contains("shareholder") || lower.contains("investor") || lower.contains("gainer")) ||
            lower.contains("files for") && (lower.contains("ipo") || lower.contains("listing"))) {
            hasPositiveSignal = true;
        }

        // === NEGATIVE phrases (price decline, risk, distress) ===
        if (lower.contains("shares fall") || lower.contains("shares drop") || lower.contains("shares plunge") ||
            lower.contains("stock falls") || lower.contains("stock drops") || lower.contains("stock slides") ||
            lower.contains("stock declines") || lower.contains("stock drops") ||
            lower.contains("share price") && (lower.contains("fall") || lower.contains("drop") || lower.contains("slide") || lower.contains("plunge") || lower.contains("decline")) ||
            lower.contains("down over") || lower.contains("down nearly") || lower.contains("down more than") ||
            lower.contains("miss estimates") || lower.contains("missed estimates") || lower.contains("miss forecasts") ||
            lower.contains("loss") && !lower.contains("unrealised") && !lower.contains("mark") && !lower.contains("market") &&
            lower.contains("downgrade") || lower.contains("cut") && (lower.contains("target") || lower.contains("estimate") || lower.contains("forecast")) ||
            lower.contains("bearish") || lower.contains("underperform") || lower.contains("underweight") ||
            lower.contains("sued") || lower.contains("fraud") || lower.contains("probe") || lower.contains("scam") ||
            lower.contains("default") || lower.contains("bankrupt") ||
            lower.contains("warn") && (lower.contains("risk") || lower.contains("loss") || lower.contains("delay")) ||
            lower.contains("fii sell") || lower.contains("di sell") || lower.contains("outflow")) {
            hasNegativeSignal = true;
        }

        // === CATALYSTS (corporate events — neutral on their own, can be positive context) ===
        if (hasPositiveSignal) {
            if (lower.contains("ipo")) {
                String entity = extractEntityNear(original, "ipo");
                extractedCatalysts.add("IPO catalyst: " + entity);
            }
            if (lower.contains("acquisition") || lower.contains("merger") || lower.contains("partnership")) {
                extractedCatalysts.add("M&A/Partnership catalyst");
            }
            if (lower.contains("upgrade") || lower.contains("raised target")) {
                extractedCatalysts.add("Analyst positive action");
            }
        }

        // Red flags
        if (hasNegativeSignal) {
            if (lower.contains("fraud") || lower.contains("sued") || lower.contains("probe") || lower.contains("scam")) {
                extractedFlags.add("Regulatory/legal investigation");
            }
            if (lower.contains("default") || lower.contains("bankrupt") || lower.contains("debt crisis")) {
                extractedFlags.add("Financial distress signal");
            }
            if (lower.contains("warn") && (lower.contains("risk") || lower.contains("loss"))) {
                extractedFlags.add("Company warning on risks/losses");
            }
        }

        Classification cls;
        if (hasPositiveSignal && hasNegativeSignal) cls = Classification.NEUTRAL;
        else if (hasPositiveSignal) cls = Classification.POSITIVE;
        else if (hasNegativeSignal) cls = Classification.NEGATIVE;
        // Check for neutral corporate events (IPO, listing, etc.) — these are catalysts, not sentiment
        else if (lower.contains("ipo") || lower.contains("listing") || lower.contains("demerger") ||
            lower.contains("split") || lower.contains("bonus") || lower.contains("right") && lower.contains("issue") ||
            lower.contains("near launch") || lower.contains("files for")) {
            cls = Classification.NEUTRAL;
        } else {
            cls = Classification.NEUTRAL;
        }

        return new HeadlineResult(cls, extractedCatalysts, extractedFlags);
    }

    /**
     * Extracts key themes from a list of headlines.
     */
    private Set<String> extractThemes(List<String> headlines) {
        Set<String> themes = new HashSet<>();
        for (String h : headlines) {
            String lower = h.toLowerCase();
            if (lower.contains("ipo") || lower.contains("listing") || lower.contains("files for") || lower.contains("near launch")) {
                themes.add("IPO activity");
            }
            if (lower.contains("climb") || lower.contains("surge") || lower.contains("rally") || lower.contains("gain")) {
                themes.add("Price momentum");
            }
            if (lower.contains("profit") || lower.contains("revenue") || lower.contains("growth")) {
                themes.add("Earnings strength");
            }
            if (lower.contains("fii") || lower.contains("di") || lower.contains("inflow")) {
                themes.add("Foreign/domestic buying");
            }
            if (lower.contains("upgrade") || lower.contains("target")) {
                themes.add("Analyst upgrade");
            }
        }
        return themes;
    }

    // ============ Weekly Sector Digest Methods ============

    /**
     * Generates a weekly sector sentiment digest.
     * Groups sentiment results by stock sector and identifies top positive/negative sectors.
     *
     * @param startDate the start date of the date range (inclusive)
     * @param endDate the end date of the date range (inclusive)
     * @return formatted digest string ready for Discord
     */
    public String generateSectorDigest(LocalDate startDate, LocalDate endDate) {
        logger.info("Generating sector digest for date range: {} to {}", startDate, endDate);

        // Fetch all sentiment results in date range
        List<SentimentResult> results = sentimentStore
                .findAllByDateBetween(startDate, endDate);

        if (results.isEmpty()) {
            return formatEmptyDigest(startDate, endDate);
        }

        // Group by sector and count sentiment
        Map<Stock.Sector, Map<SentimentResult.SentimentScore, Long>> sectorCounts =
                groupBySectorAndSentiment(results);

        // Get sector names for each symbol
        Map<String, Stock.Sector> symbolToSector = buildSymbolToSectorMap(results);

        // Identify top sectors
        List<Stock.Sector> topPositiveSectors = getTopSectors(sectorCounts, 3, true);
        List<Stock.Sector> topNegativeSectors = getTopSectors(sectorCounts, 3, false);

        // Calculate totals
        long totalStocks = symbolToSector.size();
        long totalPositive = sectorCounts.values().stream()
                .map(m -> m.get(SentimentResult.SentimentScore.POSITIVE))
                .filter(Objects::nonNull)
                .mapToLong(Long::longValue)
                .sum();
        long totalNeutral = sectorCounts.values().stream()
                .map(m -> m.get(SentimentResult.SentimentScore.NEUTRAL))
                .filter(Objects::nonNull)
                .mapToLong(Long::longValue)
                .sum();
        long totalNegative = sectorCounts.values().stream()
                .map(m -> m.get(SentimentResult.SentimentScore.NEGATIVE))
                .filter(Objects::nonNull)
                .mapToLong(Long::longValue)
                .sum();

        return formatSectorDigest(
                startDate, endDate, sectorCounts, topPositiveSectors,
                topNegativeSectors, totalStocks, totalPositive, totalNeutral, totalNegative,
                symbolToSector
        );
    }

    /**
     * Groups sentiment results by stock sector and sentiment score.
     *
     * @param results list of sentiment results
     * @return map of sector -> (sentiment score -> count)
     */
    public Map<Stock.Sector, Map<SentimentResult.SentimentScore, Long>> groupBySectorAndSentiment(
            List<SentimentResult> results) {

        Map<Stock.Sector, Map<SentimentResult.SentimentScore, Long>> sectorMap = new HashMap<>();

        for (SentimentResult result : results) {
            Stock.Sector sector = getSectorForSymbol(result.symbol());
            if (sector == null) {
                sector = Stock.Sector.OTHERS;
            }

            SentimentResult.SentimentScore score = result.score();

            sectorMap.computeIfAbsent(sector, k -> new HashMap<>());
            sectorMap.get(sector).merge(score, 1L, Long::sum);
        }

        return sectorMap;
    }

    /**
     * Gets sector for a symbol by looking up in the database.
     *
     * @param symbol stock symbol
     * @return the sector for the stock, or null if not found
     */
    public Stock.Sector getSectorForSymbol(String symbol) {
        Optional<Stock> stock = stockStore.findBySymbol(symbol);
        return stock.map(Stock::sector)
                .orElse(null);
    }

    /**
     * Builds a map of symbol to sector from sentiment results.
     *
     * @param results list of sentiment results
     * @return map of symbol -> sector
     */
    private Map<String, Stock.Sector> buildSymbolToSectorMap(List<SentimentResult> results) {
        Map<String, Stock.Sector> map = new HashMap<>();

        for (SentimentResult result : results) {
            if (!map.containsKey(result.symbol().toUpperCase())) {
                Stock.Sector sector = getSectorForSymbol(result.symbol().toUpperCase());
                map.put(result.symbol().toUpperCase(), sector != null ? sector : Stock.Sector.OTHERS);
            }
        }

        return map;
    }

    /**
     * Gets top sectors by a specific sentiment score.
     *
     * @param sectorCounts map of sector -> (sentiment score -> count)
     * @param count number of top sectors to return
     * @param positive if true, get top positive sectors; if false, get top negative
     * @return list of top sectors
     */
    public List<Stock.Sector> getTopSectors(
            Map<Stock.Sector, Map<SentimentResult.SentimentScore, Long>> sectorCounts,
            int count, boolean positive) {

        return sectorCounts.entrySet().stream()
                .sorted(Map.Entry.<Stock.Sector, Map<SentimentResult.SentimentScore, Long>>comparingByValue(
                        (m1, m2) -> {
                            Long v1 = positive ?
                                    m1.getOrDefault(SentimentResult.SentimentScore.POSITIVE, 0L) :
                                    m1.getOrDefault(SentimentResult.SentimentScore.NEGATIVE, 0L);
                            Long v2 = positive ?
                                    m2.getOrDefault(SentimentResult.SentimentScore.POSITIVE, 0L) :
                                    m2.getOrDefault(SentimentResult.SentimentScore.NEGATIVE, 0L);
                            return Long.compare(v2, v1); // Descending order
                        }
                ))
                .limit(count)
                .map(Map.Entry::getKey)
                .collect(Collectors.toList());
    }

    /**
     * Formats the sector digest output string for Telegram.
     *
     * @param startDate start date of the range
     * @param endDate end date of the range
     * @param sectorCounts map of sector sentiment counts
     * @param topPositiveSectors list of top positive sectors
     * @param topNegativeSectors list of top negative sectors
     * @param totalStocks total number of stocks analyzed
     * @param totalPositive total positive signals
     * @param totalNeutral total neutral signals
     * @param totalNegative total negative signals
     * @param symbolToSector map of symbol to sector
     * @return formatted digest string
     */
    private String formatSectorDigest(
            LocalDate startDate, LocalDate endDate,
            Map<Stock.Sector, Map<SentimentResult.SentimentScore, Long>> sectorCounts,
            List<Stock.Sector> topPositiveSectors,
            List<Stock.Sector> topNegativeSectors,
            long totalStocks, long totalPositive, long totalNeutral, long totalNegative,
            Map<String, Stock.Sector> symbolToSector) {

        StringBuilder sb = new StringBuilder();
        sb.append("📊 Weekly Sector Sentiment Digest\n");
        sb.append(String.format("Week of: %s to %s\n\n", startDate, endDate));

        // Top Positive Sectors
        sb.append("*Top Positive Sectors:*\n");
        int posIndex = 1;
        for (Stock.Sector sector : topPositiveSectors) {
            Map<SentimentResult.SentimentScore, Long> counts = sectorCounts.get(sector);
            String posCount = counts != null ?
                    counts.getOrDefault(SentimentResult.SentimentScore.POSITIVE, 0L).toString() : "0";
            String neuCount = counts != null ?
                    counts.getOrDefault(SentimentResult.SentimentScore.NEUTRAL, 0L).toString() : "0";
            String negCount = counts != null ?
                    counts.getOrDefault(SentimentResult.SentimentScore.NEGATIVE, 0L).toString() : "0";

            sb.append(String.format("%d. %s - %s POS, %s NEU, %s NEG\n",
                    posIndex++, sector.name(), posCount, neuCount, negCount));
        }

        if (topPositiveSectors.isEmpty()) {
            sb.append("No positive sectors data available\n");
        }

        sb.append("\n*Top Negative Sectors:*\n");
        int negIndex = 1;
        for (Stock.Sector sector : topNegativeSectors) {
            Map<SentimentResult.SentimentScore, Long> counts = sectorCounts.get(sector);
            String posCount = counts != null ?
                    counts.getOrDefault(SentimentResult.SentimentScore.POSITIVE, 0L).toString() : "0";
            String neuCount = counts != null ?
                    counts.getOrDefault(SentimentResult.SentimentScore.NEUTRAL, 0L).toString() : "0";
            String negCount = counts != null ?
                    counts.getOrDefault(SentimentResult.SentimentScore.NEGATIVE, 0L).toString() : "0";

            sb.append(String.format("%d. %s - %s POS, %s NEU, %s NEG\n",
                    negIndex++, sector.name(), posCount, neuCount, negCount));
        }

        if (topNegativeSectors.isEmpty()) {
            sb.append("No negative sectors data available\n");
        }

        // Summary stats
        sb.append("\n*Summary Statistics*:\n");
        sb.append(String.format("Total stocks analyzed: %d\n", totalStocks));
        sb.append(String.format("Positive signals: %d | Neutral: %d | Negative: %d\n",
                totalPositive, totalNeutral, totalNegative));

        return sb.toString();
    }

    /**
     * Formats an empty digest when no sentiment data is available.
     *
     * @param startDate start date of the range
     * @param endDate end date of the range
     * @return formatted empty digest string
     */
    private String formatEmptyDigest(LocalDate startDate, LocalDate endDate) {
        return String.format(
                "📊 Weekly Sector Sentiment Digest\n" +
                "Week of: %s to %s\n\n" +
                "No sentiment data available for the specified date range.\n\n" +
                "*Summary Statistics*:\n" +
                "Total stocks analyzed: 0\n" +
                "Positive signals: 0 | Neutral: 0 | Negative: 0",
                startDate, endDate
        );
    }

    /**
     * Generates sector digest for the last calendar week (Sunday to Saturday).
     * This is called by the scheduled job.
     *
     * @return formatted digest string ready for Discord
     */
    public String generateSectorDigestForLastWeek() {
        ZoneId ist = ZoneId.of("Asia/Kolkata");
        LocalDate today = LocalDate.now(ist);

        // Get last Sunday (or today if today is Sunday)
        LocalDate endDate = today.with(TemporalAdjusters.previousOrSame(java.time.DayOfWeek.SUNDAY));

        // If today is Sunday, use today as end, otherwise use Saturday of this week
        if (!today.equals(endDate)) {
            endDate = today.minusDays(1).with(TemporalAdjusters.previousOrSame(java.time.DayOfWeek.SUNDAY));
        }

        // Get start date (previous Monday, or 7 days before end date)
        LocalDate startDate = endDate.minusDays(6);

        logger.info("Generating sector digest for week: {} to {}", startDate, endDate);
        return generateSectorDigest(startDate, endDate);
    }

    /**
     * Extracts a short entity name from a headline near a keyword.
     * Looks for meaningful capitalized words (skipping newspaper names and generic terms).
     */
    private String extractEntityNear(String headline, String keyword) {
        String lower = headline.toLowerCase();
        int idx = lower.indexOf(keyword);
        if (idx == -1) return keyword;

        int start = Math.max(0, idx - 50);
        int end = Math.min(lower.length(), idx + 15);
        String snippet = headline.substring(start, end);

        // Words to skip (newspaper names, generic terms, connectors)
        Set<String> skipWords = Set.of("Hindu", "Times", "Standard", "Post", "Express",
                "News", "IPOs", "IPO", "The", "For", "And", "As", "At", "In", "On", "Is");

        String[] words = snippet.split("\\s+");
        for (int i = words.length - 1; i >= 0; i--) {
            String w = words[i].replaceAll("[^a-zA-Z]", "");
            if (w.length() > 2 && Character.isUpperCase(w.charAt(0)) && !skipWords.contains(w)) {
                return w;
            }
        }
        // Fallback: return first 30 chars of snippet
        return snippet.trim().length() > 30 ? snippet.trim().substring(0, 30) + "..." : snippet.trim();
    }

    // ============ Laya/Qwen Agreement Reporting ============

    /**
     * Computes the Laya/Qwen agreement rate over {@code [startDate, endDate]}
     * (inclusive), for symbol/date pairs where both a LAYA and a QWEN row exist
     * in the shadow-mode classification log. On-demand only — not wired to a
     * scheduled job.
     *
     * @return the agreement result; {@code comparedPairs} is 0 (rate 0.0) when
     *         the classification log isn't available or no pair exists in range
     */
    public LayaQwenAgreementResult computeLayaQwenAgreementRate(LocalDate startDate, LocalDate endDate) {
        if (classificationLogRepository == null) {
            return new LayaQwenAgreementResult(0, 0, 0.0);
        }

        List<SentimentClassificationLogEntity> rows =
                classificationLogRepository.findByAnalysisDateBetweenAndShadowModeTrue(startDate, endDate);

        record SymbolDate(String symbol, LocalDate date) {}
        Map<SymbolDate, String> layaSentimentByPair = new HashMap<>();
        Map<SymbolDate, String> qwenSentimentByPair = new HashMap<>();
        for (SentimentClassificationLogEntity row : rows) {
            SymbolDate key = new SymbolDate(row.getSymbol(), row.getAnalysisDate());
            if (SentimentClassificationLogEntity.ModelUsed.LAYA.name().equals(row.getModelUsed())) {
                layaSentimentByPair.put(key, row.getSentiment());
            } else if (SentimentClassificationLogEntity.ModelUsed.QWEN.name().equals(row.getModelUsed())) {
                qwenSentimentByPair.put(key, row.getSentiment());
            }
        }

        long comparedPairs = 0;
        long agreedPairs = 0;
        for (Map.Entry<SymbolDate, String> entry : layaSentimentByPair.entrySet()) {
            String qwenSentiment = qwenSentimentByPair.get(entry.getKey());
            if (qwenSentiment == null) continue;
            comparedPairs++;
            if (qwenSentiment.equals(entry.getValue())) {
                agreedPairs++;
            }
        }

        double agreementRate = comparedPairs == 0 ? 0.0 : (double) agreedPairs / comparedPairs;
        return new LayaQwenAgreementResult(comparedPairs, agreedPairs, agreementRate);
    }

    /**
     * Result of {@link #computeLayaQwenAgreementRate}.
     *
     * @param comparedPairs number of symbol/date pairs with both a LAYA and a QWEN row
     * @param agreedPairs   of those, the number where the two sentiments matched
     * @param agreementRate {@code agreedPairs / comparedPairs}, or 0.0 if comparedPairs is 0
     */
    public record LayaQwenAgreementResult(long comparedPairs, long agreedPairs, double agreementRate) {}

    // ============ Test Accessor Methods ============

    /**
     * Gets the news ingestion service for testing purposes.
     * This method is intended for test access only.
     *
     * @return the newsIngestionService
     */
    NewsIngestionService getNewsIngestionService() {
        return newsIngestionService;
    }

    /** Headline classification result. */
    private enum Classification { POSITIVE, NEGATIVE, NEUTRAL }

    /**
     * Result of classifying a single headline, including extracted signals.
     */
    private record HeadlineResult(Classification classification, List<String> extractedCatalysts, List<String> extractedFlags) {
        static HeadlineResult of(Classification c) { return new HeadlineResult(c, List.of(), List.of()); }
    }
}
