package com.swingtrade.llm.config;

import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.llm.client.SpringAiLlmClient;
import org.springframework.ai.chat.client.ChatClient;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.ai.openai.OpenAiChatOptions;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;

import java.time.Duration;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Configuration class for LLM module.
 * Provides five OpenAiChatModel beans (local, pi_ssh, openai, ollama, laya),
 * a default ChatClient/SpringAiLlmClient, and a news executor.
 * Reads base URLs from AppSettingsStore (DB) so the settings UI controls endpoints.
 *
 * Spring AI 2.0.1 — uses official openai-java SDK with OkHttp transport.
 */
@Configuration
@EnableConfigurationProperties(LlmProperties.class)
public class LlmConfig {

    private enum ApiKeySetting {
        OPENAI("openai.api_key"),
        OLLAMA("ollama.api_key");

        private final String key;

        ApiKeySetting(String key) {
            this.key = key;
        }
    }

    /**
     * Spring AI's {@code AbstractOpenAiOptions.DEFAULT_TIMEOUT} is 60 seconds with
     * {@code DEFAULT_MAX_RETRIES} of 3 (i.e. up to 4 total HTTP attempts) when a
     * chat model is built without an explicit timeout/maxRetries — see
     * {@code org.springframework.ai.openai.AbstractOpenAiOptions}. That default is
     * fine for a real remote/GPU-backed endpoint (openai, gpuhub) that reliably
     * responds within tens of seconds, but every backend that ends up running
     * llama.cpp on this Pi's CPU — local, pi_ssh, and ollama alike — can take
     * several minutes to generate a large completion (llama-server here runs with
     * only 2 threads; see PiLlamaServerManager/LlamaCppServerManager). Left
     * unconfigured, 4 attempts x 60s each burns ~4 minutes before failing — which is
     * exactly what caused the silent 243s "Stage 9" hang on the pi_ssh backend:
     * every attempt was independently timing out and being retried, not a single
     * slow-but-working call. Retrying also wastes CPU cycles (and heat) on a
     * request that's already running, for no benefit.
     *
     * Confirmed by testing: a single attempt with a 300s timeout still failed with
     * okhttp3 SocketTimeoutException while awaiting response headers (see
     * SynthesisService's WARN/DEBUG logs) — qwen3:4b on CPU genuinely needs several
     * minutes of wall time to generate a maxTokens:1024 response for a synthesis
     * prompt this size, it is not hanging/erroring on the server side. 570s gives it
     * the most runway possible while staying under SynthesisService's outer 600s
     * Mono#block deadline, so a true failure still surfaces within ~10 minutes.
     *
     * local, pi_ssh, and ollama all get this override since they're all CPU-bound
     * llama.cpp on this Pi; openai/gpuhub keep Spring AI's tighter defaults since
     * those are genuinely fast remote endpoints.
     *
     * 570s (leaving 30s under the old 600s outer deadline) turned out too tight in
     * practice: generation time on this Pi varies run to run (289s one run, >570s
     * the next, same prompt shape) — with zero retries, hitting the ceiling
     * discards an almost-finished generation, wasting the CPU/heat already spent
     * for nothing. 900s gave real, slow-but-working calls room to finish at the
     * time.
     *
     * Raised again to 2850s when SentimentService.MAX_ARTICLE_CHARS went from a
     * throughput-driven 450 to a context-budget-driven 1200 chars/article: full
     * (uncapped-for-speed) article content is worth more prompt-eval time, not
     * less content — see MAX_ARTICLE_CHARS's javadoc for the token math.
     *
     * Two earlier attempts at this number (1400s, then 1800s) were both measured
     * too tight, for two compounding reasons:
     *
     * 1. Decode ("eval") throughput on this Pi is NOT the same as prompt-eval
     *    throughput — decode is single-token/sequential (no batching benefit),
     *    so it is far more memory-latency-bound. A real run's llama-server
     *    timing log measured decode as slow as 0.54 tokens/sec.
     * 2. Prompt-eval throughput is not even CONSTANT within one request: the run
     *    that produced the 0.54 tok/s decode number started prompt eval at 8.28
     *    tokens/sec and had degraded to a 4.34 tokens/sec cumulative average by
     *    the time its ~6680-token prompt finished evaluating 26 minutes later —
     *    consistent with thermal throttling building up over a single long call,
     *    not just run-to-run noise.
     *
     * That run got lucky: the model stopped at 111 decoded tokens, not the 512
     * the request allowed. Sizing for the real worst case — MAX_ARTICLE_CHARS's
     * ~7000-token prompt ceiling, decoding the full 512-token response, at the
     * worst sustained rates actually observed (~4.0 tok/s eval, ~0.5 tok/s
     * decode) — gives:
     *   prompt eval 7000/4.0 ~= 1750s, decode 512/0.5 ~= 1024s, total ~= 2774s.
     * 2850s leaves a small margin over that. The outer Mono#block deadlines in
     * SentimentService/SynthesisService are widened to 2880s to match (see their
     * TIMEOUT_SECONDS/ANALYSIS_TIMEOUT_SECONDS), and
     * AnalysisOrchestrationController's SSE emitter timeout covers both stages
     * back-to-back at that ceiling. Yes, this means a genuinely worst-case run
     * can legitimately take well over an hour — that is the real cost of not
     * trimming article content for speed on this hardware, not a bug.
     */
    private static final Duration LOCAL_LLAMA_TIMEOUT = Duration.ofSeconds(2850);
    private static final int LOCAL_LLAMA_MAX_RETRIES = 0;

    @Bean
    public ExecutorService newsExecutor() {
        return Executors.newFixedThreadPool(4, r -> {
            Thread t = new Thread(r, "news-fetcher");
            t.setDaemon(true);
            return t;
        });
    }

    @Bean
    public OpenAiChatModel localChatModel(
            AppSettingsStore appSettingsStore,
            LlmProperties properties,
            @Value("${spring.ai.openai.api-key:none}") String apiKey) {
        // CPU-bound local inference: generous per-attempt timeout, no retries (see
        // LOCAL_LLAMA_TIMEOUT/LOCAL_LLAMA_MAX_RETRIES javadoc above).
        return createChatModel(appSettingsStore, "llm.base_url", "openai.model",
                properties.getProviders().getLocal(), ApiKeySetting.OPENAI, apiKey,
                LOCAL_LLAMA_TIMEOUT, LOCAL_LLAMA_MAX_RETRIES);
    }

    @Bean
    public OpenAiChatModel piSshChatModel(
            AppSettingsStore appSettingsStore,
            LlmProperties properties,
            @Value("${spring.ai.openai.api-key:none}") String apiKey) {
        // CPU-bound local inference on the Pi host: generous per-attempt timeout,
        // no retries (see LOCAL_LLAMA_TIMEOUT/LOCAL_LLAMA_MAX_RETRIES javadoc above).
        return createChatModel(appSettingsStore, "llm.base_url", "llamacpp.model",
                properties.getProviders().getPiSsh(), properties.getLlamaCpp().getModel(),
                ApiKeySetting.OPENAI, apiKey,
                LOCAL_LLAMA_TIMEOUT, LOCAL_LLAMA_MAX_RETRIES);
    }

    @Bean
    public OpenAiChatModel openAiChatModel(
            AppSettingsStore appSettingsStore,
            LlmProperties properties,
            @Value("${spring.ai.openai.api-key:none}") String apiKey) {
        return createChatModel(appSettingsStore, "openai.base_url", "openai.model",
                properties.getProviders().getOpenai(), ApiKeySetting.OPENAI, apiKey);
    }

    @Bean
    public OpenAiChatModel layaChatModel(
            AppSettingsStore appSettingsStore,
            LlmProperties properties,
            @Value("${spring.ai.openai.api-key:none}") String apiKey) {
        // Laya is an OpenAI-compatible remote backend: its endpoint and model are
        // discoverable and editable from the dashboard (laya.base_url / laya.model)
        // with the same tight OpenAI defaults as the OpenAI backend.
        return createChatModel(appSettingsStore, "laya.base_url", "laya.model",
                properties.getProviders().getLaya(), ApiKeySetting.OPENAI, apiKey);
    }

    @Bean
    public OpenAiChatModel ollamaChatModel(
            AppSettingsStore appSettingsStore,
            LlmProperties properties,
            @Value("${spring.ai.openai.api-key:none}") String apiKey) {
        // CPU-bound local inference: generous per-attempt timeout, no retries (see
        // LOCAL_LLAMA_TIMEOUT/LOCAL_LLAMA_MAX_RETRIES javadoc above) — a
        // slow-but-working request should be given time to finish rather than be
        // replayed.
        return createChatModel(appSettingsStore, "ollama.base_url", "ollama.model",
                properties.getProviders().getOllama(), ApiKeySetting.OLLAMA, apiKey,
                LOCAL_LLAMA_TIMEOUT, LOCAL_LLAMA_MAX_RETRIES);
    }

    private OpenAiChatModel createChatModel(AppSettingsStore settings, String urlKey,
                                            String modelKey, LlmProperties.Provider defaults,
                                            ApiKeySetting apiKeySetting, String apiKey) {
        return createChatModel(settings, urlKey, modelKey, defaults, apiKeySetting, apiKey, null, null);
    }

    private OpenAiChatModel createChatModel(AppSettingsStore settings, String urlKey,
                                            String modelKey, LlmProperties.Provider defaults,
                                            ApiKeySetting apiKeySetting, String springDefaultApiKey,
                                            Duration timeout, Integer maxRetries) {
        return createChatModel(settings, urlKey, modelKey, defaults, defaults.getModel(),
                apiKeySetting, springDefaultApiKey, timeout, maxRetries);
    }

    private OpenAiChatModel createChatModel(AppSettingsStore settings, String urlKey,
                                            String modelKey, LlmProperties.Provider defaults,
                                            String defaultModel, ApiKeySetting apiKeySetting,
                                            String springDefaultApiKey, Duration timeout,
                                            Integer maxRetries) {
        String baseUrl = settings.get(urlKey)
                .orElseGet(() -> defaults.getBaseUrl().toString());
        String model = settings.get(modelKey).orElse(defaultModel);
        String key = resolveApiKey(settings, apiKeySetting, springDefaultApiKey);

        OpenAiChatOptions.Builder optionsBuilder = OpenAiChatOptions.builder()
            .model(model)
            .baseUrl(baseUrl)
            .apiKey(key)
            // Sentiment and other structured outputs should be reproducible.
            // Providers that support temperature receive the deterministic floor.
            .temperature(0.0);
        if (timeout != null) {
            optionsBuilder.timeout(timeout);
        }
        if (maxRetries != null) {
            optionsBuilder.maxRetries(maxRetries);
        }

        return OpenAiChatModel.builder()
            .options(optionsBuilder.build())
            .build();
    }

    private String resolveApiKey(AppSettingsStore settings, ApiKeySetting setting, String springDefault) {
        String key = settings.get(setting.key).orElse(springDefault);
        if (setting == ApiKeySetting.OPENAI && ("none".equals(key) || key.isBlank())) {
            key = settings.get("gpuhub.api_key").orElse(key);
        }
        if ("none".equals(key) || key.isBlank()) {
            return "";
        }
        return key;
    }

    @Primary
    @Bean
    public ChatClient chatClient(@Qualifier("localChatModel") OpenAiChatModel localChatModel) {
        return ChatClient.create(localChatModel);
    }

    @Bean
    public SpringAiLlmClient springAiLlmClient(ChatClient chatClient,
                                               @Value("${llm.cot.enabled:false}") boolean enableCoT,
                                               @Value("${llm.reasoning-effort:}") String reasoningEffort) {
        return new SpringAiLlmClient(chatClient, enableCoT, reasoningEffort);
    }
}
