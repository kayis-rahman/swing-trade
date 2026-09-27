package com.swingtrade.llm.client;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;
import reactor.core.scheduler.Schedulers;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

/**
 * Uses the locally authenticated Pi CLI as an LLM backend.
 *
 * Pi owns the provider authentication (including openai-codex OAuth); Swing
 * Trade never reads or stores the OAuth token. The CLI is run without tools,
 * sessions, extensions, or project writes so this backend is inference-only.
 */
@Component
public class PiAgentLlmClient implements LlmClient {

    private static final Logger logger = LoggerFactory.getLogger(PiAgentLlmClient.class);
    private static final int MAX_OUTPUT_CHARS = 1_000_000;

    private final String executable;
    private volatile AgentConfig config;
    private final Duration timeout;

    @Autowired
    public PiAgentLlmClient(
            @Value("${llm.pi-agent.executable:pi}") String executable,
            @Value("${llm.pi-agent.provider:openai-codex}") String provider,
            @Value("${llm.pi-agent.model:gpt-5.6-luna}") String model,
            @Value("${llm.pi-agent.timeout:180s}") Duration timeout) {
        this(executable, provider, model, timeout, true);
    }

    /** Constructor for deterministic unit tests that do not use Spring values. */
    public PiAgentLlmClient(String executable, String provider, String model,
                            Duration timeout, boolean trimValues) {
        this.executable = trimValues ? executable.trim() : executable;
        this.config = new AgentConfig(
                trimValues ? provider.trim() : provider,
                trimValues ? model.trim() : model);
        this.timeout = timeout;
    }

    @Override
    public Mono<String> generateChatCompletion(List<Map<String, String>> messages,
                                                int maxTokens,
                                                double temperature) {
        return Mono.fromCallable(() -> runPi(messages))
                .subscribeOn(Schedulers.boundedElastic());
    }

    public void configure(String provider, String model) {
        this.config = new AgentConfig(provider.trim(), model.trim());
    }

    private String runPi(List<Map<String, String>> messages) throws IOException, InterruptedException {
        AgentConfig activeConfig = config;
        List<String> command = new ArrayList<>(List.of(
                executable,
                "--provider", activeConfig.provider(),
                "--model", activeConfig.model(),
                "--print",
                "--no-tools",
                "--no-session",
                "--no-extensions",
                "--no-skills",
                "--no-prompt-templates",
                "--no-themes",
                "--no-approve",
                buildPrompt(messages)));

        Process process = new ProcessBuilder(command)
                .redirectErrorStream(false)
                .start();
        // Pi accepts the prompt as a command-line argument. Closing stdin is
        // important when Spring launches it without a terminal: otherwise Pi
        // can keep waiting for additional interactive input after the prompt.
        process.getOutputStream().close();
        boolean completed = process.waitFor(timeout.toMillis(), TimeUnit.MILLISECONDS);
        if (!completed) {
            process.destroyForcibly();
            throw new IllegalStateException("Pi agent timed out after " + timeout);
        }

        String stdout = new String(process.getInputStream().readAllBytes(), StandardCharsets.UTF_8).trim();
        String stderr = new String(process.getErrorStream().readAllBytes(), StandardCharsets.UTF_8).trim();
        if (process.exitValue() != 0) {
            logger.warn("Pi agent exited with code {}: {}", process.exitValue(), truncate(stderr));
            throw new IllegalStateException("Pi agent failed: " + (stderr.isBlank() ? "unknown error" : stderr));
        }
        if (stdout.isBlank()) {
            throw new IllegalStateException("Pi agent returned an empty response");
        }
        return truncate(stdout);
    }

    private String buildPrompt(List<Map<String, String>> messages) {
        StringBuilder prompt = new StringBuilder();
        for (Map<String, String> message : messages) {
            String role = message.getOrDefault("role", "user").toUpperCase();
            String content = message.getOrDefault("content", "");
            prompt.append('[').append(role).append("]\n").append(content).append("\n\n");
        }
        return prompt.toString().trim();
    }

    private String truncate(String value) {
        if (value.length() <= MAX_OUTPUT_CHARS) return value;
        return value.substring(0, MAX_OUTPUT_CHARS);
    }

    private record AgentConfig(String provider, String model) { }
}
