package com.swingtrade.llm.service;

import com.swingtrade.llm.client.LlmClient;
import com.swingtrade.llm.client.LlamaCppClient;
import com.swingtrade.llm.client.SpringAiLlmClient;
import com.swingtrade.llm.client.PiAgentLlmClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.ai.chat.client.ChatClient;
import org.springframework.ai.openai.OpenAiChatModel;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.stereotype.Component;

/**
 * Provider that routes LLM requests to the correct backend.
 * Creates a SpringAiLlmClient backed by the OpenAiChatModel
 * selected by LlmBackendSelector at runtime.
 *
 * <p>The model itself is resolved from the live app_settings by
 * {@link SettingsAwareChatModels}, so a settings write through
 * {@code PUT /api/settings/llm} takes effect on the next call without a
 * container restart — the model is rebuilt when the resolved configuration
 * changes. (The startup-built model beans in LlmConfig cannot do this: the
 * OpenAI client inside OpenAiChatModel is bound to its options at construction.)
 */
@Component
public class LlmClientProvider {

    private static final Logger logger = LoggerFactory.getLogger(LlmClientProvider.class);

    private final LlmBackendSelector selector;
    private final LlamaCppClient llamaCppClient;
    private final SettingsAwareChatModels settingsAwareChatModels;
    private final PiAgentLlmClient piAgentClient;
    private final String ollamaReasoningEffort;

    public LlmClientProvider(LlmBackendSelector selector,
                             LlamaCppClient llamaCppClient,
                             SettingsAwareChatModels settingsAwareChatModels) {
        this(selector, llamaCppClient, settingsAwareChatModels,
                new PiAgentLlmClient("pi", "openai-codex", "gpt-5.6-luna", java.time.Duration.ofSeconds(180), false), null);
    }

    /**
     * @param ollamaReasoningEffort {@code reasoning_effort} sent only to the Ollama backend.
     *        Measured against qwen3.5:4b: with default thinking the answer can land in
     *        {@code reasoning} with empty {@code content}; {@code none} disables thinking
     *        (the {@code think:false} flag is ignored on the OpenAI-compatible endpoint).
     */
    @org.springframework.beans.factory.annotation.Autowired
    public LlmClientProvider(LlmBackendSelector selector,
                             LlamaCppClient llamaCppClient,
                             SettingsAwareChatModels settingsAwareChatModels,
                             PiAgentLlmClient piAgentClient,
                             @org.springframework.beans.factory.annotation.Value("${llm.providers.ollama.reasoning-effort:none}") String ollamaReasoningEffort) {
        this.ollamaReasoningEffort = ollamaReasoningEffort;
        this.selector = selector;
        this.llamaCppClient = llamaCppClient;
        this.settingsAwareChatModels = settingsAwareChatModels;
        this.piAgentClient = piAgentClient;
    }

    /**
     * Returns an LlmClient backed by the OpenAiChatModel
     * selected by the backend selector.
     */
    public LlmClient getClient() {
        LlmBackendSelector.Backend backend = selector.resolve();
        if (backend == LlmBackendSelector.Backend.PI_SSH) {
            logger.info("Using native llama.cpp HTTP client for PI_SSH backend");
            return llamaCppClient;
        }
        if (backend == LlmBackendSelector.Backend.PI_AGENT) {
            logger.info("Using Pi CLI agent provider");
            return piAgentClient;
        }
        OpenAiChatModel model = settingsAwareChatModels.resolve(backend);
        if (model.getOptions() != null) {
            logger.info("Selected LLM backend {} with model {} at {}", backend,
                    model.getOptions().getModel(), model.getOptions().getBaseUrl());
        }
        ChatClient chatClient = ChatClient.create(model);
        String effort = backend == LlmBackendSelector.Backend.OLLAMA
                ? ollamaReasoningEffort : null;
        return new SpringAiLlmClient(chatClient, false, effort);
    }

    public LlmBackendSelector.Backend getBackend() {
        return selector.resolve();
    }
}
