package com.swingtrade.llm.client;

import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;

class PiAgentLlmClientTest {

    @Test
    void usesUpdatedProviderAndModelForTheNextRequest() {
        PiAgentLlmClient client = new PiAgentLlmClient(
            "/bin/echo", "old-provider", "old-model", Duration.ofSeconds(5), false);
        client.configure("openai-codex", "gpt-5.6-luna");

        String response = client.generateChatCompletion(
            List.of(Map.of("role", "user", "content", "test")), 16, 0.0).block();

        assertThat(response).contains("--provider openai-codex --model gpt-5.6-luna");
    }
}
