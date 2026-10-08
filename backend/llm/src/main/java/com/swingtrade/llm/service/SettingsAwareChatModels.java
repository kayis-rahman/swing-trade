package com.swingtrade.llm.service;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.llm.config.LlmProperties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.ai.openai.OpenAiChatOptions;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.net.URI;
import java.time.Duration;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import okhttp3.Interceptor;
import okhttp3.Request;
import okhttp3.Response;

/**
 * Resolves the {@link OpenAiChatModel} for the selected backend from the live
 * app_settings at call time, rebuilding the model whenever the resolved
 * configuration changes.
 *
 * <p>This exists because {@code LlmConfig} builds its model beans once at container
 * startup: settings written later through {@code PUT /api/settings/llm} updated the
 * database and the settings cache but never reached the running client, while
 * {@link LlmBackendSelector} read {@code llm.backend} dynamically — so routing and
 * model configuration diverged and the orchestrator's LLM stages degraded against a
 * stale configuration (run 5721aa09: openai.base_url/openai.model/openai.api_key
 * written at 21:08:16Z; the 21:08:30Z run still used the startup bean's
 * api.openai.com with the placeholder key "not-needed" and got a 401).
 *
 * <p>The OpenAI client inside {@link OpenAiChatModel} is bound to its options at
 * construction — per-call option overrides cannot retarget it — so the model itself
 * is rebuilt when the resolved configuration changes. Rebuilds happen only on an
 * actual settings write; every other call is a cache hit.
 */
@Component
public class SettingsAwareChatModels {

    private static final Logger logger = LoggerFactory.getLogger(SettingsAwareChatModels.class);

    /** Mirrors LlmConfig's startup-time choices for the CPU-bound llama.cpp backends. */
    private static final Duration LOCAL_LLAMA_TIMEOUT = Duration.ofSeconds(2850);
    private static final int LOCAL_LLAMA_MAX_RETRIES = 0;

    /**
     * app_settings key holding extra request headers as a JSON object, for endpoints
     * that require them — e.g. {@code {"x-opencode-session":"swing-trade-stage"}} for
     * opencode.ai's /zen/go bridge, which answers 400 MissingSessionID without it.
     */
    public static final String EXTRA_HEADERS_KEY = "llm.extra_headers";

    private static final String OPENAI_API_KEY_KEY = "openai.api_key";
    private static final String OLLAMA_API_KEY_KEY = "ollama.api_key";
    private static final String GPUHUB_API_KEY_KEY = "gpuhub.api_key";
    private static final String OPENCODE_SESSION_HEADER = "x-opencode-session";
    private static final String DEFAULT_OPENCODE_SESSION = "swing-trade-orchestrator";

    private static final ObjectMapper OBJECT_MAPPER = new ObjectMapper();

    /** Everything the model build depends on; equality decides whether to rebuild. */
    public record ResolvedModelConfig(String baseUrl, String model, String apiKey, Duration timeout,
                                      Integer maxRetries, Map<String, String> extraHeaders) {
    }

    private record CachedModel(ResolvedModelConfig config, OpenAiChatModel model) {
    }

    private final AppSettingsStore settings;
    private final LlmProperties properties;
    private final String springDefaultApiKey;
    private final Map<LlmBackendSelector.Backend, CachedModel> cache = new ConcurrentHashMap<>();

    public SettingsAwareChatModels(AppSettingsStore settings,
                                   LlmProperties properties,
                                   @Value("${spring.ai.openai.api-key:none}") String springDefaultApiKey) {
        this.settings = settings;
        this.properties = properties;
        this.springDefaultApiKey = springDefaultApiKey;
    }

    /** Returns the live model for the backend, rebuilding it after a settings change. */
    public OpenAiChatModel resolve(LlmBackendSelector.Backend backend) {
        ResolvedModelConfig config = resolveConfig(backend);
        CachedModel cached = cache.get(backend);
        if (cached != null && cached.config().equals(config)) {
            return cached.model();
        }
        OpenAiChatModel model = buildModel(config);
        cache.put(backend, new CachedModel(config, model));
        if (cached == null) {
            logger.info("Built initial {} LLM model for backend {} at {}",
                    config.model(), backend, config.baseUrl());
        } else {
            logger.info("Rebuilt {} LLM model for backend {} at {} after configuration change",
                    config.model(), backend, config.baseUrl());
        }
        return model;
    }

    /** The effective configuration for a backend: app_settings over LlmProperties defaults. */
    public ResolvedModelConfig resolveConfig(LlmBackendSelector.Backend backend) {
        return switch (backend) {
            case LOCAL -> resolve("llm.base_url", "openai.model", OPENAI_API_KEY_KEY,
                    properties.getProviders().getLocal().getBaseUrl(),
                    properties.getProviders().getLocal().getModel(),
                    LOCAL_LLAMA_TIMEOUT, LOCAL_LLAMA_MAX_RETRIES);
            case PI_SSH -> resolve("llm.base_url", "llamacpp.model", OPENAI_API_KEY_KEY,
                    properties.getProviders().getPiSsh().getBaseUrl(),
                    properties.getLlamaCpp().getModel(),
                    LOCAL_LLAMA_TIMEOUT, LOCAL_LLAMA_MAX_RETRIES);
            case OPENAI -> resolve("openai.base_url", "openai.model", OPENAI_API_KEY_KEY,
                    properties.getProviders().getOpenai().getBaseUrl(),
                    properties.getProviders().getOpenai().getModel(),
                    null, null);
            case LAYA -> resolve("laya.base_url", "laya.model", OPENAI_API_KEY_KEY,
                    properties.getProviders().getLaya().getBaseUrl(),
                    properties.getProviders().getLaya().getModel(),
                    null, null);
            case OLLAMA -> resolve("ollama.base_url", "ollama.model", OLLAMA_API_KEY_KEY,
                    properties.getProviders().getOllama().getBaseUrl(),
                    properties.getProviders().getOllama().getModel(),
                    LOCAL_LLAMA_TIMEOUT, LOCAL_LLAMA_MAX_RETRIES);
            case PI_AGENT ->
                throw new IllegalStateException("Pi agent should be selected before model routing");
        };
    }

    private ResolvedModelConfig resolve(String urlKey, String modelKey, String apiKeyKey,
                                        URI defaultBaseUrl, String defaultModel,
                                        Duration timeout, Integer maxRetries) {
        String baseUrl = settings.get(urlKey)
                .orElseGet(() -> defaultBaseUrl == null ? null : defaultBaseUrl.toString());
        String model = settings.get(modelKey).orElse(defaultModel);
        return new ResolvedModelConfig(baseUrl, model, resolveApiKey(apiKeyKey), timeout, maxRetries,
                resolveExtraHeaders(baseUrl));
    }

    /**
     * Mirrors LlmConfig's key resolution (settings over the spring.ai.openai.api-key
     * default, gpuhub.api_key as the openai fallback) and additionally treats the
     * "not-needed" property placeholder as missing, so a missing key resolves to
     * empty instead of masquerading as a credential that only fails when a remote
     * API rejects it with a 401.
     */
    private String resolveApiKey(String apiKeyKey) {
        String key = settings.get(apiKeyKey).orElse(springDefaultApiKey);
        if (OPENAI_API_KEY_KEY.equals(apiKeyKey) && isPlaceholderKey(key)) {
            key = settings.get(GPUHUB_API_KEY_KEY).orElse(key);
        }
        return isPlaceholderKey(key) ? "" : key;
    }

    private boolean isPlaceholderKey(String key) {
        return key == null || key.isBlank() || "none".equals(key) || "not-needed".equals(key);
    }

    private Map<String, String> resolveExtraHeaders(String baseUrl) {
        Map<String, String> headers = new HashMap<>();
        String raw = settings.get(EXTRA_HEADERS_KEY).orElse("");
        if (!raw.isBlank()) {
            try {
                Map<String, String> parsed = OBJECT_MAPPER.readValue(raw, new TypeReference<>() {
                });
                if (parsed != null) {
                    headers.putAll(parsed);
                }
            } catch (Exception e) {
                logger.warn("Ignoring malformed {} setting: {}", EXTRA_HEADERS_KEY, e.getMessage());
            }
        }
        if (isOpenCodeEndpoint(baseUrl)) {
            headers.putIfAbsent(OPENCODE_SESSION_HEADER, DEFAULT_OPENCODE_SESSION);
        }
        return Map.copyOf(headers);
    }

    private boolean isOpenCodeEndpoint(String baseUrl) {
        if (baseUrl == null) {
            return false;
        }
        String host = URI.create(baseUrl).getHost();
        return host != null && (host.equals("opencode.ai") || host.endsWith(".opencode.ai"));
    }

    private OpenAiChatModel buildModel(ResolvedModelConfig config) {
        OpenAiChatOptions.Builder optionsBuilder = OpenAiChatOptions.builder()
            .model(config.model())
            .baseUrl(config.baseUrl())
            .apiKey(config.apiKey())
            // Sentiment and other structured outputs should be reproducible.
            // Providers that support temperature receive the deterministic floor.
            .temperature(0.0);
        if (config.timeout() != null) {
            optionsBuilder.timeout(config.timeout());
        }
        if (config.maxRetries() != null) {
            optionsBuilder.maxRetries(config.maxRetries());
        }

        OpenAiChatModel.Builder builder = OpenAiChatModel.builder()
            .options(optionsBuilder.build());
        if (!config.extraHeaders().isEmpty()) {
            builder.httpClientBuilderCustomizer(
                httpClientBuilder -> httpClientBuilder.interceptor(new ExtraHeadersInterceptor(config.extraHeaders())));
        }
        return builder.build();
    }

    /** Adds the configured extra headers to every request the model makes. */
    private record ExtraHeadersInterceptor(Map<String, String> headers) implements Interceptor {
        @Override
        public Response intercept(Chain chain) throws IOException {
            Request.Builder requestBuilder = chain.request().newBuilder();
            headers.forEach(requestBuilder::addHeader);
            return chain.proceed(requestBuilder.build());
        }
    }
}
