package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.llm.config.LlmProperties;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;

import java.net.URI;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Unit tests for {@link SettingsAwareChatModels} configuration resolution:
 * app_settings over LlmProperties defaults, the placeholder-key rule, the
 * gpuhub.api_key fallback, and the extra-headers setting.
 */
class SettingsAwareChatModelsTest {

    private InMemorySettingsStore store;
    private LlmProperties properties;
    private SettingsAwareChatModels models;

    @BeforeEach
    void setUp() {
        store = new InMemorySettingsStore();
        properties = new LlmProperties();
        properties.getProviders().getOpenai().setBaseUrl(URI.create("https://api.openai.com/v1"));
        properties.getProviders().getOpenai().setModel("gpt-4o");
        properties.getProviders().getOllama().setBaseUrl(URI.create("http://localhost:11434/v1"));
        properties.getProviders().getOllama().setModel("qwen3.5:4b");
        models = new SettingsAwareChatModels(store, properties, "none");
    }

    @Nested
    @DisplayName("resolution — defaults and overrides")
    class Resolution {

        @Test
        @DisplayName("falls back to LlmProperties defaults when nothing is set")
        void fallsBackToDefaults() {
            SettingsAwareChatModels.ResolvedModelConfig config =
                models.resolveConfig(LlmBackendSelector.Backend.OPENAI);

            assertThat(config.baseUrl()).isEqualTo("https://api.openai.com/v1");
            assertThat(config.model()).isEqualTo("gpt-4o");
            assertThat(config.apiKey()).isEmpty();
        }

        @Test
        @DisplayName("app_settings override the defaults")
        void settingsOverrideDefaults() {
            store.set("openai.base_url", "https://opencode.ai/zen/go/v1");
            store.set("openai.model", "space-bunny-free");
            store.set("openai.api_key", "real-key");

            SettingsAwareChatModels.ResolvedModelConfig config =
                models.resolveConfig(LlmBackendSelector.Backend.OPENAI);

            assertThat(config.baseUrl()).isEqualTo("https://opencode.ai/zen/go/v1");
            assertThat(config.model()).isEqualTo("space-bunny-free");
            assertThat(config.apiKey()).isEqualTo("real-key");
        }

        @Test
        @DisplayName("the not-needed property placeholder resolves to an empty key, not a credential")
        void notNeededPlaceholderResolvesToEmpty() {
            store.set("openai.api_key", "not-needed");

            assertThat(models.resolveConfig(LlmBackendSelector.Backend.OPENAI).apiKey()).isEmpty();
        }

        @Test
        @DisplayName("gpuhub.api_key is the openai key fallback")
        void gpuhubKeyIsTheOpenAiFallback() {
            store.set("gpuhub.api_key", "gpuhub-key");

            assertThat(models.resolveConfig(LlmBackendSelector.Backend.OPENAI).apiKey())
                .isEqualTo("gpuhub-key");
        }

        @Test
        @DisplayName("an explicit openai key wins over the gpuhub fallback")
        void explicitOpenAiKeyWinsOverGpuhub() {
            store.set("openai.api_key", "openai-key");
            store.set("gpuhub.api_key", "gpuhub-key");

            assertThat(models.resolveConfig(LlmBackendSelector.Backend.OPENAI).apiKey())
                .isEqualTo("openai-key");
        }
    }

    @Nested
    @DisplayName("extra headers")
    class ExtraHeaders {

        @Test
        @DisplayName("parses the llm.extra_headers JSON object")
        void parsesExtraHeaders() {
            store.set("llm.extra_headers", "{\"x-opencode-session\":\"session-1\",\"x-other\":\"v\"}");

            assertThat(models.resolveConfig(LlmBackendSelector.Backend.OPENAI).extraHeaders())
                .containsEntry("x-opencode-session", "session-1")
                .containsEntry("x-other", "v");
        }

        @Test
        @DisplayName("a blank or malformed headers setting yields no headers")
        void blankOrMalformedYieldsNoHeaders() {
            assertThat(models.resolveConfig(LlmBackendSelector.Backend.OPENAI).extraHeaders()).isEmpty();

            store.set("llm.extra_headers", "not json");
            assertThat(models.resolveConfig(LlmBackendSelector.Backend.OPENAI).extraHeaders()).isEmpty();
        }
    }

    @Nested
    @DisplayName("caching")
    class Caching {

        @Test
        @DisplayName("the same configuration resolves to the same model instance")
        void sameConfigSameInstance() {
            var first = models.resolve(LlmBackendSelector.Backend.OPENAI);
            var second = models.resolve(LlmBackendSelector.Backend.OPENAI);

            assertThat(second).isSameAs(first);
        }

        @Test
        @DisplayName("a changed configuration rebuilds the model")
        void changedConfigRebuilds() {
            var first = models.resolve(LlmBackendSelector.Backend.OPENAI);

            store.set("openai.base_url", "https://opencode.ai/zen/go/v1");
            var second = models.resolve(LlmBackendSelector.Backend.OPENAI);

            assertThat(second).isNotSameAs(first);
            assertThat(second.getOptions().getBaseUrl()).isEqualTo("https://opencode.ai/zen/go/v1");
        }

        @Test
        @DisplayName("each backend caches independently")
        void backendsCacheIndependently() {
            var openai = models.resolve(LlmBackendSelector.Backend.OPENAI);
            var ollama = models.resolve(LlmBackendSelector.Backend.OLLAMA);

            assertThat(openai).isNotSameAs(ollama);
            assertThat(models.resolve(LlmBackendSelector.Backend.OPENAI)).isSameAs(openai);
            assertThat(models.resolve(LlmBackendSelector.Backend.OLLAMA)).isSameAs(ollama);
        }
    }

    private static final class InMemorySettingsStore implements AppSettingsStore {
        private final Map<String, String> values = new HashMap<>();

        @Override
        public Optional<String> get(String key) {
            return Optional.ofNullable(values.get(key));
        }

        @Override
        public void set(String key, String value) {
            values.put(key, value);
        }
    }
}
