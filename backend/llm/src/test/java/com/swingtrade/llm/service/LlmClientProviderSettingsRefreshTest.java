package com.swingtrade.llm.service;

import com.swingtrade.domain.store.AppSettingsStore;
import com.swingtrade.llm.client.LlamaCppClient;
import com.swingtrade.llm.config.LlmProperties;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;
import org.springframework.ai.openai.OpenAiChatModel;
import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.MockWebServer;
import okhttp3.mockwebserver.RecordedRequest;

import java.io.IOException;
import java.net.URI;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.TimeUnit;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Regression test for the stage-orchestrator degradation (run 5721aa09): settings
 * written through {@code PUT /api/settings/llm} after container startup never
 * reached the running LLM client, so the orchestrator's LLM_ANALYSIS and SENTIMENT
 * stages ran against the startup-time configuration (api.openai.com with the
 * placeholder key "not-needed") and degraded with a 401.
 *
 * <p>These tests reproduce the original condition — a settings change with no
 * restart — against real HTTP servers. They fail if the client keeps serving the
 * old configuration: before the fix, the second call below hit the first server
 * with the first key again, because the model bean was built once at startup.
 */
class LlmClientProviderSettingsRefreshTest {

    private MockWebServer server1;
    private MockWebServer server2;
    private InMemorySettingsStore store;
    private SettingsAwareChatModels models;
    private LlmClientProvider provider;

    @BeforeEach
    void setUp() throws IOException {
        server1 = new MockWebServer();
        server2 = new MockWebServer();
        server1.start();
        server2.start();
        server1.enqueue(completionResponse("model-1"));
        server2.enqueue(completionResponse("model-2"));

        store = new InMemorySettingsStore();
        store.set("llm.backend", "openai");
        store.set("openai.base_url", server1.url("/v1").toString());
        store.set("openai.model", "model-1");
        store.set("openai.api_key", "key-1");

        LlmProperties properties = new LlmProperties();
        properties.getProviders().getOpenai().setBaseUrl(URI.create("https://api.openai.com/v1"));
        properties.getProviders().getOpenai().setModel("gpt-4o");

        models = new SettingsAwareChatModels(store, properties, "none");
        LlmBackendSelector selector = new LlmBackendSelector(store, "local");
        provider = new LlmClientProvider(selector, Mockito.mock(LlamaCppClient.class), models);
    }

    @AfterEach
    void tearDown() throws IOException {
        server1.shutdown();
        server2.shutdown();
    }

    @Test
    @DisplayName("a settings change is honoured on the next call without a restart")
    void settingsChangeIsHonouredOnNextCallWithoutRestart() throws Exception {
        // First call uses the initial configuration.
        String first = provider.getClient()
            .generateChatCompletion(testMessages(), 16, 0.0)
            .block();
        assertThat(first).isEqualTo("hello");
        RecordedRequest request1 = server1.takeRequest(10, TimeUnit.SECONDS);
        assertThat(request1).isNotNull();
        assertThat(request1.getHeader("Authorization")).isEqualTo("Bearer key-1");
        assertThat(request1.getPath()).isEqualTo("/v1/chat/completions");

        // Settings write through PUT /api/settings/llm — no restart, same provider.
        store.set("openai.base_url", server2.url("/v1").toString());
        store.set("openai.model", "model-2");
        store.set("openai.api_key", "key-2");

        // The next call must use the NEW configuration. Before the fix this hit
        // server1 with key-1 again (the startup-built model bean) and the
        // orchestrator degraded against the stale configuration.
        String second = provider.getClient()
            .generateChatCompletion(testMessages(), 16, 0.0)
            .block();
        assertThat(second).isEqualTo("hello");
        RecordedRequest request2 = server2.takeRequest(10, TimeUnit.SECONDS);
        assertThat(request2).isNotNull();
        assertThat(request2.getHeader("Authorization")).isEqualTo("Bearer key-2");
        assertThat(server1.getRequestCount()).isEqualTo(1);
    }

    @Test
    @DisplayName("the OpenCode endpoint gets a default session header without extra-header configuration")
    void opencodeSessionHeaderDefaultsForOpenCodeEndpoint() {
        store.set("openai.base_url", "https://opencode.ai/zen/go/v1");

        var config = models.resolveConfig(LlmBackendSelector.Backend.OPENAI);

        assertThat(config.extraHeaders()).containsEntry("x-opencode-session", "swing-trade-orchestrator");
    }

    @Test
    @DisplayName("configured extra headers are sent with every request")
    void configuredExtraHeadersAreSentWithEveryRequest() throws Exception {
        store.set("llm.extra_headers", "{\"x-opencode-session\":\"session-abc\",\"x-custom\":\"value\"}");

        provider.getClient()
            .generateChatCompletion(testMessages(), 16, 0.0)
            .block();

        RecordedRequest request = server1.takeRequest(10, TimeUnit.SECONDS);
        assertThat(request).isNotNull();
        assertThat(request.getHeader("x-opencode-session")).isEqualTo("session-abc");
        assertThat(request.getHeader("x-custom")).isEqualTo("value");
    }

    @Test
    @DisplayName("repeated resolution without a settings change returns the cached model")
    void repeatedResolutionWithoutSettingsChangeReturnsCachedModel() {
        OpenAiChatModel first = models.resolve(LlmBackendSelector.Backend.OPENAI);
        OpenAiChatModel second = models.resolve(LlmBackendSelector.Backend.OPENAI);
        assertThat(second).isSameAs(first);

        store.set("openai.api_key", "key-2");
        OpenAiChatModel third = models.resolve(LlmBackendSelector.Backend.OPENAI);
        assertThat(third).isNotSameAs(first);
    }

    private static List<Map<String, String>> testMessages() {
        return List.of(
            Map.of("role", "system", "content", "You are a test."),
            Map.of("role", "user", "content", "hi"));
    }

    private static MockResponse completionResponse(String model) {
        return new MockResponse()
            .setResponseCode(200)
            .addHeader("Content-Type", "application/json")
            .setBody(completionJson(model));
    }

    private static String completionJson(String model) {
        return "{\"id\":\"chatcmpl-1\",\"object\":\"chat.completion\",\"created\":1700000000,"
            + "\"model\":\"" + model + "\","
            + "\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"hello\"},"
            + "\"finish_reason\":\"stop\"}],"
            + "\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":1,\"total_tokens\":6}}";
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
